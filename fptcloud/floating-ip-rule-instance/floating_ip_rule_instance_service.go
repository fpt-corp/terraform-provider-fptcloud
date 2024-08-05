package fptcloud_floating_ip_rule_instance

import (
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
)

type ListFloatingIpRuleInstanceResponseDto struct {
	Status  bool                     `json:"status"`
	Message string                   `json:"message"`
	Data    []InstanceRuleFloatingIp `json:"data"`
}

type InstanceRuleFloatingIp struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IpAddress string `json:"ip_address"`
	Type      string `json:"type"`
}

// FloatingIpRuleInstanceService defines the interface for floating ip service
type FloatingIpRuleInstanceService interface {
	ListExistingInstanceOfFloatingIp(vpcId string) (*[]InstanceRuleFloatingIp, error)
}

// FloatingIpRuleInstanceServiceImpl is the implementation of FloatingIpRuleInstanceServiceImpl
type FloatingIpRuleInstanceServiceImpl struct {
	client *common.Client
}

// NewFloatingIpRuleInstanceService creates a new instance group with the given client
func NewFloatingIpRuleInstanceService(client *common.Client) FloatingIpRuleInstanceService {
	return &FloatingIpRuleInstanceServiceImpl{client: client}
}

func (s *FloatingIpRuleInstanceServiceImpl) ListExistingInstanceOfFloatingIp(vpcId string) (*[]InstanceRuleFloatingIp, error) {
	var apiPath = common.ApiPath.ListExistingInstanceOfFloatingIp(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := ListFloatingIpRuleInstanceResponseDto{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if false == response.Status {
		return nil, errors.New(response.Message)
	}
	if len(response.Data) == 0 {
		return nil, errors.New("Instance rule not found")
	}

	return &response.Data, nil
}
