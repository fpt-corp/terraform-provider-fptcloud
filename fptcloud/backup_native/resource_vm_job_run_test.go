package fptcloud_backup_native

import (
	"context"
	"strings"
	"testing"
	"time"

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

// Review focus 2.
func TestCreateVmJobRunFailureKeepsTheId(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail":        jobStatuses("job-1", "Success", "Pending", "Failed"),
		"POST " + vmBase + "/start_backup": {{Body: startedBody}},
	})
	d := vmJobRunData(t)
	diags := createVmJobRun(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "may still be retried")
	assert.True(t, strings.HasPrefix(d.Id(), "job-1/"), "the id stays so Terraform taints the resource")
	assert.Equal(t, 1, api.count(vmSyncRoute), "a run that started syncs even when it fails")
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
