package fptcloud_backup_native

import (
	"encoding/json"
	"fmt"
	"strings"

	common "terraform-provider-fptcloud/commons"
)

const (
	// volumeJobsPageSize is the page size of the v2 job list. The list counts
	// a multi-volume job once per volume, so loops stop on has_more, not total.
	volumeJobsPageSize = 100
	// maxListPages stops a paging loop that never reaches the end.
	maxListPages = 1000
)

// VolumeService calls the v2 Backup Volume user API.
type VolumeService interface {
	ListVolumeJobs(vpcId string) ([]VolumeJobListItem, error)
	CreateVolumeJob(vpcId string, p VolumeJobPayload) ([]VolumeFailure, error)
	GetVolumeJob(vpcId string, jobId string) (*VolumeJobDetail, error)
	UpdateVolumeJob(vpcId string, jobId string, p VolumeJobPayload) ([]VolumeFailure, error)
	DeleteVolumeJob(vpcId string, jobId string) error
	SetVolumeJobEnabled(vpcId string, jobId string, enabled bool) error
	ListVolumeRestorePoints(vpcId string, volumeId string) ([]VolumeRestorePoint, error)
	RestoreVolume(vpcId string, volumeId string, p VolumeRestorePayload) error
	ListVolumeHistoryPage(vpcId string, page int, pageSize int) (*VolumeHistoryPage, error)
}

func NewVolumeService(client *common.Client) VolumeService {
	return &service{client: client}
}

// volumeFailures decodes the per-volume failures that a create or update
// reports in data with HTTP 202 (some failed) or 500 (all failed).
func volumeFailures(raw []byte) []VolumeFailure {
	var env struct {
		Data []VolumeFailure `json:"data"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return nil
	}
	var failures []VolumeFailure
	for _, f := range env.Data {
		if f.VolumeId != "" || f.Error != "" {
			failures = append(failures, f)
		}
	}
	return failures
}

func formatVolumeFailures(failures []VolumeFailure) string {
	parts := make([]string, 0, len(failures))
	for _, f := range failures {
		parts = append(parts, fmt.Sprintf("volume %s (%s): %s", f.Name, f.VolumeId, f.Error))
	}
	return strings.Join(parts, "; ")
}

// sendVolumeJob runs a create or update and splits the result into
// "some volumes failed" (returned failures) and "the call failed" (error).
func (s *service) sendVolumeJob(action string, send func() ([]byte, error)) ([]VolumeFailure, error) {
	raw, err := send()
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			if failures := volumeFailures(apiErr.Body); len(failures) > 0 {
				return nil, fmt.Errorf("%s failed for every volume: %s", action, formatVolumeFailures(failures))
			}
		}
		return nil, wrapAPIError(action, err)
	}
	return volumeFailures(raw), nil
}

func (s *service) ListVolumeJobs(vpcId string) ([]VolumeJobListItem, error) {
	const action = "listing volume backup jobs"
	var jobs []VolumeJobListItem
	for page := 1; page <= maxListPages; page++ {
		raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVolumeJobs(vpcId, page, volumeJobsPageSize))
		if err != nil {
			return nil, wrapAPIError(action, err)
		}
		var resp struct {
			Data    []VolumeJobListItem `json:"data"`
			HasMore bool                `json:"has_more"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
		}
		// The list mixes v1 VM jobs (job_type 1) with volume jobs (job_type 0).
		for _, item := range resp.Data {
			if item.JobType == "0" {
				jobs = append(jobs, item)
			}
		}
		if !resp.HasMore {
			return jobs, nil
		}
	}
	return nil, fmt.Errorf("%s: stopped after %d pages without reaching the end", action, maxListPages)
}

func (s *service) CreateVolumeJob(vpcId string, p VolumeJobPayload) ([]VolumeFailure, error) {
	return s.sendVolumeJob("creating the volume backup job", func() ([]byte, error) {
		return s.client.SendPostRequest(common.ApiPath.BackupNativeVolumeJobCreate(vpcId), p)
	})
}

func (s *service) UpdateVolumeJob(vpcId string, jobId string, p VolumeJobPayload) ([]VolumeFailure, error) {
	return s.sendVolumeJob("updating the volume backup job", func() ([]byte, error) {
		return s.client.SendPutRequest(common.ApiPath.BackupNativeVolumeJob(vpcId, jobId), p)
	})
}

func (s *service) GetVolumeJob(vpcId string, jobId string) (*VolumeJobDetail, error) {
	const action = "reading the volume backup job"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVolumeJob(vpcId, jobId))
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, wrapAPIError(action, err)
	}
	env, err := decodeEnvelope(raw, action)
	if err != nil {
		return nil, err
	}
	var job VolumeJobDetail
	if err := json.Unmarshal(env.Data, &job); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	if job.Id == "" {
		return nil, nil
	}
	return &job, nil
}

func (s *service) DeleteVolumeJob(vpcId string, jobId string) error {
	_, err := s.client.SendDeleteRequest(common.ApiPath.BackupNativeVolumeJob(vpcId, jobId))
	if err != nil && !isNotFound(err) {
		return wrapAPIError("deleting the volume backup job", err)
	}
	return nil
}

func (s *service) SetVolumeJobEnabled(vpcId string, jobId string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}
	_, err := s.client.SendPostRequest(common.ApiPath.BackupNativeVolumeJobActions(vpcId, jobId), map[string]string{"action": action})
	if err != nil {
		return wrapAPIError(fmt.Sprintf("%sing the volume backup job schedule", strings.TrimSuffix(action, "e")), err)
	}
	return nil
}

func (s *service) ListVolumeRestorePoints(vpcId string, volumeId string) ([]VolumeRestorePoint, error) {
	const action = "listing the volume's restore points"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVolumeRestorePoints(vpcId, volumeId))
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
		RestorePoints []VolumeRestorePoint `json:"restore_points"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	return data.RestorePoints, nil
}

func (s *service) RestoreVolume(vpcId string, volumeId string, p VolumeRestorePayload) error {
	const action = "starting the volume restore"
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupNativeVolumeRestore(vpcId, volumeId), p)
	if err != nil {
		return wrapAPIError(action, err)
	}
	_, err = decodeEnvelope(raw, action)
	return err
}

func (s *service) ListVolumeHistoryPage(vpcId string, page int, pageSize int) (*VolumeHistoryPage, error) {
	const action = "listing backup histories"
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupNativeVolumeHistory(vpcId, page, pageSize))
	if err != nil {
		return nil, wrapAPIError(action, err)
	}
	var result VolumeHistoryPage
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("%s: could not parse the response: %w", action, err)
	}
	return &result, nil
}
