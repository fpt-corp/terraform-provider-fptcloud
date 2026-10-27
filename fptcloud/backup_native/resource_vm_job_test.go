package fptcloud_backup_native

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tenantBody = `{"status":true,"message":"","data":{"id":"tenant-1","name":"TEST"}}`

func vmJobConfig() map[string]interface{} {
	return map[string]interface{}{
		"vpc_id":                  "vpc-1",
		"name":                    "web",
		"description":             "nightly",
		"instance_ids":            []interface{}{"vm-1"},
		"notification_method_ids": []interface{}{"n-1"},
		"retention":               7,
		"schedule": []interface{}{map[string]interface{}{
			"full_backup_frequency": "daily", "time": "02:00", "time_zone": "Asia/Ho_Chi_Minh",
		}},
	}
}

func TestVmJobSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVmJob().InternalValidate(nil, true))
}

func TestExpandVmJobCreate(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	p := expandVmJobCreate(d, "tenant-1")
	assert.Equal(t, "web", p.Name)
	assert.Equal(t, []string{"n-1"}, p.Notification)
	assert.Equal(t, []string{"vm-1"}, p.AppliedInstances)
	assert.Equal(t, 7, p.NumberLatestBackup)
	assert.Equal(t, "tenant-1", p.TenantUserId)
	assert.Equal(t, "Quick", p.BackupMode)
	assert.Equal(t, "Asia/Ho_Chi_Minh", p.TimeZone)
}

func TestExpandVmJobUpdateIsAFullPayload(t *testing.T) {
	cfg := vmJobConfig()
	cfg["schedule"] = []interface{}{map[string]interface{}{"full_backup_frequency": "weekly", "full_backup_day": 0, "time": "02:00", "time_zone": "UTC"}}
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), cfg)
	p, err := expandVmJobUpdate(d, time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, p.Interval)
	assert.Equal(t, "weekly", p.FullBackupFreq)
	assert.Equal(t, "daily", p.IncrementalBackupFreq)
	require.NotNil(t, p.FullBackupDate)
	assert.Equal(t, 0, *p.FullBackupDate)
	assert.Equal(t, "2026-10-13T02:00:00Z", p.StartTime)
	assert.Equal(t, []string{"n-1"}, p.Recipients)
	assert.Equal(t, []string{"vm-1"}, p.Instances)
	assert.Equal(t, "nightly", p.Description)
}

func TestSameVmSchedule(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	a := Schedule{FullBackupFrequency: "daily", Time: "09:00", TimeZone: "Asia/Ho_Chi_Minh"}
	b := Schedule{FullBackupFrequency: "daily", Time: "09:00", TimeZone: "Asia/Bangkok"}
	assert.True(t, sameVmSchedule(a, b, now), "both are UTC+7")
	b.TimeZone = "Asia/Tokyo"
	assert.False(t, sameVmSchedule(a, b, now))
}

func vmCreateRoutes(listAfterCreate string, taskBody string) map[string][]fakeResponse {
	return map[string][]fakeResponse{
		"GET " + vmBase + "/list": {
			{Body: `{"total":0,"data":[]}`},
			{Body: listAfterCreate},
		},
		"GET /v2/tenant/TEST":                     {{Body: tenantBody}},
		"POST " + vmBase + "/create":              {{Status: 201, Body: `{"message":"creating","data":["task-1"],"status":null}`}},
		"GET " + vmBase + "/task/task-1/progress": {{Body: `{"state":"PENDING","message":"PENDING","status_code":200}`}, {Body: taskBody}},
		"GET " + vmBase + "/detail":               {{Body: vmJobDetailBody}},
	}
}

func TestCreateVmJob(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, vmCreateRoutes(
		`{"total":1,"data":[{"id":"job-1","name":"web","status":"Initiated","is_enabled":true,"instances":[]}]}`,
		`{"state":"SUCCESS","message":"Create backup web successfully","status_code":201}`,
	))
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())

	diags := createVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Empty(t, diags)
	assert.Equal(t, "job-1", d.Id())
	assert.Equal(t, "02:00", d.Get("schedule.0.time"), "read converts the UTC start time back into the zone")
	assert.Equal(t, "Initiated", d.Get("status"))
	assert.Equal(t, "2026-10-13T19:00:00Z", d.Get("next_run"))

	body := api.last("POST " + vmBase + "/create").Body
	assert.Equal(t, "tenant-1", body["tenant_user_id"])
	assert.Equal(t, "Quick", body["backup_mode"])
	assert.Equal(t, 0, api.count("POST "+vmBase+"/switch_schedule"))
	assert.Equal(t, 1, api.count(vmSyncRoute))
}

// A dev apply got one 401 from the list that finds the new job's id; the job
// already existed, so the create must not give up and leave it out of state.
func TestCreateVmJobSurvivesAPassingAPIErrorAfterCreate(t *testing.T) {
	setFastPolling(t)
	listAfterCreate := `{"total":1,"data":[{"id":"job-1","name":"web","status":"Initiated","is_enabled":true,"instances":[]}]}`
	routes := vmCreateRoutes(listAfterCreate, `{"state":"SUCCESS","message":"Create backup web successfully","status_code":201}`)
	routes["GET "+vmBase+"/list"] = []fakeResponse{
		{Body: `{"total":0,"data":[]}`},
		{Status: 401, Body: `{"message":"Unauthorized Access"}`},
		{Body: listAfterCreate},
	}
	_, client := newFakeAPI(t, routes)
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())

	diags := createVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "job-1", d.Id())
}

// Review focus 2.
func TestCreateVmJobRefusesExistingName(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/list": {{Body: `{"total":1,"data":[{"id":"job-0","name":"web"}]}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	diags := createVmJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "vpc/vpc-1/backup_native_vm_job/job-0")
	assert.Equal(t, "", d.Id())
	assert.Equal(t, 0, api.count(vmSyncRoute), "nothing changed")
}

// Review focus 3.
func TestCreateVmJobWarnsWhenQuotaDisabledSchedule(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, vmCreateRoutes(
		`{"total":1,"data":[{"id":"job-1","name":"web","status":"Initiated","is_enabled":false,"instances":[]}]}`,
		`{"state":"SUCCESS","message":"Create backup web successfully but its schedule is disabled because the backup quota is exceeded","status_code":201}`,
	))
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	diags := createVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "schedule disabled")
	assert.Equal(t, "job-1", d.Id())
}

func TestCreateVmJobDisablesScheduleWhenAsked(t *testing.T) {
	setFastPolling(t)
	routes := vmCreateRoutes(
		`{"total":1,"data":[{"id":"job-1","name":"web","status":"Initiated","is_enabled":true,"instances":[]}]}`,
		`{"state":"SUCCESS","message":"ok","status_code":201}`,
	)
	routes["POST "+vmBase+"/switch_schedule"] = []fakeResponse{{Status: 202, Body: `{"data":{"task_id":"task-1"},"status":true}`}}
	api, client := newFakeAPI(t, routes)
	cfg := vmJobConfig()
	cfg["schedule_enabled"] = false
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), cfg)

	diags := createVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, false, api.last("POST " + vmBase + "/switch_schedule").Body["enable"])
}

func TestReadVmJobRemovesDeletedJob(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")
	diags := readVmJob(context.Background(), d, client)
	require.False(t, diags.HasError())
	assert.Equal(t, "", d.Id())
}

func TestUpdateVmJobSendsFullPayloadAndWaits(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail":               {{Body: vmJobDetailBody}},
		"PUT " + vmBase + "/job-1/update":         {{Status: 202, Body: `{"data":{"task_id":"task-2"}}`}},
		"GET " + vmBase + "/task/task-2/progress": {{Body: `{"state":"SUCCESS","message":"updated","status_code":200}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")

	diags := updateVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	body := api.last("PUT " + vmBase + "/job-1/update").Body
	assert.Equal(t, float64(1), body["interval"])
	assert.Equal(t, []interface{}{"n-1"}, body["recipients"])
	assert.Equal(t, []interface{}{"vm-1"}, body["instances"])
	assert.Nil(t, body["full_backup_date"])
	assert.Regexp(t, `T19:00:00Z$`, body["start_time"])
	assert.Equal(t, 0, api.count("POST "+vmBase+"/switch_schedule"), "is_enabled already matches")
	assert.Equal(t, 1, api.count(vmSyncRoute))
}

func TestUpdateVmJobReEnablesScheduleTheBackendDisabled(t *testing.T) {
	setFastPolling(t)
	disabled := strings.Replace(vmJobDetailBody, `"is_enabled":true`, `"is_enabled":false`, 1)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail":               {{Body: disabled}, {Body: disabled}, {Body: disabled}, {Body: vmJobDetailBody}},
		"PUT " + vmBase + "/job-1/update":         {{Status: 202, Body: `{"data":{"task_id":"task-2"}}`}},
		"GET " + vmBase + "/task/task-2/progress": {{Body: `{"state":"SUCCESS","message":"updated","status_code":200}`}},
		"POST " + vmBase + "/switch_schedule":     {{Status: 202, Body: `{"data":{"task_id":"task-3"},"status":true}`}},
		"GET " + vmBase + "/task/task-3/progress": {{Body: `{"state":"SUCCESS","message":"enabled","status_code":200}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")

	diags := updateVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, 1, api.count("POST "+vmBase+"/switch_schedule"))
	assert.Equal(t, true, api.last("POST " + vmBase + "/switch_schedule").Body["enable"])
	assert.Equal(t, 1, api.count(vmSyncRoute), "an update and a switch sync once")
}

func TestDeleteVmJobWaitsUntilGone(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"DELETE " + vmBase + "/job-1/delete": {{Status: 202, Body: `{"message":"Delete successfully."}`}},
		"GET " + vmBase + "/detail": {
			{Body: `{"data":{"id":"job-1","status":"Success"}}`},
			{Body: `{"data":{"id":"job-1","status":"Deleting"}}`},
			{Body: `{}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")
	diags := deleteVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 0, api.servedBefore("GET "+vmBase+"/detail", "DELETE "+vmBase+"/job-1/delete"), "one read, no wait")
	assert.Equal(t, 3, api.count("GET "+vmBase+"/detail"))
	assert.Equal(t, 1, api.count(vmSyncRoute))
}

func TestDeleteVmJobFailsAtOnceWhileABackupRuns(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{"data":{"id":"job-1","status":"45"}}`}}, // a scheduled backup is running
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")
	diags := deleteVmJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "cannot be deleted now")
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"), "no wait")
	assert.Equal(t, 0, api.count("DELETE "+vmBase+"/job-1/delete"))
	assert.Equal(t, 0, api.count(vmSyncRoute))
}

func TestDeleteVmJobFailsAtOnceWhenTheReadFails(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Status: 401, Body: `{"message":"Unauthorized Access"}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")
	diags := deleteVmJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "Unauthorized Access")
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"), "no retry")
	assert.Equal(t, 0, api.count("DELETE "+vmBase+"/job-1/delete"))
}

// A job already deleting comes from an earlier delete that timed out.
func TestDeleteVmJobWaitsForADeleteInProgress(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{"data":{"id":"job-1","status":"Deleting"}}`}, {Body: `{}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")
	diags := deleteVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 0, api.count("DELETE "+vmBase+"/job-1/delete"))
	assert.Equal(t, 1, api.count(vmSyncRoute))
}

func TestDeleteVmJobThatIsAlreadyGone(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")
	diags := deleteVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 0, api.count(vmSyncRoute))
}

// The job is already deleted when the sync fails, so the failure only warns.
func TestDeleteVmJobWarnsWhenTheSyncFails(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"DELETE " + vmBase + "/job-1/delete": {{Status: 202, Body: `{"message":"Delete successfully."}`}},
		"GET " + vmBase + "/detail":          {{Body: `{"data":{"id":"job-1","status":"Success"}}`}, {Body: `{"data":{"id":"job-1","status":"Success"}}`}, {Body: `{}`}},
		vmSyncRoute:                          {{Status: 500, Body: `{"message":"Internal Server Error"}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")
	diags := deleteVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "sync")
}

func TestImportVmJobId(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), map[string]interface{}{})
	d.SetId("vpc/vpc-1/backup_native_vm_job/job-1")
	out, err := importJob("backup_native_vm_job")(context.Background(), d, nil)
	require.NoError(t, err)
	assert.Equal(t, "job-1", out[0].Id())
	assert.Equal(t, "vpc-1", out[0].Get("vpc_id"))

	d.SetId("job-1")
	_, err = importJob("backup_native_vm_job")(context.Background(), d, nil)
	assert.ErrorContains(t, err, "expected vpc/<vpc_id>/backup_native_vm_job/<job_id>")
}

func TestUpdateVmJobFailsAtOnceOnFailedJob(t *testing.T) {
	setFastPolling(t)
	failed := strings.Replace(vmJobDetailBody, `"status":"Initiated"`, `"status":"Failed"`, 1)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: failed}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")

	diags := updateVmJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "Failed")
	assert.Equal(t, 0, api.count("PUT "+vmBase+"/job-1/update"))
	assert.Equal(t, 0, api.count(vmSyncRoute))
}

func TestUpdateVmJobFailsAtOnceWhileAnotherUpdateRuns(t *testing.T) {
	setFastPolling(t)
	updating := strings.Replace(vmJobDetailBody, `"status":"Initiated"`, `"status":"Updating"`, 1)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: updating}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")

	diags := updateVmJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "Updating")
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"), "no wait")
	assert.Equal(t, 0, api.count("PUT "+vmBase+"/job-1/update"))
	assert.Equal(t, 0, api.count(vmSyncRoute))
}

func TestUpdateVmJobFailsAtOnceWhileABackupRuns(t *testing.T) {
	setFastPolling(t)
	backingUp := strings.Replace(vmJobDetailBody, `"status":"Initiated"`, `"status":"45"`, 1)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: backingUp}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")

	diags := updateVmJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "cannot be updated now")
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"), "no wait")
	assert.Equal(t, 0, api.count("PUT "+vmBase+"/job-1/update"))
	assert.Equal(t, 0, api.count(vmSyncRoute))
}

// A scheduled backup starts between the update and the schedule switch.
func TestUpdateVmJobScheduleSwitchFailsAtOnceWhileABackupRuns(t *testing.T) {
	setFastPolling(t)
	backingUp := strings.Replace(vmJobDetailBody, `"status":"Initiated"`, `"status":"45"`, 1)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail":               {{Body: vmJobDetailBody}, {Body: backingUp}},
		"PUT " + vmBase + "/job-1/update":         {{Status: 202, Body: `{"data":{"task_id":"task-2"}}`}},
		"GET " + vmBase + "/task/task-2/progress": {{Body: `{"state":"SUCCESS","message":"updated","status_code":200}`}},
	})
	cfg := vmJobConfig()
	cfg["schedule_enabled"] = false
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), cfg)
	d.SetId("job-1")

	diags := updateVmJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "cannot be updated now")
	assert.Equal(t, 1, api.count("PUT "+vmBase+"/job-1/update"))
	assert.Equal(t, 2, api.count("GET "+vmBase+"/detail"), "no wait")
	assert.Equal(t, 0, api.count("POST "+vmBase+"/switch_schedule"))
	assert.Equal(t, 1, api.count(vmSyncRoute), "the update went through")
}

func TestUpdateVmJobFailsAtOnceWhenTheReadFails(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Status: 401, Body: `{"message":"Unauthorized Access"}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")

	diags := updateVmJob(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "Unauthorized Access")
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"), "no retry")
	assert.Equal(t, 0, api.count("PUT "+vmBase+"/job-1/update"))
}

func TestDeleteVmJobProceedsFromFailed(t *testing.T) {
	setFastPolling(t)
	failed := `{"data":{"id":"job-1","status":"Failed"}}`
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"DELETE " + vmBase + "/job-1/delete": {{Status: 202, Body: `{"message":"Delete successfully."}`}},
		"GET " + vmBase + "/detail":          {{Body: failed}, {Body: failed}, {Body: `{}`}},
	})
	d := schema.TestResourceDataRaw(t, resourceVmJobSchema(), vmJobConfig())
	d.SetId("job-1")
	diags := deleteVmJob(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 1, api.count("DELETE "+vmBase+"/job-1/delete"))
}
