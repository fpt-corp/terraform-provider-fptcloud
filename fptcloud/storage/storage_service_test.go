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
			"created_at": "2023-10-01T00:00:00Z",
			"tag_ids": ["tag-1","tag-2"]
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

func TestUpdateStorageTags_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage/storage_id/tags": "",
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)
	response, err := service.UpdateTags("vpc_id", "storage_id", []string{"tag-1"})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestListAllStorages_UsesListShape(t *testing.T) {
	mockResponse := `{
		"total": 2,
		"data": [
			{
				"id": "storage-1",
				"vpc_id": "vpc_id",
				"name": "disk-1",
				"display_name": "disk-1",
				"description": "",
				"size": 56320,
				"status": "ENABLED",
				"vm_id": "vm-1",
				"vm_name": "vm-1",
				"storage_type": "EXTERNAL",
				"storage_policy_id": "policy-1",
				"storage_policy_name": "Standard-HDD",
				"policy_uuid": "uuid-1",
				"disk_id": "disk-id-1",
				"encrypted": "0",
				"zone_id": "zone-1",
				"created_at": "2024-01-01T00:00:00",
				"has_snapshot": false,
				"tags": [{"id": "tag-1", "key": "env", "value": "prod", "color": "red"}],
				"can_modify": true
			},
			{
				"id": "storage-2",
				"vpc_id": "vpc_id",
				"name": "disk-2",
				"display_name": "disk-2",
				"description": "",
				"size": 20480,
				"status": "ENABLED",
				"vm_id": "vm-2",
				"vm_name": "vm-2",
				"storage_type": "EXTERNAL",
				"storage_policy_id": "policy-1",
				"storage_policy_name": "Standard-HDD",
				"policy_uuid": "uuid-1",
				"disk_id": "disk-id-2",
				"encrypted": "1",
				"zone_id": "zone-1",
				"created_at": "2024-01-01T00:00:00",
				"has_snapshot": true,
				"can_modify": true
			}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/storages": mockResponse,
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)

	storages, err := service.ListAll(fptcloud_storage.StorageListDTO{VpcId: "vpc_id", PageSize: 100})
	assert.NoError(t, err)
	assert.Len(t, storages, 2)
	assert.Equal(t, "storage-1", storages[0].ID)
	assert.Equal(t, 56320, storages[0].SizeMb)
	assert.Equal(t, "tag-1", storages[0].Tags[0].ID)
	assert.Equal(t, "Standard-HDD", *storages[0].StoragePolicyName)
	assert.Equal(t, "policy-1", *storages[0].StoragePolicyId)
	assert.False(t, bool(*storages[0].Encrypted))
	assert.True(t, bool(*storages[1].Encrypted))
}

func TestListAllStorages_EmptyList(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/storages": `{"total": 0, "data": []}`,
	})
	defer server.Close()
	service := fptcloud_storage.NewStorageService(mockClient)

	storages, err := service.ListAll(fptcloud_storage.StorageListDTO{VpcId: "vpc_id", PageSize: 100})
	assert.NoError(t, err)
	assert.Empty(t, storages)
}

func TestListAllStorages_RejectsOutOfRangePageSize(t *testing.T) {
	service := fptcloud_storage.NewStorageService(nil)

	_, err := service.ListAll(fptcloud_storage.StorageListDTO{VpcId: "vpc_id", PageSize: 0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "page_size")

	_, err = service.ListAll(fptcloud_storage.StorageListDTO{VpcId: "vpc_id", PageSize: 101})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "page_size")
}
