package fptcloud_backup_native

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func volumeJobConfig() map[string]interface{} {
	return map[string]interface{}{
		"vpc_id":                  "vpc-1",
		"name":                    "pvc",
		"description":             "pv backups",
		"volume_ids":              []interface{}{"vol-1"},
		"notification_method_ids": []interface{}{"n-1"},
		"retention":               7,
		"schedule": []interface{}{map[string]interface{}{
			"full_backup_frequency": "weekly", "full_backup_day": 5, "time": "02:00", "time_zone": "Asia/Ho_Chi_Minh",
		}},
	}
}

func TestVolumeJobSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVolumeJob().InternalValidate(nil, true))
}

func TestExpandVolumeJobAlwaysCarriesObjectsAndZone(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), volumeJobConfig())
	p := expandVolumeJob(d)
	assert.Equal(t, []string{"vol-1"}, p.BackupObjects)
	assert.Equal(t, []string{"n-1"}, p.Notifications)
	assert.Equal(t, "Asia/Ho_Chi_Minh", p.BackupTimeZone)
	assert.Equal(t, 1, p.BackupMode)
	require.NotNil(t, p.FullBackupDate)
	assert.Equal(t, 5, *p.FullBackupDate)
	assert.Equal(t, "02:00:00", p.BackupTime)
}

func TestCheckDescriptionChange(t *testing.T) {
	assert.NoError(t, checkDescriptionChange("", "new"))
	assert.NoError(t, checkDescriptionChange("old", "new"))
	assert.ErrorContains(t, checkDescriptionChange("old", ""), "cannot be removed")
}

func volumeCreateRoutes(createStatus int, createBody string) map[string][]fakeResponse {
	return map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs":       {{Body: volumeJobsEmptyBody}, {Body: volumeJobsWithPvcBody}},
		"POST " + volBase + "/backup-jobs":      {{Status: createStatus, Body: createBody}},
		"GET " + volBase + "/backup-jobs/job-9": {{Body: volumeJobDetailBody}},
		"PUT " + volBase + "/backup-jobs/job-9": {{Body: `{"message":"Backup job updated successfully","data":[],"status":true}`}},
	}
}

func TestCreateVolumeJobStoresDescriptionWithAnUpdate(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, volumeCreateRoutes(201, `{"message":"Backup job created successfully","data":[],"status":true}`))
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), volumeJobConfig())

	diags := createVolumeJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "job-9", d.Id())
	assert.Equal(t, "pv backups", api.last("PUT " + volBase + "/backup-jobs/job-9").Body["description"])
	assert.Equal(t, "Asia/Ho_Chi_Minh", d.Get("schedule.0.time_zone"))
	assert.Equal(t, 5, d.Get("schedule.0.full_backup_day"))
	assert.Equal(t, []interface{}{"vol-1"}, d.Get("volume_ids").(*schema.Set).List())
}

// Review focus 2.
func TestCreateVolumeJobRefusesExistingName(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs": {{Body: volumeJobsWithPvcBody}},
	})
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), volumeJobConfig())
	diags := createVolumeJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "vpc/vpc-1/backup_native_volume_job/job-9")
}

// Review focus 4.
func TestCreateVolumeJobPartialFailureKeepsJobInState(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, volumeCreateRoutes(202,
		`{"message":"partial","status":true,"data":[{"volume_id":"vol-2","name":"pv-fke-b","error":"AWX failed"}]}`))
	cfg := volumeJobConfig()
	cfg["volume_ids"] = []interface{}{"vol-1", "vol-2"}
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), cfg)

	diags := createVolumeJob(context.Background(), d, client)
	// A warning, not an error: an error on create with the id set taints the
	// resource, and the next apply would replace the job instead of adding
	// the missing volume.
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "job-9", d.Id(), "the job exists, so it stays in state")
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "volume pv-fke-b (vol-2): AWX failed")
	put := api.last("PUT " + volBase + "/backup-jobs/job-9").Body
	assert.Equal(t, "pv backups", put["description"], "the description is still stored")
	assert.Equal(t, []interface{}{"vol-1"}, put["backup_objects"], "the failed volume is not sent again")
}

func TestReadVolumeJobRemovesDeletedJob(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9": {{Status: 400, Body: `{"message":"Backup job not found with id job-9"}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), volumeJobConfig())
	d.SetId("job-9")
	diags := readVolumeJob(context.Background(), d, client)
	require.False(t, diags.HasError())
	assert.Equal(t, "", d.Id())
}

func TestUpdateVolumeJobDisablesSchedule(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9":          {{Body: volumeJobDetailBody}},
		"PUT " + volBase + "/backup-jobs/job-9":          {{Body: `{"message":"updated","data":[],"status":true}`}},
		"POST " + volBase + "/backup-jobs/job-9/actions": {{Body: `{"message":"Request accepted"}`}},
	})
	cfg := volumeJobConfig()
	cfg["schedule_enabled"] = false
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), cfg)
	d.SetId("job-9")

	diags := updateVolumeJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, []interface{}{"vol-1"}, api.last("PUT " + volBase + "/backup-jobs/job-9").Body["backup_objects"])
	assert.Equal(t, "disable", api.last("POST " + volBase + "/backup-jobs/job-9/actions").Body["action"])
}

// v2 accepts actions only from a settled job; a running backup is busy.
func TestUpdateVolumeJobWaitsForTheJobToSettleBeforeTheAction(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9": {
			{Body: volumeJobDetailBody}, // settle before the PUT
			{Body: strings.Replace(volumeJobDetailBody, `"status":"initiated"`, `"status":"40"`, 1)}, // a scheduled backup starts
			{Body: volumeJobDetailBody},
		},
		"PUT " + volBase + "/backup-jobs/job-9":          {{Body: `{"message":"updated","data":[],"status":true}`}},
		"POST " + volBase + "/backup-jobs/job-9/actions": {{Body: `{"message":"Request accepted"}`}},
	})
	cfg := volumeJobConfig()
	cfg["schedule_enabled"] = false
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), cfg)
	d.SetId("job-9")

	diags := updateVolumeJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.GreaterOrEqual(t, api.servedBefore("GET "+volBase+"/backup-jobs/job-9", "POST "+volBase+"/backup-jobs/job-9/actions"), 2,
		"the action goes out only after the job reads as settled again")
}

func TestDeleteVolumeJobWaitsForARunningBackup(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"DELETE " + volBase + "/backup-jobs/job-9": {{Status: 202, Body: `{"message":"Request accepted","status":true}`}},
		"GET " + volBase + "/backup-jobs/job-9": {
			{Body: strings.Replace(volumeJobDetailBody, `"status":"initiated"`, `"status":"40"`, 1)},
			{Body: volumeJobDetailBody},
			{Status: 400, Body: `{"message":"Backup job not found with id job-9"}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), volumeJobConfig())
	d.SetId("job-9")
	diags := deleteVolumeJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 1, api.servedBefore("GET "+volBase+"/backup-jobs/job-9", "DELETE "+volBase+"/backup-jobs/job-9"))
}

func TestDeleteVolumeJobWaitsUntilGone(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"DELETE " + volBase + "/backup-jobs/job-9": {{Status: 202, Body: `{"message":"Request accepted","status":true}`}},
		"GET " + volBase + "/backup-jobs/job-9": {
			{Body: volumeJobDetailBody},
			{Body: volumeJobDetailBody},
			{Body: `{"data":{"id":"job-9","status":"deleting"}}`},
			{Status: 400, Body: `{"message":"Backup job not found with id job-9"}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), volumeJobConfig())
	d.SetId("job-9")
	diags := deleteVolumeJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
}

// A job that an earlier, timed-out destroy is still deleting reads as busy and
// then disappears while the delete waits for it to settle.
func TestDeleteVolumeJobThatDisappearsWhileWaiting(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9": {
			{Body: `{"data":{"id":"job-9","status":"deleting"}}`},
			{Status: 400, Body: `{"message":"Backup job not found with id job-9"}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVolumeJobSchema(), volumeJobConfig())
	d.SetId("job-9")
	diags := deleteVolumeJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 0, api.count("DELETE "+volBase+"/backup-jobs/job-9"))
}
