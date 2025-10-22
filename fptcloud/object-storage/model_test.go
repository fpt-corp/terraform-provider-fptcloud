package fptcloud_object_storage_test

import (
	"encoding/json"
	"testing"

	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"

	"github.com/stretchr/testify/assert"
)

func TestBucketRequestSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  fptcloud_object_storage.BucketRequest
		expected string
	}{
		{
			name: "bucket with object lock and versioning",
			request: fptcloud_object_storage.BucketRequest{
				Name:       "test-bucket",
				Versioning: "Enabled",
				Acl:        "private",
				ObjectLock: true,
			},
			expected: `{"name":"test-bucket","versioning":"Enabled","acl":"private","object_lock":true}`,
		},
		{
			name: "bucket without optional fields",
			request: fptcloud_object_storage.BucketRequest{
				Name: "test-bucket",
			},
			expected: `{"name":"test-bucket"}`,
		},
		{
			name: "bucket with suspended versioning",
			request: fptcloud_object_storage.BucketRequest{
				Name:       "test-bucket",
				Versioning: "Suspended",
				Acl:        "public-read",
				ObjectLock: false,
			},
			expected: `{"name":"test-bucket","versioning":"Suspended","acl":"public-read"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.request)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))
		})
	}
}

func TestBucketRequestDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected fptcloud_object_storage.BucketRequest
	}{
		{
			name:     "full bucket request",
			jsonData: `{"name":"test-bucket","versioning":"Enabled","acl":"private","object_lock":true}`,
			expected: fptcloud_object_storage.BucketRequest{
				Name:       "test-bucket",
				Versioning: "Enabled",
				Acl:        "private",
				ObjectLock: true,
			},
		},
		{
			name:     "minimal bucket request",
			jsonData: `{"name":"test-bucket"}`,
			expected: fptcloud_object_storage.BucketRequest{
				Name: "test-bucket",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request fptcloud_object_storage.BucketRequest
			err := json.Unmarshal([]byte(tt.jsonData), &request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, request)
		})
	}
}

func TestCorsRuleSerialization(t *testing.T) {
	tests := []struct {
		name     string
		rule     fptcloud_object_storage.CorsRule
		expected string
	}{
		{
			name: "full CORS rule",
			rule: fptcloud_object_storage.CorsRule{
				ID:             "rule1",
				AllowedOrigins: []string{"https://example.com", "https://test.com"},
				AllowedMethods: []string{"GET", "POST"},
				ExposeHeaders:  []string{"Content-Length", "Content-Type"},
				AllowedHeaders: []string{"Authorization", "Content-Type"},
				MaxAgeSeconds:  3600,
			},
			expected: `{"ID":"rule1","AllowedOrigins":["https://example.com","https://test.com"],"AllowedMethods":["GET","POST"],"ExposeHeaders":["Content-Length","Content-Type"],"AllowedHeaders":["Authorization","Content-Type"],"MaxAgeSeconds":3600}`,
		},
		{
			name: "minimal CORS rule",
			rule: fptcloud_object_storage.CorsRule{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET"},
				MaxAgeSeconds:  300,
			},
			expected: `{"AllowedOrigins":["*"],"AllowedMethods":["GET"],"MaxAgeSeconds":300}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.rule)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))
		})
	}
}

func TestCorsRuleDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected fptcloud_object_storage.CorsRule
	}{
		{
			name:     "full CORS rule",
			jsonData: `{"ID":"rule1","AllowedOrigins":["https://example.com"],"AllowedMethods":["GET","POST"],"ExposeHeaders":["Content-Length"],"AllowedHeaders":["Authorization"],"MaxAgeSeconds":3600}`,
			expected: fptcloud_object_storage.CorsRule{
				ID:             "rule1",
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
				ExposeHeaders:  []string{"Content-Length"},
				AllowedHeaders: []string{"Authorization"},
				MaxAgeSeconds:  3600,
			},
		},
		{
			name:     "minimal CORS rule",
			jsonData: `{"AllowedOrigins":["*"],"AllowedMethods":["GET"],"MaxAgeSeconds":300}`,
			expected: fptcloud_object_storage.CorsRule{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET"},
				MaxAgeSeconds:  300,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rule fptcloud_object_storage.CorsRule
			err := json.Unmarshal([]byte(tt.jsonData), &rule)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, rule)
		})
	}
}

func TestS3BucketLifecycleConfigSerialization(t *testing.T) {
	tests := []struct {
		name     string
		config   fptcloud_object_storage.S3BucketLifecycleConfig
		expected string
	}{
		{
			name: "lifecycle config with expiration",
			config: fptcloud_object_storage.S3BucketLifecycleConfig{
				ID: "rule1",
				Filter: fptcloud_object_storage.Filter{
					Prefix: "logs/",
				},
				Expiration: fptcloud_object_storage.Expiration{
					Days: 30,
				},
				NoncurrentVersionExpiration: fptcloud_object_storage.NoncurrentVersionExpiration{
					NoncurrentDays: 90,
				},
				AbortIncompleteMultipartUpload: fptcloud_object_storage.AbortIncompleteMultipartUpload{
					DaysAfterInitiation: 7,
				},
			},
			expected: `{"ID":"rule1","Filter":{"Prefix":"logs/"},"Expiration":{"Days":30},"NoncurrentVersionExpiration":{"NoncurrentDays":90},"AbortIncompleteMultipartUpload":{"DaysAfterInitiation":7}}`,
		},
		{
			name: "lifecycle config with delete marker expiration",
			config: fptcloud_object_storage.S3BucketLifecycleConfig{
				ID: "rule2",
				Filter: fptcloud_object_storage.Filter{
					Prefix: "temp/",
				},
				Expiration: fptcloud_object_storage.Expiration{
					ExpiredObjectDeleteMarker: true,
				},
			},
			expected: `{"ID":"rule2","Filter":{"Prefix":"temp/"},"Expiration":{"ExpiredObjectDeleteMarker":true},"NoncurrentVersionExpiration":{"NoncurrentDays":0},"AbortIncompleteMultipartUpload":{"DaysAfterInitiation":0}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.config)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))
		})
	}
}

func TestS3BucketLifecycleConfigDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected fptcloud_object_storage.S3BucketLifecycleConfig
	}{
		{
			name:     "lifecycle config with expiration",
			jsonData: `{"ID":"rule1","Filter":{"Prefix":"logs/"},"Expiration":{"Days":30},"NoncurrentVersionExpiration":{"NoncurrentDays":90},"AbortIncompleteMultipartUpload":{"DaysAfterInitiation":7}}`,
			expected: fptcloud_object_storage.S3BucketLifecycleConfig{
				ID: "rule1",
				Filter: fptcloud_object_storage.Filter{
					Prefix: "logs/",
				},
				Expiration: fptcloud_object_storage.Expiration{
					Days: 30,
				},
				NoncurrentVersionExpiration: fptcloud_object_storage.NoncurrentVersionExpiration{
					NoncurrentDays: 90,
				},
				AbortIncompleteMultipartUpload: fptcloud_object_storage.AbortIncompleteMultipartUpload{
					DaysAfterInitiation: 7,
				},
			},
		},
		{
			name:     "lifecycle config with delete marker expiration",
			jsonData: `{"ID":"rule2","Filter":{"Prefix":"temp/"},"Expiration":{"ExpiredObjectDeleteMarker":true},"NoncurrentVersionExpiration":{"NoncurrentDays":0},"AbortIncompleteMultipartUpload":{"DaysAfterInitiation":0}}`,
			expected: fptcloud_object_storage.S3BucketLifecycleConfig{
				ID: "rule2",
				Filter: fptcloud_object_storage.Filter{
					Prefix: "temp/",
				},
				Expiration: fptcloud_object_storage.Expiration{
					ExpiredObjectDeleteMarker: true,
				},
				NoncurrentVersionExpiration: fptcloud_object_storage.NoncurrentVersionExpiration{
					NoncurrentDays: 0,
				},
				AbortIncompleteMultipartUpload: fptcloud_object_storage.AbortIncompleteMultipartUpload{
					DaysAfterInitiation: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config fptcloud_object_storage.S3BucketLifecycleConfig
			err := json.Unmarshal([]byte(tt.jsonData), &config)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, config)
		})
	}
}

func TestBucketVersioningRequestSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  fptcloud_object_storage.BucketVersioningRequest
		expected string
	}{
		{
			name: "enabled versioning",
			request: fptcloud_object_storage.BucketVersioningRequest{
				Status: "Enabled",
			},
			expected: `{"status":"Enabled"}`,
		},
		{
			name: "suspended versioning",
			request: fptcloud_object_storage.BucketVersioningRequest{
				Status: "Suspended",
			},
			expected: `{"status":"Suspended"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.request)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))
		})
	}
}

func TestBucketAclRequestSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  fptcloud_object_storage.BucketAclRequest
		expected string
	}{
		{
			name: "private ACL with apply objects",
			request: fptcloud_object_storage.BucketAclRequest{
				CannedAcl:    "private",
				ApplyObjects: true,
			},
			expected: `{"cannedAcl":"private","applyObjects":true}`,
		},
		{
			name: "public-read ACL without apply objects",
			request: fptcloud_object_storage.BucketAclRequest{
				CannedAcl:    "public-read",
				ApplyObjects: false,
			},
			expected: `{"cannedAcl":"public-read","applyObjects":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.request)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))
		})
	}
}

func TestBucketWebsiteRequestSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  fptcloud_object_storage.BucketWebsiteRequest
		expected string
	}{
		{
			name: "website config with index and error documents",
			request: fptcloud_object_storage.BucketWebsiteRequest{
				Key:    "error.html",
				Suffix: "index.html",
				Bucket: "test-bucket",
			},
			expected: `{"key":"error.html","suffix":"index.html","bucket":"test-bucket"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.request)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))
		})
	}
}

func TestBucketPolicyRequestSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  fptcloud_object_storage.BucketPolicyRequest
		expected string
	}{
		{
			name: "bucket policy request",
			request: fptcloud_object_storage.BucketPolicyRequest{
				Policy: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::test-bucket/*"}]}`,
			},
			expected: `{"policy":"{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"s3:GetObject\",\"Resource\":\"arn:aws:s3:::test-bucket/*\"}]}"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.request)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))
		})
	}
}

func TestCreateAccessKeyResponseDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected fptcloud_object_storage.CreateAccessKeyResponse
	}{
		{
			name:     "successful access key creation",
			jsonData: `{"status":true,"message":"Access key created successfully","credential":{"accessKey":"AKIAIOSFODNN7EXAMPLE","secretKey":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY","active":true,"createdDate":"2024-01-01T00:00:00Z"}}`,
			expected: fptcloud_object_storage.CreateAccessKeyResponse{
				Status:  true,
				Message: "Access key created successfully",
				Credential: struct {
					AccessKey   string `json:"accessKey"`
					SecretKey   string `json:"secretKey"`
					Active      bool   `json:"active,omitempty"`
					CreatedDate string `json:"createdDate,omitempty"`
				}{
					AccessKey:   "AKIAIOSFODNN7EXAMPLE",
					SecretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					Active:      true,
					CreatedDate: "2024-01-01T00:00:00Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response fptcloud_object_storage.CreateAccessKeyResponse
			err := json.Unmarshal([]byte(tt.jsonData), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, response)
		})
	}
}

func TestSubUserCreateKeyResponseDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected fptcloud_object_storage.SubUserCreateKeyResponse
	}{
		{
			name:     "successful sub-user key creation",
			jsonData: `{"status":true,"message":"Sub-user key created successfully","credential":{"accessKey":"AKIAIOSFODNN7EXAMPLE","secretKey":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY","active":true,"createdDate":"2024-01-01T00:00:00Z"}}`,
			expected: fptcloud_object_storage.SubUserCreateKeyResponse{
				Status:  true,
				Message: "Sub-user key created successfully",
				Credential: struct {
					AccessKey   string `json:"accessKey,omitempty"`
					SecretKey   string `json:"secretKey,omitempty"`
					Active      bool   `json:"active,omitempty"`
					CreatedDate string `json:"createdDate,omitempty"`
				}{
					AccessKey:   "AKIAIOSFODNN7EXAMPLE",
					SecretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					Active:      true,
					CreatedDate: "2024-01-01T00:00:00Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response fptcloud_object_storage.SubUserCreateKeyResponse
			err := json.Unmarshal([]byte(tt.jsonData), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, response)
		})
	}
}
