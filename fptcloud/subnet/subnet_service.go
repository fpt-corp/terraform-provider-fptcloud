package fptcloud_subnet

import (
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

type CreateSubnetDTO struct {
	VpcId          string   `json:"vpc_id"`
	Name           string   `json:"name"`
	CIDR           string   `json:"cidr"`
	Type           string   `json:"type"`
	GatewayIp      string   `json:"gateway_ip"`
	IpRangeStart   string   `json:"ip_range_start"`
	IpRangeEnd     string   `json:"ip_range_end"`
	PrimaryDnsIp   string   `json:"primary_dns_ip"`
	SecondaryDnsIp string   `json:"secondary_dns_ip"`
	TagNames       []string `json:"tag_names"`
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

type UpdateResponseDto struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    Subnet `json:"data"`
}
type Subnet struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	NetworkID      string      `json:"network_id"`
	NetworkName    string      `json:"network_name"`
	Gateway        string      `json:"gateway"`
	VpcId          string      `json:"vpc_id"`
	EdgeGateway    EdgeGateway `json:"edge_gateway"`
	PrimaryDnsIp   string      `json:"primary_dns_ip"`
	SecondaryDnsIp string      `json:"secondary_dns_ip"`
	CreatedAt      string      `json:"created_at"`
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

type UpdateDnsSubnetDTO struct {
	VpcId          string `json:"vpc_id"`
	SubnetId       string `json:"subnet_id"`
	PrimaryDnsIp   string `json:"primary_dns_ip,omitempty"`
	SecondaryDnsIp string `json:"secondary_dns_ip,omitempty"`
	CIDR           string `json:"cidr"`
}

type UpdateNameSubnetDTO struct {
	VpcId    string `json:"vpc_id"`
	SubnetId string `json:"subnet_id"`
	Name     string `json:"name"`
}

type UpdateSubnetDTO struct {
	VpcId          string   `json:"vpc_id"`
	SubnetId       string   `json:"subnet_id"`
	Name           string   `json:"name,omitempty"`
	CIDR           string   `json:"cidr,omitempty"`
	GatewayIp      string   `json:"gateway_ip,omitempty"`
	IpRangeStart   string   `json:"ip_range_start,omitempty"`
	IpRangeEnd     string   `json:"ip_range_end,omitempty"`
	PrimaryDnsIp   string   `json:"primary_dns_ip,omitempty"`
	SecondaryDnsIp string   `json:"secondary_dns_ip,omitempty"`
	TagNames       []string `json:"tag_names,omitempty"`
}

type UpdateTagsSubnetDTO struct {
	VpcId    string   `json:"vpc_id"`
	SubnetId string   `json:"subnet_id"`
	TagNames []string `json:"tag_names"`
}

// SubnetService defines the interface for subnet service
type SubnetService interface {
	FindSubnet(findDto FindSubnetDTO) (*Subnet, error)
	FindSubnetByName(findDto FindSubnetDTO) (*Subnet, error)
	ListSubnet(vpcId string) (*[]Subnet, error)
	CreateSubnet(createDto CreateSubnetDTO) (*Subnet, error)
	DeleteSubnet(vpcId string, subnetId string) (bool, error)
	UpdateDns(updateDto UpdateDnsSubnetDTO) (*Subnet, error)
	UpdateReName(updateDto UpdateNameSubnetDTO) (*Subnet, error)
	UpdateTags(updateDto UpdateTagsSubnetDTO) (*Subnet, error)
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

// UpdateDns updates DNS for a subnet
func (s *SubnetServiceImpl) UpdateDns(updateDto UpdateDnsSubnetDTO) (*Subnet, error) {
	var apiPath = common.ApiPath.UpdateDnsSubnet(updateDto.VpcId, updateDto.SubnetId)
	resp, err := s.client.SendPutRequest(apiPath, updateDto)
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

func (s *SubnetServiceImpl) UpdateReName(updateDto UpdateNameSubnetDTO) (*Subnet, error) {
	var apiPath = common.ApiPath.UpdateNameSubnet(updateDto.VpcId, updateDto.SubnetId)
	resp, err := s.client.SendPutRequest(apiPath, updateDto)
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

// UpdateTags updates tags for a subnet
func (s *SubnetServiceImpl) UpdateTags(updateDto UpdateTagsSubnetDTO) (*Subnet, error) {
	apiPath := common.ApiPath.UpdateTagsSubnet(updateDto.VpcId, updateDto.SubnetId)

	resp, err := s.client.SendPostRequest(apiPath, updateDto)
	if err != nil {
		return nil, common.DecodeError(err)
	}
	response := UpdateResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}
