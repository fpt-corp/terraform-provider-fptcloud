package fptcloud_storage_test

import (
	"terraform-provider-fptcloud/fptcloud/storage"
	"testing"

	"github.com/stretchr/testify/assert"
	common "terraform-provider-fptcloud/commons"
)

func TestFindStorage_ReturnsStorage(t *testing.T) {
	mockResponse := `{
			"id": "1",
			"name": "storage-name",
			"type": "LOCAL",
			"size_gb": 100,
			"storage_policy": "storage_policy",
			"storage_policy_id": "storage_policy_id",
			"instance_id": "instance_id",
			"status": "active",
			"vpc_id": "vpc-123",
			"created_at": "2023-10-01T00:00:00Z"
		}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage": mockResponse,
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)
	searchModel := fptcloud_storage.FindStorageDTO{VpcId: "vpc_id", Name: "storage-name"}
	storage, err := service.FindStorage(searchModel)
	assert.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, "1", storage.ID)
	assert.Equal(t, "storage-name", storage.Name)
}

func TestFindStorage_ReturnsErrorOnRequestFailure(t *testing.T) {
	mockResponse := `invalid`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage": mockResponse,
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)
	searchModel := fptcloud_storage.FindStorageDTO{VpcId: "vpc_id", Name: "storage-name"}
	storage, err := service.FindStorage(searchModel)
	assert.Error(t, err)
	assert.Nil(t, storage)
}

func TestCreateStorage_ReturnsStorageId(t *testing.T) {
	mockResponse := `{"storage_id": "1"}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage": mockResponse,
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)
	createModel := fptcloud_storage.StorageDTO{VpcId: "vpc_id", Name: "storage-name", Type: "LOCAL", SizeGb: 100, StoragePolicyId: "policy-123"}
	storageId, err := service.CreateStorage(createModel)
	assert.NoError(t, err)
	assert.Equal(t, "1", storageId)
}

func TestUpdateStorage_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage": "",
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)
	updateModel := fptcloud_storage.UpdateStorageDTO{Name: "storage-rename", SizeGb: 200, StoragePolicyId: "policy_id"}
	response, err := service.UpdateStorage("vpc_id", "storage-name", updateModel)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestDeleteStorage_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage": "",
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)
	response, err := service.DeleteStorage("vpc-123", "storage_id")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestUpdateAttachedInstance_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage/storage_id/update-attached": "",
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)
	instanceId := "instance-123"
	response, err := service.UpdateAttachedInstance("vpc-123", "storage_id", &instanceId)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}
