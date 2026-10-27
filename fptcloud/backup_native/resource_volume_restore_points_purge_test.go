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

const (
	volumePurgeRoute       = "DELETE " + volBase + "/volume/vol-1/restore_points"
	volumeRestorePoints    = "GET " + volBase + "/backups"
	volumeHistory          = "GET " + volBase + "/history/combined-history"
	twoVolumeRestorePoints = `{"data":{"volume_id":"vol-1","restore_points":[{"id":"rp-1","created_at":"2026-10-12T19:00:00","status":"available"},{"id":"rp-2","created_at":"2026-10-11T19:00:00","status":"available"}]}}`
	noVolumeRestorePoints  = `{"data":{"volume_id":"vol-1","restore_points":[]}}`
	noVolumeHistory        = `{"items":[],"has_more":false}`
)

func volumePurgeRow(status, start string) string {
	return `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Delete all backup restore point","status":"` + status +
		`","start_time":"` + start + `","history_metadata":{"extract_info":{"status":"cinder error"}}}],"has_more":false}`
}

func volumePurgeData(t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, resourceVolumeRestorePointsPurgeSchema(), map[string]interface{}{"vpc_id": "vpc-1", "volume_id": "vol-1"})
}

func TestVolumeRestorePointsPurgeSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVolumeRestorePointsPurge().InternalValidate(nil, true))
}

func TestVolumeRestorePointsPurgeDescriptionWarnsAgainstConcurrentPurges(t *testing.T) {
	assert.Contains(t, ResourceVolumeRestorePointsPurge().Description, "Run one purge per volume at a time")
}

func TestCreateVolumeRestorePointsPurgeWithNothingToDeleteSendsNoRequest(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: noVolumeRestorePoints}},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, 0, d.Get("restore_point_count"))
	assert.Equal(t, "success", d.Get("status"))
	assert.Equal(t, 0, api.count(volumePurgeRoute))
}

func TestCreateVolumeRestorePointsPurgeWaitsForTheSuccessRow(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}, {Body: twoVolumeRestorePoints}, {Body: noVolumeRestorePoints}},
		volumePurgeRoute:    {{Body: purgeAccepted}},
		volumeHistory: {
			{Body: noVolumeHistory},
			{Body: volumePurgeRow("deleting", start)},
			{Body: volumePurgeRow("success", start)},
		},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Empty(t, warnings(diags))
	assert.Equal(t, 2, d.Get("restore_point_count"))
	assert.Equal(t, "success", d.Get("status"))
	assert.Equal(t, 1, api.count(volumePurgeRoute))
}

// The worker writes no history row when it finds no restore point on the
// platform, so an empty list ends the wait.
func TestCreateVolumeRestorePointsPurgeEndsWhenTheListEmpties(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}, {Body: twoVolumeRestorePoints}, {Body: noVolumeRestorePoints}},
		volumePurgeRoute:    {{Body: purgeAccepted}},
		volumeHistory:       {{Body: noVolumeHistory}},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Empty(t, warnings(diags))
	assert.NotEmpty(t, d.Id())
}

func TestCreateVolumeRestorePointsPurgeRetriesWhileBusy(t *testing.T) {
	setFastPolling(t)
	busy := fakeResponse{Status: 400, Body: `{"message":"Resource is not available for delete"}`}
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}, {Body: noVolumeRestorePoints}},
		volumePurgeRoute:    {busy, busy, {Body: purgeAccepted}},
		volumeHistory:       {{Body: noVolumeHistory}},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 3, api.count(volumePurgeRoute))
}

func TestSubmitVolumeRestorePointsPurgeReportsTheLastRefusalOnTimeout(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumePurgeRoute: {{Status: 400, Body: `{"message":"Resource is not available for delete"}`}},
		volumeHistory:    {{Body: noVolumeHistory}},
	})
	_, _, err := submitVolumeRestorePointsPurge(context.Background(), NewVolumeService(client), "vpc-1", "vol-1", time.Now().Add(200*time.Millisecond))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "waiting for the API to accept deleting the restore points of volume vol-1")
	assert.Contains(t, err.Error(), "a backup or restore of the VM or volume is running")
}

func TestCreateVolumeRestorePointsPurgeStopsOnOtherErrors(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}},
		volumePurgeRoute:    {{Status: 403, Body: `{"message":"Tenant or vpc service is disabled, please try again later."}`}},
		volumeHistory:       {{Body: noVolumeHistory}},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "deleting the volume's restore points failed with HTTP 403")
	assert.Equal(t, 1, api.count(volumePurgeRoute))
	assert.Equal(t, "", d.Id())
}

// A purge the API accepted is saved whatever its outcome.
func TestCreateVolumeRestorePointsPurgeFailureSavesTheAPIStatus(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}},
		volumePurgeRoute:    {{Body: purgeAccepted}},
		volumeHistory:       {{Body: noVolumeHistory}, {Body: volumePurgeRow("failed", start)}},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Contains(t, diags[0].Summary, "cinder error")
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, "failed", d.Get("status"))
	assert.Equal(t, 2, d.Get("restore_point_count"))
	assert.Empty(t, d.Get("finished_at"))
}

func TestCreateVolumeRestorePointsPurgeTimeoutSavesThePurge(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}},
		volumePurgeRoute:    {{Body: purgeAccepted}},
		volumeHistory:       {{Body: noVolumeHistory}},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(ctx, d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.NotEmpty(t, d.Id())
	assert.Equal(t, runStatusTimeout, d.Get("status"))
}

// While the API refuses the purge as busy, nothing has started yet.
func TestCreateVolumeRestorePointsPurgeNeverAcceptedLeavesNoState(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}},
		volumePurgeRoute:    {{Status: 400, Body: `{"message":"Resource is not available for delete"}`}},
		volumeHistory:       {{Body: noVolumeHistory}},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(ctx, d, client)
	require.True(t, diags.HasError())
	assert.Empty(t, d.Id())
}

func volumePurgeState(t *testing.T, started time.Time) *schema.ResourceData {
	d := volumePurgeData(t)
	d.SetId(fmt.Sprintf("vol-1/%d", started.Unix()))
	require.NoError(t, d.Set("status", runStatusTimeout))
	return d
}

func TestReadVolumeRestorePointsPurgeSyncsFromTheRow(t *testing.T) {
	started := time.Now().UTC().Add(-time.Hour)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeHistory:       {{Body: volumePurgeRow("success", started.Add(5*time.Second).Format(apiTimeLayout))}},
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}},
	})
	d := volumePurgeState(t, started)
	require.Empty(t, readVolumeRestorePointsPurge(context.Background(), d, client))
	assert.Equal(t, "success", d.Get("status"))
	assert.NotEmpty(t, d.Get("finished_at"))
}

func TestReadVolumeRestorePointsPurgeSyncsFromAnEmptyList(t *testing.T) {
	started := time.Now().UTC().Add(-time.Hour)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeHistory:       {{Body: noVolumeHistory}},
		volumeRestorePoints: {{Body: noVolumeRestorePoints}},
	})
	d := volumePurgeState(t, started)
	require.Empty(t, readVolumeRestorePointsPurge(context.Background(), d, client))
	assert.Equal(t, "success", d.Get("status"))
}

func TestReadVolumeRestorePointsPurgeStoresAFailedPurge(t *testing.T) {
	started := time.Now().UTC().Add(-time.Hour)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeHistory:       {{Body: volumePurgeRow("failed", started.Add(5*time.Second).Format(apiTimeLayout))}},
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}},
	})
	d := volumePurgeState(t, started)
	require.Empty(t, readVolumeRestorePointsPurge(context.Background(), d, client))
	assert.Equal(t, "failed", d.Get("status"))
	assert.Contains(t, d.Get("message"), "cinder error")
	assert.Empty(t, d.Get("finished_at"))
}

func TestReadVolumeRestorePointsPurgeKeepsARunningPurge(t *testing.T) {
	started := time.Now().UTC().Add(-time.Hour)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeHistory:       {{Body: volumePurgeRow("deleting", started.Add(5*time.Second).Format(apiTimeLayout))}},
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}},
	})
	d := volumePurgeState(t, started)
	require.Empty(t, readVolumeRestorePointsPurge(context.Background(), d, client))
	assert.Equal(t, runStatusTimeout, d.Get("status"))
}

// The row of a purge that finished just before this one was submitted is
// inside the clock-skew window and must not end the wait.
func TestCreateVolumeRestorePointsPurgeIgnoresTheRowOfAnEarlierPurge(t *testing.T) {
	setFastPolling(t)
	earlier := time.Now().UTC().Add(-30 * time.Second).Format(apiTimeLayout)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}, {Body: twoVolumeRestorePoints}, {Body: twoVolumeRestorePoints}, {Body: noVolumeRestorePoints}},
		volumePurgeRoute:    {{Body: purgeAccepted}},
		volumeHistory:       {{Body: volumePurgeRow("success", earlier)}},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 5, api.count(volumeRestorePoints), "the first list, three polls until the list emptied, and the check for leftovers")
}

// A restore row of the volume is not the purge's row.
func TestCreateVolumeRestorePointsPurgeIgnoresOtherHistoryTypes(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	restore := `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"success","start_time":"` + start + `"}],"has_more":false}`
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}, {Body: twoVolumeRestorePoints}, {Body: noVolumeRestorePoints}},
		volumePurgeRoute:    {{Body: purgeAccepted}},
		volumeHistory:       {{Body: noVolumeHistory}, {Body: restore}},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 4, api.count(volumeRestorePoints), "the first list, two polls until the list emptied, and the check for leftovers")
}

func TestCreateVolumeRestorePointsPurgeWarnsAboutRestorePointsLeft(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		volumeRestorePoints: {{Body: twoVolumeRestorePoints}},
		volumePurgeRoute:    {{Body: purgeAccepted}},
		volumeHistory:       {{Body: noVolumeHistory}, {Body: volumePurgeRow("success", start)}},
	})
	d := volumePurgeData(t)
	diags := createVolumeRestorePointsPurge(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, []string{"2 restore points of volume vol-1 remain after the purge"}, warnings(diags))
	assert.NotEmpty(t, d.Id())
}
