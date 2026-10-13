package fptcloud_backup_native

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVmRestoreSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVmRestore().InternalValidate(nil, true))
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
}

func TestCreateVmRestoreFailureLeavesNoState(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + vmBase + "/restore": {{Status: 202, Body: `{"status":true}`}},
		"GET " + vmBase + "/histories/list": {
			{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"Failed","reason":"disk error","start_time":"` + start + `"}]}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVmRestoreSchema(), map[string]interface{}{
		"vpc_id": "vpc-1", "vm_id": "vm-1", "restore_point_created_at": "2026-10-12T19:00:00", "mode": "overwrite",
	})
	diags := createVmRestore(context.Background(), d, client)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "disk error")
	assert.Equal(t, "", d.Id())
}
