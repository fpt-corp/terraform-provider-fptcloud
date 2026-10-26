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

// retryRead calls read until it returns anything but a failed API read,
// riding out up to maxPollErrors of those pollInterval apart, as a poll does.
func retryRead(ctx context.Context, read func() error) error {
	for failures := 0; ; failures++ {
		err := read()
		var readErr pollReadError
		if !errors.As(err, &readErr) {
			return err
		}
		if failures >= maxPollErrors {
			return fmt.Errorf("%w (failed %d reads in a row)", readErr.err, failures+1)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
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

// Statuses in which a v1 job accepts update, switch_schedule and a manual
// backup (SwitchScheduleController.py:50). Anything else, including a
// percentage while a backup runs, means busy, except "failed": see
// settlePurpose.
var vmJobStableStatuses = map[string]bool{"initiated": true, "available": true, "success": true, "error": true}

// settlePurpose says what a settle wait is for. A v1 job reads "Failed" after
// a run in which any VM failed, and keeps it until another run starts. Update,
// switch_schedule and a manual backup refuse that state, so waiting for it to
// change would only run out the timeout; delete accepts it. An update waits
// only for another update to finish: v1 also accepts it while a backup or a
// restore runs (UpdateBackupJobController.py:79-100).
type settlePurpose int

const (
	settleToChange settlePurpose = iota
	settleToUpdate
	settleToDelete
)

// vmJobFailedError reports a v1 job in state Failed, which refuses every
// change and run (see settlePurpose).
func vmJobFailedError(jobId string) error {
	return fmt.Errorf("VM backup job %s is in state Failed because a VM failed in its last run. "+
		"The API refuses to change or start the job until another run starts, either a scheduled run or a backend retry; "+
		"check the backup history in the portal", jobId)
}

// jobBusyError reports a job that cannot start a manual backup now. A manual
// backup reads the status once instead of waiting: the run in progress may be
// a scheduled backup that outlasts the apply's timeout.
func jobBusyError(kind, jobId, status string) error {
	return fmt.Errorf("the %s backup job %s cannot start a backup now: it is in state %s, so a backup, a restore or a change of the job is still running. "+
		"Apply again after it ends", kind, jobId, status)
}

func waitVmJobSettled(ctx context.Context, svc VmService, vpcId, jobId string, purpose settlePurpose, timeout time.Duration) (*VmJobDetail, error) {
	conf := newStateConf([]string{"busy"}, []string{"settled"}, func() (interface{}, string, error) {
		job, err := svc.GetVmJob(vpcId, jobId)
		if err != nil {
			return nil, "", readFailed(err)
		}
		if job == nil {
			return nil, "", fmt.Errorf("VM backup job %s no longer exists", jobId)
		}
		status := strings.ToLower(job.Status)
		if status == "failed" {
			if purpose == settleToDelete {
				return job, "settled", nil
			}
			return nil, "", vmJobFailedError(jobId)
		}
		if purpose == settleToUpdate {
			if status == "updating" {
				return job, "busy", nil
			}
			return job, "settled", nil
		}
		if vmJobStableStatuses[status] {
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

// findVmHistoryRow returns the newest history row of an action submitted at
// since, or nil. History rows carry no id, so a row matches on VM, type and
// a start time no earlier than since minus historyClockSkew. seen holds the
// start times of matching rows read before the action was submitted; they
// belong to earlier actions.
func findVmHistoryRow(rows []VmHistory, vmId, historyType string, since time.Time, seen map[string]bool) *VmHistory {
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

// vmHistoryRowsSeen returns the start times of the rows findVmHistoryRow
// would match before an action is submitted.
func vmHistoryRowsSeen(svc VmService, vpcId, vmId, historyType string, since time.Time) (map[string]bool, error) {
	rows, err := svc.ListVmHistories(vpcId, historyType, since.Add(-historyClockSkew))
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for {
		row := findVmHistoryRow(rows, vmId, historyType, since, seen)
		if row == nil {
			return seen, nil
		}
		seen[row.StartTime] = true
	}
}

func waitVmRestore(ctx context.Context, svc VmService, vpcId, vmId, historyType string, since time.Time, seen map[string]bool, timeout time.Duration) (*VmHistory, error) {
	conf := newStateConf([]string{"waiting"}, []string{"done"}, func() (interface{}, string, error) {
		rows, err := svc.ListVmHistories(vpcId, historyType, since.Add(-historyClockSkew))
		if err != nil {
			return nil, "", readFailed(err)
		}
		row := findVmHistoryRow(rows, vmId, historyType, since, seen)
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
// that is not in seen, or nil. seen works as in findVmHistoryRow.
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

// runFailureHold is how long a failed job status must hold before a manual
// backup counts as failed. A job with one VM or volume is retried after a
// failure, and a v2 job with several volumes reads "error" while the others
// still back up, so a failed status can turn back into a running one. Tests
// shorten it.
var runFailureHold = 60 * time.Second

func waitVmJobRun(ctx context.Context, svc VmService, vpcId, jobId string, timeout time.Duration) (string, error) {
	return waitJobRun(ctx, "VM", jobId, func() (string, bool, error) {
		job, err := svc.GetVmJob(vpcId, jobId)
		if err != nil || job == nil {
			return "", false, err
		}
		return job.Status, true, nil
	}, map[string]bool{"failed": true, "error": true}, timeout)
}

func waitVolumeJobRun(ctx context.Context, svc VolumeService, vpcId, jobId string, timeout time.Duration) (string, error) {
	return waitJobRun(ctx, "volume", jobId, func() (string, bool, error) {
		job, err := svc.GetVolumeJob(vpcId, jobId)
		if err != nil || job == nil {
			return "", false, err
		}
		return job.Status, true, nil
	}, map[string]bool{"error": true}, timeout)
}

// waitJobRun polls a job whose manual backup was just started and returns the
// job's final status. A job runs one backup at a time and the start call
// writes Pending before it returns, so every status read here belongs to this
// run. read returns found=false when the job is gone.
func waitJobRun(ctx context.Context, kind, jobId string, read func() (status string, found bool, err error), failed map[string]bool, timeout time.Duration) (string, error) {
	var outcome error
	var failingSince time.Time
	lastStatus := "Pending"
	conf := newStateConf([]string{"running"}, []string{"done"}, func() (interface{}, string, error) {
		status, found, err := read()
		if err != nil {
			return nil, "", readFailed(err)
		}
		if found {
			lastStatus = status
		}
		if !found {
			outcome = fmt.Errorf("the %s backup job %s was deleted while its backup ran", kind, jobId)
			return nil, "", outcome
		}
		s := strings.ToLower(status)
		if s == "success" {
			return status, "done", nil
		}
		if !failed[s] {
			failingSince = time.Time{}
			return status, "running", nil
		}
		if failingSince.IsZero() {
			failingSince = time.Now()
		}
		if time.Since(failingSince) < runFailureHold {
			return status, "running", nil
		}
		outcome = fmt.Errorf("the backup of %s backup job %s failed: the job is in state %s. "+
			"A job with one VM or volume may still be retried by the backend; check the backup history in the portal", kind, jobId, status)
		return nil, "", outcome
	}, timeout)
	raw, err := conf.WaitForStateContext(ctx)
	if outcome != nil {
		return "", outcome
	}
	var timeoutErr *retry.TimeoutError
	if errors.As(err, &timeoutErr) {
		if strings.EqualFold(lastStatus, "pending") {
			return "", fmt.Errorf("waiting for the backup of %s backup job %s to finish: %w. "+
				"The job may be stuck in Pending; the API refuses a new run until the job settles", kind, jobId, err)
		}
		return "", fmt.Errorf("waiting for the backup of %s backup job %s to finish: %w. "+
			"The job was last in state %s: the backup goes on without Terraform. Raise timeouts.create to wait longer", kind, jobId, err, lastStatus)
	}
	if err != nil {
		return "", fmt.Errorf("waiting for the backup of %s backup job %s to finish: %w", kind, jobId, err)
	}
	return raw.(string), nil
}
