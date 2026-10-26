package fptcloud_backup_native

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// historyPageSize is the page size of the history lists. Rows come back
// newest first, and a restore's row is written when it starts.
const historyPageSize = 50

// VmService calls the v1 OSP native backup user API.
type VmService interface {
	ListVmJobs(vpcId string) ([]VmJobListItem, error)
	CreateVmJob(vpcId string, p VmJobCreatePayload) (string, error)
	GetVmJob(vpcId string, jobId string) (*VmJobDetail, error)
	UpdateVmJob(vpcId string, jobId string, p VmJobUpdatePayload) (string, error)
	DeleteVmJob(vpcId string, jobId string) error
	SwitchVmJobSchedule(vpcId string, jobId string, enable bool) (string, error)
	StartVmJobBackup(vpcId string, jobId string) error
	GetTaskProgress(vpcId string, taskId string) (*TaskProgress, error)
	ListVmRestorePoints(vpcId string, vmId string) ([]VmRestorePoint, error)
	// DeleteAllVmRestorePoints starts deleting every restore point of the VM.
	// It returns a restorePointsBusy error while the API refuses the request.
	DeleteAllVmRestorePoints(vpcId string, vmId string) error
	// ListVmHistories returns the rows of historyType that started at or after
	// cutoff, newest first, plus at most one page of older rows.
	ListVmHistories(vpcId string, historyType string, cutoff time.Time) ([]VmHistory, error)
	RestoreVm(vpcId string, p VmRestorePayload) error
	RestoreVmKeep(vpcId string, p VmRestoreKeepPayload) error
	// SyncVmJobs asks the API to bring its records of the VPC's VM backups in
	// line with the platform. The sync runs in the background.
	SyncVmJobs(vpcId string) error
	ListFloatingIps(vpcId string) ([]FloatingIpItem, error)
}

type service struct {
	client *common.Client
}

func NewVmService(client *common.Client) VmService {
	return &service{client: client}
}

// decodeEnvelope parses {message, data, status}. v1 create reports a disabled
// service as HTTP 200 with status false (CreateBackupController.py:777-844),
// so status false is an error. A missing or null status is not.
func decodeEnvelope(raw []byte, action string) (*envelope, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	if env.Status != nil && !*env.Status {
		message := env.Message
		if message == "" {
			message = "the API rejected the request without giving a reason"
		}
		return &env, fmt.Errorf("%s failed: %s", action, describe(200, message))
	}
	return &env, nil
}

func taskIdFrom(raw []byte, action string) (string, error) {
	env, err := decodeEnvelope(raw, action)
	if err != nil {
		return "", err
	}
	var data struct {
		TaskId string `json:"task_id"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil || data.TaskId == "" {
		return "", fmt.Errorf("%s: the API did not return a task id: %s", action, string(raw))
	}
	return data.TaskId, nil
}

func (s *service) ListVmJobs(vpcId string) ([]VmJobListItem, error) {
	const action = "listing VM backup jobs"
	return common.ListAllPages(common.ListPageSize, func(page int) ([]VmJobListItem, int, error) {
		raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVmList(vpcId, page, common.ListPageSize))
		if err != nil {
			return nil, 0, wrapAPIError(action, err)
		}
		var resp struct {
			Data  []VmJobListItem `json:"data"`
			Total int             `json:"total"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, 0, fmt.Errorf("%s: could not parse the response: %w", action, err)
		}
		return resp.Data, resp.Total, nil
	}, func(job VmJobListItem) string { return job.Id })
}

func (s *service) CreateVmJob(vpcId string, p VmJobCreatePayload) (string, error) {
	const action = "creating the VM backup job"
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupNativeVmCreate(vpcId), p)
	if err != nil {
		return "", wrapAPIError(action, err)
	}
	env, err := decodeEnvelope(raw, action)
	if err != nil {
		return "", err
	}
	// data is a one-element list holding the Celery task id, not the job id.
	var taskIds []string
	if err := json.Unmarshal(env.Data, &taskIds); err != nil || len(taskIds) == 0 || taskIds[0] == "" {
		return "", fmt.Errorf("%s: the API did not return a task id: %s", action, string(raw))
	}
	return taskIds[0], nil
}

func (s *service) GetVmJob(vpcId string, jobId string) (*VmJobDetail, error) {
	const action = "reading the VM backup job"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVmDetail(vpcId, jobId))
	if err != nil {
		return nil, wrapAPIError(action, err)
	}
	env, err := decodeEnvelope(raw, action)
	if err != nil {
		return nil, err
	}
	// A missing or deleted job is HTTP 200 with body {} (BackupDetailController.py:70-75).
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil, nil
	}
	var job VmJobDetail
	if err := json.Unmarshal(env.Data, &job); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	if job.Id == "" {
		return nil, nil
	}
	return &job, nil
}

func (s *service) UpdateVmJob(vpcId string, jobId string, p VmJobUpdatePayload) (string, error) {
	const action = "updating the VM backup job"
	raw, err := s.client.SendPutRequest(common.ApiPath.BackupNativeVmUpdate(vpcId, jobId), p)
	if err != nil {
		return "", wrapAPIError(action, err)
	}
	return taskIdFrom(raw, action)
}

func (s *service) DeleteVmJob(vpcId string, jobId string) error {
	_, err := s.client.SendDeleteRequest(common.ApiPath.BackupNativeVmDelete(vpcId, jobId))
	if err != nil && !isNotFound(err) {
		return wrapAPIError("deleting the VM backup job", err)
	}
	return nil
}

func (s *service) SwitchVmJobSchedule(vpcId string, jobId string, enable bool) (string, error) {
	const action = "switching the VM backup job schedule"
	body := map[string]interface{}{"backup_id": jobId, "type": "backup", "enable": enable}
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupNativeVmSwitchSchedule(vpcId), body)
	if err != nil {
		return "", wrapAPIError(action, err)
	}
	return taskIdFrom(raw, action)
}

// StartVmJobBackup starts one run of the job outside its schedule. The API
// returns no task id; the job's status tracks the run.
func (s *service) StartVmJobBackup(vpcId string, jobId string) error {
	const action = "starting the VM backup job"
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupNativeVmStartBackup(vpcId), map[string]string{"resource_id": jobId})
	if err != nil {
		return wrapAPIError(action, err)
	}
	_, err = decodeEnvelope(raw, action)
	return err
}

func (s *service) GetTaskProgress(vpcId string, taskId string) (*TaskProgress, error) {
	const action = "reading the task progress"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVmTaskProgress(vpcId, taskId))
	if err != nil {
		return nil, wrapAPIError(action, err)
	}
	var progress TaskProgress
	if err := json.Unmarshal(raw, &progress); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	return &progress, nil
}

func (s *service) ListVmRestorePoints(vpcId string, vmId string) ([]VmRestorePoint, error) {
	const action = "listing the VM's restore points"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVmRestorePoints(vpcId, vmId))
	if err != nil {
		return nil, wrapAPIError(action, err)
	}
	env, err := decodeEnvelope(raw, action)
	if err != nil {
		return nil, err
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil, nil
	}
	var data struct {
		RestorePoints []VmRestorePoint `json:"restore_points"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	return data.RestorePoints, nil
}

// restorePointsBusy is the API refusing to delete restore points while a
// backup or restore of the VM runs, or while the job's AWX template or
// inventory is not ready (DeleteBackupController.py:513-522).
type restorePointsBusy struct{ error }

func (e restorePointsBusy) Unwrap() error { return e.error }

func (s *service) DeleteAllVmRestorePoints(vpcId string, vmId string) error {
	const action = "deleting the VM's restore points"
	raw, err := s.client.SendDeleteRequest(common.ApiPath.BackupNativeVmDeleteRestorePoints(vpcId, vmId))
	if err != nil {
		wrapped := wrapAPIError(action, err)
		if apiErr := asAPIError(err); apiErr != nil && apiErr.Code == 400 && strings.Contains(strings.ToLower(apiErr.Message), msgNotAvailableForDelete) {
			return restorePointsBusy{wrapped}
		}
		return wrapped
	}
	_, err = decodeEnvelope(raw, action)
	return err
}

func (s *service) ListVmHistories(vpcId string, historyType string, cutoff time.Time) ([]VmHistory, error) {
	const action = "listing backup histories"
	var rows []VmHistory
	for page := 1; page <= maxListPages; page++ {
		raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVmHistories(vpcId, historyType, page, historyPageSize))
		if err != nil {
			return nil, wrapAPIError(action, err)
		}
		var resp struct {
			Data  []VmHistory `json:"data"`
			Total int         `json:"total"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
		}
		rows = append(rows, resp.Data...)
		if len(resp.Data) < historyPageSize || page*historyPageSize >= resp.Total || olderThan(resp.Data[len(resp.Data)-1], cutoff) {
			return rows, nil
		}
	}
	return nil, fmt.Errorf("%s: the list did not end after %d pages", action, maxListPages)
}

// olderThan reports whether row started before cutoff. A row whose time does
// not parse is not older, so paging goes on past it.
func olderThan(row VmHistory, cutoff time.Time) bool {
	start, err := parseAPITime(row.StartTime)
	return err == nil && start.Before(cutoff)
}

func (s *service) RestoreVm(vpcId string, p VmRestorePayload) error {
	const action = "starting the VM restore"
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupNativeVmRestore(vpcId), p)
	if err != nil {
		return wrapAPIError(action, err)
	}
	_, err = decodeEnvelope(raw, action)
	return err
}

func (s *service) RestoreVmKeep(vpcId string, p VmRestoreKeepPayload) error {
	const action = "starting the VM restore to a new VM"
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupNativeVmRestoreKeep(vpcId), p)
	if err != nil {
		return wrapAPIError(action, err)
	}
	_, err = decodeEnvelope(raw, action)
	return err
}

// SyncVmJobs returns once the API has queued the sync; it does not wait for it.
func (s *service) SyncVmJobs(vpcId string) error {
	const action = "syncing the VM backups"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVmSync(vpcId))
	if err != nil {
		return wrapAPIError(action, err)
	}
	_, err = decodeEnvelope(raw, action)
	return err
}

// syncVmJobs syncs the VPC's VM backups once after an action the API
// accepted, so that the portal and the next read see what the action changed.
// The action itself is done, so a failed sync only warns. Defer it with the
// action's named diagnostics.
func syncVmJobs(svc VmService, vpcId string, diags *diag.Diagnostics) {
	if err := svc.SyncVmJobs(vpcId); err != nil {
		*diags = append(*diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("could not sync the VM backups of VPC %s", vpcId),
			Detail:   err.Error() + ". The portal syncs them again while its backup list is open.",
		})
	}
}

func (s *service) ListFloatingIps(vpcId string) ([]FloatingIpItem, error) {
	const action = "listing floating IPs"
	return common.ListAllPages(common.ListPageSize, func(page int) ([]FloatingIpItem, int, error) {
		raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeFloatingIps(vpcId, page, common.ListPageSize))
		if err != nil {
			return nil, 0, wrapAPIError(action, err)
		}
		var resp struct {
			Data  []FloatingIpItem `json:"data"`
			Total int              `json:"total"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, 0, fmt.Errorf("%s: could not parse the response: %w", action, err)
		}
		return resp.Data, resp.Total, nil
	}, func(ip FloatingIpItem) string { return ip.Id })
}
