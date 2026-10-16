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

// A v2 job with several volumes reads "error" once one volume fails while the
// others still back up; the manual run must not start on top of that run.
func TestCreateVolumeBackupWaitsOutAMidRunError(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9":             jobStatuses("job-9", "backing up", "error", "backing up", "success", "pending", "success"),
		"POST " + volBase + "/backup-jobs/job-9/executions": {{Body: `{"message":"Backup job started successfully","status":true}`}},
	})
	diags := createVolumeBackup(context.Background(), volumeBackupData(t), client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 3, api.servedBefore("GET "+volBase+"/backup-jobs/job-9", "POST "+volBase+"/backup-jobs/job-9/executions"))
}
