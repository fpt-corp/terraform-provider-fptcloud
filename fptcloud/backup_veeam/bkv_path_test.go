package fptcloud_backup_veeam_test

import (
	"testing"

	common "terraform-provider-fptcloud/commons"

	"github.com/stretchr/testify/assert"
)

func TestBackupVeeamApiPaths(t *testing.T) {
	vpcId := "vpc-1"
	jobId := "job-1"

	assert.Equal(t, "/v1/vmware/vpc/vpc-1/backup/jobs/create",
		common.ApiPath.BackupVeeamCreateJob(vpcId))
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/backup/jobs/job-1/update",
		common.ApiPath.BackupVeeamUpdateJob(vpcId, jobId))
	// detail dùng "job" SỐ ÍT - đây là bất thường của API, không phải lỗi gõ
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/backup/job/job-1/detail",
		common.ApiPath.BackupVeeamJobDetail(vpcId, jobId))
	// delete đặt id Ở CUỐI
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/backup/jobs/delete/job-1",
		common.ApiPath.BackupVeeamDeleteJob(vpcId, jobId))
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/backup/jobs?page=1&page_size=100&name=abc",
		common.ApiPath.BackupVeeamListJobs(vpcId, 1, 100, "abc", ""))
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/backup/jobs?page=1&page_size=100",
		common.ApiPath.BackupVeeamListJobs(vpcId, 1, 100, "", ""))
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/backup/instances?page=1&page_size=9999&not_backup=true&job_id=job-1",
		common.ApiPath.BackupVeeamInstances(vpcId, true, jobId, ""))
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/alert/alarm-notification/list?level=VPC",
		common.ApiPath.AlertNotificationMethods(vpcId, "VPC"))
}
