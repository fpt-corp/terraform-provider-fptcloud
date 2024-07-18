package fptcloud_storage

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

const (
	External = "EXTERNAL"
	Local    = "LOCAL"
)

// FindStorageDTO find storage model defined
type FindStorageDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	VpcId string `json:"vpc_id"`
}

// StorageDTO storage dto model to create or update storage
type StorageDTO struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	SizeGb          int     `json:"size_gb"`
	StoragePolicyId string  `json:"storage_policy_id"`
	InstanceId      *string `json:"instance_id"`
	VpcId           string  `json:"vpc_id"`
}

// Storage represents a storage model
type Storage struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	SizeGb        int    `json:"size_gb"`
	StoragePolicy string `json:"storage_policy"`
	InstanceId    string `json:"instance_id"`
	VpcId         string `json:"vpc_id"`
	CreatedAt     string `json:"created_at"`
}

// StorageService defines the interface for storage service
type StorageService interface {
	FindStorage(searchModel FindStorageDTO) (*Storage, error)
	CreateStorage(createdModel StorageDTO) (string, error)
	UpdateStorage(storageId string, updatedModel StorageDTO) (*common.SimpleResponse, error)
	DeleteStorage(vpcId string, storageId string) (*common.SimpleResponse, error)
}

// StorageServiceImpl is the implementation of StorageServiceImpl
type StorageServiceImpl struct {
	client *common.Client
}

// NewStorageService creates a new instance of storage Service with the given client
func NewStorageService(client *common.Client) StorageService {
	return &StorageServiceImpl{client: client}
}

// FindStorage finds a storage by either part of the ID or part of the name
func (s *StorageServiceImpl) FindStorage(searchModel FindStorageDTO) (*Storage, error) {
	var apiPath = common.ApiPath.Storage(searchModel.VpcId) + utils.ToQueryParams(searchModel)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, err
	}

	result := Storage{}
	err = json.Unmarshal(resp, &result)

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateStorage create a new storage
func (s *StorageServiceImpl) CreateStorage(createdModel StorageDTO) (string, error) {
	var apiPath = common.ApiPath.Storage(createdModel.VpcId)
	resp, err := s.client.SendPostRequest(apiPath, createdModel)

	if err != nil {
		return "", err
	}

	var createStorageResponse struct {
		StorageId string `json:"storage_id"`
	}

	err = json.Unmarshal(resp, &createStorageResponse)

	if err != nil {
		return "", err
	}

	return createStorageResponse.StorageId, nil
}

// UpdateStorage update a storage
func (s *StorageServiceImpl) UpdateStorage(storageId string, updatedModel StorageDTO) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.Storage(updatedModel.VpcId) + "/" + storageId
	_, err := s.client.SendPutRequest(apiPath, updatedModel)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}

// DeleteStorage delete a storage
func (s *StorageServiceImpl) DeleteStorage(vpcId string, storageId string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.Storage(vpcId) + "/" + storageId
	_, err := s.client.SendDeleteRequest(apiPath)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}
