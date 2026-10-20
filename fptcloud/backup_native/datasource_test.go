package fptcloud_backup_native

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataSourceSchemasAreValid(t *testing.T) {
	for name, r := range map[string]*schema.Resource{
		"vm_jobs":               DataSourceVmJobs(),
		"vm_restore_points":     DataSourceVmRestorePoints(),
		"volume_jobs":           DataSourceVolumeJobs(),
		"volume_restore_points": DataSourceVolumeRestorePoints(),
	} {
		assert.NoError(t, r.InternalValidate(nil, false), name)
	}
}

func TestNewestFirst(t *testing.T) {
	values := []string{"2026-10-11T00:00:00", "garbage", "2026-10-13T00:00:00", "2026-10-12T00:00:00"}
	order := newestFirst(func(i int) string { return values[i] }, len(values))
	assert.Equal(t, []int{2, 3, 0, 1}, order)
}

func TestVmJobsDataSourceFiltersByName(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/list": {{Body: `{"total":2,"data":[
			{"id":"job-1","name":"web","status":"Success","is_enabled":true,"next_run":"2026-10-13T19:00:00","instances":[{"id":"vm-1"}]},
			{"id":"job-2","name":"db","status":"Error","is_enabled":false,"instances":[{"id":"vm-2"}]}]}`}},
	})
	d := schema.TestResourceDataRaw(t, DataSourceVmJobs().Schema, map[string]interface{}{"vpc_id": "vpc-1", "name": "web"})
	diags := readVmJobsDataSource(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 1, d.Get("jobs.#"))
	assert.Equal(t, "job-1", d.Get("jobs.0.id"))
	assert.Equal(t, "2026-10-13T19:00:00Z", d.Get("jobs.0.next_run"))
	assert.Equal(t, "vm-1", d.Get("jobs.0.instance_ids.0"))
}

func TestVmRestorePointsDataSourceKeepsNaiveCreatedAt(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/restore_points/list": {{Body: `{"data":{"vm_id":"vm-1","vm_name":"web","restore_points":[
			{"vm_id":"vm-1","size":1.0,"created_at":"2026-10-11T19:00:00","is_incremental":true,"is_available":true,"backup_job_id":"job-1","backup_job_name":"web"},
			{"vm_id":"vm-1","size":2.0,"created_at":"2026-10-12T19:00:00","is_incremental":false,"is_available":true,"backup_job_id":"job-1","backup_job_name":"web"}]}}`}},
	})
	d := schema.TestResourceDataRaw(t, DataSourceVmRestorePoints().Schema, map[string]interface{}{"vpc_id": "vpc-1", "vm_id": "vm-1"})
	diags := readVmRestorePointsDataSource(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "2026-10-12T19:00:00", d.Get("restore_points.0.created_at"), "newest first, naive form kept for the restore resource")
	assert.Equal(t, 2.0, d.Get("restore_points.0.size"))
}

func TestVolumeJobsDataSource(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs": {{Body: volumeJobsWithPvcBody}},
	})
	d := schema.TestResourceDataRaw(t, DataSourceVolumeJobs().Schema, map[string]interface{}{"vpc_id": "vpc-1"})
	diags := readVolumeJobsDataSource(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 1, d.Get("jobs.#"), "VM jobs in the v2 list are left out")
	assert.Equal(t, "job-9", d.Get("jobs.0.id"))
	assert.Equal(t, "2026-10-13T19:00:00Z", d.Get("jobs.0.next_run"))
	assert.Equal(t, "vol-1", d.Get("jobs.0.volume_ids.0"))
}

func TestVolumeRestorePointsDataSource(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backups": {{Body: `{"data":{"volume_id":"vol-1","restore_points":[
			{"id":"rp-1","created_at":"2026-10-11T08:00:00","status":"available","is_available":true,"is_incremental":true,"size":0.4},
			{"id":"rp-2","created_at":"2026-10-12T08:00:00","status":"available","is_available":true,"is_incremental":false,"size":1.2}]}}`}},
	})
	d := schema.TestResourceDataRaw(t, DataSourceVolumeRestorePoints().Schema, map[string]interface{}{"vpc_id": "vpc-1", "volume_id": "vol-1"})
	diags := readVolumeRestorePointsDataSource(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "rp-2", d.Get("restore_points.0.id"))
	assert.Equal(t, "2026-10-12T08:00:00Z", d.Get("restore_points.0.created_at"))
}
