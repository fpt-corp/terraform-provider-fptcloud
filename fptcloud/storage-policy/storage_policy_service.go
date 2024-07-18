package fptcloud_storage_policy

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
)

// StoragePolicy represents a storage policy model
type StoragePolicy struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// StoragePolicyService defines the interface for storage policy service
type StoragePolicyService interface {
	ListStoragePolicy(vpcId string) (*[]StoragePolicy, error)
}

// StoragePolicyServiceImpl is the implementation of StoragePolicyServiceImpl
type StoragePolicyServiceImpl struct {
	client *common.Client
}

// NewStoragePolicyService creates a new instance of storage policy Service with the given client
func NewStoragePolicyService(client *common.Client) StoragePolicyService {
	return &StoragePolicyServiceImpl{client: client}
}

// ListStoragePolicy get list storage policy
func (s *StoragePolicyServiceImpl) ListStoragePolicy(vpcId string) (*[]StoragePolicy, error) {
	var apiPath = common.ApiPath.StoragePolicy(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, err
	}

	var storagePolicyResponse struct {
		Data []StoragePolicy `json:"data"`
	}
	err = json.Unmarshal(resp, &storagePolicyResponse)

	if err != nil {
		return nil, err
	}
	return &storagePolicyResponse.Data, nil
}
