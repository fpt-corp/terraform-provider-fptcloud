package fptcloud_backup_veeam

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

// Instant recovery is an action, the same shape as the two restore resources:
// no update path, no importer, and nothing that waits.
func TestResourceInstantRecoveryShape(t *testing.T) {
	res := ResourceBackupVeeamInstantRecovery()

	assert.NotNil(t, res.CreateContext)
	assert.NotNil(t, res.ReadContext)
	assert.NotNil(t, res.DeleteContext)
	assert.Nil(t, res.UpdateContext)
	assert.Nil(t, res.Importer)
	// No CustomizeDiff and no Timeouts: there is no transition to validate
	// and nothing to wait for.
	assert.Nil(t, res.CustomizeDiff)
	assert.Nil(t, res.Timeouts)
	assert.Nil(t, res.InternalValidate(nil, true))
}

// Every argument starts a new session, and the mounted instance always gets a
// name of its own - there is no in-place variant.
func TestInstantRecoverySchema(t *testing.T) {
	name, ok := resourceBackupVeeamInstantRecoverySchema["new_instance_name"]
	assert.True(t, ok, "the session always mounts under a name")
	assert.True(t, name.Required)
	assert.True(t, name.ForceNew)

	for _, field := range []string{
		"vpc_id", "backup_job_id", "vm_id", "restore_point_id",
		"power_up", "nics_enabled", "vm_tags_restore_enabled",
	} {
		s, ok := resourceBackupVeeamInstantRecoverySchema[field]
		assert.True(t, ok, "%s must exist", field)
		assert.True(t, s.ForceNew, "%s must be ForceNew", field)
	}

	// The session lifecycle is not modelled at all, and the session type is
	// not configurable.
	for _, gone := range []string{"migrate", "migrated", "type"} {
		_, exists := resourceBackupVeeamInstantRecoverySchema[gone]
		assert.False(t, exists, "%s must not be part of the schema", gone)
	}
}

// The payload always asks for a Customized session with a destination: the
// controller reads destination.restored_vm_name without checking, and the
// in-place type is not offered at all because the portal hides it.
func TestInstantRecoveryPayloadShape(t *testing.T) {
	payload := InstantRecoveryPayload{
		RestorePointId: "point-1",
		Type:           InstantRecoveryTypeCustomized,
		Destination:    &InstantRecoveryDestination{RestoredVmName: "db-ir"},
	}

	raw, err := json.Marshal(payload)
	assert.Nil(t, err)
	assert.Contains(t, string(raw), `"type":"Customized"`)
	assert.Contains(t, string(raw), `"restored_vm_name":"db-ir"`)
	// destination_host and datastore are dropped by the backend, so they must
	// not be part of the payload at all.
	assert.NotContains(t, string(raw), "destination_host")
	assert.NotContains(t, string(raw), "datastore")
}

type stubMountService struct {
	BackupVeeamService
	mounts []MountItem
	points []RestorePointItem
}

func (s *stubMountService) ListMounts(_ string) (MountListResponse, error) {
	return MountListResponse{Data: s.mounts}, nil
}

func (s *stubMountService) GetRestorePoint(_ string, _ string, _ string, id string) (*RestorePointItem, error) {
	for i := range s.points {
		if s.points[i].Id == id {
			return &s.points[i], nil
		}
	}
	return nil, nil
}

func (s *stubMountService) FindMountForRestorePoint(_ string, mountName string, recoveredVmName string, restorePointTime string) (*MountItem, error) {
	for i := range s.mounts {
		if mountName != "" && s.mounts[i].VmMountName == mountName {
			return &s.mounts[i], nil
		}
		if mountName == "" && s.mounts[i].RecoveredVmName == recoveredVmName && s.mounts[i].RestorePointTime == restorePointTime {
			return &s.mounts[i], nil
		}
	}
	return nil, nil
}

func instantRecoveryData(t *testing.T, values map[string]interface{}) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamInstantRecoverySchema, values)
	d.SetId("point-1/db-ir")
	return d
}

func TestReadReportsTheSessionWhenItIsThere(t *testing.T) {
	d := instantRecoveryData(t, map[string]interface{}{
		"vpc_id":            "vpc-1",
		"restore_point_id":  "point-1",
		"new_instance_name": "db-ir",
	})
	svc := &stubMountService{
		points: []RestorePointItem{{Id: "point-1", VmDisplayName: "vm-db-01", RestoreAt: "2026-09-15T15:02:18"}},
		mounts: []MountItem{{
			VmMountId: "951fa975", VmMountName: "db-ir", VmId: "vm-9",
			State: "Mounted", Mode: "Customized", BackupId: "backup-1",
			RestorePointTime: "2026-09-15T15:02:18", ReadyMigrate: true,
		}},
	}

	diags := readInstantRecoveryWith(svc, d)

	assert.False(t, diags.HasError())
	assert.Equal(t, "951fa975", d.Get("mount_id"))
	assert.Equal(t, "Mounted", d.Get("state"))
	assert.Equal(t, "Customized", d.Get("mode"))
	assert.True(t, d.Get("ready_migrate").(bool))
}

// A session that cannot be found is NOT drift: it may not have mounted yet, or
// somebody may have migrated or stopped it. Clearing the id for any of those
// would make the next apply mount the backup a second time.
func TestReadKeepsTheIdWhenNoSessionIsFound(t *testing.T) {
	d := instantRecoveryData(t, map[string]interface{}{
		"vpc_id":            "vpc-1",
		"restore_point_id":  "point-1",
		"new_instance_name": "db-ir",
	})
	svc := &stubMountService{
		points: []RestorePointItem{{Id: "point-1", VmDisplayName: "vm-db-01", RestoreAt: "2026-09-15T15:02:18"}},
	}

	diags := readInstantRecoveryWith(svc, d)

	assert.False(t, diags.HasError())
	assert.Equal(t, "point-1/db-ir", d.Id())
	assert.Equal(t, "", d.Get("mount_id"))
}

// Destroy must not stop the session: whatever was written to the mounted
// instance would be thrown away without anybody asking for it. It warns
// instead, and the warning has to say what the open session costs.
func TestDeleteLeavesTheSessionOpenAndWarns(t *testing.T) {
	d := instantRecoveryData(t, map[string]interface{}{
		"vpc_id":            "vpc-1",
		"new_instance_name": "db-ir",
	})

	diags := deleteInstantRecovery(context.Background(), d, nil)

	assert.Equal(t, "", d.Id())
	assert.Len(t, diags, 1)
	assert.Equal(t, "The instant recovery session was left open", diags[0].Summary)
	assert.Contains(t, diags[0].Detail, "db-ir")
	assert.Contains(t, diags[0].Detail, "backup job")
}
