package fptcloud_backup_native

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVmRestoreSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVmRestore().InternalValidate(nil, true))
}

// A restore is matched to its history row by VM, type and start time, so two
// restores of one VM in the same apply can each finish on the other's row.
func TestVmRestoreDescriptionWarnsAgainstConcurrentRestores(t *testing.T) {
	assert.Contains(t, ResourceVmRestore().Description, "Run one restore per VM at a time")
}

func TestCheckRestoreArguments(t *testing.T) {
	assert.NoError(t, checkRestoreArguments("overwrite", map[string]bool{}))
	assert.ErrorContains(t, checkRestoreArguments("overwrite", map[string]bool{"flavor_id": true}), "flavor_id can only be set when mode is keep")
	assert.ErrorContains(t, checkRestoreArguments("keep", map[string]bool{"flavor_id": true}), "subnet_id is required when mode is keep")
	assert.NoError(t, checkRestoreArguments("keep", map[string]bool{"flavor_id": true, "subnet_id": true}))
}

func TestRestorePointCreatedAtRejectsSuffix(t *testing.T) {
	validate := resourceVmRestoreSchema()["restore_point_created_at"].ValidateFunc
	_, errs := validate("2026-07-01T02:00:00", "restore_point_created_at")
	assert.Empty(t, errs)
	_, errs = validate("2026-07-01T02:00:00Z", "restore_point_created_at")
	assert.NotEmpty(t, errs)
	_, errs = validate("2026-07-01T02:00:00+07:00", "restore_point_created_at")
	assert.NotEmpty(t, errs)
}

func TestCreateVmRestoreOverwrite(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/restore": {{Status: 202, Body: `{"message":"Your request have been accepted.","status":true}`}},
		"GET " + vmBase + "/histories/list": {
			{Body: `{"total":0,"data":[]}`},
			{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"Restoring","start_time":"` + start + `"}]}`},
			{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"Success","start_time":"` + start + `","end_time":"` + start + `"}]}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVmRestoreSchema(), map[string]interface{}{
		"vpc_id": "vpc-1", "vm_id": "vm-1", "restore_point_created_at": "2026-10-12T19:00:00", "mode": "overwrite",
	})

	diags := createVmRestore(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "Success", d.Get("status"))
	assert.NotEmpty(t, d.Id())
	body := api.last("POST " + vmBase + "/restore").Body
	assert.Equal(t, true, body["is_overwrite"])
	assert.Equal(t, "2026-10-12T19:00:00", body["created_at"])
	assert.Equal(t, true, body["power_on"])
	assert.Equal(t, "Restore", api.last("GET "+vmBase+"/histories/list").Query.Get("type"))
	assert.Equal(t, 1, api.count(vmSyncRoute))
}

// Review focus 5: the row of a restore that finished just before this one was
// submitted is inside the clock-skew window and must not complete it.
func TestCreateVmRestoreIgnoresTheRowOfAnEarlierRestore(t *testing.T) {
	setFastPolling(t)
	earlier := time.Now().UTC().Add(-30 * time.Second).Format(apiTimeLayout)
	start := time.Now().UTC().Add(time.Second).Format(apiTimeLayout)
	old := `{"vm_id":"vm-1","type":"Restore","status":"Success","start_time":"` + earlier + `","end_time":"` + earlier + `"}`
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/restore": {{Status: 202, Body: `{"status":true}`}},
		"GET " + vmBase + "/histories/list": {
			{Body: `{"total":1,"data":[` + old + `]}`},
			{Body: `{"total":2,"data":[{"vm_id":"vm-1","type":"Restore","status":"Restoring","start_time":"` + start + `"},` + old + `]}`},
			{Body: `{"total":2,"data":[{"vm_id":"vm-1","type":"Restore","status":"Success","start_time":"` + start + `","end_time":"` + start + `"},` + old + `]}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVmRestoreSchema(), map[string]interface{}{
		"vpc_id": "vpc-1", "vm_id": "vm-1", "restore_point_created_at": "2026-10-12T19:00:00", "mode": "overwrite",
	})
	diags := createVmRestore(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, formatAPITime(start), d.Get("started_at"))
}

func vmRestoreData(t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, resourceVmRestoreSchema(), map[string]interface{}{
		"vpc_id": "vpc-1", "vm_id": "vm-1", "restore_point_created_at": "2026-10-12T19:00:00", "mode": "overwrite",
	})
}

func vmRestoreRow(status, start, end string) string {
	return `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"` + status + `","reason":"disk error","start_time":"` + start + `","end_time":"` + end + `"}]}`
}

// A restore that was sent is saved whatever its outcome, so the next apply
// does not restore again.
func TestCreateVmRestoreFailureSavesTheAPIStatus(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/restore":       {{Status: 202, Body: `{"status":true}`}},
		"GET " + vmBase + "/histories/list": {{Body: `{"total":0,"data":[]}`}, {Body: vmRestoreRow("Failed", start, start)}},
	})
	d := vmRestoreData(t)
	diags := createVmRestore(context.Background(), d, client)
	require.False(t, diags.HasError(), "a failed restore must not taint the resource: %v", diags)
	assert.Contains(t, diags[0].Summary, "disk error")
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, "Failed", d.Get("status"))
	assert.Contains(t, d.Get("message"), "disk error")
	assert.Equal(t, formatAPITime(start), d.Get("started_at"))
	assert.Equal(t, 1, api.count(vmSyncRoute), "a restore that was sent syncs even when it fails")
}

func TestCreateVmRestoreTimeoutKeepsTheRowStart(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/restore":       {{Status: 202, Body: `{"status":true}`}},
		"GET " + vmBase + "/histories/list": {{Body: `{"total":0,"data":[]}`}, {Body: vmRestoreRow("Restoring", start, "")}},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	d := vmRestoreData(t)
	diags := createVmRestore(ctx, d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, runStatusTimeout, d.Get("status"))
	assert.Equal(t, formatAPITime(start), d.Get("started_at"), "the row's start finds the row again on refresh")
	assert.Empty(t, d.Get("finished_at"))
}

func TestCreateVmRestoreInterruptSavesTheRestore(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/restore":       {{Status: 202, Body: `{"status":true}`}},
		"GET " + vmBase + "/histories/list": {{Body: `{"total":0,"data":[]}`}},
	})
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	d := vmRestoreData(t)
	diags := createVmRestore(ctx, d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, runStatusInterrupted, d.Get("status"))
	assert.Empty(t, d.Get("started_at"))
}

func TestCreateVmRestoreRefusedLeavesNoState(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/restore":       {{Status: 400, Body: `{"message":"Resource is unavailable"}`}},
		"GET " + vmBase + "/histories/list": {{Body: `{"total":0,"data":[]}`}},
	})
	d := vmRestoreData(t)
	diags := createVmRestore(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Empty(t, d.Id())
}

func vmRestoreState(t *testing.T, startedAt string, submitted time.Time) *schema.ResourceData {
	d := vmRestoreData(t)
	d.SetId(fmt.Sprintf("vm-1/overwrite/%d", submitted.Unix()))
	require.NoError(t, d.Set("status", runStatusTimeout))
	require.NoError(t, d.Set("started_at", startedAt))
	return d
}

func TestReadVmRestoreSyncsByTheRowStart(t *testing.T) {
	submitted := time.Now().UTC().Add(-time.Hour)
	start := submitted.Add(10 * time.Second).Format(apiTimeLayout)
	end := submitted.Add(20 * time.Minute).Format(apiTimeLayout)
	later := submitted.Add(30 * time.Minute).Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Body: `{"total":2,"data":[` +
			`{"vm_id":"vm-1","type":"Restore","status":"Failed","reason":"another restore","start_time":"` + later + `"},` +
			`{"vm_id":"vm-1","type":"Restore","status":"Success","start_time":"` + start + `","end_time":"` + end + `"}]}`}},
	})
	d := vmRestoreState(t, formatAPITime(start), submitted)
	require.Empty(t, readVmRestore(context.Background(), d, client))
	assert.Equal(t, "Success", d.Get("status"))
	assert.Equal(t, formatAPITime(end), d.Get("finished_at"))
	assert.Empty(t, d.Get("message"))
}

// A restore that timed out before its row appeared is matched by the time it
// was submitted, which the id records.
func TestReadVmRestoreFindsARowFirstSeenOnRefresh(t *testing.T) {
	submitted := time.Now().UTC().Add(-time.Hour)
	start := submitted.Add(10 * time.Second).Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Body: vmRestoreRow("Failed", start, start)}},
	})
	d := vmRestoreState(t, "", submitted)
	require.Empty(t, readVmRestore(context.Background(), d, client))
	assert.Equal(t, "Failed", d.Get("status"))
	assert.Contains(t, d.Get("message"), "disk error")
	assert.Equal(t, formatAPITime(start), d.Get("started_at"))
}

func TestReadVmRestoreKeepsARunningRestore(t *testing.T) {
	submitted := time.Now().UTC().Add(-time.Hour)
	start := submitted.Add(10 * time.Second).Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Body: vmRestoreRow("Restoring", start, "")}},
	})
	d := vmRestoreState(t, "", submitted)
	require.Empty(t, readVmRestore(context.Background(), d, client))
	assert.Equal(t, runStatusTimeout, d.Get("status"))
	assert.Equal(t, formatAPITime(start), d.Get("started_at"), "the row is remembered for the next refresh")
}

func TestReadVmRestoreLeavesAFinalStatus(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Body: `{"total":0,"data":[]}`}},
	})
	d := vmRestoreState(t, "", time.Now())
	require.NoError(t, d.Set("status", "Success"))
	require.Empty(t, readVmRestore(context.Background(), d, client))
	assert.Equal(t, 0, api.count("GET "+vmBase+"/histories/list"))
}
