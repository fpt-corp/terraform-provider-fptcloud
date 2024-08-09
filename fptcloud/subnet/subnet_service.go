package fptcloud_subnet

import (
	"errors"
	common "terraform-provider-fptcloud/commons"
)

type CreateSubnetDTO struct {
	VpcId        string `json:"vpc_id"`
	Name         string `json:"name"`
	CIDR         string `json:"cidr"`
	Type         string `json:"type"`
	GatewayIp    string `json:"gateway_ip"`
	IpRangeStart string `json:"ip_range_start"`
	IpRangeEnd   string `json:"ip_range_end"`
}

type FindSubnetDTO struct {
	SubnetID   string `json:"subnet_id"`
	SubnetName string `json:"subnet_name"`
	VpcId      string `json:"vpc_id"`
}

type SubnetResponseDto struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    Subnet `json:"data"`
}

type Subnet struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	NetworkName string      `json:"network_name"`
	Gateway     string      `json:"gateway"`
	VpcId       string      `json:"vpc_id"`
	EdgeGateway EdgeGateway `json:"edge_gateway"`
	CreatedAt   string      `json:"created_at"`
}

type EdgeGateway struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	EdgeGatewayId string `json:"edge_gateway_id"`
}

// SubnetService defines the interface for subnet service
type SubnetService interface {
	FindSubnet(findDto FindSubnetDTO) (*Subnet, error)
	FindSubnetByName(findDto FindSubnetDTO) (*Subnet, error)
	ListNetwork(vpcId string) (*[]Subnet, error)
	CreateSubnet(createDto CreateSubnetDTO) (*Subnet, error)
	DeleteSubnet(vpcId string, subnetId string) (bool, error)
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
	if false == response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}
