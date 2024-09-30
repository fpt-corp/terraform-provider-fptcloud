package fptcloud_instance_group

import (
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

// InstanceGroupDTO instance group dto model to create instance group
type CreateInstanceGroupDTO struct {
	VpcId    string   `json:"vpc_id"`
	Name     string   `json:"name"`
	PolicyId string   `json:"policy_id"`
	VmIds    []string `json:"vm_ids"`
}

// FindInstanceGroupDTO find instance group model defined
type FindInstanceGroupDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	VpcId string `json:"vpc_id"`
}

// InstanceGroup represents a instance group model
type InstanceGroup struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Policy    interface{}   `json:"policy"`
	Vms       []interface{} `json:"vms"`
	VpcId     string        `json:"vpc_id"`
	CreatedAt string        `json:"created_at"`
}

// InstanceGroupService defines the interface for the instance group service
type InstanceGroupService interface {
	FindInstanceGroup(searchModel FindInstanceGroupDTO) (*[]InstanceGroup, error)
	CreateInstanceGroup(createdModel CreateInstanceGroupDTO) (bool, error)
	DeleteInstanceGroup(vpcId string, instanceGroupId string) (bool, error)
}

// InstanceGroupServiceImpl is the implementation of InstanceGroupServiceImpl
type InstanceGroupServiceImpl struct {
	client *common.Client
}

// NewInstanceGroupService creates a new instance of instance group with the given client
func NewInstanceGroupService(client *common.Client) InstanceGroupService {
	return &InstanceGroupServiceImpl{client: client}
}

// FindInstanceGroup finds an instance group by either part of the ID or part of the name
func (s *InstanceGroupServiceImpl) FindInstanceGroup(searchModel FindInstanceGroupDTO) (*[]InstanceGroup, error) {
	var apiPath = common.ApiPath.FindInstanceGroup(searchModel.VpcId) + utils.ToQueryParams(searchModel)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var instanceGroupResponse struct {
		Status  bool            `json:"status"`
		Message string          `json:"message"`
		Data    []InstanceGroup `json:"data"`
	}
	err = json.Unmarshal(resp, &instanceGroupResponse)
	if err != nil {
		return nil, err
	}
	if !instanceGroupResponse.Status || len(instanceGroupResponse.Data) == 0 {
		return nil, errors.New(instanceGroupResponse.Message)
	}

	return &instanceGroupResponse.Data, nil
}

// CreateInstanceGroup create a new instance group
func (s *InstanceGroupServiceImpl) CreateInstanceGroup(createdModel CreateInstanceGroupDTO) (bool, error) {
	var apiPath = common.ApiPath.CreateInstanceGroup(createdModel.VpcId)
	resp, err := s.client.SendPostRequest(apiPath, createdModel)
	if err != nil {
		return false, common.DecodeError(err)
	}

	var result struct {
		Status bool `json:"status"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return false, err
	}

	return result.Status, nil
}

// DeleteInstanceGroup delete an instance group
func (s *InstanceGroupServiceImpl) DeleteInstanceGroup(vpcId string, instanceGroupId string) (bool, error) {
	var apiPath = common.ApiPath.DeleteInstanceGroup(vpcId, instanceGroupId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return false, common.DecodeError(err)
	}

	var result struct {
		Status bool `json:"status"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return false, err
	}

	return result.Status, nil
}
