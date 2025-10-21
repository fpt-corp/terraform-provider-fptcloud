package fptcloud_floating_ip

import (
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

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

type ListFloatingIpResponseDto struct {
	Status  bool                `json:"status"`
	Message string              `json:"message"`
	Data    *ListFloatingIpData `json:"data"`
}

type ListFloatingIpData struct {
	Data  []FloatingIp `json:"data"`
	Total int16        `json:"total"`
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

// FloatingIpService defines the interface for floating ip service
type FloatingIpService interface {
	FindFloatingIp(findDto FindFloatingIpDTO) (*FloatingIp, error)
	FindFloatingIpByAddress(findDto FindFloatingIpDTO) (*FloatingIp, error)
	ListFloatingIp(vpcId string) (*[]FloatingIp, error)
	CreateFloatingIp(vpcId string) (*FloatingIp, error)
	DeleteFloatingIp(vpcId string, floatingIpId string) (bool, error)
}

// FloatingIpServiceImpl is the implementation of FloatingIpServiceImpl
type FloatingIpServiceImpl struct {
	client *common.Client
}

// NewFloatingIpService creates a new floating ip with the given client
func NewFloatingIpService(client *common.Client) FloatingIpService {
	return &FloatingIpServiceImpl{client: client}
}

func (s *FloatingIpServiceImpl) FindFloatingIp(findDto FindFloatingIpDTO) (*FloatingIp, error) {
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
	if !response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}

// FindFloatingIpByAddress find a floating ip
func (s *FloatingIpServiceImpl) FindFloatingIpByAddress(findDto FindFloatingIpDTO) (*FloatingIp, error) {
	var apiPath = common.ApiPath.FindFloatingIpByAddress(findDto.VpcId) + utils.ToQueryParams(findDto)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := FloatingIpResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}

// ListFloatingIp list floating ip
func (s *FloatingIpServiceImpl) ListFloatingIp(vpcId string) (*[]FloatingIp, error) {
	var apiPath = common.ApiPath.ListFloatingIp(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := ListFloatingIpResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}
	if response.Data == nil || len(response.Data.Data) == 0 {
		return nil, errors.New("floating ip not found")
	}

	return &response.Data.Data, nil
}

// CreateFloatingIp create a floating ip
func (s *FloatingIpServiceImpl) CreateFloatingIp(vpcId string) (*FloatingIp, error) {
	body := map[string]interface{}{"vpc_id": vpcId, "floating_ip_id": "new"}
	var apiPath = common.ApiPath.CreateFloatingIp(vpcId)
	resp, err := s.client.SendPostRequest(apiPath, body)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := FloatingIpResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}

// DeleteFloatingIp delete an floating ip
func (s *FloatingIpServiceImpl) DeleteFloatingIp(vpcId string, floatingIpId string) (bool, error) {
	var apiPath = common.ApiPath.DeleteFloatingIp(vpcId, floatingIpId)
	resp, err := s.client.SendPostRequest(apiPath, nil)
	if err != nil {
		return false, common.DecodeError(err)
	}

	response := FloatingIpResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return false, err
	}
	if !response.Status {
		return false, errors.New(response.Message)
	}

	return response.Status, nil
}
