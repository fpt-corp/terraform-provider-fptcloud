package fptcloud_floating_ip

import (
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

// CreateFloatingIpDTO
type CreateFloatingIpDTO struct {
	FloatingIpId   string `json:"floating_ip_id"`
	FloatingIpPort int8   `json:"floating_ip_port"`
	InstanceId     string `json:"instance_id"`
	InstancePort   int8   `json:"instance_port"`
}

// FloatingIp represents a instance group model
type FloatingIp struct {
	ID        string             `json:"id"`
	IpAddress string             `json:"ip_address"`
	NatType   string             `json:"nat_type"`
	Instance  FloatingIpInstance `json:"instance"`
	Status    string             `json:"status"`
	CreatedAt string             `json:"created_at"`
}

type FloatingIpInstance struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// FloatingIpService defines the interface for floating ip service
type FloatingIpService interface {
	FindFloatingIp(vpcId string, floatingIpId string) (*[]FloatingIp, error)
	CreateFloatingIp(createdModel CreateFloatingIpDTO) (bool, error)
	DeleteFloatingIp(vpcId string, floatingIpId string) (bool, error)
}

// FloatingIpServiceImpl is the implementation of FloatingIpServiceImpl
type FloatingIpServiceImpl struct {
	client *common.Client
}

// NewFloatingIpService creates a new instance group with the given client
func NewFloatingIpService(client *common.Client) FloatingIpService {
	return &FloatingIpServiceImpl{client: client}
}

func (s *FloatingIpServiceImpl) FindFloatingIp(vpcId string, floatingIpId string) (*[]FloatingIp, error) {
	var apiPath = common.ApiPath.FindFloatingIp(searchModel.VpcId) + utils.ToQueryParams(searchModel)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var floatingIpResponse struct {
		Status  bool         `json:"status"`
		Message string       `json:"message"`
		Data    []FloatingIp `json:"data"`
	}
	err = json.Unmarshal(resp, &floatingIpResponse)
	if err != nil {
		return nil, err
	}
	if false == floatingIpResponse.Status {
		return nil, errors.New(instanceGroupResponse.Message)
	}

	return &instanceGroupResponse.Data, nil
}

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
