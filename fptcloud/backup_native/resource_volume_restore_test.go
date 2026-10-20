package fptcloud_backup_native

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVolumeRestoreSchemaIsValid(t *testing.T) {
	assert.NoError(t, ResourceVolumeRestore().InternalValidate(nil, true))
}

func TestCreateVolumeRestore(t *testing.T) {
	setFastPolling(t)
	start := time.Now().UTC().Format(apiTimeLayout)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + volBase + "/volumes/vol-1/restore": {{Status: 202, Body: `{"message":"Request accepted","status":true}`}},
		"GET " + volBase + "/history/combined-history": {
			{Body: `{"items":[],"has_more":false}`},
			{Body: `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"pending","start_time":"` + start + `"}],"has_more":false}`},
			{Body: `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"success","start_time":"` + start + `","end_time":"` + start + `"}],"has_more":false}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVolumeRestoreSchema(), map[string]interface{}{
		"vpc_id": "vpc-1", "volume_id": "vol-1", "restore_point_id": "rp-1", "name": "pv-restored", "size_gb": 20,
	})

	diags := createVolumeRestore(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "success", d.Get("status"))
	assert.NotEmpty(t, d.Id())
	body := api.last("POST " + volBase + "/volumes/vol-1/restore").Body
	assert.Equal(t, "rp-1", body["backup_id"])
	assert.Equal(t, map[string]interface{}{"name": "pv-restored", "size": float64(20)}, body["metadata"])
}

func TestCreateVolumeRestoreIgnoresTheRowOfAnEarlierRestore(t *testing.T) {
	setFastPolling(t)
	earlier := time.Now().UTC().Add(-30 * time.Second).Format(apiTimeLayout)
	start := time.Now().UTC().Add(time.Second).Format(apiTimeLayout)
	old := `{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"success","start_time":"` + earlier + `","end_time":"` + earlier + `"}`
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + volBase + "/volumes/vol-1/restore": {{Status: 202, Body: `{"message":"Request accepted","status":true}`}},
		"GET " + volBase + "/history/combined-history": {
			{Body: `{"items":[` + old + `],"has_more":false}`},
			{Body: `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"pending","start_time":"` + start + `"},` + old + `],"has_more":false}`},
			{Body: `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"success","start_time":"` + start + `","end_time":"` + start + `"},` + old + `],"has_more":false}`},
		},
	})
	d := schema.TestResourceDataRaw(t, resourceVolumeRestoreSchema(), map[string]interface{}{
		"vpc_id": "vpc-1", "volume_id": "vol-1", "restore_point_id": "rp-1", "name": "pv-restored", "size_gb": 20,
	})
	diags := createVolumeRestore(context.Background(), d, client)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, formatAPITime(start), d.Get("started_at"))
}

func TestVolumeRestoreDescriptionWarnsAgainstConcurrentRestores(t *testing.T) {
	assert.Contains(t, ResourceVolumeRestore().Description, "Run one restore per volume at a time")
}

func TestVolumeRestoreNameValidation(t *testing.T) {
	validate := resourceVolumeRestoreSchema()["name"].ValidateFunc
	_, errs := validate("pv_restored", "name")
	assert.NotEmpty(t, errs)
	_, errs = validate("pv-restored", "name")
	assert.Empty(t, errs)
}
