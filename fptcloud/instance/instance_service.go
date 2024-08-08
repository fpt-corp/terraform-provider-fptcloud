package fptcloud_instance

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

// InstanceService defines the interface for instance service
type InstanceService interface {
	Find(searchModel FindInstanceDTO) (*InstanceModel, error)
	Create(createdModel CreateInstanceDTO) (string, error)
	Delete(vpcId string, instanceId string) (*common.SimpleResponse, error)
	Rename(vpcId string, instanceId string, newName string) (*common.SimpleResponse, error)
	ChangeStatus(vpcId string, instanceId string, status string) (*common.SimpleResponse, error)
	Resize(vpcId string, instanceId string, flavorId string) (*common.SimpleResponse, error)
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
