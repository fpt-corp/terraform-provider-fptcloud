package fptcloud_object_storage_test

import (
	"testing"

	common "terraform-provider-fptcloud/commons"
	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"

	"github.com/stretchr/testify/assert"
)

func TestResourceBucket(t *testing.T) {
	resource := fptcloud_object_storage.ResourceBucket()

	assert.NotNil(t, resource)
	assert.NotNil(t, resource.Schema)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.ReadContext)

	// Test schema fields
	schema := resource.Schema
	assert.Contains(t, schema, "name")
	assert.Contains(t, schema, "versioning")
	assert.Contains(t, schema, "object_lock_enabled")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "acl")
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "status")

	// Test field properties
	assert.True(t, schema["name"].Required)
	assert.True(t, schema["name"].ForceNew)
	assert.NotNil(t, schema["name"].ValidateFunc)

	assert.True(t, schema["versioning"].Optional)
	assert.True(t, schema["versioning"].ForceNew)
	assert.NotNil(t, schema["versioning"].ValidateFunc)

	assert.True(t, schema["object_lock_enabled"].Optional)
	assert.True(t, schema["object_lock_enabled"].ForceNew)

	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["region_name"].ForceNew)
	assert.NotNil(t, schema["region_name"].ValidateFunc)

	assert.True(t, schema["acl"].Optional)
	assert.True(t, schema["acl"].ForceNew)
	assert.NotNil(t, schema["acl"].ValidateFunc)

	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["vpc_id"].ForceNew)

	assert.True(t, schema["status"].Computed)
	assert.True(t, schema["status"].ForceNew)
}

func TestGetServiceEnableRegion(t *testing.T) {
	// Create a mock service
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/check-service-enabled?check_unlimited=undefined": `{
			"data": [
				{
					"s3_service_name": "HCM-01",
					"s3_service_id": "s3_service_id",
					"s3_platform": "ceph"
				}
			],
			"total": 1
		}`,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)

	// Test successful case
	result := fptcloud_object_storage.GetServiceEnableRegion(service, "vpc_id", "HCM-01")
	assert.Equal(t, "HCM-01", result.S3ServiceName)
	assert.Equal(t, "s3_service_id", result.S3ServiceId)
	assert.Equal(t, "ceph", result.S3Platform)

	// Test region not found
	result = fptcloud_object_storage.GetServiceEnableRegion(service, "vpc_id", "INVALID")
	assert.Empty(t, result.S3ServiceName)
	assert.Empty(t, result.S3ServiceId)
	assert.Empty(t, result.S3Platform)
}

func TestResourceBucketCreate(t *testing.T) {
	// This is a more complex test that would require mocking the entire Terraform context
	// For now, we'll test the basic structure

	resource := fptcloud_object_storage.ResourceBucket()
	assert.NotNil(t, resource.CreateContext)

	// Test that the function exists and can be called
	// Note: Full integration testing would require setting up Terraform test framework
}

func TestResourceBucketDelete(t *testing.T) {
	resource := fptcloud_object_storage.ResourceBucket()
	assert.NotNil(t, resource.DeleteContext)

	// Test that the function exists and can be called
	// Note: Full integration testing would require setting up Terraform test framework
}
