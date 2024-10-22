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
type CorsRule struct {
	ID             string   `json:"ID"`
	AllowedOrigins []string `json:"AllowedOrigins"`
	AllowedMethods []string `json:"AllowedMethods"`
	MaxAgeSeconds  int      `json:"MaxAgeSeconds,omitempty"`
	ExposeHeaders  []string `json:"ExposeHeaders,omitempty"`
	AllowedHeaders []string `json:"AllowedHeaders,omitempty"`
}
type BucketCors struct {
	Status    bool       `json:"status"`
	CorsRules []CorsRule `json:"cors_rules"`
	Total     int        `json:"total"`
}

// ObjectStorageService defines the interface for object storage operations
type ObjectStorageService interface {
	CreateBucket(req BucketCreateRequest) (*Bucket, error)
	CreateSubUser(req SubUser) (*SubUser, error)
	CreateAccessKey() (*CreateAccessKeyResponse, error)
	ListBuckets() ([]Bucket, error)
	ListSubUsers() ([]SubUser, error)
	ListAccessKeys() ([]AccessKey, error)
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
