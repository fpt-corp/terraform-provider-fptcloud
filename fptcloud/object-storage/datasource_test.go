package fptcloud_object_storage_test

import (
	"testing"

	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"

	"github.com/stretchr/testify/assert"
)

func TestDataSourceBucket(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceBucket()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "page")
	assert.Contains(t, schema, "page_size")
	assert.Contains(t, schema, "list_bucket_result")

	// Test field properties
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["page"].Optional)
	assert.True(t, schema["page_size"].Optional)
	assert.True(t, schema["list_bucket_result"].Computed)
}

func TestDataSourceAccessKey(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceAccessKey()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "credentials")

	// Test field properties
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["credentials"].Computed)
}

func TestDataSourceBucketAcl(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceBucketAcl()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "bucket_name")
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "canned_acl")
	assert.Contains(t, schema, "status")
	assert.Contains(t, schema, "bucket_acl")

	// Test field properties
	assert.True(t, schema["bucket_name"].Required)
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["canned_acl"].Computed)
	assert.True(t, schema["status"].Computed)
	assert.True(t, schema["bucket_acl"].Computed)
}

func TestDataSourceBucketCors(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceBucketCors()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "bucket_name")
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "page_size")
	assert.Contains(t, schema, "page")
	assert.Contains(t, schema, "cors_rule")

	// Test field properties
	assert.True(t, schema["bucket_name"].Required)
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["page_size"].Optional)
	assert.True(t, schema["page"].Optional)
	assert.True(t, schema["cors_rule"].Computed)
}

func TestDataSourceBucketLifecycle(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceBucketLifecycle()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "bucket_name")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "page_size")
	assert.Contains(t, schema, "page")
	assert.Contains(t, schema, "life_cycle_rules")

	// Test field properties
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["bucket_name"].Required)
	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["page_size"].Optional)
	assert.True(t, schema["page"].Optional)
	assert.True(t, schema["life_cycle_rules"].Computed)
}

func TestDataSourceBucketPolicy(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceBucketPolicy()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "bucket_name")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "policy")

	// Test field properties
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["bucket_name"].Required)
	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["policy"].Computed)
}

func TestDataSourceBucketStaticWebsite(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceBucketStaticWebsite()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "bucket_name")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "index_document_suffix")
	assert.Contains(t, schema, "error_document_key")

	// Test field properties
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["bucket_name"].Required)
	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["index_document_suffix"].Optional)
	assert.True(t, schema["error_document_key"].Optional)
}

func TestDataSourceBucketVersioning(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceBucketVersioning()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "bucket_name")
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "versioning_status")
	assert.Contains(t, schema, "region_name")

	// Test field properties
	assert.True(t, schema["bucket_name"].Required)
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["versioning_status"].Optional)
	assert.True(t, schema["region_name"].Required)
}

func TestDataSourceS3ServiceEnableResponse(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceS3ServiceEnableResponse()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "s3_enable_services")
	assert.Contains(t, schema, "vpc_id")

	// Test field properties
	assert.True(t, schema["s3_enable_services"].Computed)
	assert.True(t, schema["vpc_id"].Required)
}

func TestDataSourceSubUser(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceSubUser()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "region_name")
	assert.Contains(t, schema, "page")
	assert.Contains(t, schema, "page_size")
	assert.Contains(t, schema, "list_sub_user")

	// Test field properties
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["region_name"].Required)
	assert.True(t, schema["page"].Optional)
	assert.True(t, schema["page_size"].Optional)
	assert.True(t, schema["list_sub_user"].Computed)
}

func TestDataSourceSubUserDetail(t *testing.T) {
	dataSource := fptcloud_object_storage.DataSourceSubUserDetail()

	assert.NotNil(t, dataSource)
	assert.NotNil(t, dataSource.Schema)
	assert.NotNil(t, dataSource.ReadContext)

	// Test schema fields
	schema := dataSource.Schema
	assert.Contains(t, schema, "vpc_id")
	assert.Contains(t, schema, "user_id")
	assert.Contains(t, schema, "arn")
	assert.Contains(t, schema, "active")
	assert.Contains(t, schema, "role")
	assert.Contains(t, schema, "created_at")
	assert.Contains(t, schema, "access_keys")

	// Test field properties
	assert.True(t, schema["vpc_id"].Required)
	assert.True(t, schema["user_id"].Computed)
	assert.True(t, schema["arn"].Computed)
	assert.True(t, schema["active"].Computed)
	assert.True(t, schema["role"].Computed)
	assert.True(t, schema["created_at"].Computed)
	assert.True(t, schema["access_keys"].Computed)
}
