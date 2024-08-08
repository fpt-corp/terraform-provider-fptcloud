package fptcloud_instance_group_policy

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
)

// InstanceGroupPolicy represents an instance group policy model
type InstanceGroupPolicy struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InstanceGroupPolicyService defines the interface for the instance group policy service
type InstanceGroupPolicyService interface {
	ListInstanceGroupPolicies(vpcId string) (*[]InstanceGroupPolicy, error)
}

// InstanceGroupPolicyServiceImpl is the implementation of InstanceGroupPolicyService
type InstanceGroupPolicyServiceImpl struct {
	client *common.Client
}

// NewInstanceGroupPolicyService creates a new instance of instance group policy with the given client
func NewInstanceGroupPolicyService(client *common.Client) InstanceGroupPolicyService {
	return &InstanceGroupPolicyServiceImpl{client: client}
}

// ListInstanceGroupPolicies get list instance group policies
func (s *InstanceGroupPolicyServiceImpl) ListInstanceGroupPolicies(vpcId string) (*[]InstanceGroupPolicy, error) {
	var apiPath = common.ApiPath.VMGroupPolicies(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, err
	}

	var instanceGroupPolicyResponse struct {
		Data []InstanceGroupPolicy `json:"data"`
	}
	err = json.Unmarshal(resp, &instanceGroupPolicyResponse)

	if err != nil {
		return nil, err
	}
	return &instanceGroupPolicyResponse.Data, nil
}
