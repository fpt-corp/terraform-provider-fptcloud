package fptcloud_floating_ip_association

import (
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
)

type AssociateFloatingIpDTO struct {
	VpcId          string `json:"vpc_id"`
	FloatingIpId   string `json:"floating_ip_id"`
	FloatingIpPort int    `json:"floating_ip_port"`
	InstanceId     string `json:"instance_id"`
	InstancePort   int    `json:"instance_port"`
}

type FindFloatingIpDTO struct {
	FloatingIpID string `json:"floating_ip_id"`
	IpAddress    string `json:"ip_address"`
	VpcId        string `json:"vpc_id"`
}

type FloatingIpResponseDto struct {
	Status  bool       `json:"status"`
	Message string     `json:"message"`
	Data    FloatingIp `json:"data"`
}

// FloatingIp represents a floating ip model
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

// FloatingIpAssociationService defines the interface for floating ip association service
type FloatingIpAssociationService interface {
	FindFloatingIp(findDto FindFloatingIpDTO) (*FloatingIp, error)
	Associate(associateData AssociateFloatingIpDTO) (*bool, error)
	Disassociate(vpcId string, floatingIpId string) (bool, error)
}

// FloatingIpAssociationServiceImpl is the implementation of FloatingIpAssociationServiceImpl
type FloatingIpAssociationServiceImpl struct {
	client *common.Client
}

// NewFloatingIpAssociationService creates a new floating ip association with the given client
func NewFloatingIpAssociationService(client *common.Client) FloatingIpAssociationService {
	return &FloatingIpAssociationServiceImpl{client: client}
}

func (s *FloatingIpAssociationServiceImpl) FindFloatingIp(findDto FindFloatingIpDTO) (*FloatingIp, error) {
	var apiPath = common.ApiPath.FindFloatingIp(findDto.VpcId, findDto.FloatingIpID)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := FloatingIpResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if false == response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}

// Associate add floating ip to instance
func (s *FloatingIpAssociationServiceImpl) Associate(associateData AssociateFloatingIpDTO) (*bool, error) {
	var apiPath = common.ApiPath.AssociateFloatingIp(associateData.VpcId)
	resp, err := s.client.SendPostRequest(apiPath, associateData)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := FloatingIpResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if false == response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Status, nil
}

// Disassociate remove instance from floating ip
func (s *FloatingIpAssociationServiceImpl) Disassociate(vpcId string, floatingIpId string) (bool, error) {
	var apiPath = common.ApiPath.DisassociateFloatingIp(vpcId, floatingIpId)
	_, err := s.client.SendPostRequest(apiPath, nil)
	if err != nil {
		return false, common.DecodeError(err)
	}
	return true, nil
}
