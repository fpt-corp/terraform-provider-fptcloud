package fptcloud_object_storage

import (
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"
	"time"
)

// SubUserCreateRequest represents the request body for creating a sub-user
type SubUserCreateRequest struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions"`
}
type AccessKey struct {
	Credentials []struct {
		ID          string `json:"id"`
		Credentials []struct {
			AccessKey   string      `json:"accessKey"`
			Active      bool        `json:"active"`
			CreatedDate interface{} `json:"createdDate"`
		} `json:"credentials"`
	} `json:"credentials"`
}
type CreateAccessKeyResponse struct {
	Status     bool   `json:"status"`
	Message    string `json:"message,omitempty"`
	Credential struct {
		AccessKey   string      `json:"accessKey"`
		SecretKey   string      `json:"secretKey"`
		Active      interface{} `json:"active"`
		CreatedDate interface{} `json:"createdDate"`
	} `json:"credential,omitempty"`
}
type SubUserCreateKeyResponse struct {
	Status     bool `json:"status"`
	Credential struct {
		AccessKey   string      `json:"accessKey"`
		SecretKey   string      `json:"secretKey"`
		Active      interface{} `json:"active"`
		CreatedDate interface{} `json:"createdDate"`
	} `json:"credential"`
}

type SubUser struct {
	Role   string `json:"role"`
	UserId string `json:"user_id"`
}
type SubUserListResponse struct {
	SubUsers []struct {
		UserID     string      `json:"user_id"`
		Arn        string      `json:"arn"`
		Active     bool        `json:"active"`
		Role       string      `json:"role"`
		CreatedAt  interface{} `json:"created_at"`
		AccessKeys interface{} `json:"access_keys"`
	} `json:"sub_users"`
	Total int `json:"total"`
}
type CommonResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message,omitempty"`
}
type CorsRule struct {
	ID             string   `json:"ID,omitempty"`
	AllowedOrigins []string `json:"AllowedOrigins"`
	AllowedMethods []string `json:"AllowedMethods"`
	MaxAgeSeconds  int      `json:"MaxAgeSeconds,omitempty"`
	ExposeHeaders  []string `json:"ExposeHeaders,omitempty"`
	AllowedHeaders []string `json:"AllowedHeaders,omitempty"`
}
type BucketCors struct {
	CorsRules []CorsRule `json:"CORSRules"`
}
type BucketCorsResponse struct {
	Status    bool       `json:"status"`
	Total     int        `json:"total"`
	CorsRules []CorsRule `json:"cors_rules"`
}

type BucketPolicyResponse struct {
	Status bool   `json:"status"`
	Policy string `json:"policy"`
}
type BucketPolicyRequest struct {
	Policy string `json:"policy"`
}
type Statement struct {
	Sid       string                 `json:"Sid"`
	Effect    string                 `json:"Effect"`
	Principal map[string]interface{} `json:"Principal"`
	Action    []string               `json:"Action"`
	Resource  []string               `json:"Resource"`
}

type BucketVersioningRequest struct {
	Status string `json:"status"` // "Enabled" or "Suspended"
}

type BucketAclResponse struct {
	Status bool `json:"status"`
	Owner  struct {
		DisplayName string `json:"DisplayName"`
		ID          string `json:"ID"`
	} `json:"Owner"`
	Grants []struct {
		Grantee struct {
			DisplayName string `json:"DisplayName"`
			ID          string `json:"ID"`
			Type        string `json:"Type"`
		} `json:"Grantee"`
		Permission string `json:"Permission"`
	} `json:"Grants"`
	CannedACL string `json:"CannedACL"`
}
type BucketAclRequest struct {
	CannedAcl    string `json:"cannedAcl"`
	ApplyObjects bool   `json:"applyObjects"`
}
type PutBucketAclResponse struct {
	Status bool `json:"status"`
	// TaskID may be empty if applyObjects is false, if applyObjects is true, the taskID will be returned
	TaskID string `json:"taskId"`
}
type BucketWebsiteRequest struct {
	Key    string `json:"key"`
	Suffix string `json:"suffix"`
	Bucket string `json:"bucket"`
}
type BucketWebsiteResponse struct {
	Status bool `json:"status"`
	Config struct {
		ResponseMetadata struct {
			RequestID      string `json:"RequestId"`
			HostID         string `json:"HostId"`
			HTTPStatusCode int    `json:"HTTPStatusCode"`
			HTTPHeaders    struct {
				XAmzRequestID string `json:"x-amz-request-id"`
				ContentType   string `json:"content-type"`
				ContentLength string `json:"content-length"`
				Date          string `json:"date"`
			} `json:"HTTPHeaders"`
			RetryAttempts int `json:"RetryAttempts"`
		} `json:"ResponseMetadata"`
		IndexDocument struct {
			Suffix string `json:"Suffix"`
		} `json:"IndexDocument"`
		ErrorDocument struct {
			Key string `json:"Key"`
		} `json:"ErrorDocument"`
	} `json:"config,omitempty"`
}

type S3ServiceEnableResponse struct {
	Data []struct {
		S3ServiceName      string      `json:"s3_service_name"`
		S3ServiceID        string      `json:"s3_service_id"`
		S3Platform         string      `json:"s3_platform"`
		DefaultUser        interface{} `json:"default_user"`
		MigrateQuota       int         `json:"migrate_quota"`
		SyncQuota          int         `json:"sync_quota"`
		RgwTotalNodes      int         `json:"rgw_total_nodes"`
		RgwUserActiveNodes int         `json:"rgw_user_active_nodes"`
		HasUnusualConfig   interface{} `json:"has_unusual_config"`
	} `json:"data"`
	Total int `json:"total"`
}

// Bucket represents the response structure for a created bucket
type BucketRequest struct {
	Name       string `json:"name"`
	Region     string `json:"region"`
	Versioning string `json:"versioning"`
	Acl        string `json:"acl"`
}
type ListBucketResponse struct {
	Buckets []struct {
		Name             string    `json:"Name"`
		CreationDate     time.Time `json:"CreationDate"`
		IsEmpty          bool      `json:"isEmpty"`
		S3ServiceID      string    `json:"s3_service_id"`
		IsEnabledLogging bool      `json:"isEnabledLogging"`
		Endpoint         string    `json:"endpoint"`
	} `json:"buckets"`
	Total int `json:"total"`
}
type BucketLifecycleResponse struct {
	Status bool `json:"status"`
	Rules  []struct {
		Expiration struct {
			Days int `json:"Days"`
		} `json:"Expiration"`
		ID     string `json:"ID"`
		Filter struct {
			Prefix string `json:"Prefix"`
		} `json:"Filter"`
		Status                         string `json:"Status"`
		AbortIncompleteMultipartUpload struct {
			DaysAfterInitiation int `json:"DaysAfterInitiation"`
		} `json:"AbortIncompleteMultipartUpload"`
	} `json:"rules"`
	Total int `json:"total"`
}

type DetailSubUser struct {
	UserID     string      `json:"user_id"`
	Arn        interface{} `json:"arn"`
	Active     bool        `json:"active"`
	Role       string      `json:"role"`
	CreatedAt  interface{} `json:"created_at"`
	AccessKeys []string    `json:"access_keys"`
}

// ObjectStorageService defines the interface for object storage operations
type ObjectStorageService interface {
	CheckServiceEnable(vpcId string) S3ServiceEnableResponse

	// Bucket
	ListBuckets(vpcId, s3ServiceId string, page, pageSize int) ListBucketResponse
	CreateBucket(req BucketRequest, vpcId, s3ServiceId string) CommonResponse
	DeleteBucket(vpcId, s3ServiceId, bucketName string) CommonResponse

	// Access key
	ListAccessKeys(vpcId, s3ServiceId string) (AccessKey, error)
	DeleteAccessKey(vpcId, s3ServiceId, accessKeyId string) error
	CreateAccessKey(vpcId, s3ServiceId string) *CreateAccessKeyResponse

	// Sub user
	CreateSubUser(req SubUser, vpcId, s3ServiceId string) *CommonResponse
	DeleteSubUser(vpcId, s3ServiceId, subUserId string) error
	ListSubUsers(vpcId, s3ServiceId string) ([]SubUserListResponse, error)
	DetailSubUser(vpcId, s3ServiceId, subUserId string) *DetailSubUser
	CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId string) *SubUserCreateKeyResponse
	DeleteSubUserAccessKey(vpcId, s3ServiceId, subUserId, accessKeyId string) CommonResponse

	// bucket configuration
	PutBucketPolicy(vpcId, s3ServiceId, bucketName string, policy BucketPolicyRequest) CommonResponse
	GetBucketPolicy(vpcId, s3ServiceId, bucketName string) *BucketPolicyResponse

	// CORS configuration
	PutBucketCors(bucketName, vpcId, s3ServiceId string, cors CorsRule) (CommonResponse, error)
	UpdateBucketCors(bucketName, vpcId, s3ServiceId string, cors BucketCors) (CommonResponse, error)
	GetBucketCors(vpcId, s3ServiceId, bucketName string) (*BucketCors, error)

	// Versioning configuration
	PutBucketVersioning(vpcId, s3ServiceId, bucketName string, versioning BucketVersioningRequest) error
	GetBucketVersioning(vpcId, s3ServiceId, bucketName string) *BucketVersioningRequest

	// Acl configuration
	PutBucketAcl(vpcId, s3ServiceId, bucketName string, acl BucketAclRequest) PutBucketAclResponse
	GetBucketAcl(vpcId, s3ServiceId, bucketName string) *BucketAclResponse

	// Static website configuration
	PutBucketWebsite(vpcId, s3ServiceId, bucketName string, website BucketWebsiteRequest) CommonResponse
	GetBucketWebsite(vpcId, s3ServiceId, bucketName string) *BucketWebsiteResponse
	DeleteBucketStaticWebsite(vpcId, s3ServiceId, bucketName string) CommonResponse

	// Lifecycle configuration
	GetBucketLifecycle(vpcId, s3ServiceId, bucketName, page, pageSize string) (*BucketLifecycleResponse, error)
	PutBucketLifecycle(vpcId, s3ServiceId, bucketName string, lifecycle interface{}) (*BucketLifecycleResponse, error)
	DeleteBucketLifecycle(vpcId, s3ServiceId, bucketName string, lifecycle interface{}) (*BucketLifecycleResponse, error)
}

// ObjectStorageServiceImpl is the implementation of ObjectStorageService
type ObjectStorageServiceImpl struct {
	client *common.Client
}

// NewObjectStorageService creates a new instance of ObjectStorageService
func NewObjectStorageService(client *common.Client) ObjectStorageService {
	return &ObjectStorageServiceImpl{client: client}
}

func (s *ObjectStorageServiceImpl) CheckServiceEnable(vpcId string) S3ServiceEnableResponse {
	apiPath := common.ApiPath.CheckS3ServiceEnable(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return S3ServiceEnableResponse{}
	}

	var response S3ServiceEnableResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return S3ServiceEnableResponse{}
	}
	return response
}

func (s *ObjectStorageServiceImpl) CreateBucket(req BucketRequest, vpcId, s3ServiceId string) CommonResponse {

	apiPath := common.ApiPath.CreateBucket(vpcId, s3ServiceId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}

	var bucket BucketRequest
	err = json.Unmarshal(resp, &bucket)
	if err != nil {
		return CommonResponse{Status: false, Message: err.Error()}
	}

	return CommonResponse{Status: true, Message: "Bucket created successfully"}
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

func (s *ObjectStorageServiceImpl) ListSubUsers(vpcId, s3ServiceId string) ([]SubUserListResponse, error) {
	apiPath := common.ApiPath.ListSubUsers(vpcId, s3ServiceId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-users: %v", err)
	}

	var subUsers []SubUserListResponse
	err = json.Unmarshal(resp, &subUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sub-user list response: %v", err)
	}

	return subUsers, nil
}

func (s *ObjectStorageServiceImpl) ListAccessKeys(vpcId, s3ServiceId string) (AccessKey, error) {
	apiPath := common.ApiPath.ListAccessKeys(vpcId, s3ServiceId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return AccessKey{}, fmt.Errorf("failed to list access keys: %v", err)
	}

	var accessKeys AccessKey
	err = json.Unmarshal(resp, &accessKeys)
	if err != nil {
		return AccessKey{}, fmt.Errorf("failed to unmarshal access key list response: %v", err)
	}

	return accessKeys, nil
}

func (s *ObjectStorageServiceImpl) DeleteBucket(vpcId, s3ServiceId, bucketName string) CommonResponse {
	apiPath := common.ApiPath.DeleteBucket(vpcId, s3ServiceId)
	payload := map[string]string{"name": bucketName}

	if _, err := s.client.SendDeleteRequestWithBody(apiPath, payload); err != nil {

		return CommonResponse{Status: false}
	}
	return CommonResponse{Status: true, Message: "Bucket deleted successfully"}
}

func (s *ObjectStorageServiceImpl) DeleteAccessKey(vpcId, s3ServiceId, accessKeyId string) error {
	apiPath := common.ApiPath.DeleteAccessKey(vpcId, s3ServiceId)
	body := map[string]string{"accessKey": accessKeyId}
	if _, err := s.client.SendDeleteRequestWithBody(apiPath, body); err != nil {
		return fmt.Errorf("failed to delete access key: %v", err)
	}
	return nil
}

// Implement bucket policy methods
func (s *ObjectStorageServiceImpl) PutBucketPolicy(vpcId, s3ServiceId, bucketName string, policy BucketPolicyRequest) CommonResponse {
	apiPath := common.ApiPath.PutBucketPolicy(vpcId, s3ServiceId, bucketName)
	if _, err := s.client.SendPutRequest(apiPath, policy); err != nil {
		return CommonResponse{Status: false}
	}
	return CommonResponse{Status: true}
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
func (s *ObjectStorageServiceImpl) PutBucketCors(bucketName, vpcId, s3ServiceId string, cors CorsRule) (CommonResponse, error) {
	apiPath := common.ApiPath.PutBucketCORS(vpcId, s3ServiceId, bucketName)
	if _, err := s.client.SendPutRequest(apiPath, cors); err != nil {
		return CommonResponse{Status: false}, fmt.Errorf("failed to update bucket CORS: %v", err)
	}
	return CommonResponse{Status: true}, nil
}

func (s *ObjectStorageServiceImpl) UpdateBucketCors(bucketName, vpcId, s3ServiceId string, cors BucketCors) (CommonResponse, error) {
	apiPath := common.ApiPath.PutBucketCORS(vpcId, s3ServiceId, bucketName)
	if _, err := s.client.SendPutRequest(apiPath, cors); err != nil {
		return CommonResponse{Status: false}, fmt.Errorf("failed to update bucket CORS: %v", err)
	}
	return CommonResponse{Status: true}, nil
}

func (s *ObjectStorageServiceImpl) GetBucketCors(vpcId, s3ServiceId, bucketName string) (*BucketCors, error) {
	apiPath := common.ApiPath.GetBucketCORS(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket CORS: %v", err)
	}

	var cors BucketCors
	if err := json.Unmarshal(resp, &cors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket CORS: %v", err)
	}
	return &cors, nil
}

// Implement versioning methods
func (s *ObjectStorageServiceImpl) PutBucketVersioning(vpcId, s3ServiceId, bucketName string, versioning BucketVersioningRequest) error {
	apiPath := common.ApiPath.PutBucketVersioning(vpcId, s3ServiceId, bucketName)
	if _, err := s.client.SendPutRequest(apiPath, versioning); err != nil {
		return fmt.Errorf("failed to put bucket versioning: %v", err)
	}
	return nil
}

func (s *ObjectStorageServiceImpl) GetBucketVersioning(vpcId, s3ServiceId, bucketName string) *BucketVersioningRequest {
	apiPath := common.ApiPath.GetBucketVersioning(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return &BucketVersioningRequest{}
	}

	var versioning BucketVersioningRequest
	if err := json.Unmarshal(resp, &versioning); err != nil {
		return &BucketVersioningRequest{}
	}
	return &versioning
}

func (s *ObjectStorageServiceImpl) PutBucketWebsite(vpcId, s3ServiceId, bucketName string, website BucketWebsiteRequest) CommonResponse {
	apiPath := common.ApiPath.PutBucketWebsite(vpcId, s3ServiceId, bucketName)
	if _, err := s.client.SendPutRequest(apiPath, website); err != nil {
		return CommonResponse{Status: false}
	}
	return CommonResponse{Status: true}
}
func (s *ObjectStorageServiceImpl) DeleteBucketStaticWebsite(vpcId, s3ServiceId, bucketName string) CommonResponse {
	apiPath := common.ApiPath.DeleteBucketStaticWebsite(vpcId, s3ServiceId, bucketName)
	if _, err := s.client.SendDeleteRequest(apiPath); err != nil {
		return CommonResponse{Status: false}
	}
	return CommonResponse{Status: true}
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
	if _, err := s.client.SendDeleteRequest(apiPath); err != nil {
		return fmt.Errorf("failed to delete sub-user: %v", err)
	}
	return nil
}

func (s *ObjectStorageServiceImpl) GetBucketLifecycle(vpcId, s3ServiceId, bucketName, page, pageSize string) (*BucketLifecycleResponse, error) {
	apiPath := common.ApiPath.GetBucketLifecycle(vpcId, s3ServiceId, bucketName, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket lifecycle: %v", err)
	}

	var bucketLifecycle BucketLifecycleResponse
	if err := json.Unmarshal(resp, &bucketLifecycle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket lifecycle: %v", err)
	}
	return &bucketLifecycle, nil
}

func (s *ObjectStorageServiceImpl) PutBucketLifecycle(vpcId, s3ServiceId, bucketName string, lifecycle interface{}) (*BucketLifecycleResponse, error) {
	apiPath := common.ApiPath.PutBucketLifecycle(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPutRequest(apiPath, lifecycle)
	if err != nil {
		return nil, fmt.Errorf("failed to put bucket lifecycle: %v", err)
	}

	var bucketLifecycle BucketLifecycleResponse
	if err := json.Unmarshal(resp, &bucketLifecycle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket lifecycle: %v", err)
	}
	return &bucketLifecycle, nil
}

func (s *ObjectStorageServiceImpl) DeleteBucketLifecycle(vpcId, s3ServiceId, bucketName string, lifecycle interface{}) (*BucketLifecycleResponse, error) {
	apiPath := common.ApiPath.DeleteBucketLifecycle(vpcId, s3ServiceId, bucketName)
	resp, err := s.client.SendPutRequest(apiPath, lifecycle)
	if err != nil {
		return nil, fmt.Errorf("failed to delete bucket lifecycle: %v", err)
	}

	var bucketLifecycle BucketLifecycleResponse
	if err := json.Unmarshal(resp, &bucketLifecycle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket lifecycle: %v", err)
	}
	return &bucketLifecycle, nil
}

func (s *ObjectStorageServiceImpl) CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId string) *SubUserCreateKeyResponse {
	apiPath := common.ApiPath.CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId)
	resp, err := s.client.SendPostRequest(apiPath, nil)
	if err != nil {
		return nil
	}

	var subUserKeys SubUserCreateKeyResponse
	if err := json.Unmarshal(resp, &subUserKeys); err != nil {
		return nil
	}
	return &subUserKeys
}

func (s *ObjectStorageServiceImpl) DeleteSubUserAccessKey(vpcId, s3ServiceId, subUserId, accessKeyId string) CommonResponse {
	apiPath := common.ApiPath.DeleteSubUserAccessKey(vpcId, s3ServiceId, subUserId, accessKeyId)
	if _, err := s.client.SendDeleteRequest(apiPath); err != nil {
		return CommonResponse{Status: false}
	}
	return CommonResponse{Status: true}
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
