package fptcloud_object_storage

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeService serves canned read-back responses. The embedded interface is nil,
// so a reconcile helper that reaches for anything beyond the read it is supposed
// to use panics rather than passing quietly.
type fakeService struct {
	ObjectStorageService

	lifecycle  BucketLifecycleResponse
	cors       *BucketCorsResponse
	corsErr    error
	buckets    ListBucketResponse
	policy     *BucketPolicyResponse
	acl        *BucketAclResponse
	versioning *BucketVersioningResponse
	website    *BucketWebsiteResponse
	subUser    *DetailSubUser
}

func (f fakeService) GetBucketLifecycle(_, _, _ string, _, _ int) BucketLifecycleResponse {
	return f.lifecycle
}
func (f fakeService) GetBucketCors(_, _, _ string, _, _ int) (*BucketCorsResponse, error) {
	return f.cors, f.corsErr
}
func (f fakeService) ListBuckets(_, _ string, _, _ int) ListBucketResponse { return f.buckets }
func (f fakeService) GetBucketPolicy(_, _, _ string) *BucketPolicyResponse { return f.policy }
func (f fakeService) GetBucketAcl(_, _, _ string) *BucketAclResponse       { return f.acl }
func (f fakeService) GetBucketVersioning(_, _, _ string) *BucketVersioningResponse {
	return f.versioning
}
func (f fakeService) GetBucketWebsite(_, _, _ string) *BucketWebsiteResponse { return f.website }
func (f fakeService) DetailSubUser(_, _, _ string) *DetailSubUser            { return f.subUser }

// The response models nest anonymous structs, so canned data is built from JSON
// rather than from struct literals.
func decode[T any](t *testing.T, raw string) T {
	t.Helper()
	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("bad test fixture: %v", err)
	}
	return out
}

func rule(id string) S3BucketLifecycleConfig {
	return S3BucketLifecycleConfig{ID: id}
}

func TestReconcileLifeCycleRule(t *testing.T) {
	const stored = `{"status":true,"total":1,"rules":[{
		"ID":"expire-tmp","Status":"Disabled",
		"Filter":{"Prefix":"tmp/"},"Expiration":{"Days":30},
		"NoncurrentVersionExpiration":{"NoncurrentDays":0},
		"AbortIncompleteMultipartUpload":{"DaysAfterInitiation":0}}]}`

	want := func(mutate func(*S3BucketLifecycleConfig)) S3BucketLifecycleConfig {
		c := rule("expire-tmp")
		c.Status = "Disabled"
		c.Filter = &Filter{Prefix: "tmp/"}
		c.Expiration = &Expiration{Days: 30}
		if mutate != nil {
			mutate(&c)
		}
		return c
	}

	tests := []struct {
		name     string
		response string
		want     S3BucketLifecycleConfig
		expected createOutcome
	}{
		{
			// The 504 case: the rule committed, only the response was lost.
			name: "adopts the rule it asked for", response: stored,
			want: want(nil), expected: createAdopted,
		},
		{
			// Fields the config omits are the backend's business, not a conflict.
			name: "adopts when config omits fields the backend defaulted", response: stored,
			want: rule("expire-tmp"), expected: createAdopted,
		},
		{
			name: "conflict when status differs", response: stored,
			want: want(func(c *S3BucketLifecycleConfig) { c.Status = "Enabled" }), expected: createConflict,
		},
		{
			name: "conflict when expiration differs", response: stored,
			want: want(func(c *S3BucketLifecycleConfig) { c.Expiration = &Expiration{Days: 60} }), expected: createConflict,
		},
		{
			name: "conflict when prefix differs", response: stored,
			want: want(func(c *S3BucketLifecycleConfig) { c.Filter = &Filter{Prefix: "other/"} }), expected: createConflict,
		},
		{
			name: "conflict when a declared zero differs from stored", response: stored,
			want: want(func(c *S3BucketLifecycleConfig) {
				c.NoncurrentVersionExpiration = &NoncurrentVersionExpiration{NoncurrentDays: 30}
			}), expected: createConflict,
		},
		{
			name: "adopts when a declared zero matches stored", response: stored,
			want: want(func(c *S3BucketLifecycleConfig) {
				c.NoncurrentVersionExpiration = &NoncurrentVersionExpiration{NoncurrentDays: 0}
			}), expected: createAdopted,
		},
		{
			name: "failed when the rule is absent", response: `{"status":true,"total":0,"rules":[]}`,
			want: want(nil), expected: createFailed,
		},
		{
			name: "failed when another rule holds the bucket", response: stored,
			want: rule("something-else"), expected: createFailed,
		},
		{
			name: "failed when the read itself fails", response: `{"status":false}`,
			want: want(nil), expected: createFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := fakeService{lifecycle: decode[BucketLifecycleResponse](t, tt.response)}
			assert.Equal(t, tt.expected, reconcileLifeCycleRule(svc, "vpc", "s3", "bucket", tt.want))
		})
	}
}

func TestReconcileCorsRule(t *testing.T) {
	const stored = `{"status":true,"total":1,"cors_rules":[{
		"ID":"app-cors","AllowedMethods":["GET","PUT"],
		"AllowedOrigins":["https://app.example.com"],"MaxAgeSeconds":3600}]}`

	base := CorsRule{
		ID:             "app-cors",
		AllowedMethods: []string{"GET", "PUT"},
		AllowedOrigins: []string{"https://app.example.com"},
		MaxAgeSeconds:  3600,
	}

	t.Run("adopts the rule it asked for", func(t *testing.T) {
		svc := fakeService{cors: ptr(decode[BucketCorsResponse](t, stored))}
		assert.Equal(t, createAdopted, reconcileCorsRule(svc, "vpc", "s3", "bucket", base))
	})

	t.Run("conflict when max age differs", func(t *testing.T) {
		other := base
		other.MaxAgeSeconds = 600
		svc := fakeService{cors: ptr(decode[BucketCorsResponse](t, stored))}
		assert.Equal(t, createConflict, reconcileCorsRule(svc, "vpc", "s3", "bucket", other))
	})

	t.Run("conflict when methods differ", func(t *testing.T) {
		other := base
		other.AllowedMethods = []string{"GET"}
		svc := fakeService{cors: ptr(decode[BucketCorsResponse](t, stored))}
		assert.Equal(t, createConflict, reconcileCorsRule(svc, "vpc", "s3", "bucket", other))
	})

	t.Run("adopts when undeclared optional headers are absent", func(t *testing.T) {
		other := base
		other.AllowedHeaders = nil
		svc := fakeService{cors: ptr(decode[BucketCorsResponse](t, stored))}
		assert.Equal(t, createAdopted, reconcileCorsRule(svc, "vpc", "s3", "bucket", other))
	})

	t.Run("failed when the read errors", func(t *testing.T) {
		svc := fakeService{corsErr: errors.New("boom")}
		assert.Equal(t, createFailed, reconcileCorsRule(svc, "vpc", "s3", "bucket", base))
	})

	t.Run("failed when the rule is absent", func(t *testing.T) {
		svc := fakeService{cors: ptr(decode[BucketCorsResponse](t, `{"status":true,"total":0,"cors_rules":[]}`))}
		assert.Equal(t, createFailed, reconcileCorsRule(svc, "vpc", "s3", "bucket", base))
	})
}

func TestReconcileBucket(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected createOutcome
	}{
		{
			// Empty means it is plausibly the orphan we just created.
			name:     "adopts an empty bucket",
			response: `{"total":1,"buckets":[{"Name":"b1","isEmpty":true}]}`,
			expected: createAdopted,
		},
		{
			// Data in it means it is somebody else's; destroy would delete it.
			name:     "refuses a bucket that holds objects",
			response: `{"total":1,"buckets":[{"Name":"b1","isEmpty":false}]}`,
			expected: createConflict,
		},
		{
			name:     "failed when the bucket is absent",
			response: `{"total":1,"buckets":[{"Name":"other","isEmpty":true}]}`,
			expected: createFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := fakeService{buckets: decode[ListBucketResponse](t, tt.response)}
			assert.Equal(t, tt.expected, reconcileBucket(svc, "vpc", "s3", "b1"))
		})
	}
}

func TestReconcileBucketPolicy(t *testing.T) {
	const want = `{"Version":"2012-10-17","Statement":[{"Effect":"Deny"}]}`

	t.Run("adopts a policy that differs only in formatting", func(t *testing.T) {
		svc := fakeService{policy: &BucketPolicyResponse{
			Status: true,
			Policy: "{\n  \"Statement\": [ { \"Effect\": \"Deny\" } ],\n  \"Version\": \"2012-10-17\"\n}",
		}}
		assert.Equal(t, createAdopted, reconcileBucketPolicy(svc, "vpc", "s3", "bucket", want))
	})

	t.Run("conflict when the document differs", func(t *testing.T) {
		svc := fakeService{policy: &BucketPolicyResponse{
			Status: true,
			Policy: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow"}]}`,
		}}
		assert.Equal(t, createConflict, reconcileBucketPolicy(svc, "vpc", "s3", "bucket", want))
	})

	t.Run("failed when no policy is set", func(t *testing.T) {
		svc := fakeService{policy: &BucketPolicyResponse{Status: true, Policy: ""}}
		assert.Equal(t, createFailed, reconcileBucketPolicy(svc, "vpc", "s3", "bucket", want))
	})
}

func TestReconcileSubUser(t *testing.T) {
	t.Run("adopts a user with the requested role", func(t *testing.T) {
		svc := fakeService{subUser: &DetailSubUser{UserID: "u1", Role: "SubUserReadWrite"}}
		assert.Equal(t, createAdopted, reconcileSubUser(svc, "vpc", "s3", "u1", "SubUserReadWrite"))
	})

	t.Run("conflict when the role differs", func(t *testing.T) {
		svc := fakeService{subUser: &DetailSubUser{UserID: "u1", Role: "SubUserRead"}}
		assert.Equal(t, createConflict, reconcileSubUser(svc, "vpc", "s3", "u1", "SubUserReadWrite"))
	})

	t.Run("failed when the user is absent", func(t *testing.T) {
		svc := fakeService{subUser: nil}
		assert.Equal(t, createFailed, reconcileSubUser(svc, "vpc", "s3", "u1", "SubUserReadWrite"))
	})
}

func TestReconcileOverwriteSettings(t *testing.T) {
	t.Run("acl adopts when already applied", func(t *testing.T) {
		svc := fakeService{acl: &BucketAclResponse{Status: true, CannedACL: "private"}}
		assert.Equal(t, createAdopted, reconcileBucketAcl(svc, "vpc", "s3", "bucket", "private"))
		assert.Equal(t, createFailed, reconcileBucketAcl(svc, "vpc", "s3", "bucket", "public-read"))
	})

	t.Run("versioning adopts when already applied", func(t *testing.T) {
		svc := fakeService{versioning: &BucketVersioningResponse{Status: true, Config: "Enabled"}}
		assert.Equal(t, createAdopted, reconcileBucketVersioning(svc, "vpc", "s3", "bucket", "Enabled"))
		assert.Equal(t, createFailed, reconcileBucketVersioning(svc, "vpc", "s3", "bucket", "Suspended"))
	})

	t.Run("website adopts when a configuration is present", func(t *testing.T) {
		assert.Equal(t, createAdopted,
			reconcileBucketWebsite(fakeService{website: &BucketWebsiteResponse{Status: true}}, "vpc", "s3", "bucket"))
		assert.Equal(t, createFailed,
			reconcileBucketWebsite(fakeService{website: nil}, "vpc", "s3", "bucket"))
	})
}

func TestEqualHelpers(t *testing.T) {
	assert.True(t, equalStrings(nil, nil))
	assert.True(t, equalStrings([]string{}, nil))
	assert.True(t, equalStrings([]string{"a", "b"}, []string{"a", "b"}))
	assert.False(t, equalStrings([]string{"a", "b"}, []string{"b", "a"}))

	assert.True(t, equalJSON(`{"a":1,"b":2}`, `{"b":2,"a":1}`))
	assert.False(t, equalJSON(`{"a":1}`, `{"a":2}`))
	assert.False(t, equalJSON(`not json`, `{"a":1}`))
}

func ptr[T any](v T) *T { return &v }
