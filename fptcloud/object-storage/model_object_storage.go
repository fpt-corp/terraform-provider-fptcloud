package fptcloud_object_storage

type AbortIncompleteMultipartUpload struct {
	DaysAfterInitiation int `json:"DaysAfterInitiation"`
}

type AccessKey struct {
	Credentials []struct {
		ID          string `json:"id"`
		Credentials []struct {
			AccessKey   string      `json:"accessKey"`
			Active      bool        `json:"active"`
			CreatedDate interface{} `json:"createdDate,omitempty"`
		} `json:"credentials"`
	} `json:"credentials"`
}

type BucketAclRequest struct {
	CannedAcl    string `json:"cannedAcl"`
	ApplyObjects bool   `json:"applyObjects"`
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

type BucketCors struct {
	CorsRules []CorsRule `json:"CORSRules"`
}

type BucketCorsResponse struct {
	Status    bool `json:"status"`
	CorsRules []struct {
		ID             string   `json:"ID"`
		AllowedHeaders []string `json:"AllowedHeaders,omitempty"`
		AllowedMethods []string `json:"AllowedMethods"`
		AllowedOrigins []string `json:"AllowedOrigins"`
		ExposeHeaders  []string `json:"ExposeHeaders,omitempty"`
		MaxAgeSeconds  int      `json:"MaxAgeSeconds"`
	} `json:"cors_rules"`
	Total int `json:"total"`
}

type BucketLifecycleResponse struct {
	Status bool `json:"status"`
	Rules  []struct {
		Expiration struct {
			ExpiredObjectDeleteMarker bool `json:"ExpiredObjectDeleteMarker,omitempty"`
			Days                      int  `json:"Days,omitempty"`
		} `json:"Expiration"`
		ID     string `json:"ID"`
		Filter struct {
			Prefix string `json:"Prefix"`
		} `json:"Filter,omitempty"`
		Status                      string `json:"Status"`
		NoncurrentVersionExpiration struct {
			NoncurrentDays int `json:"NoncurrentDays"`
		} `json:"NoncurrentVersionExpiration"`
		AbortIncompleteMultipartUpload struct {
			DaysAfterInitiation int `json:"DaysAfterInitiation"`
		} `json:"AbortIncompleteMultipartUpload"`
		Prefix string `json:"Prefix,omitempty"`
	} `json:"rules"`
	Total int `json:"total"`
}

type BucketPolicyRequest struct {
	Policy string `json:"policy"`
}

type BucketPolicyResponse struct {
	Status bool   `json:"status"`
	Policy string `json:"policy"`
}

type BucketRequest struct {
	Name       string `json:"name"`
	Versioning string `json:"versioning,omitempty"`
	Acl        string `json:"acl"`
	ObjectLock bool   `json:"object_lock"`
}

type BucketVersioningRequest struct {
	Status string `json:"status"` // "Enabled" or "Suspended"
}

type BucketVersioningResponse struct {
	Status bool   `json:"status"`
	Config string `json:"config"` // "Enabled" or "Suspended"
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

type CommonResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message,omitempty"`
}

type CorsRule struct {
	ID             string   `json:"ID,omitempty"`
	AllowedOrigins []string `json:"AllowedOrigins"`
	AllowedMethods []string `json:"AllowedMethods"`
	ExposeHeaders  []string `json:"ExposeHeaders,omitempty"`
	AllowedHeaders []string `json:"AllowedHeaders,omitempty"`
	MaxAgeSeconds  int      `json:"MaxAgeSeconds"`
}

type CreateAccessKeyResponse struct {
	Status     bool   `json:"status"`
	Message    string `json:"message,omitempty"`
	Credential struct {
		AccessKey   string      `json:"accessKey"`
		SecretKey   string      `json:"secretKey"`
		Active      interface{} `json:"active"`
		CreatedDate interface{} `json:"createdDate,omitempty"`
	} `json:"credential,omitempty"`
}

type DetailSubUser struct {
	UserID     string      `json:"user_id"`
	Arn        interface{} `json:"arn,omitempty"`
	Active     bool        `json:"active"`
	Role       string      `json:"role"`
	CreatedAt  interface{} `json:"created_at,omitempty"`
	AccessKeys []string    `json:"access_keys"`
}

type Expiration struct {
	Days                      int  `json:"Days,omitempty"`
	ExpiredObjectDeleteMarker bool `json:"ExpiredObjectDeleteMarker,omitempty"`
}

type Filter struct {
	Prefix string `json:"Prefix"`
}

type ListBucketResponse struct {
	Buckets []struct {
		Name             string `json:"Name"`
		CreationDate     string `json:"CreationDate"`
		IsEmpty          bool   `json:"isEmpty"`
		S3ServiceID      string `json:"s3_service_id"`
		IsEnabledLogging bool   `json:"isEnabledLogging"`
		Endpoint         string `json:"endpoint"`
	} `json:"buckets"`
	Total int `json:"total"`
}

type NoncurrentVersionExpiration struct {
	NoncurrentDays int `json:"NoncurrentDays"`
}

type PutBucketAclResponse struct {
	Status bool   `json:"status"`
	TaskID string `json:"taskId"`
}

type S3BucketLifecycleConfig struct {
	ID                             string                         `json:"ID"`
	Filter                         Filter                         `json:"Filter"`
	Expiration                     Expiration                     `json:"Expiration"`
	NoncurrentVersionExpiration    NoncurrentVersionExpiration    `json:"NoncurrentVersionExpiration"`
	AbortIncompleteMultipartUpload AbortIncompleteMultipartUpload `json:"AbortIncompleteMultipartUpload"`
}

type S3ServiceEnableResponse struct {
	Data []struct {
		S3ServiceName      string      `json:"s3_service_name"`
		S3ServiceID        string      `json:"s3_service_id"`
		S3Platform         string      `json:"s3_platform"`
		DefaultUser        interface{} `json:"default_user,omitempty"`
		MigrateQuota       int         `json:"migrate_quota"`
		SyncQuota          int         `json:"sync_quota"`
		RgwTotalNodes      int         `json:"rgw_total_nodes,omitempty"`
		RgwUserActiveNodes int         `json:"rgw_user_active_nodes,omitempty"`
		HasUnusualConfig   interface{} `json:"has_unusual_config,omitempty"`
	} `json:"data"`
	Total int `json:"total"`
}

type Statement struct {
	Sid       string                 `json:"Sid"`
	Effect    string                 `json:"Effect"`
	Principal map[string]interface{} `json:"Principal"`
	Action    []string               `json:"Action"`
	Resource  []string               `json:"Resource"`
}

type SubUser struct {
	Role   string `json:"role"`
	UserId string `json:"user_id"`
}

type SubUserCreateKeyResponse struct {
	Status     bool   `json:"status"`
	Message    string `json:"message,omitempty"`
	Credential struct {
		AccessKey   string      `json:"accessKey,omitempty"`
		SecretKey   string      `json:"secretKey,omitempty"`
		Active      interface{} `json:"active,omitempty"`
		CreatedDate interface{} `json:"createdDate,omitempty"`
	} `json:"credential,omitempty"`
}

type SubUserCreateRequest struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions"`
}

type SubUserListResponse struct {
	SubUsers []struct {
		UserID     string      `json:"user_id"`
		Arn        string      `json:"arn"`
		Active     bool        `json:"active"`
		Role       string      `json:"role"`
		CreatedAt  interface{} `json:"created_at,omitempty"`
		AccessKeys interface{} `json:"access_keys,omitempty"`
	} `json:"sub_users"`
	Total int `json:"total"`
}
