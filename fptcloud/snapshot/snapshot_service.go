package fptcloud_snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	common "terraform-provider-fptcloud/commons"
)

// SnapshotService defines the interface for the instance snapshot service
type SnapshotService interface {
	// CreateSnapshot returns the recorded snapshot, still being created.
	CreateSnapshot(vpcId string, createdModel CreateSnapshotDTO) (*Snapshot, error)
	// GetSnapshot returns nil, nil when the snapshot no longer exists.
	GetSnapshot(vpcId string, snapshotId string) (*Snapshot, error)
	// DeleteSnapshot is a no-op when the snapshot is already gone.
	DeleteSnapshot(vpcId string, snapshotId string) error
}

// SnapshotServiceImpl is the implementation of SnapshotService
type SnapshotServiceImpl struct {
	client *common.Client
}

// NewSnapshotService creates a new instance snapshot service with the given client
func NewSnapshotService(client *common.Client) SnapshotService {
	return &SnapshotServiceImpl{client: client}
}

// CreateSnapshot creates a new snapshot of an instance
func (s *SnapshotServiceImpl) CreateSnapshot(vpcId string, createdModel CreateSnapshotDTO) (*Snapshot, error) {
	apiPath := common.ApiPath.Snapshot(vpcId)
	resp, err := s.client.SendPostRequest(apiPath, createdModel)
	if err != nil {
		return nil, decodeApiError(err)
	}

	var result snapshotResponse
	if err = json.Unmarshal(resp, &result); err != nil {
		return nil, common.DecodeError(err)
	}
	if result.Data.Id == "" {
		return nil, fmt.Errorf("the platform accepted the request without returning a snapshot")
	}

	return &result.Data, nil
}

// GetSnapshot reads one snapshot, reporting a snapshot that is gone as nil
func (s *SnapshotServiceImpl) GetSnapshot(vpcId string, snapshotId string) (*Snapshot, error) {
	apiPath := common.ApiPath.SnapshotDetail(vpcId, snapshotId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, decodeApiError(err)
	}

	var result snapshotResponse
	if err = json.Unmarshal(resp, &result); err != nil {
		return nil, common.DecodeError(err)
	}
	// A 200 without a snapshot is not "gone": reporting it as such would clear
	// the resource out of state.
	if result.Data.Id == "" {
		return nil, fmt.Errorf("the platform returned no snapshot for %s", snapshotId)
	}

	return &result.Data, nil
}

// DeleteSnapshot deletes a snapshot together with its volume snapshots
func (s *SnapshotServiceImpl) DeleteSnapshot(vpcId string, snapshotId string) error {
	apiPath := common.ApiPath.SnapshotDetail(vpcId, snapshotId)
	_, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		// Already gone is the desired state; a destroy must not fail for it.
		if isNotFound(err) {
			return nil
		}
		return decodeApiError(err)
	}

	return nil
}

func isNotFound(err error) bool {
	var httpErr common.HTTPError
	return errors.As(err, &httpErr) && httpErr.Code == http.StatusNotFound
}

// decodeApiError keeps the API's own error code, which common.DecodeError
// discards for anything that is not a 400.
func decodeApiError(err error) error {
	var httpErr common.HTTPError
	if !errors.As(err, &httpErr) {
		return common.DecodeError(err)
	}

	var body apiError
	if json.Unmarshal([]byte(httpErr.Reason), &body) == nil && body.ErrorCode != "" {
		return fmt.Errorf("%s: %s", body.ErrorCode, body.Message)
	}

	return common.DecodeError(err)
}
