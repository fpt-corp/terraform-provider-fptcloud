package fptcloud_mfke_storage_policy

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
)

// MfkeStoragePolicy represents a MFKE storage policy model
type MfkeStoragePolicy struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
	IDDb      string `json:"id_db"`
	Zone      string `json:"zone"`
}

// MfkeStoragePolicyService defines the interface for MFKE storage policy service
type MfkeStoragePolicyService interface {
	ListMfkeStoragePolicy(vpcId string) (*[]MfkeStoragePolicy, error)
}

// MfkeStoragePolicyServiceImpl is the implementation of MfkeStoragePolicyService
type MfkeStoragePolicyServiceImpl struct {
	client *common.Client
}

// NewMfkeStoragePolicyService creates a new instance of MFKE storage policy Service with the given client
func NewMfkeStoragePolicyService(client *common.Client) MfkeStoragePolicyService {
	return &MfkeStoragePolicyServiceImpl{client: client}
}

// ListMfkeStoragePolicy get list MFKE storage policy
func (s *MfkeStoragePolicyServiceImpl) ListMfkeStoragePolicy(vpcId string) (*[]MfkeStoragePolicy, error) {
	var apiPath = common.ApiPath.ManagedFKEStoragePolicy(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, err
	}

	var storagePolicyResponse struct {
		Data []MfkeStoragePolicy `json:"data"`
	}
	err = json.Unmarshal(resp, &storagePolicyResponse)

	if err != nil {
		return nil, err
	}
	return &storagePolicyResponse.Data, nil
}
