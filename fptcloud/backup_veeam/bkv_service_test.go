package fptcloud_backup_veeam_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	common "terraform-provider-fptcloud/commons"
	bkv "terraform-provider-fptcloud/fptcloud/backup_veeam"

	"github.com/stretchr/testify/assert"
)

func samplePayload() bkv.CreateJobPayload {
	return bkv.CreateJobPayload{
		Name:            "job-1",
		Enabled:         true,
		ScheduleEnabled: true,
		Retention:       bkv.RetentionPayload{Cycles: 7, LimitType: "Days"},
		VmIds:           []string{"vm-1"},
	}
}

// clientReturningStatus spins up a test server that answers with a specific
// HTTP status. It reuses the existing NewClientForTestingWithServer rather
// than adding a helper to commons/client.go: no need to touch shared code for
// something only the tests want.
func clientReturningStatus(t *testing.T, statusCode int, body string) (*common.Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(statusCode)
		_, _ = rw.Write([]byte(body))
	}))
	client, err := common.NewClientForTestingWithServer(server)
	assert.Nil(t, err)
	return client, server
}

func TestCreateJobSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/jobs/create": `{
			"status": true,
			"resource_id": "11111111-1111-1111-1111-111111111111",
			"backup_job_id": "11111111-1111-1111-1111-111111111111",
			"resource_name": "job-1"
		}`,
	})
	defer server.Close()

	resp, err := bkv.NewBackupVeeamService(mockClient).CreateJob("vpc-1", samplePayload())
	assert.Nil(t, err)
	assert.True(t, resp.Status)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", resp.BackupJobId)
}

// The API answers HTTP 200 with status:false when the job name is taken.
func TestCreateJobDuplicateNameReturnsError(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/jobs/create": `{
			"status": false,
			"message": "Job with the same name already exists"
		}`,
	})
	defer server.Close()

	_, err := bkv.NewBackupVeeamService(mockClient).CreateJob("vpc-1", samplePayload())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Job with the same name already exists")
}

// The business-error branch does NOT set "status" - only error_type. All seven
// codes must produce their own message rather than collapsing into one.
func TestCreateJobErrorTypesAllReturnDistinctMessages(t *testing.T) {
	cases := []struct {
		errorType string
		wantHint  string
	}{
		{"duplicateVm", "only one active backup job"},
		{"vmNotFound", "does not exist"},
		{"vmNotInVpc", "does not belong to this VPC"},
		{"reachLimitQuota", "quota"},
		{"jobNotEligible", "does not allow this operation"},
		{"requestInProgress", "already being processed"},
		{"idempotencyKeyReused", "idempotency"},
	}

	for _, c := range cases {
		t.Run(c.errorType, func(t *testing.T) {
			mockClient, server, _ := common.NewClientForTesting(map[string]string{
				"/v1/vmware/vpc/vpc-1/backup/jobs/create": `{
					"error_type": "` + c.errorType + `",
					"message": "server message"
				}`,
			})
			defer server.Close()

			_, err := bkv.NewBackupVeeamService(mockClient).CreateJob("vpc-1", samplePayload())
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "server message")
			assert.Contains(t, err.Error(), c.wantHint)
		})
	}
}

// Update checks quota while create does not, so this is a real error users hit.
func TestUpdateJobQuotaErrorReturnsError(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/jobs/job-1/update": `{
			"status": false,
			"message": "Limited backup quota"
		}`,
	})
	defer server.Close()

	_, err := bkv.NewBackupVeeamService(mockClient).UpdateJob("vpc-1", "job-1", samplePayload())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Limited backup quota")
}

// A VPC without the Backup Veeam service means the backend does not null-check
// the tenant and answers 500. The message must point at the real cause instead
// of leaving the user to guess.
func TestCreateJobServerErrorHintsAtServiceNotEnabled(t *testing.T) {
	mockClient, server := clientReturningStatus(t, http.StatusInternalServerError, `{"message": "Internal Server Error"}`)
	defer server.Close()

	_, err := bkv.NewBackupVeeamService(mockClient).CreateJob("vpc-1", samplePayload())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Backup Veeam")
}

func TestGetJobDetailMapsFields(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/job/job-1/detail": `{
			"data": {
				"id": "job-1",
				"name": "job-1",
				"description": "d",
				"schedule_enabled": true,
				"backup_object": [{"vm_id": "vm-1", "vm_display_name": "VM One"}],
				"backup_retention": {"cycles": 7, "limit_type": "Days"},
				"backup_schedule": {"schedule_type": "daily",
					"daily_schedule": {"enabled": true, "type": "Everyday", "run_at": "22:00:00"}},
				"notification_method_ids": [],
				"is_capacity_tier_enabled": false
			}
		}`,
	})
	defer server.Close()

	detail, err := bkv.NewBackupVeeamService(mockClient).GetJobDetail("vpc-1", "job-1")
	assert.Nil(t, err)
	assert.Equal(t, "job-1", detail.Name)
	assert.Len(t, detail.BackupObject, 1)
	assert.Equal(t, "vm-1", detail.BackupObject[0].VmId)
	assert.Equal(t, "daily", detail.BackupSchedule.ScheduleType)
}

// A deleted job returns no data from detail, so this is nil and NOT an error.
func TestGetJobDetailNotFoundReturnsNil(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/job/job-1/detail": `{"error": "can't get job detail."}`,
	})
	defer server.Close()

	detail, err := bkv.NewBackupVeeamService(mockClient).GetJobDetail("vpc-1", "job-1")
	assert.Nil(t, err)
	assert.Nil(t, detail)
}

func TestGetJobDetail404ReturnsNil(t *testing.T) {
	mockClient, server := clientReturningStatus(t, http.StatusNotFound, `{"error": "can't get job detail."}`)
	defer server.Close()

	detail, err := bkv.NewBackupVeeamService(mockClient).GetJobDetail("vpc-1", "job-1")
	assert.Nil(t, err)
	assert.Nil(t, detail)
}

// FindJobInList must fall back to an unfiltered scan when the job was renamed
// outside Terraform - otherwise it concludes the job is gone, recreates it, and
// hits duplicateVm.
func TestFindJobInListFallsBackWhenRenamed(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/jobs?page=1&page_size=1000&name=old-name": `{"items": [], "total_count": 0}`,
		"/v1/vmware/vpc/vpc-1/backup/jobs?page=1&page_size=1000":               `{"items": [{"id": "job-1", "name": "new-name", "status": "NOT_AVAILABLE", "enabled": true}], "total_count": 1}`,
	})
	defer server.Close()

	item, err := bkv.NewBackupVeeamService(mockClient).FindJobInList("vpc-1", "job-1", "old-name")
	assert.Nil(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, "new-name", item.Name)
	assert.Equal(t, "NOT_AVAILABLE", item.Status)
}

func TestFindJobInListReturnsNilWhenGone(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/jobs": `{"items": [], "total_count": 0}`,
	})
	defer server.Close()

	item, err := bkv.NewBackupVeeamService(mockClient).FindJobInList("vpc-1", "job-1", "old-name")
	assert.Nil(t, err)
	assert.Nil(t, item)
}

// Deleting an already-deleted job answers HTTP 500 because the backend does
// not null-check. That must NOT count as an error: "the job is gone" is exactly
// what Delete is asking for.
func TestDeleteJobTolerates500(t *testing.T) {
	mockClient, server := clientReturningStatus(t, http.StatusInternalServerError, `{"message": "Internal Server Error"}`)
	defer server.Close()

	err := bkv.NewBackupVeeamService(mockClient).DeleteJob("vpc-1", "job-1")
	assert.Nil(t, err)
}

func TestDeleteJobSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/jobs/delete/job-1": `{"status": true, "message": "success"}`,
	})
	defer server.Close()

	err := bkv.NewBackupVeeamService(mockClient).DeleteJob("vpc-1", "job-1")
	assert.Nil(t, err)
}

func TestListInstances(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/instances": `{
			"data": [{"id": "vm-1", "name": "VM One", "status": "POWERED_ON"}],
			"total": 1
		}`,
	})
	defer server.Close()

	resp, err := bkv.NewBackupVeeamService(mockClient).ListInstances("vpc-1", true, "", "")
	assert.Nil(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "vm-1", resp.Data[0].Id)
}

// The key must be DETERMINISTIC: only if a retry carries the same key can the
// backend recognise the same request and return the cached result instead of
// creating a second job.
func TestIdempotencyKeyIsDeterministic(t *testing.T) {
	first := bkv.BuildIdempotencyKey("vpc-1", samplePayload())
	second := bkv.BuildIdempotencyKey("vpc-1", samplePayload())
	assert.Equal(t, first, second)
	assert.NotEmpty(t, first)

	other := samplePayload()
	other.Name = "job-2"
	assert.NotEqual(t, first, bkv.BuildIdempotencyKey("vpc-1", other))
	assert.NotEqual(t, first, bkv.BuildIdempotencyKey("vpc-2", samplePayload()))
}
