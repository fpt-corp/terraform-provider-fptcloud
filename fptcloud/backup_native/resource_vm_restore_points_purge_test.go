package fptcloud_backup_native

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	purgeRoute       = "DELETE " + vmBase + "/instance/vm-1/restore_points"
	restorePoints    = "GET " + vmBase + "/restore_points/list"
	histories        = "GET " + vmBase + "/histories/list"
	twoRestorePoints = `{"status":true,"data":{"vm_id":"vm-1","restore_points":[{"vm_id":"vm-1","created_at":"2026-10-12T19:00:00"},{"vm_id":"vm-1","created_at":"2026-10-11T19:00:00"}]}}`
	noRestorePoints  = `{"status":true,"data":{"vm_id":"vm-1","restore_points":[]}}`
	purgeAccepted    = `{"message":"Request accepted","status":true}`
)

func purgeRow(status, start string) string {
	return `{"total":1,"data":[{"vm_id":"vm-1","type":"Delete all backup restore point","status":"` + status + `","reason":"cinder error","start_time":"` + start + `"}]}`
}

func purgeData(t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, resourceVmRestorePointsPurgeSchema(), map[string]interface{}{"vpc_id": "vpc-1", "vm_id": "vm-1"})
}

func warnings(diags diag.Diagnostics) []string {
	var out []string
	for _, d := range diags {
		if d.Severity == diag.Warning {
			out = append(out, d.Summary)
		}
	}
	return out
}

func TestVmRestorePointsPurgeSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVmRestorePointsPurge().InternalValidate(nil, true))
}

func TestVmRestorePointsPurgeDescriptionWarnsAgainstConcurrentPurges(t *testing.T) {
	assert.Contains(t, ResourceVmRestorePointsPurge().Description, "Run one purge per VM at a time")
}

func TestCreateVmRestorePointsPurgeWithNothingToDeleteSendsNoRequest(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: noRestorePoints}},
	})
	d := purgeData(t)
	diags := createVmRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, 0, d.Get("restore_point_count"))
	assert.Equal(t, "Success", d.Get("status"))
	assert.Equal(t, 0, api.count(purgeRoute))
}

func TestCreateVmRestorePointsPurgeWaitsForTheSuccessRow(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: twoRestorePoints}, {Body: twoRestorePoints}, {Body: noRestorePoints}},
		purgeRoute:    {{Body: purgeAccepted}},
		histories: {
			{Body: `{"total":0,"data":[]}`},
			{Body: purgeRow("Deleting", start)},
			{Body: purgeRow("Success", start)},
		},
	})
	d := purgeData(t)
	diags := createVmRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Empty(t, warnings(diags))
	assert.Equal(t, 2, d.Get("restore_point_count"))
	assert.Equal(t, "Success", d.Get("status"))
	assert.Equal(t, 1, api.count(purgeRoute))
	assert.Equal(t, purgeHistoryType, api.last(histories).Query.Get("type"))
}

// The worker writes no history row when it finds no restore point on the
// platform, so an empty list ends the wait.
func TestCreateVmRestorePointsPurgeEndsWhenTheListEmpties(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: twoRestorePoints}, {Body: twoRestorePoints}, {Body: noRestorePoints}},
		purgeRoute:    {{Body: purgeAccepted}},
		histories:     {{Body: `{"total":0,"data":[]}`}},
	})
	d := purgeData(t)
	diags := createVmRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Empty(t, warnings(diags))
	assert.NotEmpty(t, d.Id())
}

func TestCreateVmRestorePointsPurgeRetriesWhileBusy(t *testing.T) {
	setFastPolling(t)
	busy := fakeResponse{Status: 400, Body: `{"message":"Resource is not available for delete"}`}
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: twoRestorePoints}, {Body: noRestorePoints}},
		purgeRoute:    {busy, busy, {Body: purgeAccepted}},
		histories:     {{Body: `{"total":0,"data":[]}`}},
	})
	d := purgeData(t)
	diags := createVmRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 3, api.count(purgeRoute))
}

func TestPurgeVmRestorePointsReportsTheLastRefusalOnTimeout(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: twoRestorePoints}},
		purgeRoute:    {{Status: 400, Body: `{"message":"Resource is not available for delete"}`}},
		histories:     {{Body: `{"total":0,"data":[]}`}},
	})
	err := purgeVmRestorePoints(context.Background(), NewVmService(client), "vpc-1", "vm-1", time.Now().Add(200*time.Millisecond))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "waiting for the API to accept deleting the restore points of VM vm-1")
	assert.Contains(t, err.Error(), "a backup or restore of the VM is running")
}

func TestCreateVmRestorePointsPurgeStopsOnOtherErrors(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: twoRestorePoints}},
		purgeRoute:    {{Status: 403, Body: `{"message":"Tenant or vpc service is disabled, please try again later."}`}},
		histories:     {{Body: `{"total":0,"data":[]}`}},
	})
	d := purgeData(t)
	diags := createVmRestorePointsPurge(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "deleting the VM's restore points failed with HTTP 403")
	assert.Equal(t, 1, api.count(purgeRoute))
	assert.Equal(t, "", d.Id())
}

func TestCreateVmRestorePointsPurgeFailureLeavesNoState(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: twoRestorePoints}},
		purgeRoute:    {{Body: purgeAccepted}},
		histories:     {{Body: `{"total":0,"data":[]}`}, {Body: purgeRow("Failed", start)}},
	})
	d := purgeData(t)
	diags := createVmRestorePointsPurge(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "cinder error")
	assert.Equal(t, "", d.Id())
}

// The row of a purge that finished just before this one was submitted is
// inside the clock-skew window and must not end the wait.
func TestCreateVmRestorePointsPurgeIgnoresTheRowOfAnEarlierPurge(t *testing.T) {
	setFastPolling(t)
	earlier := time.Now().UTC().Add(-30 * time.Second).Format(apiTimeLayout)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: twoRestorePoints}, {Body: twoRestorePoints}, {Body: twoRestorePoints}, {Body: noRestorePoints}},
		purgeRoute:    {{Body: purgeAccepted}},
		histories:     {{Body: purgeRow("Success", earlier)}},
	})
	d := purgeData(t)
	diags := createVmRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 5, api.count(restorePoints), "the first list, three polls until the list emptied, and the check for leftovers")
}

func TestCreateVmRestorePointsPurgeWarnsAboutRestorePointsLeft(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		restorePoints: {{Body: twoRestorePoints}},
		purgeRoute:    {{Body: purgeAccepted}},
		histories:     {{Body: `{"total":0,"data":[]}`}, {Body: purgeRow("Success", start)}},
	})
	d := purgeData(t)
	diags := createVmRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, []string{"2 restore points of VM vm-1 remain after the purge"}, warnings(diags))
	assert.NotEmpty(t, d.Id())
}
