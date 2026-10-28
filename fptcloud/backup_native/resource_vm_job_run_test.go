package fptcloud_backup_native

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func vmJobRunData(t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, jobRunSchema("fptcloud_backup_native_vm_job"), map[string]interface{}{
		"vpc_id": "vpc-1", "job_id": "job-1",
	})
}

const startedBody = `{"message":"Started backup job","status":true}`

func TestVmJobRunSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVmJobRun().InternalValidate(nil, true))
}

func TestJobRunArgumentsForceNew(t *testing.T) {
	for name, r := range map[string]*schema.Resource{"vm": ResourceVmJobRun(), "volume": ResourceVolumeJobRun()} {
		for key, s := range r.Schema {
			if !s.Computed {
				assert.True(t, s.ForceNew, "%s: %s must be ForceNew", name, key)
			}
		}
		assert.Nil(t, r.Importer, name)
		assert.Nil(t, r.UpdateContext, name)
		require.NotNil(t, r.Timeouts, name)
		assert.Equal(t, 60*time.Minute, *r.Timeouts.Create, name)
	}
}

func TestCreateVmJobRun(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail":        jobStatuses("job-1", "Success", "Pending", "Backing up", "Success"),
		"POST " + vmBase + "/start_backup": {{Body: startedBody}},
	})
	d := vmJobRunData(t)

	diags := createVmJobRun(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "Success", d.Get("status"))
	assert.True(t, strings.HasPrefix(d.Id(), "job-1/"), d.Id())
	for _, key := range []string{"started_at", "finished_at"} {
		_, err := time.Parse(time.RFC3339, d.Get(key).(string))
		assert.NoError(t, err, key)
	}
	assert.Equal(t, map[string]interface{}{"resource_id": "job-1"}, api.last("POST "+vmBase+"/start_backup").Body)
	assert.Equal(t, 1, api.count(vmSyncRoute))
}

// Review focus 1.
func TestCreateVmJobRunRefusesABusyJob(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": jobStatuses("job-1", "Backing up", "Success"),
	})
	d := vmJobRunData(t)
	diags := createVmJobRun(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "Backing up")
	assert.Empty(t, d.Id())
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"), "the status is read once, not waited on")
	assert.Equal(t, 0, api.count("POST "+vmBase+"/start_backup"))
	assert.Equal(t, 0, api.count(vmSyncRoute))
}

func TestCreateVmJobRunFailsAtOnceOnAFailedJob(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": jobStatuses("job-1", "Failed"),
	})
	d := vmJobRunData(t)
	diags := createVmJobRun(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "Failed")
	assert.Empty(t, d.Id())
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"))
	assert.Equal(t, 0, api.count("POST "+vmBase+"/start_backup"))
}

// Review focus 2: a run that started is saved whatever its outcome, so the
// next apply does not run it again.
func TestCreateVmJobRunFailureSavesTheAPIStatus(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail":        jobStatuses("job-1", "Success", "Pending", "Failed"),
		"POST " + vmBase + "/start_backup": {{Body: startedBody}},
	})
	d := vmJobRunData(t)
	diags := createVmJobRun(context.Background(), d, client)
	require.False(t, diags.HasError(), "a failed run must not taint the resource: %v", diags)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "may still be retried")
	assert.True(t, strings.HasPrefix(d.Id(), "job-1/"), d.Id())
	assert.Equal(t, "Failed", d.Get("status"))
	assert.Contains(t, d.Get("message"), "may still be retried")
	assert.Empty(t, d.Get("finished_at"))
	assert.Equal(t, 1, api.count(vmSyncRoute), "a run that started syncs even when it fails")
}

func TestCreateJobRunTimeoutSavesTheRun(t *testing.T) {
	d := vmJobRunData(t)
	diags := createJobRun(context.Background(), d, jobRunSteps{
		check: func() error { return nil },
		start: func() error { return nil },
		wait: func(context.Context, time.Duration) (string, error) {
			return "", fmt.Errorf("waiting for the backup: %w", &retry.TimeoutError{LastState: "running"})
		},
	})
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, runStatusTimeout, d.Get("status"))
	assert.NotEmpty(t, d.Get("started_at"))
}

// Ctrl+C cancels the create's context: the run goes on without Terraform, and
// the resource is saved so that the next apply does not start another.
func TestCreateJobRunInterruptSavesTheRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	d := vmJobRunData(t)
	diags := createJobRun(ctx, d, jobRunSteps{
		check: func() error { return nil },
		start: func() error { return nil },
		wait: func(ctx context.Context, _ time.Duration) (string, error) {
			cancel()
			<-ctx.Done()
			return "", fmt.Errorf("waiting for the backup: %w", ctx.Err())
		},
	})
	require.False(t, diags.HasError(), "%v", diags)
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, runStatusInterrupted, d.Get("status"))
}

// An interrupt before the run starts leaves nothing to save.
func TestCreateJobRunInterruptBeforeStartFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := vmJobRunData(t)
	diags := createJobRun(ctx, d, jobRunSteps{
		check: func() error { return readFailed(errors.New("HTTP 502")) },
		start: func() error { t.Fatal("the run must not start"); return nil },
	})
	require.True(t, diags.HasError())
	assert.Empty(t, d.Id())
}

func vmJobRunState(t *testing.T, status string) *schema.ResourceData {
	d := vmJobRunData(t)
	d.SetId("job-1/1767225600")
	require.NoError(t, d.Set("status", status))
	require.NoError(t, d.Set("message", "waiting timed out"))
	return d
}

func TestReadVmJobRunSyncsATimedOutRun(t *testing.T) {
	for _, status := range []string{runStatusTimeout, runStatusInterrupted} {
		_, client := newFakeAPI(t, map[string][]fakeResponse{
			"GET " + vmBase + "/detail": jobStatuses("job-1", "Success"),
		})
		d := vmJobRunState(t, status)
		diags := readVmJobRun(context.Background(), d, client)
		require.Empty(t, diags)
		assert.Equal(t, "Success", d.Get("status"), status)
		assert.Empty(t, d.Get("message"))
		_, err := time.Parse(time.RFC3339, d.Get("finished_at").(string))
		assert.NoError(t, err)
	}
}

func TestReadVmJobRunStoresAFailedRun(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": jobStatuses("job-1", "Failed"),
	})
	d := vmJobRunState(t, runStatusTimeout)
	require.Empty(t, readVmJobRun(context.Background(), d, client))
	assert.Equal(t, "Failed", d.Get("status"))
	assert.NotEmpty(t, d.Get("message"))
}

func TestReadVmJobRunKeepsARunningRun(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": jobStatuses("job-1", "45"),
	})
	d := vmJobRunState(t, runStatusTimeout)
	require.Empty(t, readVmJobRun(context.Background(), d, client))
	assert.Equal(t, runStatusTimeout, d.Get("status"))
	assert.Empty(t, d.Get("finished_at"))
}

func TestReadVmJobRunWarnsWhenTheCheckFails(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Status: 502, Body: `{"message":"Bad gateway"}`}},
	})
	d := vmJobRunState(t, runStatusTimeout)
	diags := readVmJobRun(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	require.Len(t, diags, 1)
	assert.Equal(t, runStatusTimeout, d.Get("status"))
	assert.Equal(t, "job-1/1767225600", d.Id())
}

func TestReadVmJobRunMarksADeletedJobFailed(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		// A missing v1 job is HTTP 200 with an empty body.
		"GET " + vmBase + "/detail": {{Body: `{}`}},
	})
	d := vmJobRunState(t, runStatusTimeout)
	require.Empty(t, readVmJobRun(context.Background(), d, client))
	assert.Equal(t, runStatusFailed, d.Get("status"))
	assert.Equal(t, "job-1/1767225600", d.Id(), "the run is kept: deleting the job does not undo it")
}

// A run that ended is never read again.
func TestReadVmJobRunLeavesAFinalStatus(t *testing.T) {
	for _, status := range []string{"Success", "Failed", runStatusFailed} {
		api, client := newFakeAPI(t, map[string][]fakeResponse{
			"GET " + vmBase + "/detail": jobStatuses("job-1", "45"),
		})
		d := vmJobRunState(t, status)
		require.Empty(t, readVmJobRun(context.Background(), d, client))
		assert.Equal(t, status, d.Get("status"))
		assert.Equal(t, 0, api.count("GET "+vmBase+"/detail"), status)
	}
}

func TestCreateVmJobRunStartRefused(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail":        jobStatuses("job-1", "Success"),
		"POST " + vmBase + "/start_backup": {{Status: 400, Body: `{"message":"Resource is unavailable. Please try again later"}`}},
	})
	d := vmJobRunData(t)
	diags := createVmJobRun(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "busy")
	assert.Empty(t, d.Id())
	assert.Equal(t, 0, api.count(vmSyncRoute))
}
