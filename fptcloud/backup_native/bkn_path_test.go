package fptcloud_backup_native

import (
	"testing"

	common "terraform-provider-fptcloud/commons"

	"github.com/stretchr/testify/assert"
)

func TestBackupNativePaths(t *testing.T) {
	p := common.ApiPath
	cases := map[string]string{
		p.BackupNativeVmList("v", 2, 25):                      "/v1/vmware/vpc/v/osp_native_backup/list?page=2&page_size=25",
		p.BackupNativeVmCreate("v"):                           "/v1/vmware/vpc/v/osp_native_backup/create",
		p.BackupNativeVmDetail("v", "j"):                      "/v1/vmware/vpc/v/osp_native_backup/detail?backup_id=j",
		p.BackupNativeVmUpdate("v", "j"):                      "/v1/vmware/vpc/v/osp_native_backup/j/update",
		p.BackupNativeVmDelete("v", "j"):                      "/v1/vmware/vpc/v/osp_native_backup/j/delete",
		p.BackupNativeVmSwitchSchedule("v"):                   "/v1/vmware/vpc/v/osp_native_backup/switch_schedule",
		p.BackupNativeVmStartBackup("v"):                      "/v1/vmware/vpc/v/osp_native_backup/start_backup",
		p.BackupNativeVmTaskProgress("v", "t"):                "/v1/vmware/vpc/v/osp_native_backup/task/t/progress",
		p.BackupNativeVmRestorePoints("v", "m"):               "/v1/vmware/vpc/v/osp_native_backup/restore_points/list?vm_id=m",
		p.BackupNativeVmHistories("v", "Restore keep", 1, 50): "/v1/vmware/vpc/v/osp_native_backup/histories/list?type=Restore+keep&page=1&page_size=50",
		p.BackupNativeVmRestore("v"):                          "/v1/vmware/vpc/v/osp_native_backup/restore",
		p.BackupNativeVmRestoreKeep("v"):                      "/v1/vmware/vpc/v/osp_native_backup/restore-new",
		p.BackupNativeFloatingIps("v", 2, 25):                 "/v1/vmware/vpc/v/ip-address?page=2&page_size=25&is_all=false",
		p.BackupNativeVolumeJobs("v", 2, 100):                 "/v2/vpc/v/backup-volume/backup-jobs?page=2&page_size=100",
		p.BackupNativeVolumeJobCreate("v"):                    "/v2/vpc/v/backup-volume/backup-jobs",
		p.BackupNativeVolumeJob("v", "j"):                     "/v2/vpc/v/backup-volume/backup-jobs/j",
		p.BackupNativeVolumeJobActions("v", "j"):              "/v2/vpc/v/backup-volume/backup-jobs/j/actions",
		p.BackupNativeVolumeJobExecutions("v", "j"):           "/v2/vpc/v/backup-volume/backup-jobs/j/executions",
		p.BackupNativeVolumeRestorePoints("v", "o"):           "/v2/vpc/v/backup-volume/backups?volume_id=o",
		p.BackupNativeVolumeRestore("v", "o"):                 "/v2/vpc/v/backup-volume/volumes/o/restore",
		p.BackupNativeVolumeHistory("v", 3, 50):               "/v2/vpc/v/backup-volume/history/combined-history?page=3&page_size=50",
	}
	for got, want := range cases {
		assert.Equal(t, want, got)
	}
}
