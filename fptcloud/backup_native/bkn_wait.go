package fptcloud_backup_native

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// pollInterval is the wait between polls. Tests shorten it.
var pollInterval = 5 * time.Second

// historyClockSkew widens the "after submit" test on history rows, because
// the backend clock and this machine's clock are not the same clock.
const historyClockSkew = 2 * time.Minute

// pollTimeout trims the CRUD timeout before handing it to StateChangeConf, so
// the poll reports its own timeout (naming the object and its last state)
// before the SDK cancels the context. Same rule as backup_veeam.
func pollTimeout(crudTimeout time.Duration) time.Duration {
	const margin = 30 * time.Second
	if crudTimeout <= 2*margin {
		return crudTimeout / 2
	}
	return crudTimeout - margin
}

func newStateConf(pending, target []string, refresh retry.StateRefreshFunc, timeout time.Duration) *retry.StateChangeConf {
	return &retry.StateChangeConf{
		Pending:                   pending,
		Target:                    target,
		Refresh:                   refresh,
		Timeout:                   pollTimeout(timeout),
		Delay:                     pollInterval,
		MinTimeout:                pollInterval,
		PollInterval:              pollInterval,
		ContinuousTargetOccurence: 1,
	}
}

// waitForTask polls a v1 Celery task and returns its final message.
// status_code stays 200 until the task has a result (blueprint.py:644-656),
// so both state and status_code decide.
func waitForTask(ctx context.Context, svc VmService, vpcId, taskId, action string, timeout time.Duration) (string, error) {
	conf := newStateConf([]string{"running"}, []string{"done"}, func() (interface{}, string, error) {
		progress, err := svc.GetTaskProgress(vpcId, taskId)
		if err != nil {
			return nil, "", err
		}
		switch strings.ToUpper(progress.State) {
		case "SUCCESS":
			if progress.StatusCode >= 400 {
				return nil, "", fmt.Errorf("%s failed (status %d): %s", action, progress.StatusCode, describe(progress.StatusCode, progress.Message))
			}
			return progress, "done", nil
		case "FAILURE", "REVOKED":
			return nil, "", fmt.Errorf("%s failed: the task ended in state %s: %s", action, progress.State, progress.Message)
		default:
			return progress, "running", nil
		}
	}, timeout)
	raw, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return "", fmt.Errorf("waiting for %s (task %s): %w", action, taskId, err)
	}
	return raw.(*TaskProgress).Message, nil
}

// Statuses in which a v1 job accepts update and switch_schedule
// (SwitchScheduleController.py:50). Anything else, including a percentage
// while a backup runs, means busy.
var vmJobStableStatuses = map[string]bool{"initiated": true, "available": true, "success": true, "error": true}

func waitVmJobSettled(ctx context.Context, svc VmService, vpcId, jobId string, timeout time.Duration) (*VmJobDetail, error) {
	conf := newStateConf([]string{"busy"}, []string{"settled"}, func() (interface{}, string, error) {
		job, err := svc.GetVmJob(vpcId, jobId)
		if err != nil {
			return nil, "", err
		}
		if job == nil {
			return nil, "", fmt.Errorf("VM backup job %s no longer exists", jobId)
		}
		if vmJobStableStatuses[strings.ToLower(job.Status)] {
			return job, "settled", nil
		}
		return job, "busy", nil
	}, timeout)
	raw, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for VM backup job %s to finish its running operation: %w", jobId, err)
	}
	return raw.(*VmJobDetail), nil
}

func waitVmJobGone(ctx context.Context, svc VmService, vpcId, jobId string, timeout time.Duration) error {
	conf := newStateConf([]string{"deleting"}, []string{"gone"}, func() (interface{}, string, error) {
		job, err := svc.GetVmJob(vpcId, jobId)
		if err != nil {
			return nil, "", err
		}
		if job == nil {
			return struct{}{}, "gone", nil
		}
		switch strings.ToLower(job.Status) {
		case "error", "error_deleting":
			return nil, "", fmt.Errorf("deleting VM backup job %s failed: the job is in state %s", jobId, job.Status)
		}
		return job, "deleting", nil
	}, timeout)
	if _, err := conf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for VM backup job %s to be deleted: %w", jobId, err)
	}
	return nil
}

// Statuses in which a v2 job accepts update, actions and executions.
var volumeJobStableStatuses = map[string]bool{"initiated": true, "available": true, "success": true, "error": true}

func waitVolumeJobSettled(ctx context.Context, svc VolumeService, vpcId, jobId string, timeout time.Duration) (*VolumeJobDetail, error) {
	conf := newStateConf([]string{"busy"}, []string{"settled"}, func() (interface{}, string, error) {
		job, err := svc.GetVolumeJob(vpcId, jobId)
		if err != nil {
			return nil, "", err
		}
		if job == nil {
			return nil, "", fmt.Errorf("volume backup job %s no longer exists", jobId)
		}
		if volumeJobStableStatuses[strings.ToLower(job.Status)] {
			return job, "settled", nil
		}
		return job, "busy", nil
	}, timeout)
	raw, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for volume backup job %s to finish its running operation: %w", jobId, err)
	}
	return raw.(*VolumeJobDetail), nil
}

func waitVolumeJobGone(ctx context.Context, svc VolumeService, vpcId, jobId string, timeout time.Duration) error {
	conf := newStateConf([]string{"deleting"}, []string{"gone"}, func() (interface{}, string, error) {
		job, err := svc.GetVolumeJob(vpcId, jobId)
		if err != nil {
			return nil, "", err
		}
		if job == nil {
			return struct{}{}, "gone", nil
		}
		switch strings.ToLower(job.Status) {
		case "error", "deleting error":
			return nil, "", fmt.Errorf("deleting volume backup job %s failed: the job is in state %s", jobId, job.Status)
		}
		return job, "deleting", nil
	}, timeout)
	if _, err := conf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for volume backup job %s to be deleted: %w", jobId, err)
	}
	return nil
}

// findVmRestoreRow returns the newest history row of a restore submitted at
// since, or nil. History rows carry no id, so a row matches on VM, type and
// a start time no earlier than since minus historyClockSkew.
func findVmRestoreRow(rows []VmHistory, vmId, historyType string, since time.Time) *VmHistory {
	cutoff := since.Add(-historyClockSkew)
	var best *VmHistory
	var bestStart time.Time
	for i := range rows {
		row := &rows[i]
		if row.VmId != vmId || row.Type != historyType {
			continue
		}
		start, err := parseAPITime(row.StartTime)
		if err != nil || start.Before(cutoff) {
			continue
		}
		if best == nil || start.After(bestStart) {
			best, bestStart = row, start
		}
	}
	return best
}

func waitVmRestore(ctx context.Context, svc VmService, vpcId, vmId, historyType string, since time.Time, timeout time.Duration) (*VmHistory, error) {
	conf := newStateConf([]string{"waiting"}, []string{"done"}, func() (interface{}, string, error) {
		rows, err := svc.ListVmHistories(vpcId, historyType)
		if err != nil {
			return nil, "", err
		}
		row := findVmRestoreRow(rows, vmId, historyType, since)
		if row == nil {
			return struct{}{}, "waiting", nil
		}
		switch row.Status {
		case "Success":
			return row, "done", nil
		case "Failed":
			return nil, "", fmt.Errorf("restoring VM %s failed: %s", vmId, row.Reason)
		}
		return row, "waiting", nil
	}, timeout)
	raw, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for the restore of VM %s: %w", vmId, err)
	}
	return raw.(*VmHistory), nil
}

// findVolumeRestoreRow reads combined history newest first until it finds the
// "Restore keep" row of this volume, or reaches rows older than the cutoff.
// The endpoint ignores every filter, so matching happens here.
func findVolumeRestoreRow(svc VolumeService, vpcId, volumeId string, since time.Time) (*VolumeHistoryItem, error) {
	cutoff := since.Add(-historyClockSkew)
	for page := 1; page <= maxListPages; page++ {
		result, err := svc.ListVolumeHistoryPage(vpcId, page, historyPageSize)
		if err != nil {
			return nil, err
		}
		for i := range result.Items {
			item := &result.Items[i]
			start, err := parseAPITime(item.StartTime)
			if err != nil {
				continue
			}
			if start.Before(cutoff) {
				return nil, nil
			}
			if item.SourceType == "volume" && item.ObjectId == volumeId && item.Type == "Restore keep" {
				return item, nil
			}
		}
		if !result.HasMore {
			return nil, nil
		}
	}
	return nil, nil
}

func waitVolumeRestore(ctx context.Context, svc VolumeService, vpcId, volumeId string, since time.Time, timeout time.Duration) (*VolumeHistoryItem, error) {
	conf := newStateConf([]string{"waiting"}, []string{"done"}, func() (interface{}, string, error) {
		row, err := findVolumeRestoreRow(svc, vpcId, volumeId, since)
		if err != nil {
			return nil, "", err
		}
		if row == nil {
			return struct{}{}, "waiting", nil
		}
		switch strings.ToLower(row.Status) {
		case "success":
			return row, "done", nil
		case "failed":
			reason := fmt.Sprint(row.HistoryMetadata.ExtractInfo["status"])
			return nil, "", fmt.Errorf("restoring volume %s failed: %s", volumeId, reason)
		}
		return row, "waiting", nil
	}, timeout)
	raw, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for the restore of volume %s: %w", volumeId, err)
	}
	return raw.(*VolumeHistoryItem), nil
}
