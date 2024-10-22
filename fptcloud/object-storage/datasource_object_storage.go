package fptcloud_object_storage

import (
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"
)

// BucketCreateRequest represents the request body for creating a bucket
type BucketCreateRequest struct {
	Name         string            `json:"name"`
	Region       string            `json:"region"`
	StorageClass string            `json:"storage_class"`
	Versioning   bool              `json:"versioning"`
	Tags         map[string]string `json:"tags,omitempty"`
}

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
	Message    string `json:"message"`
	Credential struct {
		AccessKey   string      `json:"accessKey"`
		SecretKey   string      `json:"secretKey"`
		Active      interface{} `json:"active"`
		CreatedDate interface{} `json:"createdDate"`
	} `json:"credential"`
}
type SubUser struct {
	Role   string `json:"role"`
	UserId string `json:"user_id,omitempty"`
}
type CommonResponse struct {
	Status bool `json:"status"`
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

// ObjectStorageService defines the interface for object storage operations
type ObjectStorageService interface {
	CreateBucket(req BucketCreateRequest) (*Bucket, error)
	CreateSubUser(req SubUser) (*SubUser, error)
	CreateAccessKey() (*CreateAccessKeyResponse, error)
	DeleteBucket(bucketName string) error
	DeleteAccessKey(username, accessKey string) error
	ListBuckets() ([]Bucket, error)
	ListSubUsers() ([]SubUser, error)
	ListAccessKeys() ([]AccessKey, error)

	// bucket configuration
	PutBucketPolicy(bucketName string, policy BucketPolicyRequest) CommonResponse
	GetBucketPolicy(bucketName string) *BucketPolicyResponse

	// CORS configuration
	PutBucketCors(bucketName string, cors CorsRule) (CommonResponse, error)
	UpdateBucketCors(bucketName string, cors BucketCors) (CommonResponse, error)
	GetBucketCors(bucketName string) (*BucketCors, error)

	// Versioning configuration
	PutBucketVersioning(bucketName string, versioning BucketVersioningRequest) error
	GetBucketVersioning(bucketName string) (*BucketVersioningRequest, error)

	// Acl configuration
	PutBucketAcl(bucketName string, acl BucketAclRequest) PutBucketAclResponse
	GetBucketAcl(bucketName string) (*BucketAclResponse, error)

	// Static website configuration
	PutBucketWebsite(bucketName string, website BucketWebsiteRequest) CommonResponse
	GetBucketWebsite(bucketName string) *BucketWebsiteResponse
	DeleteBucketWebsite(bucketName string) CommonResponse
}

// ObjectStorageServiceImpl is the implementation of ObjectStorageService
type ObjectStorageServiceImpl struct {
	client *common.Client
}

// NewObjectStorageService creates a new instance of ObjectStorageService
func NewObjectStorageService(client *common.Client) ObjectStorageService {
	return &ObjectStorageServiceImpl{client: client}
}

// Bucket represents the response structure for a created bucket
type Bucket struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Region       string            `json:"region"`
	StorageClass string            `json:"storage_class"`
	Versioning   bool              `json:"versioning"`
	Tags         map[string]string `json:"tags,omitempty"`
	CreationDate string            `json:"creation_date"`
}

// CreateBucket creates a new bucket
func (s *ObjectStorageServiceImpl) CreateBucket(req BucketCreateRequest) (*Bucket, error) {
	apiPath := "/v1/object-storage/buckets"
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %v", err)
	}

	var bucket Bucket
	err = json.Unmarshal(resp, &bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket response: %v", err)
	}

	return &bucket, nil
}

// CreateSubUser creates a new sub-user
func (s *ObjectStorageServiceImpl) CreateSubUser(req SubUser) (*SubUser, error) {
	apiPath := "/v1/object-storage/sub-users"
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create sub-user: %v", err)
	}

	var subUser SubUser
	err = json.Unmarshal(resp, &subUser)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sub-user response: %v", err)
	}

	return &subUser, nil
}

func (s *ObjectStorageServiceImpl) CreateAccessKey() (*CreateAccessKeyResponse, error) {
	apiPath := "/v1/object-storage/access-keys"
	resp, err := s.client.SendPostRequest(apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create access key: %v", err)
	}

	var accessKey CreateAccessKeyResponse
	err = json.Unmarshal(resp, &accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal access key response: %v", err)
	}

	return &accessKey, nil
}

func (s *ObjectStorageServiceImpl) ListBuckets() ([]Bucket, error) {
	apiPath := "/v1/object-storage/buckets"
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %v", err)
	}

	var buckets []Bucket
	err = json.Unmarshal(resp, &buckets)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket list response: %v", err)
	}

	return buckets, nil
}

func (s *ObjectStorageServiceImpl) ListSubUsers() ([]SubUser, error) {
	apiPath := "/v1/object-storage/sub-users"
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-users: %v", err)
	}

	var subUsers []SubUser
	err = json.Unmarshal(resp, &subUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sub-user list response: %v", err)
	}

	return subUsers, nil
}

func (s *ObjectStorageServiceImpl) ListAccessKeys() ([]AccessKey, error) {
	apiPath := "/v1/object-storage/access-keys"
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list access keys: %v", err)
	}

	var accessKeys []AccessKey
	err = json.Unmarshal(resp, &accessKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal access key list response: %v", err)
	}

	return accessKeys, nil
}

func (s *ObjectStorageServiceImpl) DeleteBucket(bucketName string) error {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s", bucketName)
	if _, err := s.client.SendDeleteRequest(apiPath); err != nil {
		return fmt.Errorf("failed to delete bucket: %v", err)
	}
	return nil
}

func (s *ObjectStorageServiceImpl) DeleteAccessKey(username, accessKey string) error {
	apiPath := fmt.Sprintf("/v1/object-storage/users/%s/access-keys/%s", username, accessKey)
	if _, err := s.client.SendDeleteRequest(apiPath); err != nil {
		return fmt.Errorf("failed to delete access key: %v", err)
	}
	return nil
}

// Implement bucket policy methods
func (s *ObjectStorageServiceImpl) PutBucketPolicy(bucketName string, policy BucketPolicyRequest) CommonResponse {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/policy", bucketName)
	if _, err := s.client.SendPutRequest(apiPath, policy); err != nil {
		return CommonResponse{Status: false}
	}
	return CommonResponse{Status: true}
}

func (s *ObjectStorageServiceImpl) GetBucketPolicy(bucketName string) *BucketPolicyResponse {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/policy", bucketName)
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
func (s *ObjectStorageServiceImpl) PutBucketCors(bucketName string, cors CorsRule) (CommonResponse, error) {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/cors", bucketName)
	if _, err := s.client.SendPutRequest(apiPath, cors); err != nil {
		return CommonResponse{Status: false}, fmt.Errorf("failed to update bucket CORS: %v", err)
	}
	return CommonResponse{Status: true}, nil
}

func (s *ObjectStorageServiceImpl) UpdateBucketCors(bucketName string, cors BucketCors) (CommonResponse, error) {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/cors", bucketName)
	if _, err := s.client.SendPutRequest(apiPath, cors); err != nil {
		return CommonResponse{Status: false}, fmt.Errorf("failed to update bucket CORS: %v", err)
	}
	return CommonResponse{Status: true}, nil
}

func (s *ObjectStorageServiceImpl) GetBucketCors(bucketName string) (*BucketCors, error) {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/cors", bucketName)
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
func (s *ObjectStorageServiceImpl) PutBucketVersioning(bucketName string, versioning BucketVersioningRequest) error {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/versioning", bucketName)
	if _, err := s.client.SendPutRequest(apiPath, versioning); err != nil {
		return fmt.Errorf("failed to put bucket versioning: %v", err)
	}
	return nil
}

func (s *ObjectStorageServiceImpl) GetBucketVersioning(bucketName string) (*BucketVersioningRequest, error) {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/versioning", bucketName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket versioning: %v", err)
	}

	var versioning BucketVersioningRequest
	if err := json.Unmarshal(resp, &versioning); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket versioning: %v", err)
	}
	return &versioning, nil
}

func (s *ObjectStorageServiceImpl) PutBucketWebsite(bucketName string, website BucketWebsiteRequest) CommonResponse {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/website", bucketName)
	if _, err := s.client.SendPutRequest(apiPath, website); err != nil {
		return CommonResponse{Status: false}
	}
	return CommonResponse{Status: true}
}

func (s *ObjectStorageServiceImpl) DeleteBucketWebsite(bucketName string) CommonResponse {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/website", bucketName)
	if _, err := s.client.SendDeleteRequest(apiPath); err != nil {
		return CommonResponse{Status: false}
	}
	return CommonResponse{Status: true}
}

func (s *ObjectStorageServiceImpl) GetBucketWebsite(bucketName string) *BucketWebsiteResponse {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/website", bucketName)
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

func (s *ObjectStorageServiceImpl) PutBucketAcl(bucketName string, acl BucketAclRequest) PutBucketAclResponse {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/acl", bucketName)
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

func (s *ObjectStorageServiceImpl) GetBucketAcl(bucketName string) (*BucketAclResponse, error) {
	apiPath := fmt.Sprintf("/v1/object-storage/buckets/%s/acl", bucketName)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket ACL: %v", err)
	}

	var acl BucketAclResponse
	if err := json.Unmarshal(resp, &acl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bucket ACL: %v", err)
	}
	return &acl, nil
}
