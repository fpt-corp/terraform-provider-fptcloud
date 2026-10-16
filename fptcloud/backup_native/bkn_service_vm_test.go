package fptcloud_backup_native

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	vmBase = "/v1/vmware/vpc/vpc-1/osp_native_backup"

	vmJobDetailBody = `{"data":{"id":"job-1","name":"web","description":"nightly","interval":1,
		"incremental_backup_freq":"","incremental_backup_date":null,"number_latest_backup":7,
		"full_backup_freq":"daily","full_backup_date":null,"status":"Initiated",
		"start_time":"2026-10-12T19:00:00","next_run":"2026-10-13T19:00:00",
		"created_at":"2026-10-12T08:00:00","updated_at":"2026-10-12T08:00:00","is_deleted":false,
		"is_enabled":true,"vpc_id":"vpc-1",
		"instances":[{"id":"vm-1","name":"vm-1","vpc_id":"vpc-1","is_deleted":false}],
		"recipients":[{"id":"n-1","name":"ops@example.com","address":"ops@example.com"}],
		"is_quick_mode":true}}`
)

func TestCreateVmJobReturnsTaskId(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/create": {{Status: 201, Body: `{"message":"Instance backup web is creating...","data":["task-1"],"status":null}`}},
	})
	taskId, err := NewVmService(client).CreateVmJob("vpc-1", VmJobCreatePayload{Name: "web"})
	require.NoError(t, err)
	assert.Equal(t, "task-1", taskId)
	assert.Equal(t, "web", api.last("POST " + vmBase + "/create").Body["name"])
}

// v1 create reports a disabled service as HTTP 200 with status false.
func TestCreateVmJobStatusFalseIsAnError(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/create": {{Body: `{"status":false,"message":"Tenant or vpc service is disabled, please try again later."}`}},
	})
	_, err := NewVmService(client).CreateVmJob("vpc-1", VmJobCreatePayload{Name: "web"})
	assert.ErrorContains(t, err, "Backup Native is not enabled for this VPC")
}

func TestCreateVmJobHttpErrorHasHint(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/create": {{Status: 400, Body: `{"message":"Instance is registered to a backup","data":[],"status":null}`}},
	})
	_, err := NewVmService(client).CreateVmJob("vpc-1", VmJobCreatePayload{Name: "web"})
	assert.ErrorContains(t, err, "only one backup job")
}

func TestGetVmJobParsesDetail(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: vmJobDetailBody}},
	})
	job, err := NewVmService(client).GetVmJob("vpc-1", "job-1")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "web", job.Name)
	assert.Equal(t, 7, job.NumberLatestBackup)
	assert.True(t, job.IsEnabled)
	assert.Equal(t, "vm-1", job.Instances[0].Id)
	assert.Equal(t, "n-1", job.Recipients[0].Id)
	assert.Nil(t, job.FullBackupDate)
	assert.Equal(t, "job-1", api.last("GET "+vmBase+"/detail").Query.Get("backup_id"))
}

// A missing or deleted job is HTTP 200 with body {}.
func TestGetVmJobEmptyBodyMeansGone(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{}`}},
	})
	job, err := NewVmService(client).GetVmJob("vpc-1", "job-1")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestUpdateVmJobReturnsTaskId(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"PUT " + vmBase + "/job-1/update": {{Status: 202, Body: `{"message":"Your request have been accepted.","data":{"task_id":"task-2"}}`}},
	})
	taskId, err := NewVmService(client).UpdateVmJob("vpc-1", "job-1", VmJobUpdatePayload{Name: "web"})
	require.NoError(t, err)
	assert.Equal(t, "task-2", taskId)
}

func TestDeleteVmJobTreats404AsDone(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"DELETE " + vmBase + "/job-1/delete": {{Status: 404, Body: `{"message":"This backup is no longer exist"}`}},
	})
	assert.NoError(t, NewVmService(client).DeleteVmJob("vpc-1", "job-1"))
}

func TestSwitchVmJobScheduleSendsBackupType(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/switch_schedule": {{Status: 202, Body: `{"message":"accepted","data":{"task_id":"task-3"},"status":true}`}},
	})
	taskId, err := NewVmService(client).SwitchVmJobSchedule("vpc-1", "job-1", false)
	require.NoError(t, err)
	assert.Equal(t, "task-3", taskId)
	body := api.last("POST " + vmBase + "/switch_schedule").Body
	assert.Equal(t, "job-1", body["backup_id"])
	assert.Equal(t, "backup", body["type"])
	assert.Equal(t, false, body["enable"])
}

func TestGetTaskProgress(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/task/task-1/progress": {{Body: `{"state":"SUCCESS","message":"done","status_code":201}`}},
	})
	p, err := NewVmService(client).GetTaskProgress("vpc-1", "task-1")
	require.NoError(t, err)
	assert.Equal(t, TaskProgress{State: "SUCCESS", Message: "done", StatusCode: 201}, *p)
}

func TestListVmRestorePoints(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/restore_points/list": {{Body: `{"data":{"vm_id":"vm-1","vm_name":"web","restore_points":[
			{"vm_id":"vm-1","size":1.5,"created_at":"2026-10-12T19:00:00","is_incremental":false,"is_available":true,"backup_job_id":"job-1","backup_job_name":"web"}]}}`}},
	})
	points, err := NewVmService(client).ListVmRestorePoints("vpc-1", "vm-1")
	require.NoError(t, err)
	require.Len(t, points, 1)
	assert.Equal(t, "2026-10-12T19:00:00", points[0].CreatedAt)
	assert.Equal(t, 1.5, points[0].Size)
	assert.Equal(t, "vm-1", api.last("GET "+vmBase+"/restore_points/list").Query.Get("vm_id"))
}

func TestListVmHistoriesSendsTypeAndPaging(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore keep","status":"Success","start_time":"2026-10-13T08:00:00","end_time":"2026-10-13T08:10:00"}]}`}},
	})
	rows, err := NewVmService(client).ListVmHistories("vpc-1", "Restore keep")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	q := api.last("GET " + vmBase + "/histories/list").Query
	assert.Equal(t, "Restore keep", q.Get("type"))
	assert.Equal(t, "1", q.Get("page"))
	assert.Equal(t, "50", q.Get("page_size"))
}

func TestRestoreVmKeepSendsObjects(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/restore-new": {{Status: 202, Body: `{"message":"Your request have been accepted.","status":true,"data":[]}`}},
	})
	err := NewVmService(client).RestoreVmKeep("vpc-1", VmRestoreKeepPayload{
		VmId: "vm-1", CreatedAt: "2026-10-12T19:00:00", PowerOn: true,
		HwFlavor: RestoreFlavor{Id: "f-1", Name: "2C4G", Cpu: 2, Ram: 4096},
		Subnet:   IdName{Id: "net-1", Name: "subnet-a"}, SecurityGroups: []IdName{},
	})
	require.NoError(t, err)
	body := api.last("POST " + vmBase + "/restore-new").Body
	assert.Equal(t, []interface{}{}, body["security_groups"], "never null")
	assert.Equal(t, map[string]interface{}{"id": "", "ip": ""}, body["floating_ip"], "never null")
	assert.Equal(t, false, body["is_overwrite"])
}

func TestListFloatingIps(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET /v1/vmware/vpc/vpc-1/ip-address": {{Body: `{"status":true,"total":1,"data":[{"id":"ip-1","ip_type":"PUBLIC","ip_address":"203.0.113.5"}]}`}},
	})
	ips, err := NewVmService(client).ListFloatingIps("vpc-1")
	require.NoError(t, err)
	assert.Equal(t, []FloatingIpItem{{Id: "ip-1", IpAddress: "203.0.113.5"}}, ips)
}

func TestStartVmJobBackup(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/start_backup": {{Body: `{"message":"Started backup job","status":true}`}},
	})
	require.NoError(t, NewVmService(client).StartVmJobBackup("vpc-1", "job-1"))
	assert.Equal(t, map[string]interface{}{"resource_id": "job-1"}, api.last("POST "+vmBase+"/start_backup").Body)
}

func TestStartVmJobBackupBusy(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/start_backup": {{Status: 400, Body: `{"message":"Resource is unavailable. Please try again later"}`}},
	})
	err := NewVmService(client).StartVmJobBackup("vpc-1", "job-1")
	assert.ErrorContains(t, err, "starting the VM backup job")
	assert.ErrorContains(t, err, "busy")
}
