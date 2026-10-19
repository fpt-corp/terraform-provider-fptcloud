package fptcloud_backup_native

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func volumeBackupData(t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, manualBackupSchema("fptcloud_backup_native_volume_job"), map[string]interface{}{
		"vpc_id": "vpc-1", "job_id": "job-9",
	})
}

func TestVolumeBackupSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVolumeBackup().InternalValidate(nil, true))
}

func TestCreateVolumeBackup(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9":             jobStatuses("job-9", "success", "pending", "backing up", "success"),
		"POST " + volBase + "/backup-jobs/job-9/executions": {{Body: `{"message":"Backup job started successfully","status":true}`}},
	})
	d := volumeBackupData(t)
	diags := createVolumeBackup(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "success", d.Get("status"))
	assert.Empty(t, api.last("POST "+volBase+"/backup-jobs/job-9/executions").Body)
}

// Review focus 3.
func TestCreateVolumeBackupQuotaExhausted(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9":             jobStatuses("job-9", "success"),
		"POST " + volBase + "/backup-jobs/job-9/executions": {{Status: 403, Body: `{"message":"Quota exceeded"}`}},
	})
	d := volumeBackupData(t)
	diags := createVolumeBackup(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "backup quota is exhausted")
	assert.Empty(t, d.Id())
}

func TestCreateVolumeBackupRefusesABusyJob(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9": jobStatuses("job-9", "backing up", "success"),
	})
	d := volumeBackupData(t)
	diags := createVolumeBackup(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "backing up")
	assert.Empty(t, d.Id())
	assert.Equal(t, 1, api.count("GET "+volBase+"/backup-jobs/job-9"), "the status is read once, not waited on")
	assert.Equal(t, 0, api.count("POST "+volBase+"/backup-jobs/job-9/executions"))
}

// The executions endpoint accepts a job in "error", so the run starts at once.
func TestCreateVolumeBackupStartsOnAnErrorJob(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9":             jobStatuses("job-9", "error", "pending", "success"),
		"POST " + volBase + "/backup-jobs/job-9/executions": {{Body: `{"message":"Backup job started successfully","status":true}`}},
	})
	diags := createVolumeBackup(context.Background(), volumeBackupData(t), client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 0, api.servedBefore("GET "+volBase+"/backup-jobs/job-9", "POST "+volBase+"/backup-jobs/job-9/executions"))
}

// The dev gateway answers a passing 401 now and then; one failed read must
// not fail the apply before the run starts.
func TestCreateVolumeBackupCheckSurvivesAPassingAPIError(t *testing.T) {
	setFastPolling(t)
	statuses := jobStatuses("job-9", "success", "pending", "success")
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9":             append([]fakeResponse{{Status: 401, Body: `{"message":"Unauthorized Access"}`}}, statuses...),
		"POST " + volBase + "/backup-jobs/job-9/executions": {{Body: `{"message":"Backup job started successfully","status":true}`}},
	})
	diags := createVolumeBackup(context.Background(), volumeBackupData(t), client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 1, api.servedBefore("GET "+volBase+"/backup-jobs/job-9", "POST "+volBase+"/backup-jobs/job-9/executions"))
}
