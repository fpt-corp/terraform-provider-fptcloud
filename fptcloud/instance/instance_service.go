package fptcloud_instance

import (
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

// Storage types of the boot disk: the portal persists ROOT on VMW and LOCAL on OSP, the infrastructure reports root
const (
	RootStorageType      = "ROOT"
	LocalStorageType     = "LOCAL"
	RootInfraStorageType = "root"
)

// InstanceService defines the interface for instance service
type InstanceService interface {
	Find(searchModel FindInstanceDTO) (*InstanceModel, error)
	Create(createdModel CreateInstanceDTO) (string, error)
	Delete(vpcId string, instanceId string) (*common.SimpleResponse, error)
	Rename(vpcId string, instanceId string, newName string) (*common.SimpleResponse, error)
	ChangeStatus(vpcId string, instanceId string, status string) (*common.SimpleResponse, error)
	Resize(vpcId string, instanceId string, flavorId string) (*common.SimpleResponse, error)
	GetFlavorByName(vpcId string, flavorName string) (*FlavorDTO, error)
	UpdateTags(vpcId string, instanceId string, tagIds []string) (*common.SimpleResponse, error)
	ListStorages(vpcId string, instanceId string) ([]InstanceStorageModel, error)
	ListStoragesFromInfra(vpcId string, instanceId string) ([]InstanceStorageInfraModel, error)
	FindRootStorage(vpcId string, instanceId string) (*RootStorageModel, error)
	ResizeRootDisk(vpcId string, instanceId string, resizeModel ResizeRootDiskDTO) (*common.SimpleResponse, error)
	FindStoragePolicy(vpcId string, storagePolicyId string) (*StoragePolicyDTO, error)
}

// InstanceServiceImpl is the implementation of InstanceService
type InstanceServiceImpl struct {
	client *common.Client
}

// NewInstanceService creates a new instance service with the given client
func NewInstanceService(client *common.Client) InstanceService {
	return &InstanceServiceImpl{client: client}
}

// Find get instance by id or name
func (s *InstanceServiceImpl) Find(searchModel FindInstanceDTO) (*InstanceModel, error) {
	var apiPath = common.ApiPath.Instance(searchModel.VpcId) + utils.ToQueryParams(searchModel)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var responseModel struct {
		Data InstanceModel `json:"data"`
	}
	err = json.Unmarshal(resp, &responseModel)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &responseModel.Data, nil
}

// Create created a new instance
func (s *InstanceServiceImpl) Create(createdModel CreateInstanceDTO) (string, error) {
	var apiPath = common.ApiPath.Instance(createdModel.VpcId)
	resp, err := s.client.SendPostRequest(apiPath, createdModel)
	if err != nil {
		return "", common.DecodeError(err)
	}

	var createdResponse struct {
		InstanceId string `json:"instance_id"`
	}

	err = json.Unmarshal(resp, &createdResponse)

	if err != nil {
		return "", common.DecodeError(err)
	}

	return createdResponse.InstanceId, nil
}

// Delete deleted a instance
func (s *InstanceServiceImpl) Delete(vpcId string, instanceId string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.Instance(vpcId) + "/" + instanceId
	_, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data: "Successfully",
	}

	return result, nil
}

// Rename update name a instance
func (s *InstanceServiceImpl) Rename(vpcId string, instanceId string, newName string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.RenameInstance(vpcId, instanceId)
	_, err := s.client.SendPutRequest(apiPath, map[string]string{"new_name": newName})
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data: "Successfully",
	}

	return result, nil
}

// ChangeStatus update status an instance
func (s *InstanceServiceImpl) ChangeStatus(vpcId string, instanceId string, status string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.ChangeStatusInstance(vpcId, instanceId)
	_, err := s.client.SendPutRequest(apiPath, map[string]string{"status": status})
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data: "Successfully",
	}

	return result, nil
}

// Resize update flavor an instance
func (s *InstanceServiceImpl) Resize(vpcId string, instanceId string, flavorId string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.ResizeInstance(vpcId, instanceId)
	_, err := s.client.SendPostRequest(apiPath, map[string]string{"hw_flavor": flavorId})
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data: "Successfully",
	}

	return result, nil
}

// GetFlavorByName get flavor by name
func (s *InstanceServiceImpl) GetFlavorByName(vpcId string, flavorName string) (*FlavorDTO, error) {
	var apiPath = common.ApiPath.GetFlavorByName(vpcId)
	resp, err := s.client.SendPostRequest(apiPath, map[string]string{"flavor_name": flavorName})
	if err != nil {
		return nil, common.DecodeError(err)
	}

	flavor := FlavorDTO{}
	err = json.Unmarshal(resp, &flavor)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &flavor, nil
}

// UpdateTags updates tags associated with an instance
func (s *InstanceServiceImpl) UpdateTags(vpcId string, instanceId string, tagIds []string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.UpdateInstanceTags(vpcId, instanceId)
	payload := map[string][]string{
		"tag_ids": tagIds,
	}
	_, err := s.client.SendPutRequest(apiPath, payload)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data: "Successfully",
	}

	return result, nil
}

// ListStorages get all disks of an instance as the portal persists them
func (s *InstanceServiceImpl) ListStorages(vpcId string, instanceId string) ([]InstanceStorageModel, error) {
	var apiPath = common.ApiPath.InstanceStorages(vpcId, instanceId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var responseModel struct {
		Data []InstanceStorageModel `json:"data"`
	}
	err = json.Unmarshal(resp, &responseModel)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	return responseModel.Data, nil
}

// ListStoragesFromInfra get all disks of an instance as the infrastructure reports them
func (s *InstanceServiceImpl) ListStoragesFromInfra(vpcId string, instanceId string) ([]InstanceStorageInfraModel, error) {
	var apiPath = common.ApiPath.InstanceStoragesInfra(vpcId, instanceId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var responseModel struct {
		Data []InstanceStorageInfraModel `json:"data"`
	}
	err = json.Unmarshal(resp, &responseModel)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	return responseModel.Data, nil
}

// FindRootStorage get the boot disk of an instance, preferring the portal listing because it also carries the policy id
func (s *InstanceServiceImpl) FindRootStorage(vpcId string, instanceId string) (*RootStorageModel, error) {
	rootStorage, err := s.findRootStorageInPortal(vpcId, instanceId)
	if err == nil && rootStorage.DiskId != "" {
		return rootStorage, nil
	}

	// the portal only lists a disk once its row is ENABLED, until then the infrastructure is the only source
	return s.findRootStorageInInfra(vpcId, instanceId)
}

func (s *InstanceServiceImpl) findRootStorageInPortal(vpcId string, instanceId string) (*RootStorageModel, error) {
	storages, err := s.ListStorages(vpcId, instanceId)
	if err != nil {
		return nil, err
	}

	var localStorages []InstanceStorageModel
	for _, storage := range storages {
		switch storage.StorageType {
		case RootStorageType:
			return &RootStorageModel{
				DiskId:          storage.DiskId,
				SizeMb:          storage.SizeMb,
				StoragePolicyId: storage.StoragePolicyId,
			}, nil
		case LocalStorageType:
			localStorages = append(localStorages, storage)
		}
	}

	// OSP persists the boot disk as LOCAL and never as ROOT, and it is the only local disk it can have
	if len(localStorages) == 1 {
		return &RootStorageModel{
			DiskId:          localStorages[0].DiskId,
			SizeMb:          localStorages[0].SizeMb,
			StoragePolicyId: localStorages[0].StoragePolicyId,
		}, nil
	}

	return nil, fmt.Errorf("root disk of instance %s not found in the portal", instanceId)
}

func (s *InstanceServiceImpl) findRootStorageInInfra(vpcId string, instanceId string) (*RootStorageModel, error) {
	storages, err := s.ListStoragesFromInfra(vpcId, instanceId)
	if err != nil {
		return nil, err
	}

	var unnamed []InstanceStorageInfraModel
	for _, storage := range storages {
		if storage.IsRoot || storage.StorageType == RootInfraStorageType {
			return &RootStorageModel{
				DiskId:            storage.DiskId,
				SizeMb:            storage.SizeMb,
				StoragePolicyName: storage.StorageProfileName,
			}, nil
		}
		// OSP does not report the bus/unit of a disk so none is flagged as root, only the boot disk has no name
		if storage.DiskName == nil {
			unnamed = append(unnamed, storage)
		}
	}

	if len(unnamed) == 1 {
		return &RootStorageModel{
			DiskId:            unnamed[0].DiskId,
			SizeMb:            unnamed[0].SizeMb,
			StoragePolicyName: unnamed[0].StorageProfileName,
		}, nil
	}

	if len(unnamed) > 1 {
		return nil, fmt.Errorf("instance %s has %d unnamed disks, cannot tell which one is the root disk", instanceId, len(unnamed))
	}

	return nil, fmt.Errorf("root disk of instance %s not found", instanceId)
}

// ResizeRootDisk resize and/or change the storage policy of the root disk of an instance
func (s *InstanceServiceImpl) ResizeRootDisk(vpcId string, instanceId string, resizeModel ResizeRootDiskDTO) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.ResizeInstanceRootDisk(vpcId, instanceId)
	_, err := s.client.SendPostRequest(apiPath, resizeModel)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data: "Successfully",
	}

	return result, nil
}

// FindStoragePolicy get a storage policy by the id the provider exposes, or by its infrastructure id
func (s *InstanceServiceImpl) FindStoragePolicy(vpcId string, storagePolicyId string) (*StoragePolicyDTO, error) {
	var apiPath = common.ApiPath.StoragePolicy(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var responseModel struct {
		Data []StoragePolicyDTO `json:"data"`
	}
	err = json.Unmarshal(resp, &responseModel)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	for _, storagePolicy := range responseModel.Data {
		// tolerate a config that already holds the infrastructure id
		if storagePolicy.ID == storagePolicyId || storagePolicy.InfraId == storagePolicyId {
			found := storagePolicy
			return &found, nil
		}
	}

	return nil, fmt.Errorf("storage policy %s not found in vpc %s", storagePolicyId, vpcId)
}
