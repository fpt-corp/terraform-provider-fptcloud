package fptcloud_storage_policy_test

import (
	"terraform-provider-fptcloud/fptcloud/storage-policy"
	"testing"

	"github.com/stretchr/testify/assert"
	common "terraform-provider-fptcloud/commons"
)

func TestListStoragePolicy_ReturnsStoragePolicies(t *testing.T) {
	mockResponse := `{
		"data": [{
			"id": "11111111-aaaa-1111-bbbb-111111111111",
			"name": "PREMIUM-SSD"
		}]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/storage-policies": mockResponse,
	})
	defer server.Close()
	service := fptcloud_storage_policy.NewStoragePolicyService(mockClient)
	storagePolicies, err := service.ListStoragePolicy("vpc_id")
	assert.NoError(t, err)
	assert.NotNil(t, storagePolicies)
	assert.Equal(t, 1, len(*storagePolicies))
	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", (*storagePolicies)[0].ID)
	assert.Equal(t, "PREMIUM-SSD", (*storagePolicies)[0].Name)
}

func TestListStoragePolicy_ReturnsEmptyWhenNotFound(t *testing.T) {
	mockResponse := `{
		"data": []
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/storage-policies": mockResponse,
	})
	defer server.Close()
	service := fptcloud_storage_policy.NewStoragePolicyService(mockClient)
	storagePolicies, err := service.ListStoragePolicy("vpc_id")
	assert.NoError(t, err)
	assert.NotNil(t, storagePolicies)
	assert.Equal(t, 0, len(*storagePolicies))
}
