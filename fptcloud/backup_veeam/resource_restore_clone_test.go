package fptcloud_backup_veeam

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestResourceBackupVeeamRestoreCloneShape(t *testing.T) {
	res := ResourceBackupVeeamRestoreClone()
	assert.NotNil(t, res.CreateContext)
	assert.NotNil(t, res.ReadContext)
	assert.NotNil(t, res.DeleteContext)
	// Same reasoning as the plain restore: a restore that already happened can
	// neither be edited nor adopted.
	assert.Nil(t, res.UpdateContext)
	assert.Nil(t, res.Importer)
	assert.Nil(t, res.InternalValidate(nil, true))
}

func TestRestoreCloneArgumentsAreAllForceNew(t *testing.T) {
	for _, name := range []string{
		"vpc_id", "backup_job_id", "vm_id", "restore_point_id",
		"new_instance_name", "power_on_after_restore", "triggers",
	} {
		field, ok := resourceBackupVeeamRestoreCloneSchema[name]
		assert.True(t, ok, "%s must exist", name)
		assert.True(t, field.ForceNew, "%s must be ForceNew", name)
	}
}

// quick_rollback exists on the plain restore endpoint but not on this one, and
// new_instance_name is what this endpoint cannot work without.
func TestRestoreCloneSchemaMatchesTheEndpoint(t *testing.T) {
	_, hasQuickRollback := resourceBackupVeeamRestoreCloneSchema["quick_rollback"]
	assert.False(t, hasQuickRollback, "quick_rollback is not accepted by the clone endpoint")

	name := resourceBackupVeeamRestoreCloneSchema["new_instance_name"]
	assert.True(t, name.Required)
	assert.NotNil(t, name.StateFunc, "the server trims the name, so state has to store it trimmed")
	assert.Equal(t, "db-recovered", name.StateFunc("  db-recovered  "))
}

// keep_original_vm has to be sent, and has to be true. The boundary reads it as
// KeepOriginalVM and as the inverse of QuickRollback, so false here would turn
// this into an overwrite of the original instance.
func TestRestoreClonePayloadAlwaysKeepsTheOriginal(t *testing.T) {
	payload := RestoreClonePayload{
		RestoreVmPointId: "point-1",
		NewVmName:        "db-recovered",
		KeepOriginalVm:   true,
	}

	raw, err := json.Marshal(payload)
	assert.Nil(t, err)
	body := string(raw)
	assert.Contains(t, body, `"keep_original_vm":true`)
	assert.Contains(t, body, `"new_vm_name":"db-recovered"`)
	assert.Contains(t, body, "power_on_after_restore")
	assert.NotContains(t, body, "quick_rollback")
}

// Destroy must not delete the instance the restore created. Terraform never
// created it as a managed instance, and deleting somebody's instance on a
// destroy they ran to "untrack a restore" would be a nasty surprise.
func TestRestoreCloneDeleteKeepsTheNewInstanceAndWarns(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamRestoreCloneSchema, map[string]interface{}{
		"new_instance_name": "db-recovered",
	})
	d.SetId("point-1/db-recovered")

	diags := deleteBackupVeeamRestoreClone(context.Background(), d, nil)

	assert.Equal(t, "", d.Id())
	assert.Len(t, diags, 1)
	assert.Equal(t, "The restored instance was kept", diags[0].Summary)
	assert.Contains(t, diags[0].Detail, "db-recovered")
}

type stubInstanceService struct {
	BackupVeeamService
	instances []InstanceItem
	notBackup bool
}

func (s *stubInstanceService) ListInstances(_ string, notBackup bool, _ string, _ string) (InstanceListResponse, error) {
	s.notBackup = notBackup
	return InstanceListResponse{Data: s.instances, Total: len(s.instances)}, nil
}

func TestFindInstanceIdByNameIgnoresCaseAndWhitespace(t *testing.T) {
	svc := &stubInstanceService{instances: []InstanceItem{
		{Id: "vm-1", Name: "db-recovered"},
		{Id: "vm-2", Name: "app-01"},
	}}

	id, err := findInstanceIdByName(svc, "vpc-1", "  DB-Recovered ")
	assert.Nil(t, err)
	assert.Equal(t, "vm-1", id)

	// The lookup has to see every instance, not just the unprotected ones: a
	// name is taken whether or not the instance holding it is in a backup job.
	assert.False(t, svc.notBackup)
}

func TestFindInstanceIdByNameReturnsEmptyWhenAbsent(t *testing.T) {
	svc := &stubInstanceService{instances: []InstanceItem{{Id: "vm-2", Name: "app-01"}}}

	id, err := findInstanceIdByName(svc, "vpc-1", "db-recovered")
	assert.Nil(t, err)
	assert.Equal(t, "", id)

	taken, err := instanceNameTaken(svc, "vpc-1", "app-01")
	assert.Nil(t, err)
	assert.True(t, taken)
}
