package fptcloud_backup_native

import (
	"encoding/json"
	"fmt"
	"time"

	common "terraform-provider-fptcloud/commons"
)

// historyPageSize is how many history rows a restore poll reads. Rows come
// back newest first, and a restore's row is written when it starts.
const historyPageSize = 50

// VmService calls the v1 OSP native backup user API.
type VmService interface {
	ListVmJobs(vpcId string) ([]VmJobListItem, error)
	CreateVmJob(vpcId string, p VmJobCreatePayload) (string, error)
	GetVmJob(vpcId string, jobId string) (*VmJobDetail, error)
	UpdateVmJob(vpcId string, jobId string, p VmJobUpdatePayload) (string, error)
	DeleteVmJob(vpcId string, jobId string) error
	SwitchVmJobSchedule(vpcId string, jobId string, enable bool) (string, error)
	GetTaskProgress(vpcId string, taskId string) (*TaskProgress, error)
	ListVmRestorePoints(vpcId string, vmId string) ([]VmRestorePoint, error)
	ListVmHistories(vpcId string, historyType string) ([]VmHistory, error)
	RestoreVm(vpcId string, p VmRestorePayload) error
	RestoreVmKeep(vpcId string, p VmRestoreKeepPayload) error
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
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVmList(vpcId))
	if err != nil {
		return nil, wrapAPIError(action, err)
	}
	var resp struct {
		Data []VmJobListItem `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	return resp.Data, nil
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

func (s *service) ListVmHistories(vpcId string, historyType string) ([]VmHistory, error) {
	const action = "listing backup histories"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVmHistories(vpcId, historyType, 1, historyPageSize))
	if err != nil {
		return nil, wrapAPIError(action, err)
	}
	var resp struct {
		Data []VmHistory `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	return resp.Data, nil
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

func (s *service) ListFloatingIps(vpcId string) ([]FloatingIpItem, error) {
	const action = "listing floating IPs"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeFloatingIps(vpcId))
	if err != nil {
		return nil, wrapAPIError(action, err)
	}
	var resp struct {
		Data []FloatingIpItem `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	return resp.Data, nil
}

var pollInterval = 5 * time.Second // moved to bkn_wait.go in Task 7
