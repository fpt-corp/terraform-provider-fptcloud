package fptcloud_backup_veeam

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	common "terraform-provider-fptcloud/commons"
)

// listPageSize is the page_size used for every list call. It is large enough
// that pagination never kicks in for a real tenant. If a tenant ever exceeds
// it, FindJobInList stops seeing the job and Terraform reports drift, which is
// an acceptable failure mode for this version.
const listPageSize = 1000

type BackupVeeamService interface {
	CreateJob(vpcId string, payload CreateJobPayload) (JobMutationResponse, error)
	UpdateJob(vpcId string, jobId string, payload CreateJobPayload) (JobMutationResponse, error)
	GetJobDetail(vpcId string, jobId string) (*JobDetail, error)
	ListJobs(vpcId string, name string, status string) (JobListResponse, error)
	FindJobInList(vpcId string, jobId string, name string) (*JobListItem, error)
	DeleteJob(vpcId string, jobId string) error
	ListInstances(vpcId string, notBackup bool, jobId string, status string) (InstanceListResponse, error)
}

type backupVeeamServiceImpl struct {
	client *common.Client
}

func NewBackupVeeamService(client *common.Client) BackupVeeamService {
	return &backupVeeamServiceImpl{client: client}
}

// describeErrorType turns the API's error_type into a message that tells the
// user what to do about it. The seven codes come from the backend's
// BackupErrorType class.
func describeErrorType(errorType string, message string) string {
	hints := map[string]string{
		"duplicateVm":          "an instance can belong to only one active backup job; use the fptcloud_backup_veeam_instances data source with not_backup = true to list the instances still available",
		"vmNotFound":           "the instance does not exist; check vm_ids",
		"vmNotInVpc":           "the instance does not belong to this VPC; check vpc_id and vm_ids",
		"reachLimitQuota":      "the tenant's backup quota is exhausted; contact FPT Cloud to raise it, Terraform cannot resolve this",
		"jobNotEligible":       "the job is in a state that does not allow this operation; wait for it to settle and apply again",
		"requestInProgress":    "an identical request is already being processed; run terraform apply again in a few minutes",
		"idempotencyKeyReused": "the idempotency key was already used with a different request body (a provider bug, not a problem with your configuration)",
	}

	if hint, ok := hints[errorType]; ok {
		return fmt.Sprintf("%s (%s): %s", message, errorType, hint)
	}
	return fmt.Sprintf("%s (%s)", message, errorType)
}

// checkMutationResponse is the single guard against the API returning HTTP 200
// on failure. It checks error_type BEFORE status, because the business-error
// branch does not set status at all - the zero value false would still catch
// it, but error_type gives a far better message.
func checkMutationResponse(resp JobMutationResponse) error {
	if resp.ErrorType != "" {
		return fmt.Errorf("%s", describeErrorType(resp.ErrorType, resp.Message))
	}
	if !resp.Status {
		message := resp.Message
		if message == "" {
			message = "the API rejected the request without giving a reason"
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

// BuildIdempotencyKey derives a key DETERMINISTICALLY from the request body.
// It must not be a random UUID: a retry has to carry the same key for the
// backend to recognise it as the same request and return the cached result
// instead of creating a second job.
func BuildIdempotencyKey(vpcId string, payload CreateJobPayload) string {
	// The key must not depend on itself, so clear the field before hashing.
	payload.IdempotencyKey = ""

	body, err := json.Marshal(payload)
	if err != nil {
		body = []byte(payload.Name)
	}
	sum := sha256.Sum256(append([]byte(vpcId+"|"), body...))
	return "tf-" + hex.EncodeToString(sum[:16])
}

// decorateServerError adds a hint for HTTP 500 on create and update. The
// backend does not null-check the tenant, so a VPC without the Backup Veeam
// service returns 500 with a meaningless body - and this is the first error a
// new user runs into.
func decorateServerError(action string, err error) error {
	if strings.Contains(err.Error(), "500") {
		return fmt.Errorf("%s failed (HTTP 500): %v - check whether the Backup Veeam service is enabled for this VPC", action, err)
	}
	return fmt.Errorf("%s failed: %v", action, err)
}

func (s *backupVeeamServiceImpl) CreateJob(vpcId string, payload CreateJobPayload) (JobMutationResponse, error) {
	payload.IdempotencyKey = BuildIdempotencyKey(vpcId, payload)

	raw, err := s.client.SendPostRequest(common.ApiPath.BackupVeeamCreateJob(vpcId), payload)
	if err != nil {
		return JobMutationResponse{}, decorateServerError("creating the backup job", err)
	}

	var result JobMutationResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return JobMutationResponse{}, fmt.Errorf("could not parse the create backup job response: %v", err)
	}
	if err := checkMutationResponse(result); err != nil {
		return result, err
	}
	return result, nil
}

func (s *backupVeeamServiceImpl) UpdateJob(vpcId string, jobId string, payload CreateJobPayload) (JobMutationResponse, error) {
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupVeeamUpdateJob(vpcId, jobId), payload)
	if err != nil {
		return JobMutationResponse{}, decorateServerError("updating the backup job", err)
	}

	var result JobMutationResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return JobMutationResponse{}, fmt.Errorf("could not parse the update backup job response: %v", err)
	}
	if err := checkMutationResponse(result); err != nil {
		return result, err
	}
	return result, nil
}

func (s *backupVeeamServiceImpl) GetJobDetail(vpcId string, jobId string) (*JobDetail, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamJobDetail(vpcId, jobId))
	if err != nil {
		// A 404 means the job is gone. That is drift, not an error.
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, fmt.Errorf("reading the backup job failed: %v", err)
	}

	var result JobDetailResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("could not parse the backup job detail: %v", err)
	}
	// The blueprint answers {"error": "can't get job detail."} - usually with a
	// 404, but guard against it arriving as a 200 with an empty body too.
	if result.Data.Id == "" {
		return nil, nil
	}
	return &result.Data, nil
}

func (s *backupVeeamServiceImpl) ListJobs(vpcId string, name string, status string) (JobListResponse, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamListJobs(vpcId, 1, listPageSize, name, status))
	if err != nil {
		return JobListResponse{}, fmt.Errorf("listing backup jobs failed: %v", err)
	}

	var result JobListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return JobListResponse{}, fmt.Errorf("could not parse the backup job list: %v", err)
	}
	return result, nil
}

// FindJobInList looks a job up by ID through the list endpoint, which is the
// ONLY source of status and enabled - the detail endpoint returns neither.
//
// It filters by name first because that is cheaper, but when the job is not
// found it MUST rescan without the filter: the job may have been renamed in
// the portal. Without that second pass Terraform concludes the job is gone,
// recreates it, and walks straight into a duplicateVm error.
func (s *backupVeeamServiceImpl) FindJobInList(vpcId string, jobId string, name string) (*JobListItem, error) {
	if name != "" {
		list, err := s.ListJobs(vpcId, name, "")
		if err != nil {
			return nil, err
		}
		if item := pickJobById(list.Items, jobId); item != nil {
			return item, nil
		}
	}

	list, err := s.ListJobs(vpcId, "", "")
	if err != nil {
		return nil, err
	}
	return pickJobById(list.Items, jobId), nil
}

func pickJobById(items []JobListItem, jobId string) *JobListItem {
	for i := range items {
		if items[i].Id == jobId {
			return &items[i]
		}
	}
	return nil
}

// DeleteJob ignores HTTP 500: the backend reads backup_job.name without a
// null check, so deleting an already-deleted job answers 500 rather than 404.
// As far as Terraform is concerned, "the job is gone" is exactly the outcome
// Delete is asking for.
func (s *backupVeeamServiceImpl) DeleteJob(vpcId string, jobId string) error {
	raw, err := s.client.SendDeleteRequest(common.ApiPath.BackupVeeamDeleteJob(vpcId, jobId))
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "500") {
			return nil
		}
		return fmt.Errorf("deleting the backup job failed: %v", err)
	}

	var result JobMutationResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		// The delete succeeded but the body is unexpected. Not an error.
		return nil
	}
	return checkMutationResponse(result)
}

func (s *backupVeeamServiceImpl) ListInstances(vpcId string, notBackup bool, jobId string, status string) (InstanceListResponse, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamInstances(vpcId, notBackup, jobId, status))
	if err != nil {
		return InstanceListResponse{}, fmt.Errorf("listing the instances available for a backup job failed: %v", err)
	}

	var result InstanceListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return InstanceListResponse{}, fmt.Errorf("could not parse the instance list: %v", err)
	}
	return result, nil
}
