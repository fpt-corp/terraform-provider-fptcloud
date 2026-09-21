package fptcloud_object_storage

import (
	"encoding/json"
	"fmt"
	"log"
	common "terraform-provider-fptcloud/commons"
)

const (
	// CheckServiceEnable reports an API failure as an empty service list, which is
	// indistinguishable here from a region that genuinely is not enabled. The
	// message therefore names both causes, and the real API error is in the
	// provider log (TF_LOG=DEBUG).
	regionError = "region %s is not enabled, or the object storage service list could not be read - check credentials and connectivity, and see the provider log for the API error"
)

// ObjectStorageService defines the interface for object storage operations
type ObjectStorageService interface {
	CheckServiceEnable(vpcId string) S3ServiceEnableResponse

	// Bucket
	ListBuckets(vpcId, s3ServiceId string, page, pageSize int) ListBucketResponse
	CreateBucket(req BucketRequest, vpcId, s3ServiceId string) CommonResponse
	DeleteBucket(vpcId, s3ServiceId, bucketName string) CommonResponse

	// IAM user
	CreateIamUser(vpcId, s3ServiceId, userName string) *CreateIamUserResponse
	GetIamUser(vpcId, s3ServiceId, userName string) *IamUserDetail
	ListIamUsers(vpcId, s3ServiceId string, page, pageSize int) (IamUserListResponse, error)
	DeleteIamUser(vpcId, s3ServiceId, userName string) error
	ListIamUserAccessKeys(vpcId, s3ServiceId, userName string) (IamUserAccessKeyListResponse, error)
	CreateIamUserAccessKey(vpcId, s3ServiceId, userName string) *CreateIamUserAccessKeyResponse
	DeleteIamUserAccessKey(vpcId, s3ServiceId, userName, accessKeyId string) CommonResponse
	GetIamUserPolicy(vpcId, s3ServiceId, userName string) *IamPolicyResponse
	PutIamUserPolicy(vpcId, s3ServiceId, userName, policy string) CommonResponse
	DeleteIamUserPolicy(vpcId, s3ServiceId, userName string) CommonResponse

	// IAM role
	CreateIamRole(vpcId, s3ServiceId, roleName string, trustedUsers []string) *CreateIamRoleResponse
	GetIamRole(vpcId, s3ServiceId, roleName string) *IamRole
	ListIamRoles(vpcId, s3ServiceId string, page, pageSize int) (IamRoleListResponse, error)
	UpdateIamRoleTrustedUsers(vpcId, s3ServiceId, roleName string, trustedUsers []string) *IamRoleTrustedUsersResponse
	DeleteIamRole(vpcId, s3ServiceId, roleName string) error
	GetIamRolePolicy(vpcId, s3ServiceId, roleName string) *IamPolicyResponse
	PutIamRolePolicy(vpcId, s3ServiceId, roleName, policy string) CommonResponse
	DeleteIamRolePolicy(vpcId, s3ServiceId, roleName string) CommonResponse

	// Access key
	ListAccessKeys(vpcId, s3ServiceId string) (AccessKey, error)
	DeleteAccessKey(vpcId, s3ServiceId, accessKeyId string) CommonResponse
	CreateAccessKey(vpcId, s3ServiceId string) *CreateAccessKeyResponse

	// Sub user
	CreateSubUser(req SubUser, vpcId, s3ServiceId string) *CommonResponse
	DeleteSubUser(vpcId, s3ServiceId, subUserId string) error
	ListSubUsers(vpcId, s3ServiceId string, page, pageSize int) (SubUserListResponse, error)
	DetailSubUser(vpcId, s3ServiceId, subUserId string) *DetailSubUser
	CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId string) *SubUserCreateKeyResponse
	DeleteSubUserAccessKey(vpcId, s3ServiceId, subUserId, accessKeyId string) CommonResponse

	// bucket configuration
	PutBucketPolicy(vpcId, s3ServiceId, bucketName string, policy interface{}) CommonResponse
	GetBucketPolicy(vpcId, s3ServiceId, bucketName string) *BucketPolicyResponse

	// CORS configuration
	CreateBucketCors(vpcId, s3ServiceId, bucketName string, cors map[string]interface{}) CommonResponse
	UpdateBucketCors(vpcId, s3ServiceId, bucketName string, cors []map[string]interface{}) CommonResponse
	GetBucketCors(vpcId, s3ServiceId, bucketName string, page, pageSize int) (*BucketCorsResponse, error)

	// Versioning configuration
	PutBucketVersioning(vpcId, s3ServiceId, bucketName string, versioning BucketVersioningRequest) error
	GetBucketVersioning(vpcId, s3ServiceId, bucketName string) *BucketVersioningResponse

	// Acl configuration
	PutBucketAcl(vpcId, s3ServiceId, bucketName string, acl BucketAclRequest) PutBucketAclResponse
	GetBucketAcl(vpcId, s3ServiceId, bucketName string) *BucketAclResponse

	// Static website configuration
	PutBucketWebsite(vpcId, s3ServiceId, bucketName string, website BucketWebsiteRequest) CommonResponse
	GetBucketWebsite(vpcId, s3ServiceId, bucketName string) *BucketWebsiteResponse
	DeleteBucketStaticWebsite(vpcId, s3ServiceId, bucketName string) CommonResponse

	// Lifecycle configuration
	GetBucketLifecycle(vpcId, s3ServiceId, bucketName string, page, pageSize int) BucketLifecycleResponse
	PutBucketLifecycle(vpcId, s3ServiceId, bucketName string, lifecycle map[string]interface{}) CommonResponse
	DeleteBucketLifecycle(vpcId, s3ServiceId, bucketName string, lifecycle map[string]interface{}) CommonResponse
}

// ObjectStorageServiceImpl is the implementation of ObjectStorageService
type ObjectStorageServiceImpl struct {
	client *common.Client
}

// NewObjectStorageService creates a new instance of ObjectStorageService
func NewObjectStorageService(client *common.Client) ObjectStorageService {
	return &ObjectStorageServiceImpl{client: client}
}

// decodeCommonResponse interprets the body of a call that did not fail at the
// HTTP layer. The API reports some logical failures as {"status": false} under a
// 2xx, so assuming success from the status code alone records state for work the
// backend refused to do.
//
// Status is a pointer because an absent field must not be read as an explicit
// false: endpoints that answer with no body, or with a body that carries no
// status, have genuinely succeeded.
func decodeCommonResponse(resp []byte, successMessage string) CommonResponse {
	var body struct {
		Status  *bool  `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp, &body); err != nil {
		// Not the usual envelope; the call itself succeeded, so take it as success.
		return CommonResponse{Status: true, Message: successMessage}
	}
	if body.Status != nil && !*body.Status {
		message := body.Message
		if message == "" {
			message = "the API reported a failure without a message"
		}
		return CommonResponse{Status: false, Message: message}
	}
	return CommonResponse{Status: true, Message: successMessage}
}

func (s *ObjectStorageServiceImpl) CheckServiceEnable(vpcId string) S3ServiceEnableResponse {
	apiPath := common.ApiPath.CheckS3ServiceEnable(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		log.Printf("[ERROR] FPTCloud Provider - CheckServiceEnable API request failed: %v", err)
		return S3ServiceEnableResponse{Total: 0}
	}

	var response S3ServiceEnableResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		log.Printf("[ERROR] FPTCloud Provider - CheckServiceEnable JSON Unmarshal failed: %v, Body: %s", err, string(resp))
		return S3ServiceEnableResponse{Total: 0}
	}
	return response
}

func (s *ObjectStorageServiceImpl) CreateBucket(req BucketRequest, vpcId, s3ServiceId string) CommonResponse {

	apiPath := common.ApiPath.CreateBucket(vpcId, s3ServiceId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}

	var bucket CommonResponse
	err = json.Unmarshal(resp, &bucket)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}

	return CommonResponse{Status: bucket.Status, Message: bucket.Message}
}

// CreateSubUser creates a new sub-user
func (s *ObjectStorageServiceImpl) CreateSubUser(req SubUser, vpcId, s3ServiceId string) *CommonResponse {
	apiPath := common.ApiPath.CreateSubUser(vpcId, s3ServiceId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return &CommonResponse{Status: false, Message: err.Error()}
	}

	var subUser CommonResponse
	err = json.Unmarshal(resp, &subUser)
	if err != nil {
		return &CommonResponse{Status: false, Message: err.Error()}
	}
	return &CommonResponse{Status: subUser.Status, Message: "Sub-user created successfully"}
}

func (s *ObjectStorageServiceImpl) CreateAccessKey(vpcId, s3ServiceId string) *CreateAccessKeyResponse {
	apiPath := common.ApiPath.CreateAccessKey(vpcId, s3ServiceId)
	resp, err := s.client.SendPostRequest(apiPath, nil)
	if err != nil {
		return &CreateAccessKeyResponse{Status: false, Message: err.Error()}
	}

	var accessKey CreateAccessKeyResponse
	err = json.Unmarshal(resp, &accessKey)
	if err != nil {

		return &CreateAccessKeyResponse{Status: false, Message: err.Error()}
	}
	return &accessKey
}

func (s *ObjectStorageServiceImpl) ListBuckets(vpcId, s3ServiceId string, page, pageSize int) ListBucketResponse {
	apiPath := common.ApiPath.ListBuckets(vpcId, s3ServiceId, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return ListBucketResponse{Total: 0}
	}

	var buckets ListBucketResponse
	err = json.Unmarshal(resp, &buckets)
	if err != nil {
		return ListBucketResponse{Total: 0}
	}

	return buckets
}

func (s *ObjectStorageServiceImpl) ListSubUsers(vpcId, s3ServiceId string, page, pageSize int) (SubUserListResponse, error) {
	apiPath := common.ApiPath.ListSubUsers(vpcId, s3ServiceId, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return SubUserListResponse{Total: 0}, fmt.Errorf("failed to list sub-users: %v", err)
	}

	var subUsers SubUserListResponse
	err = json.Unmarshal(resp, &subUsers)
	if err != nil {
		return SubUserListResponse{Total: 0}, fmt.Errorf("failed to unmarshal sub-user list response: %v", err)
	}

	return subUsers, nil
}

func (s *ObjectStorageServiceImpl) ListAccessKeys(vpcId, s3ServiceId string) (AccessKey, error) {
	apiPath := common.ApiPath.ListAccessKeys(vpcId, s3ServiceId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return AccessKey{}, fmt.Errorf("failed to list access keys: %v", err)
	}

	var accessKey AccessKey
	err = json.Unmarshal(resp, &accessKey)
	if err != nil {
		return AccessKey{}, fmt.Errorf("failed to unmarshal access key list response: %v", err)
	}

	return accessKey, nil
}

func (s *ObjectStorageServiceImpl) DeleteBucket(vpcId, s3ServiceId, bucketName string) CommonResponse {
	apiPath := common.ApiPath.DeleteBucket(vpcId, s3ServiceId)
	payload := map[string]string{"name": bucketName}

	resp, err := s.client.SendDeleteRequestWithBody(apiPath, payload)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "Bucket deleted successfully")
}

func (s *ObjectStorageServiceImpl) DeleteAccessKey(vpcId, s3ServiceId, accessKeyId string) CommonResponse {
	apiPath := common.ApiPath.DeleteAccessKey(vpcId, s3ServiceId)
	body := map[string]string{"accessKey": accessKeyId}

	resp, err := s.client.SendDeleteRequestWithBody(apiPath, body)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "Access key deleted successfully")
}

// Implement bucket policy methods
func (s *ObjectStorageServiceImpl) PutBucketPolicy(vpcId, s3ServiceId, bucketName string, policy interface{}) CommonResponse {
	apiPath := common.ApiPath.PutBucketPolicy(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPutRequest(apiPath, policy)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "Bucket policy updated successfully")
}

func (s *ObjectStorageServiceImpl) GetBucketPolicy(vpcId, s3ServiceId, bucketName string) *BucketPolicyResponse {
	apiPath := common.ApiPath.GetBucketPolicy(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return &BucketPolicyResponse{Status: false}
	}

	var policy BucketPolicyResponse
	if err := json.Unmarshal(resp, &policy); err != nil {
		return &BucketPolicyResponse{Status: false}
	}
	return &policy
}

// Implement CORS methods
func (s *ObjectStorageServiceImpl) CreateBucketCors(vpcId, s3ServiceId, bucketName string, cors map[string]interface{}) CommonResponse {
	apiPath := common.ApiPath.CreateBucketCors(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPostRequest(apiPath, cors)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "Bucket CORS configuration updated successfully")
}

func (s *ObjectStorageServiceImpl) UpdateBucketCors(vpcId, s3ServiceId, bucketName string, cors []map[string]interface{}) CommonResponse {
	apiPath := common.ApiPath.PutBucketCORS(vpcId, s3ServiceId, bucketName)
	// Backend expects { "CORSRules": [ ... ] }
	payload := map[string]interface{}{"CORSRules": cors}
	resp, err := s.client.SendPutRequest(apiPath, payload)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "Bucket CORS configuration updated successfully")
}

func (s *ObjectStorageServiceImpl) GetBucketCors(vpcId, s3ServiceId, bucketName string, page, pageSize int) (*BucketCorsResponse, error) {
	apiPath := common.ApiPath.GetBucketCORS(vpcId, s3ServiceId, bucketName, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket CORS: %v", err)
	}

	var cors BucketCorsResponse
	if err := json.Unmarshal(resp, &cors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket CORS: %v", err)
	}
	return &cors, nil
}

// Implement versioning methods
func (s *ObjectStorageServiceImpl) PutBucketVersioning(vpcId, s3ServiceId, bucketName string, versioning BucketVersioningRequest) error {
	apiPath := common.ApiPath.PutBucketVersioning(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPutRequest(apiPath, versioning)
	if err != nil {
		return fmt.Errorf("failed to put bucket versioning: %v", err)
	}
	if r := decodeCommonResponse(resp, ""); !r.Status {
		return fmt.Errorf("failed to put bucket versioning: %s", r.Message)
	}
	return nil
}

func (s *ObjectStorageServiceImpl) GetBucketVersioning(vpcId, s3ServiceId, bucketName string) *BucketVersioningResponse {
	apiPath := common.ApiPath.GetBucketVersioning(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return &BucketVersioningResponse{Status: false}
	}

	var versioning BucketVersioningResponse
	if err := json.Unmarshal(resp, &versioning); err != nil {
		return &BucketVersioningResponse{Status: false}
	}
	return &versioning
}

func (s *ObjectStorageServiceImpl) PutBucketWebsite(vpcId, s3ServiceId, bucketName string, website BucketWebsiteRequest) CommonResponse {
	apiPath := common.ApiPath.PutBucketWebsite(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPutRequest(apiPath, website)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "Bucket website configuration updated successfully")
}
func (s *ObjectStorageServiceImpl) DeleteBucketStaticWebsite(vpcId, s3ServiceId, bucketName string) CommonResponse {
	apiPath := common.ApiPath.DeleteBucketStaticWebsite(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "Bucket website configuration deleted successfully")
}
func (s *ObjectStorageServiceImpl) GetBucketWebsite(vpcId, s3ServiceId, bucketName string) *BucketWebsiteResponse {
	apiPath := common.ApiPath.GetBucketWebsite(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return &BucketWebsiteResponse{Status: false}
	}

	var website BucketWebsiteResponse
	if err := json.Unmarshal(resp, &website); err != nil {
		return &BucketWebsiteResponse{Status: false}
	}
	return &website
}

func (s *ObjectStorageServiceImpl) PutBucketAcl(vpcId, s3ServiceId, bucketName string, acl BucketAclRequest) PutBucketAclResponse {
	apiPath := common.ApiPath.PutBucketAcl(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPutRequest(apiPath, acl)
	if err != nil {
		return PutBucketAclResponse{Status: false}
	}

	var putBucketAclResponse PutBucketAclResponse
	if err := json.Unmarshal(resp, &putBucketAclResponse); err != nil {
		return PutBucketAclResponse{Status: false}
	}
	return putBucketAclResponse
}

func (s *ObjectStorageServiceImpl) GetBucketAcl(vpcId, s3ServiceId, bucketName string) *BucketAclResponse {
	apiPath := common.ApiPath.GetBucketAcl(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return &BucketAclResponse{Status: false}
	}

	var acl BucketAclResponse
	if err := json.Unmarshal(resp, &acl); err != nil {
		return &BucketAclResponse{Status: false}
	}
	return &acl
}

func (s *ObjectStorageServiceImpl) DeleteSubUser(vpcId, s3ServiceId, subUserId string) error {
	apiPath := common.ApiPath.DeleteSubUser(vpcId, s3ServiceId, subUserId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return fmt.Errorf("failed to delete sub-user: %v", err)
	}
	if r := decodeCommonResponse(resp, ""); !r.Status {
		return fmt.Errorf("failed to delete sub-user: %s", r.Message)
	}
	return nil
}

func (s *ObjectStorageServiceImpl) GetBucketLifecycle(vpcId, s3ServiceId, bucketName string, page, pageSize int) BucketLifecycleResponse {
	apiPath := common.ApiPath.GetBucketLifecycle(vpcId, s3ServiceId, bucketName, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return BucketLifecycleResponse{Total: 0, Status: false}
	}

	var bucketLifecycle BucketLifecycleResponse
	if err := json.Unmarshal(resp, &bucketLifecycle); err != nil {
		return BucketLifecycleResponse{Total: 0, Status: false}
	}
	return bucketLifecycle
}

func (s *ObjectStorageServiceImpl) PutBucketLifecycle(vpcId, s3ServiceId, bucketName string, lifecycle map[string]interface{}) CommonResponse {
	apiPath := common.ApiPath.PutBucketLifecycle(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPostRequest(apiPath, lifecycle)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}

	var bucketLifecycle CommonResponse
	if err := json.Unmarshal(resp, &bucketLifecycle); err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return bucketLifecycle
}

func (s *ObjectStorageServiceImpl) DeleteBucketLifecycle(vpcId, s3ServiceId, bucketName string, lifecycle map[string]interface{}) CommonResponse {
	apiPath := common.ApiPath.DeleteBucketLifecycle(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPutRequest(apiPath, lifecycle)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}

	var bucketLifecycle CommonResponse
	if err := json.Unmarshal(resp, &bucketLifecycle); err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return bucketLifecycle
}

func (s *ObjectStorageServiceImpl) CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId string) *SubUserCreateKeyResponse {
	apiPath := common.ApiPath.CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId)
	resp, err := s.client.SendPostRequest(apiPath, nil)
	if err != nil {
		return &SubUserCreateKeyResponse{Status: false, Message: err.Error()}
	}

	var subUserKeys SubUserCreateKeyResponse
	if err := json.Unmarshal(resp, &subUserKeys); err != nil {
		return &SubUserCreateKeyResponse{Status: false, Message: err.Error()}
	}
	return &subUserKeys
}

func (s *ObjectStorageServiceImpl) DeleteSubUserAccessKey(vpcId, s3ServiceId, subUserId, accessKeyId string) CommonResponse {
	apiPath := common.ApiPath.DeleteSubUserAccessKey(vpcId, s3ServiceId, subUserId)
	payload := map[string]string{"accessKey": accessKeyId}
	resp, err := s.client.SendDeleteRequestWithBody(apiPath, payload)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}
	return decodeCommonResponse(resp, "Access key deleted successfully")
}

func (s *ObjectStorageServiceImpl) DetailSubUser(vpcId, s3ServiceId, subUserId string) *DetailSubUser {
	apiPath := common.ApiPath.DetailSubUser(vpcId, s3ServiceId, subUserId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil
	}

	var detail DetailSubUser
	if err := json.Unmarshal(resp, &detail); err != nil {
		return nil
	}
	return &detail
}
