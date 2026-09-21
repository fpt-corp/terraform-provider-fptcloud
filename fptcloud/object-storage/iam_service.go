package fptcloud_object_storage

import (
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"
)

// IAM users and roles are a Ceph-only surface: on every other S3 platform the
// backend answers 501 with {"status": false}. That reads here as an ordinary
// failure carrying the API's own message, which is what the tenant needs to see.

func (s *ObjectStorageServiceImpl) CreateIamUser(vpcId, s3ServiceId, userName string) *CreateIamUserResponse {
	apiPath := common.ApiPath.CreateIamUser(vpcId, s3ServiceId)
	resp, err := s.client.SendPostRequest(apiPath, IamUserRequest{UserName: userName})
	if err != nil {
		return &CreateIamUserResponse{Status: false, Message: err.Error()}
	}

	var created CreateIamUserResponse
	if err := json.Unmarshal(resp, &created); err != nil {
		return &CreateIamUserResponse{Status: false, Message: err.Error()}
	}
	return &created
}

// GetIamUser reads one IAM user. The endpoint answers with the user object
// itself rather than an envelope, so nil means "not there" - either a transport
// or 404 error, or a body that carries an explicit failure.
func (s *ObjectStorageServiceImpl) GetIamUser(vpcId, s3ServiceId, userName string) *IamUserDetail {
	apiPath := common.ApiPath.GetIamUser(vpcId, s3ServiceId, userName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil
	}
	if r := decodeCommonResponse(resp, ""); !r.Status {
		return nil
	}

	var detail IamUserDetail
	if err := json.Unmarshal(resp, &detail); err != nil {
		return nil
	}
	return &detail
}

func (s *ObjectStorageServiceImpl) ListIamUsers(vpcId, s3ServiceId string, page, pageSize int) (IamUserListResponse, error) {
	apiPath := common.ApiPath.ListIamUsers(vpcId, s3ServiceId, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return IamUserListResponse{Total: 0}, fmt.Errorf("failed to list IAM users: %v", err)
	}

	var list IamUserListResponse
	if err := json.Unmarshal(resp, &list); err != nil {
		return IamUserListResponse{Total: 0}, fmt.Errorf("failed to unmarshal IAM user list response: %v", err)
	}
	return list, nil
}

// DeleteIamUser removes the user after the backend has revoked its access keys
// and removed its policies. A cleanup that aborts part way answers with a
// failure envelope under a 2xx, so the body is what decides here, not the
// status code.
func (s *ObjectStorageServiceImpl) DeleteIamUser(vpcId, s3ServiceId, userName string) error {
	apiPath := common.ApiPath.DeleteIamUser(vpcId, s3ServiceId, userName)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return fmt.Errorf("failed to delete IAM user: %v", err)
	}
	if r := decodeCommonResponse(resp, ""); !r.Status {
		return fmt.Errorf("failed to delete IAM user: %s", r.Message)
	}
	return nil
}

func (s *ObjectStorageServiceImpl) ListIamUserAccessKeys(vpcId, s3ServiceId, userName string) (IamUserAccessKeyListResponse, error) {
	apiPath := common.ApiPath.ListIamUserAccessKeys(vpcId, s3ServiceId, userName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return IamUserAccessKeyListResponse{Total: 0}, fmt.Errorf("failed to list IAM user access keys: %v", err)
	}

	var list IamUserAccessKeyListResponse
	if err := json.Unmarshal(resp, &list); err != nil {
		return IamUserAccessKeyListResponse{Total: 0}, fmt.Errorf("failed to unmarshal IAM user access key list response: %v", err)
	}
	return list, nil
}

// CreateIamUserAccessKey mints a credential. The secret in the response is the
// only copy that will ever exist, so a caller that discards it cannot recover
// it - which is also why access keys are never adopted by a reconcile.
func (s *ObjectStorageServiceImpl) CreateIamUserAccessKey(vpcId, s3ServiceId, userName string) *CreateIamUserAccessKeyResponse {
	apiPath := common.ApiPath.CreateIamUserAccessKey(vpcId, s3ServiceId, userName)
	resp, err := s.client.SendPostRequest(apiPath, nil)
	if err != nil {
		return &CreateIamUserAccessKeyResponse{Status: false, Message: err.Error()}
	}

	var created CreateIamUserAccessKeyResponse
	if err := json.Unmarshal(resp, &created); err != nil {
		return &CreateIamUserAccessKeyResponse{Status: false, Message: err.Error()}
	}
	return &created
}

func (s *ObjectStorageServiceImpl) DeleteIamUserAccessKey(vpcId, s3ServiceId, userName, accessKeyId string) CommonResponse {
	apiPath := common.ApiPath.DeleteIamUserAccessKey(vpcId, s3ServiceId, userName, accessKeyId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "IAM user access key revoked successfully")
}

func (s *ObjectStorageServiceImpl) GetIamUserPolicy(vpcId, s3ServiceId, userName string) *IamPolicyResponse {
	return s.getIamPolicy(common.ApiPath.GetIamUserPolicy(vpcId, s3ServiceId, userName))
}

func (s *ObjectStorageServiceImpl) PutIamUserPolicy(vpcId, s3ServiceId, userName, policy string) CommonResponse {
	return s.putIamPolicy(common.ApiPath.PutIamUserPolicy(vpcId, s3ServiceId, userName), policy)
}

func (s *ObjectStorageServiceImpl) DeleteIamUserPolicy(vpcId, s3ServiceId, userName string) CommonResponse {
	return s.deleteIamPolicy(common.ApiPath.DeleteIamUserPolicy(vpcId, s3ServiceId, userName))
}

// The three helpers below are shared by the user and role policy endpoints,
// which are the same operation on a different identity.

func (s *ObjectStorageServiceImpl) getIamPolicy(apiPath string) *IamPolicyResponse {
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil
	}
	if r := decodeCommonResponse(resp, ""); !r.Status {
		return nil
	}

	var policy IamPolicyResponse
	if err := json.Unmarshal(resp, &policy); err != nil {
		return nil
	}
	return &policy
}

func (s *ObjectStorageServiceImpl) putIamPolicy(apiPath, policy string) CommonResponse {
	// Rejected here rather than at the API: the payload must carry a JSON
	// object, and sending an unparseable document would arrive as a quoted
	// string and come back as a validator error about the wrong thing.
	if !json.Valid([]byte(policy)) {
		return CommonResponse{Status: false, Message: "the policy is not valid JSON"}
	}

	resp, err := s.client.SendPutRequest(apiPath, IamPolicyRequest{Policy: json.RawMessage(policy)})
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "IAM policy applied successfully")
}

func (s *ObjectStorageServiceImpl) deleteIamPolicy(apiPath string) CommonResponse {
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "IAM policy deleted successfully")
}

func (s *ObjectStorageServiceImpl) CreateIamRole(vpcId, s3ServiceId, roleName string, trustedUsers []string) *CreateIamRoleResponse {
	apiPath := common.ApiPath.CreateIamRole(vpcId, s3ServiceId)
	payload := IamRoleRequest{RoleName: roleName, TrustedUsers: trustedUsers}
	if payload.TrustedUsers == nil {
		// The field is not optional on the wire; a nil slice would marshal to
		// null, which the backend reads as a missing list rather than an empty
		// one.
		payload.TrustedUsers = []string{}
	}

	resp, err := s.client.SendPostRequest(apiPath, payload)
	if err != nil {
		return &CreateIamRoleResponse{Status: false, Message: err.Error()}
	}

	var created CreateIamRoleResponse
	if err := json.Unmarshal(resp, &created); err != nil {
		return &CreateIamRoleResponse{Status: false, Message: err.Error()}
	}
	return &created
}

func (s *ObjectStorageServiceImpl) GetIamRole(vpcId, s3ServiceId, roleName string) *IamRole {
	apiPath := common.ApiPath.GetIamRole(vpcId, s3ServiceId, roleName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil
	}
	if r := decodeCommonResponse(resp, ""); !r.Status {
		return nil
	}

	var role IamRole
	if err := json.Unmarshal(resp, &role); err != nil {
		return nil
	}
	return &role
}

func (s *ObjectStorageServiceImpl) ListIamRoles(vpcId, s3ServiceId string, page, pageSize int) (IamRoleListResponse, error) {
	apiPath := common.ApiPath.ListIamRoles(vpcId, s3ServiceId, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return IamRoleListResponse{Total: 0}, fmt.Errorf("failed to list IAM roles: %v", err)
	}

	var list IamRoleListResponse
	if err := json.Unmarshal(resp, &list); err != nil {
		return IamRoleListResponse{Total: 0}, fmt.Errorf("failed to unmarshal IAM role list response: %v", err)
	}
	return list, nil
}

// UpdateIamRoleTrustedUsers rewrites the whole trust policy. It is a replace,
// not a merge, and the response reports the list that landed.
func (s *ObjectStorageServiceImpl) UpdateIamRoleTrustedUsers(vpcId, s3ServiceId, roleName string, trustedUsers []string) *IamRoleTrustedUsersResponse {
	apiPath := common.ApiPath.UpdateIamRoleTrustedUsers(vpcId, s3ServiceId, roleName)
	payload := IamRoleTrustedUsersRequest{TrustedUsers: trustedUsers}
	if payload.TrustedUsers == nil {
		payload.TrustedUsers = []string{}
	}

	resp, err := s.client.SendPutRequest(apiPath, payload)
	if err != nil {
		return &IamRoleTrustedUsersResponse{Status: false, Message: err.Error()}
	}

	var updated IamRoleTrustedUsersResponse
	if err := json.Unmarshal(resp, &updated); err != nil {
		return &IamRoleTrustedUsersResponse{Status: false, Message: err.Error()}
	}
	return &updated
}

func (s *ObjectStorageServiceImpl) DeleteIamRole(vpcId, s3ServiceId, roleName string) error {
	apiPath := common.ApiPath.DeleteIamRole(vpcId, s3ServiceId, roleName)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return fmt.Errorf("failed to delete IAM role: %v", err)
	}
	if r := decodeCommonResponse(resp, ""); !r.Status {
		return fmt.Errorf("failed to delete IAM role: %s", r.Message)
	}
	return nil
}

func (s *ObjectStorageServiceImpl) GetIamRolePolicy(vpcId, s3ServiceId, roleName string) *IamPolicyResponse {
	return s.getIamPolicy(common.ApiPath.GetIamRolePolicy(vpcId, s3ServiceId, roleName))
}

func (s *ObjectStorageServiceImpl) PutIamRolePolicy(vpcId, s3ServiceId, roleName, policy string) CommonResponse {
	return s.putIamPolicy(common.ApiPath.PutIamRolePolicy(vpcId, s3ServiceId, roleName), policy)
}

func (s *ObjectStorageServiceImpl) DeleteIamRolePolicy(vpcId, s3ServiceId, roleName string) CommonResponse {
	return s.deleteIamPolicy(common.ApiPath.DeleteIamRolePolicy(vpcId, s3ServiceId, roleName))
}
