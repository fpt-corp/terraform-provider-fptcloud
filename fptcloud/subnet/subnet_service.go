package fptcloud_subnet

import (
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

type CreateSubnetDTO struct {
	VpcId        string   `json:"vpc_id"`
	Name         string   `json:"name"`
	CIDR         string   `json:"cidr"`
	Type         string   `json:"type"`
	GatewayIp    string   `json:"gateway_ip"`
	IpRangeStart string   `json:"ip_range_start"`
	IpRangeEnd   string   `json:"ip_range_end"`
	TagIds       []string `json:"tag_ids,omitempty"`
}

type FindSubnetDTO struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	VpcId       string `json:"vpc_id"`
}

type SubnetResponseDto struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    Subnet `json:"data"`
}

type Subnet struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	NetworkID   string      `json:"network_id"`
	NetworkName string      `json:"network_name"`
	Gateway     string      `json:"gateway"`
	VpcId       string      `json:"vpc_id"`
	EdgeGateway EdgeGateway `json:"edge_gateway"`
	CreatedAt   string      `json:"created_at"`
	TagIds      []string    `json:"tag_ids,omitempty"`
}

type EdgeGateway struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	EdgeGatewayId string `json:"edge_gateway_id"`
}

type ListSubnetResponseDto struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    *ListSubnet `json:"data"`
}

type ListSubnet struct {
	Data  []Subnet `json:"data"`
	Total int16    `json:"total"`
}

// SubnetService defines the interface for subnet service
type SubnetService interface {
	FindSubnet(findDto FindSubnetDTO) (*Subnet, error)
	FindSubnetByName(findDto FindSubnetDTO) (*Subnet, error)
	ListSubnet(vpcId string) (*[]Subnet, error)
	CreateSubnet(createDto CreateSubnetDTO) (*Subnet, error)
	DeleteSubnet(vpcId string, subnetId string) (bool, error)
	UpdateTags(vpcId string, subnetId string, tagIds []string) (*common.SimpleResponse, error)
}

// SubnetServiceImpl is the implementation of SubnetServiceImpl
type SubnetServiceImpl struct {
	client *common.Client
}

// NewSubnetService creates a new subnet with the given client
func NewSubnetService(client *common.Client) SubnetService {
	return &SubnetServiceImpl{client: client}
}

// CreateSubnet create a floating ip
func (s *SubnetServiceImpl) CreateSubnet(createDto CreateSubnetDTO) (*Subnet, error) {
	var apiPath = common.ApiPath.CreateSubnet(createDto.VpcId)
	resp, err := s.client.SendPostRequest(apiPath, createDto)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := SubnetResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}

// FindSubnetByName find a subnet name
func (s *SubnetServiceImpl) FindSubnetByName(findDto FindSubnetDTO) (*Subnet, error) {
	var apiPath = common.ApiPath.FindSubnetByName(findDto.VpcId) + utils.ToQueryParams(findDto)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := SubnetResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}

// FindSubnet find a subnet id
func (s *SubnetServiceImpl) FindSubnet(findDto FindSubnetDTO) (*Subnet, error) {
	var apiPath = common.ApiPath.FindSubnet(findDto.VpcId, findDto.NetworkID)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := SubnetResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}

// ListSubnet list subnet
func (s *SubnetServiceImpl) ListSubnet(vpcId string) (*[]Subnet, error) {
	var apiPath = common.ApiPath.ListSubnets(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := ListSubnetResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}
	if response.Data == nil || len(response.Data.Data) == 0 {
		return nil, errors.New("Subnet not found")
	}

	return &response.Data.Data, nil
}

// DeleteSubnet delete a subnet
func (s *SubnetServiceImpl) DeleteSubnet(vpcId string, subnetId string) (bool, error) {
	var apiPath = common.ApiPath.DeleteSubnet(vpcId, subnetId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return false, common.DecodeError(err)
	}

	response := SubnetResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return false, err
	}
	if !response.Status {
		return false, errors.New(response.Message)
	}

	return response.Status, nil
}

// UpdateTags updates the tags associated with a subnet
func (s *SubnetServiceImpl) UpdateTags(vpcId string, subnetId string, tagIds []string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.UpdateSubnetTags(vpcId, subnetId)
	payload := map[string][]string{
		"tag_ids": tagIds,
	}
	_, err := s.client.SendPutRequest(apiPath, payload)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}
