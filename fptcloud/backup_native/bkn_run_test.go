package fptcloud_backup_native

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/stretchr/testify/assert"
)

func TestRunStatusesDoNotCollideWithAPIStatuses(t *testing.T) {
	api := []string{
		// v1 jobs and history (osp_native_backup/common/constant.py)
		"Creating", "Available", "Updating", "Deleting", "Error", "Restoring", "error_deleting",
		"Initiated", "Pending", "Backing up", "Success", "Deleted", "Undefined", "Failed",
		// v2 (backup_volume/common/constant.py Status)
		"failed", "deleting error", "running", "warning", "canceled", "successful",
	}
	for _, status := range []string{runStatusTimeout, runStatusInterrupted, runStatusFailed} {
		for _, other := range api {
			assert.False(t, strings.EqualFold(other, status), "%s collides with %s", status, other)
		}
	}
}

func TestRunOutcomeKeepsTheAPIFailureStatus(t *testing.T) {
	err := fmt.Errorf("waiting: %w", runFailed("Failed", errors.New("restoring VM vm-1 failed: disk error")))
	status, message := runOutcome(context.Background(), err)
	assert.Equal(t, "Failed", status)
	assert.Equal(t, "restoring VM vm-1 failed: disk error", message)
}

func TestRunOutcomeFailureWithoutAPIStatus(t *testing.T) {
	status, message := runOutcome(context.Background(), runFailed("", errors.New("the job was deleted")))
	assert.Equal(t, runStatusFailed, status)
	assert.Equal(t, "the job was deleted", message)
}

func TestRunOutcomeTimeout(t *testing.T) {
	err := fmt.Errorf("waiting for the backup: %w", &retry.TimeoutError{LastState: "running"})
	status, message := runOutcome(context.Background(), err)
	assert.Equal(t, runStatusTimeout, status)
	assert.Contains(t, message, "waiting for the backup")
}

// Failed API reads in a row also stop the wait before the action ends.
func TestRunOutcomeLostTrackCountsAsTimeout(t *testing.T) {
	status, _ := runOutcome(context.Background(), errors.New("HTTP 502 (failed 6 reads in a row)"))
	assert.Equal(t, runStatusTimeout, status)
}

func TestRunOutcomeInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	status, message := runOutcome(ctx, fmt.Errorf("waiting: %w", context.Canceled))
	assert.Equal(t, runStatusInterrupted, status)
	assert.Contains(t, message, "interrupted")
}

func TestRunInProgress(t *testing.T) {
	assert.True(t, runInProgress(runStatusTimeout))
	assert.True(t, runInProgress(runStatusInterrupted))
	for _, final := range []string{runStatusFailed, "Success", "success", "Failed", "error", ""} {
		assert.False(t, runInProgress(final), final)
	}
}

func TestRunWarningIsNotAnError(t *testing.T) {
	diags := runWarning("tf_timeout", "waiting for the backup timed out")
	assert.False(t, diags.HasError())
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Detail, "triggers")
}
