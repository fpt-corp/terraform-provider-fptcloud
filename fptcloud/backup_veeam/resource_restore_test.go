package fptcloud_backup_veeam

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestResourceBackupVeeamRestoreShape(t *testing.T) {
	res := ResourceBackupVeeamRestore()
	assert.NotNil(t, res.CreateContext)
	assert.NotNil(t, res.ReadContext)
	assert.NotNil(t, res.DeleteContext)
	// A restore cannot be edited after the fact, so there must be no update
	// path and no importer.
	assert.Nil(t, res.UpdateContext)
	assert.Nil(t, res.Importer)
	assert.Nil(t, res.InternalValidate(nil, true))
}

// Every argument has to be ForceNew. Without that the resource would accept an
// edit and quietly do nothing, leaving state claiming a restore that never ran.
func TestRestoreArgumentsAreAllForceNew(t *testing.T) {
	for _, name := range []string{
		"vpc_id", "backup_job_id", "vm_id", "restore_point_id",
		"quick_rollback", "power_on_after_restore", "triggers",
	} {
		field, ok := resourceBackupVeeamRestoreSchema[name]
		assert.True(t, ok, "%s must exist", name)
		assert.True(t, field.ForceNew, "%s must be ForceNew", name)
	}
}

// Destroy must not call the API: a restore cannot be rolled back. It warns
// instead, so nobody reads "Destroy complete" as "the instance went back".
func TestRestoreDeleteOnlyForgetsAndWarns(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamRestoreSchema, map[string]interface{}{})
	d.SetId("point-1")

	diags := deleteBackupVeeamRestore(context.Background(), d, nil)

	assert.Equal(t, "", d.Id())
	assert.Len(t, diags, 1)
	assert.Equal(t, "A restore cannot be undone", diags[0].Summary)
}

// keep_original_vm is accepted by the API's model and then dropped by the
// handler, so sending it would look supported and do nothing. The payload must
// carry only the three fields the handler actually reads.
func TestRestorePayloadOmitsKeepOriginalVm(t *testing.T) {
	payload := RestorePayload{RestoreVmPointId: "point-1", QuickRollback: true}

	raw, err := json.Marshal(payload)
	assert.Nil(t, err)
	body := string(raw)
	assert.Contains(t, body, "restore_vm_point_id")
	assert.Contains(t, body, "quick_rollback")
	assert.Contains(t, body, "power_on_after_restore")
	assert.NotContains(t, body, "keep_original_vm")
}
