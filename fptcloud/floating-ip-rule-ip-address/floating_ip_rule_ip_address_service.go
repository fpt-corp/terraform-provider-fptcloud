package fptcloud_floating_ip_rule_ip_address

import (
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
)

type ListFloatingIpRuleIpAddressResponseDto struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    []IpAddress `json:"data"`
}

type IpAddress struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FloatingIpRuleIpAddressService defines the interface for floating ip service
type FloatingIpRuleIpAddressService interface {
	ListExistingIpOfFloatingIp(vpcId string) (*[]IpAddress, error)
}

// FloatingIpRuleIpAddressServiceImpl is the implementation of FloatingIpRuleIpAddressServiceImpl
type FloatingIpRuleIpAddressServiceImpl struct {
	client *common.Client
}

// NewFloatingIpRuleIpAddressService creates a new instance group with the given client
func NewFloatingIpRuleIpAddressService(client *common.Client) FloatingIpRuleIpAddressService {
	return &FloatingIpRuleIpAddressServiceImpl{client: client}
}

func (s *FloatingIpRuleIpAddressServiceImpl) ListExistingIpOfFloatingIp(vpcId string) (*[]IpAddress, error) {
	var apiPath = common.ApiPath.ListExistingIpOfFloatingIp(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := ListFloatingIpRuleIpAddressResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if false == response.Status {
		return nil, errors.New(response.Message)
	}
	if len(response.Data) == 0 {
		return nil, errors.New("Ip address rule not found")
	}

	return &response.Data, nil
}
