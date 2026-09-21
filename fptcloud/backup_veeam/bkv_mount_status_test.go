package fptcloud_backup_veeam

import (
	"testing"

	common "terraform-provider-fptcloud/commons"

	"github.com/stretchr/testify/assert"
)

const mountListPath = "/v1/vmware/vpc/vpc-1/backup/vm-instant-recovery"

// The session list carries no restore point id, so a session started under a
// chosen name is recognised by that name.
func TestFindMountForRestorePointMatchesTheMountName(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		mountListPath: `{"data": [
			{"vm_mount_id": "mount-1", "vm_mount_name": "db-ir", "recovered_vm_name": "vm-db-01"},
			{"vm_mount_id": "mount-2", "vm_mount_name": "app-ir", "recovered_vm_name": "vm-app-01"}
		]}`,
	})
	defer server.Close()

	mount, err := NewBackupVeeamService(mockClient).FindMountForRestorePoint("vpc-1", "app-ir", "", "")
	assert.Nil(t, err)
	assert.NotNil(t, mount)
	assert.Equal(t, "mount-2", mount.VmMountId)
}

// An in-place session has no name of its own, so it is matched on the instance
// the restore point belongs to plus when the point was taken.
func TestFindMountForRestorePointMatchesTheInstanceAndTime(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		mountListPath: `{"data": [
			{"vm_mount_id": "mount-1", "recovered_vm_name": "vm-db-01", "restore_point_time": "2026-09-13T22:00:00"},
			{"vm_mount_id": "mount-2", "recovered_vm_name": "vm-db-01", "restore_point_time": "2026-09-15T15:02:18"},
			{"vm_mount_id": "mount-3", "recovered_vm_name": "vm-app-01", "restore_point_time": "2026-09-15T15:02:18"}
		]}`,
	})
	defer server.Close()

	mount, err := NewBackupVeeamService(mockClient).FindMountForRestorePoint(
		"vpc-1", "", "vm-db-01", "2026-09-15T15:02:18")
	assert.Nil(t, err)
	assert.NotNil(t, mount)
	assert.Equal(t, "mount-2", mount.VmMountId)
}

// backup_id must NOT be used for this, even though the mount and the restore
// point both have a field by that name. Measured on the dev backend: a session
// mounted from a point whose backup_id was 71cdb10b-... reported 331edde5-...
// on the mount. Matching on it finds nothing.
func TestFindMountForRestorePointIgnoresBackupId(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		mountListPath: `{"data": [
			{"vm_mount_id": "mount-1", "recovered_vm_name": "vm-db-01",
			 "restore_point_time": "2026-09-15T15:02:18", "backup_id": "331edde5-mount-side"}
		]}`,
	})
	defer server.Close()

	mount, err := NewBackupVeeamService(mockClient).FindMountForRestorePoint(
		"vpc-1", "", "vm-db-01", "2026-09-15T15:02:18")
	assert.Nil(t, err)
	assert.NotNil(t, mount, "the session has to be found even though its backup_id is a different id")
	assert.Equal(t, "331edde5-mount-side", mount.BackupId, "the mount's own backup_id is what migrate needs")
}

// When two sessions cannot be told apart, guessing would risk unmounting
// somebody else's session on the next destroy. It has to be an error.
func TestFindMountForRestorePointRefusesToGuess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		mountListPath: `{"data": [
			{"vm_mount_id": "mount-1", "recovered_vm_name": "vm-db-01", "restore_point_time": "2026-09-15T15:02:18"},
			{"vm_mount_id": "mount-2", "recovered_vm_name": "vm-db-01", "restore_point_time": "2026-09-15T15:02:18"}
		]}`,
	})
	defer server.Close()

	mount, err := NewBackupVeeamService(mockClient).FindMountForRestorePoint(
		"vpc-1", "", "vm-db-01", "2026-09-15T15:02:18")
	assert.Nil(t, mount)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot be told apart")
}

func TestFindMountForRestorePointReturnsNothingWhenAbsent(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		mountListPath: `{"data": []}`,
	})
	defer server.Close()

	mount, err := NewBackupVeeamService(mockClient).FindMountForRestorePoint(
		"vpc-1", "", "vm-db-01", "2026-09-15T15:02:18")
	assert.Nil(t, err)
	assert.Nil(t, mount)
}

// With neither a name nor the pair, there is nothing to match on - better to
// say so than to scan the list and pick something.
func TestFindMountForRestorePointNeedsSomethingToMatchOn(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		mountListPath: `{"data": [{"vm_mount_id": "mount-1"}]}`,
	})
	defer server.Close()

	_, err := NewBackupVeeamService(mockClient).FindMountForRestorePoint("vpc-1", "", "", "")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot identify")
}

func TestFlattenMounts(t *testing.T) {
	sessions := flattenMounts([]MountItem{{
		VmMountId:        "mount-1",
		VmMountName:      "db-ir",
		RecoveredVmName:  "db-01",
		VmId:             "vm-9",
		State:            "Mounted",
		Mode:             "Customized",
		RestorePointTime: "2026-09-14T22:00:07",
		BackupId:         "backup-1",
		ReadyMigrate:     true,
	}})

	assert.Len(t, sessions, 1)
	session := sessions[0].(map[string]interface{})
	assert.Equal(t, "mount-1", session["vm_mount_id"])
	assert.Equal(t, "db-ir", session["vm_mount_name"])
	assert.Equal(t, "db-01", session["recovered_vm_name"])
	assert.Equal(t, "vm-9", session["vm_id"])
	assert.Equal(t, "Customized", session["mode"])
	assert.Equal(t, true, session["ready_migrate"])
}
