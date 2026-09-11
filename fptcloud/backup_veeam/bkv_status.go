package fptcloud_backup_veeam

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// Statuses that mean work is still in progress. DISABLING_SCHEDULE and
// ENABLING_SCHEDULE are here even though the provider never calls toggle: the
// job can be in either state because somebody clicked it in the portal while
// Terraform was running.
var pendingStatuses = []string{
	"CREATING", "UPDATING", "DELETING", "STARTING",
	"DISABLING_SCHEDULE", "ENABLING_SCHEDULE",
}

// CREATE_FAILED means the job never reached Veeam - either the create failed
// outright, or it lost the tie-break against another job competing for the
// same instance. ERROR means the job exists but is unhealthy. Both are
// failures as far as Terraform is concerned.
var failedStatuses = []string{"ERROR", "CREATE_FAILED", "FAILED"}

// Resting states that count as success. NOT_AVAILABLE is the NORMAL state of a
// job that has just been created and has not run yet.
var settledStatuses = []string{"NOT_AVAILABLE", "WORKING", "SUCCESS", "WARNING"}

func containsStatus(list []string, status string) bool {
	for _, s := range list {
		if s == status {
			return true
		}
	}
	return false
}

func IsPendingStatus(status string) bool { return containsStatus(pendingStatuses, status) }

func IsFailedStatus(status string) bool { return containsStatus(failedStatuses, status) }

// pollTimeout trims the CRUD timeout before handing it to StateChangeConf.
//
// The SDK cancels the context exactly at the CRUD timeout, so a StateChangeConf
// given the same duration races that cancellation: whichever fires first
// decides the message the user sees, and when the context wins they get a bare
// "context deadline exceeded" instead of one naming the job and its last
// status. HashiCorp's guidance is to keep the StateChangeConf timeout below the
// CRUD timeout for exactly this reason.
func pollTimeout(crudTimeout time.Duration) time.Duration {
	const margin = 30 * time.Second
	// A timeout too small to trim safely is halved instead, so the margin can
	// never turn into a zero or negative budget.
	if crudTimeout <= 2*margin {
		return crudTimeout / 2
	}
	return crudTimeout - margin
}

// WaitForJobSettled waits until the job leaves its pending state.
//
// It polls the LIST endpoint, not detail: detail answers 200 as soon as the
// database row is written - before the Celery task even runs - so polling
// detail would "succeed" instantly and miss CREATE_FAILED entirely.
func WaitForJobSettled(ctx context.Context, svc BackupVeeamService, vpcId string, jobId string, name string, timeout time.Duration) (*JobListItem, error) {
	stateConf := &retry.StateChangeConf{
		Pending: pendingStatuses,
		Target:  settledStatuses,
		Refresh: func() (interface{}, string, error) {
			item, err := svc.FindJobInList(vpcId, jobId, name)
			if err != nil {
				return nil, "", err
			}
			if item == nil {
				return nil, "", fmt.Errorf("backup job %s was not found in the VPC's job list", jobId)
			}
			if IsFailedStatus(item.Status) {
				return nil, "", failureMessage(item.Status, jobId)
			}
			return item, item.Status, nil
		},
		Timeout:                   pollTimeout(timeout),
		Delay:                     3 * time.Second,
		MinTimeout:                3 * time.Second,
		ContinuousTargetOccurence: 1,
	}

	raw, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return nil, err
	}
	item, _ := raw.(*JobListItem)
	return item, nil
}

func failureMessage(status string, jobId string) error {
	if status == "CREATE_FAILED" {
		return fmt.Errorf("backup job %s is in state CREATE_FAILED - the job was never created on Veeam. "+
			"When an apply creates several jobs at once this usually means two of them competed for the same instance; "+
			"run terraform apply again, or use -parallelism=1", jobId)
	}
	return fmt.Errorf("backup job %s is in state %s - the job exists but is unhealthy, check it in the portal", jobId, status)
}

// WaitForJobGone waits until detail stops returning the job. This is the ONLY
// place detail can be polled, because the signal here is "the job disappeared"
// rather than a status value.
func WaitForJobGone(ctx context.Context, svc BackupVeeamService, vpcId string, jobId string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending: []string{"EXISTS"},
		Target:  []string{"GONE"},
		Refresh: func() (interface{}, string, error) {
			detail, err := svc.GetJobDetail(vpcId, jobId)
			if err != nil {
				return nil, "", err
			}
			if detail == nil {
				return jobId, "GONE", nil
			}
			return detail, "EXISTS", nil
		},
		Timeout:    pollTimeout(timeout),
		Delay:      3 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForStateContext(ctx)
	return err
}

// WaitForVmIdsSettled waits until the job's instance list matches what was
// sent.
//
// Status settles before the backup_object rows do: measured against the dev
// backend, an update that removes an instance returns with status out of
// UPDATING while detail still reports the old list, and the new list appears
// about 15 seconds later. Reading straight after WaitForJobSettled therefore
// writes a stale vm_ids into state, and the next plan shows a diff on a job
// that is in fact correct.
func WaitForVmIdsSettled(ctx context.Context, svc BackupVeeamService, vpcId string, jobId string, want []string, timeout time.Duration) error {
	wanted := make(map[string]bool, len(want))
	for _, id := range want {
		wanted[id] = true
	}

	stateConf := &retry.StateChangeConf{
		Pending: []string{"stale"},
		Target:  []string{"match"},
		Refresh: func() (interface{}, string, error) {
			detail, err := svc.GetJobDetail(vpcId, jobId)
			if err != nil {
				return nil, "", err
			}
			if detail == nil {
				return nil, "", fmt.Errorf("backup job %s disappeared while waiting for its instance list", jobId)
			}
			if sameVmIds(detail.BackupObject, wanted) {
				return detail, "match", nil
			}
			return detail, "stale", nil
		},
		Timeout:    pollTimeout(timeout),
		Delay:      3 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return fmt.Errorf("the backup job was updated but its instance list did not settle: %v", err)
	}
	return nil
}

func sameVmIds(objects []BackupObject, wanted map[string]bool) bool {
	if len(objects) != len(wanted) {
		return false
	}
	for _, obj := range objects {
		if !wanted[obj.VmId] {
			return false
		}
	}
	return true
}
