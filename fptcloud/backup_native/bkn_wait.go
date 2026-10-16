package fptcloud_backup_native

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// pollInterval is the wait between polls. Tests shorten it.
var pollInterval = 5 * time.Second

// maxPollErrors is how many failed API reads in a row a wait rides out. The
// operation being waited on is already running, so a passing error (a dev
// gateway answered one poll with 401) must not abandon it.
const maxPollErrors = 5

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

// pollReadError marks a failed API read inside a refresh function, as opposed
// to an outcome such as "the restore failed", which ends the wait at once.
type pollReadError struct{ err error }

func (e pollReadError) Error() string { return e.err.Error() }
func (e pollReadError) Unwrap() error { return e.err }

func readFailed(err error) error { return pollReadError{err} }

// tolerateReadErrors keeps a wait in its pending state through up to
// maxPollErrors failed reads in a row; one more ends it with the last error.
func tolerateReadErrors(refresh retry.StateRefreshFunc, pending string) retry.StateRefreshFunc {
	failures := 0
	return func() (interface{}, string, error) {
		result, state, err := refresh()
		var readErr pollReadError
		if !errors.As(err, &readErr) {
			failures = 0
			return result, state, err
		}
		failures++
		if failures > maxPollErrors {
			return nil, "", fmt.Errorf("%w (failed %d reads in a row)", readErr.err, failures)
		}
		return struct{}{}, pending, nil
	}
}

func newStateConf(pending, target []string, refresh retry.StateRefreshFunc, timeout time.Duration) *retry.StateChangeConf {
	return &retry.StateChangeConf{
		Pending:                   pending,
		Target:                    target,
		Refresh:                   tolerateReadErrors(refresh, pending[0]),
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
			return nil, "", readFailed(err)
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
			return nil, "", readFailed(err)
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

// deleteFailed reports whether a job status during a delete wait means the
// delete failed. A job whose last backup failed already reads "error" before
// the delete starts, so "error" counts only after another status was seen.
func deleteFailed(status, failedStatus string, sawOther *bool) bool {
	status = strings.ToLower(status)
	if status == failedStatus {
		return true
	}
	if status == "error" {
		return *sawOther
	}
	*sawOther = true
	return false
}

func waitVmJobGone(ctx context.Context, svc VmService, vpcId, jobId string, timeout time.Duration) error {
	sawOther := false
	conf := newStateConf([]string{"deleting"}, []string{"gone"}, func() (interface{}, string, error) {
		job, err := svc.GetVmJob(vpcId, jobId)
		if err != nil {
			return nil, "", readFailed(err)
		}
		if job == nil {
			return struct{}{}, "gone", nil
		}
		if deleteFailed(job.Status, "error_deleting", &sawOther) {
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

func waitVolumeJobSettled(ctx context.Context, svc VolumeService, vpcId, jobId string, timeout time.Duration) error {
	conf := newStateConf([]string{"busy"}, []string{"settled"}, func() (interface{}, string, error) {
		job, err := svc.GetVolumeJob(vpcId, jobId)
		if err != nil {
			return nil, "", readFailed(err)
		}
		if job == nil {
			return nil, "", fmt.Errorf("volume backup job %s no longer exists", jobId)
		}
		if volumeJobStableStatuses[strings.ToLower(job.Status)] {
			return job, "settled", nil
		}
		return job, "busy", nil
	}, timeout)
	if _, err := conf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for volume backup job %s to finish its running operation: %w", jobId, err)
	}
	return nil
}

func waitVolumeJobGone(ctx context.Context, svc VolumeService, vpcId, jobId string, timeout time.Duration) error {
	sawOther := false
	conf := newStateConf([]string{"deleting"}, []string{"gone"}, func() (interface{}, string, error) {
		job, err := svc.GetVolumeJob(vpcId, jobId)
		if err != nil {
			return nil, "", readFailed(err)
		}
		if job == nil {
			return struct{}{}, "gone", nil
		}
		if deleteFailed(job.Status, "deleting error", &sawOther) {
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
// a start time no earlier than since minus historyClockSkew. seen holds the
// start times of matching rows read before the restore was submitted; they
// belong to earlier restores.
func findVmRestoreRow(rows []VmHistory, vmId, historyType string, since time.Time, seen map[string]bool) *VmHistory {
	cutoff := since.Add(-historyClockSkew)
	var best *VmHistory
	var bestStart time.Time
	for i := range rows {
		row := &rows[i]
		if row.VmId != vmId || row.Type != historyType || seen[row.StartTime] {
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

// vmRestoreRowsSeen returns the start times of the rows findVmRestoreRow
// would match before a restore is submitted.
func vmRestoreRowsSeen(svc VmService, vpcId, vmId, historyType string, since time.Time) (map[string]bool, error) {
	rows, err := svc.ListVmHistories(vpcId, historyType)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for {
		row := findVmRestoreRow(rows, vmId, historyType, since, seen)
		if row == nil {
			return seen, nil
		}
		seen[row.StartTime] = true
	}
}

func waitVmRestore(ctx context.Context, svc VmService, vpcId, vmId, historyType string, since time.Time, seen map[string]bool, timeout time.Duration) (*VmHistory, error) {
	conf := newStateConf([]string{"waiting"}, []string{"done"}, func() (interface{}, string, error) {
		rows, err := svc.ListVmHistories(vpcId, historyType)
		if err != nil {
			return nil, "", readFailed(err)
		}
		row := findVmRestoreRow(rows, vmId, historyType, since, seen)
		if row == nil {
			return struct{}{}, "waiting", nil
		}
		switch strings.ToLower(row.Status) {
		case "success":
			return row, "done", nil
		case "failed":
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

// findVolumeRestoreRow returns the newest "Restore keep" row of this volume
// that is not in seen, or nil. seen works as in findVmRestoreRow.
func findVolumeRestoreRow(svc VolumeService, vpcId, volumeId string, since time.Time, seen map[string]bool) (*VolumeHistoryItem, error) {
	var found *VolumeHistoryItem
	err := scanVolumeRestoreRows(svc, vpcId, volumeId, since, func(item *VolumeHistoryItem) bool {
		if seen[item.StartTime] {
			return false
		}
		found = item
		return true
	})
	return found, err
}

// volumeRestoreRowsSeen returns the start times of the rows
// findVolumeRestoreRow would match before a restore is submitted.
func volumeRestoreRowsSeen(svc VolumeService, vpcId, volumeId string, since time.Time) (map[string]bool, error) {
	seen := map[string]bool{}
	err := scanVolumeRestoreRows(svc, vpcId, volumeId, since, func(item *VolumeHistoryItem) bool {
		seen[item.StartTime] = true
		return false
	})
	return seen, err
}

// scanVolumeRestoreRows reads combined history newest first and calls visit
// for each "Restore keep" row of this volume, until visit returns true or the
// rows get older than the cutoff. The endpoint ignores every filter, so
// matching happens here.
func scanVolumeRestoreRows(svc VolumeService, vpcId, volumeId string, since time.Time, visit func(*VolumeHistoryItem) bool) error {
	cutoff := since.Add(-historyClockSkew)
	for page := 1; page <= maxListPages; page++ {
		result, err := svc.ListVolumeHistoryPage(vpcId, page, historyPageSize)
		if err != nil {
			return err
		}
		for i := range result.Items {
			item := &result.Items[i]
			start, err := parseAPITime(item.StartTime)
			if err != nil {
				continue
			}
			if start.Before(cutoff) {
				return nil
			}
			if item.SourceType == "volume" && item.ObjectId == volumeId && item.Type == "Restore keep" && visit(item) {
				return nil
			}
		}
		if !result.HasMore {
			return nil
		}
	}
	return nil
}

func waitVolumeRestore(ctx context.Context, svc VolumeService, vpcId, volumeId string, since time.Time, seen map[string]bool, timeout time.Duration) (*VolumeHistoryItem, error) {
	conf := newStateConf([]string{"waiting"}, []string{"done"}, func() (interface{}, string, error) {
		row, err := findVolumeRestoreRow(svc, vpcId, volumeId, since, seen)
		if err != nil {
			return nil, "", readFailed(err)
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
