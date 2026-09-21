package fptcloud_backup_veeam

import (
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

// historyPageSize is small on purpose: the history list is only ever read to
// find the newest row of one instance, and it comes back newest first.

type BackupVeeamService interface {
	CreateJob(vpcId string, payload CreateJobPayload) (JobMutationResponse, error)
	UpdateJob(vpcId string, jobId string, payload CreateJobPayload) (JobMutationResponse, error)
	GetJobDetail(vpcId string, jobId string) (*JobDetail, error)
	ListJobs(vpcId string, name string, status string) (JobListResponse, error)
	FindJobInList(vpcId string, jobId string, name string) (*JobListItem, error)
	DeleteJob(vpcId string, jobId string) error
	ListInstances(vpcId string, notBackup bool, jobId string, status string) (InstanceListResponse, error)

	ListRestoreGroups(vpcId string) (RestoreGroupListResponse, error)
	ListRestorePoints(vpcId string, jobId string, vmId string) (RestorePointListResponse, error)
	GetRestorePoint(vpcId string, jobId string, vmId string, restorePointId string) (*RestorePointItem, error)
	Restore(vpcId string, payload RestorePayload) (RestoreResponse, error)
	RestoreClone(vpcId string, payload RestoreClonePayload) (RestoreResponse, error)

	StartInstantRecovery(vpcId string, pointId string, payload InstantRecoveryPayload) (InstantRecoveryResponse, error)
	ListMounts(vpcId string) (MountListResponse, error)
	FindMountForRestorePoint(vpcId string, mountName string, recoveredVmName string, restorePointTime string) (*MountItem, error)
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
		"duplicateVm":          "an instance can belong to only one active backup job; use the fptcloud_backup_veeam_instances data source to list the instances still available",
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
	// No idempotency key is sent. A deterministic key - the only kind a
	// provider can produce, since it has no per-attempt handle from Terraform -
	// makes the backend answer a re-create of an identical configuration with
	// the PREVIOUS job's id without doing any work. That breaks
	// `terraform destroy` followed by `terraform apply`, and `-replace`, both of
	// which send a byte-identical payload. Verified against the dev backend.
	//
	// Nothing is lost that matters: job names are unique per VPC, so a retry
	// after a half-finished create is rejected with "Job with the same name
	// already exists" rather than silently creating a second job.
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

// --- Restore ---------------------------------------------------------------

func (s *backupVeeamServiceImpl) ListRestoreGroups(vpcId string) (RestoreGroupListResponse, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamRestoreGroups(vpcId, 1, listPageSize))
	if err != nil {
		return RestoreGroupListResponse{}, fmt.Errorf("listing the instances that have restore points failed: %v", err)
	}

	var result RestoreGroupListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return RestoreGroupListResponse{}, fmt.Errorf("could not parse the restore group list: %v", err)
	}
	return result, nil
}

func (s *backupVeeamServiceImpl) ListRestorePoints(vpcId string, jobId string, vmId string) (RestorePointListResponse, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamRestorePoints(vpcId, jobId, vmId))
	if err != nil {
		return RestorePointListResponse{}, fmt.Errorf("listing restore points failed: %v", err)
	}

	var result RestorePointListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return RestorePointListResponse{}, fmt.Errorf("could not parse the restore point list: %v", err)
	}
	return result, nil
}

// GetRestorePoint finds one restore point. There is no endpoint that reads a
// restore point by id on its own, so this filters the job's list - which is
// also the only place the point's status can be read.
func (s *backupVeeamServiceImpl) GetRestorePoint(vpcId string, jobId string, vmId string, restorePointId string) (*RestorePointItem, error) {
	list, err := s.ListRestorePoints(vpcId, jobId, vmId)
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].Id == restorePointId {
			return &list.Items[i], nil
		}
	}
	return nil, nil
}

func (s *backupVeeamServiceImpl) Restore(vpcId string, payload RestorePayload) (RestoreResponse, error) {
	// The id goes in the path as well as the body. The backend reads only the
	// body, but the portal sends both and this keeps the two callers identical.
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupVeeamRestore(vpcId, payload.RestoreVmPointId), payload)
	if err != nil {
		return RestoreResponse{}, decorateServerError("restoring the instance", err)
	}

	var result RestoreResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return RestoreResponse{}, fmt.Errorf("could not parse the restore response: %v", err)
	}

	// Same HTTP-200-on-failure shape as the job endpoints: a rejection arrives
	// as 200 with status:false, or with error_type set and no status at all.
	// The success path answers with resource_id and no status field, so an
	// empty error_type plus an empty message counts as accepted.
	if result.ErrorType != "" {
		return result, fmt.Errorf("%s", describeErrorType(result.ErrorType, result.Message))
	}
	if !result.Status && result.ResourceId == "" {
		message := result.Message
		if message == "" {
			message = "the API rejected the restore without giving a reason"
		}
		return result, fmt.Errorf("%s", message)
	}
	return result, nil
}

// RestoreClone runs a "Restore keep": the restore point is brought back as a
// new instance and the original is left alone.
//
// The response shape is identical to a plain restore, including the fact that
// the success path carries no status field, so the same three-way check
// applies.
func (s *backupVeeamServiceImpl) RestoreClone(vpcId string, payload RestoreClonePayload) (RestoreResponse, error) {
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupVeeamRestoreClone(vpcId, payload.RestoreVmPointId), payload)
	if err != nil {
		return RestoreResponse{}, decorateServerError("restoring the instance to a new one", err)
	}

	var result RestoreResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return RestoreResponse{}, fmt.Errorf("could not parse the restore response: %v", err)
	}

	if result.ErrorType != "" {
		return result, fmt.Errorf("%s", describeErrorType(result.ErrorType, result.Message))
	}
	if !result.Status && result.ResourceId == "" {
		message := result.Message
		if message == "" {
			message = "the API rejected the restore without giving a reason"
		}
		return result, fmt.Errorf("%s", message)
	}
	return result, nil
}

// StartInstantRecovery mounts a backup as a new instance that runs straight
// from it.
//
// Unlike restore, this response DOES carry status on the success path, along
// with a history_id.
func (s *backupVeeamServiceImpl) StartInstantRecovery(vpcId string, pointId string, payload InstantRecoveryPayload) (InstantRecoveryResponse, error) {
	action := "starting the instant recovery session"

	raw, err := s.client.SendPostRequest(common.ApiPath.BackupVeeamInstantRecoveryClone(vpcId, pointId), payload)
	if err != nil {
		return InstantRecoveryResponse{}, decorateServerError(action, err)
	}

	var result InstantRecoveryResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return InstantRecoveryResponse{}, fmt.Errorf("could not parse the instant recovery response: %v", err)
	}

	if result.ErrorType != "" {
		return result, fmt.Errorf("%s", describeErrorType(result.ErrorType, result.Message))
	}
	if !result.Status {
		message := result.Message
		if message == "" {
			message = "the API rejected the instant recovery without giving a reason"
		}
		return result, fmt.Errorf("%s", message)
	}
	return result, nil
}

// ListMounts lists the instant recovery sessions of the whole VPC.
func (s *backupVeeamServiceImpl) ListMounts(vpcId string) (MountListResponse, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamMounts(vpcId))
	if err != nil {
		return MountListResponse{}, fmt.Errorf("listing instant recovery sessions failed: %v", err)
	}

	var result MountListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return MountListResponse{}, fmt.Errorf("could not parse the instant recovery session list: %v", err)
	}
	return result, nil
}

// FindMountForRestorePoint identifies the session this provider just started.
//
// The list carries no restore point id, so the session has to be recognised
// from what it does carry:
//
//   - A session started under a chosen name is found by that name, which is
//     unique because an instance name has to be.
//   - A session mounted in place has no name of its own, so it is matched on
//     the pair (recovered_vm_name, restore_point_time) - the instance the
//     restore point belongs to, and when the point was taken. Both values come
//     straight off the restore point, and restore_point_time uses the same
//     format as the point's restore_at.
//
// NOT backup_id, even though both the mount and the restore point have a field
// by that name: measured against the dev backend, a session mounted from a
// point whose backup_id was 71cdb10b-... reported 331edde5-... on the mount.
// They are different ids that happen to share a name.
//
// Matching several sessions is reported as an error rather than guessed at: the
// id picked here is what a later `terraform destroy` unmounts, and unmounting
// somebody else's session discards their data.
func (s *backupVeeamServiceImpl) FindMountForRestorePoint(vpcId string, mountName string, recoveredVmName string, restorePointTime string) (*MountItem, error) {
	if mountName == "" && (recoveredVmName == "" || restorePointTime == "") {
		return nil, fmt.Errorf("cannot identify the instant recovery session: the restore point reports neither an " +
			"instance name nor a timestamp to match it against")
	}

	list, err := s.ListMounts(vpcId)
	if err != nil {
		return nil, err
	}

	var candidates []MountItem
	for _, item := range list.Data {
		if mountName != "" {
			if item.VmMountName == mountName {
				candidates = append(candidates, item)
			}
			continue
		}
		if item.RecoveredVmName == recoveredVmName && item.RestorePointTime == restorePointTime {
			candidates = append(candidates, item)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}
	if len(candidates) == 1 {
		return &candidates[0], nil
	}
	return nil, fmt.Errorf(
		"found %d instant recovery sessions that all look like this one, so the right one cannot be told apart. "+
			"Check the Instant Recovery tab in the portal and stop the sessions that are not wanted, then apply again",
		len(candidates))
}
