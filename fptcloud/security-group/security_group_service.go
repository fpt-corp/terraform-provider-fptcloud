package fptcloud_security_group

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

// SecurityGroupService defines the interface for security service
type SecurityGroupService interface {
	Find(searchModel FindSecurityGroupDTO) (*SecurityGroup, error)
	Create(createdModel CreatedSecurityGroupDTO) (string, error)
	Delete(vpcId string, securityGroupId string) (*common.SimpleResponse, error)
	Rename(vpcId string, securityGroupId string, newName string) (*common.SimpleResponse, error)
	UpdateApplyTo(vpcId string, securityGroupId string, applyTo []string) (*common.SimpleResponse, error)
}

// SecurityGroupServiceImpl is the implementation of SecurityGroupService
type SecurityGroupServiceImpl struct {
	client *common.Client
}

// NewSecurityGroupService creates a new instance of security group service with the given client
func NewSecurityGroupService(client *common.Client) SecurityGroupService {
	return &SecurityGroupServiceImpl{client: client}
}

// Find search security group by id or name
func (s *SecurityGroupServiceImpl) Find(searchModel FindSecurityGroupDTO) (*SecurityGroup, error) {
	var apiPath = common.ApiPath.SecurityGroup(searchModel.VpcId) + utils.ToQueryParams(searchModel)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	response := FindSecurityGroupResponse{}
	err = json.Unmarshal(resp, &response)

	if err != nil {
		return nil, common.DecodeError(err)
	}
	return &response.Data, nil
}

// Create created a new security group
func (s *SecurityGroupServiceImpl) Create(createdModel CreatedSecurityGroupDTO) (string, error) {
	var apiPath = common.ApiPath.SecurityGroup(createdModel.VpcId)
	resp, err := s.client.SendPostRequest(apiPath, createdModel)
	if err != nil {
		return "", common.DecodeError(err)
	}

	var createdResponse struct {
		SecurityGroupId string `json:"security_group_id"`
	}

	err = json.Unmarshal(resp, &createdResponse)

	if err != nil {
		return "", common.DecodeError(err)
	}

	return createdResponse.SecurityGroupId, nil
}

// Delete deleted a security group
func (s *SecurityGroupServiceImpl) Delete(vpcId string, securityGroupId string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.SecurityGroup(vpcId) + "/" + securityGroupId
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

// Rename update name a security group
func (s *SecurityGroupServiceImpl) Rename(vpcId string, securityGroupId string, newName string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.RenameSecurityGroup(vpcId, securityGroupId)
	_, err := s.client.SendPutRequest(apiPath, map[string]string{"new_name": newName})
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}

// UpdateApplyTo update apply to from security group
func (s *SecurityGroupServiceImpl) UpdateApplyTo(vpcId string, securityGroupId string, applyTo []string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.UpdateApplyToSecurityGroup(vpcId, securityGroupId)
	_, err := s.client.SendPutRequest(apiPath, map[string]interface{}{"apply_to": applyTo})
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}
