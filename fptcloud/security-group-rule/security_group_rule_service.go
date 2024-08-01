package fptcloud_security_group_rule

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
)

type SecurityGroupRule struct {
	ID              string   `json:"id"`
	Direction       string   `json:"direction"`
	Action          string   `json:"action"`
	Protocol        string   `json:"protocol"`
	PortRange       string   `json:"port_range"`
	Sources         []string `json:"sources"`
	IpType          string   `json:"ip_type"`
	Description     string   `json:"description"`
	Status          string   `json:"status"`
	VpcId           string   `json:"vpc_id"`
	SecurityGroupId string   `json:"security_group_id"`
}

type FindSecurityGroupRuleResponse struct {
	Data SecurityGroupRule `json:"data"`
}

type CreateSecurityGroupRuleDto struct {
	Direction       string   `json:"direction"`
	Action          string   `json:"action"`
	Protocol        string   `json:"protocol"`
	PortRange       string   `json:"port_range"`
	Sources         []string `json:"sources"`
	Description     *string  `json:"description,omitempty"`
	SecurityGroupId string   `json:"security_group_id"`
}

// SecurityGroupRuleService defines the interface for security group rule service
type SecurityGroupRuleService interface {
	Find(vpcId string, ruleId string) (*SecurityGroupRule, error)
	Create(vpcId string, createdModel CreateSecurityGroupRuleDto) (string, error)
	Delete(vpcId string, ruleId string) (*common.SimpleResponse, error)
}

// SecurityGroupRuleServiceImpl is the implementation of SecurityGroupRuleService
type SecurityGroupRuleServiceImpl struct {
	client *common.Client
}

// NewSecurityGroupRuleService creates a new instance of security group rule service with the given client
func NewSecurityGroupRuleService(client *common.Client) SecurityGroupRuleService {
	return &SecurityGroupRuleServiceImpl{client: client}
}

// Find search security group rule by id
func (s *SecurityGroupRuleServiceImpl) Find(vpcId string, ruleId string) (*SecurityGroupRule, error) {
	var apiPath = common.ApiPath.SecurityGroupRule(vpcId, ruleId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := FindSecurityGroupRuleResponse{}
	err = json.Unmarshal(resp, &response)

	if err != nil {
		return nil, common.DecodeError(err)
	}
	return &response.Data, nil
}

// Create created a new security group rule
func (s *SecurityGroupRuleServiceImpl) Create(vpcId string, createdModel CreateSecurityGroupRuleDto) (string, error) {
	var apiPath = common.ApiPath.CreateSecurityGroupRule(vpcId)
	resp, err := s.client.SendPostRequest(apiPath, createdModel)
	if err != nil {
		return "", common.DecodeError(err)
	}

	var createdResponse struct {
		SecurityGroupRuleId string `json:"security_group_rule_id"`
	}

	err = json.Unmarshal(resp, &createdResponse)

	if err != nil {
		return "", common.DecodeError(err)
	}

	return createdResponse.SecurityGroupRuleId, nil
}

// Delete deleted a security group rule
func (s *SecurityGroupRuleServiceImpl) Delete(vpcId string, ruleId string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.SecurityGroupRule(vpcId, ruleId)
	_, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}
