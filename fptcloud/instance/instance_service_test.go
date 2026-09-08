package fptcloud_instance_test

import (
	"terraform-provider-fptcloud/fptcloud/instance"
	"testing"

	"github.com/stretchr/testify/assert"
	common "terraform-provider-fptcloud/commons"
)

func TestFindInstance_ReturnsInstance(t *testing.T) {
	mockResponse := `{
		"data": {
			"id": "11111111-aaaa-1111-bbbb-111111111111",
			"vpc_id": "22222222-bbbb-2222-cccc-222222222222",
			"name": "vm-12345678901-xyzxyzxyz",
			"guest_os": "Ubuntu Linux (64-bit)",
			"host_name": null,
			"status": "POWERED_OFF",
			"private_ip": "10.0.0.1",
			"public_ip": null,
			"memory_mb": 2048,
			"cpu_number": 2,
			"flavor_id": "None",
			"subnet_id": "33333333-cccc-3333-dddd-333333333333",
			"storage_size_gb": 20,
			"storage_policy": "standard",
			"storage_policy_id": "44444444-dddd-4444-eeee-444444444444",
			"security_group_ids": [],
			"instance_group_id": "55555555-eeee-5555-ffff-555555555555",
			"created_at": "2024-01-01T00:00:00",
			"tag_ids": ["tag-id-1","tag-id-2"]
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	searchModel := fptcloud_instance.FindInstanceDTO{VpcId: "vpc_id", Name: "vm-12345678901-xyzxyzxyz"}
	instance, err := service.Find(searchModel)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", instance.ID)
	assert.Equal(t, "vm-12345678901-xyzxyzxyz", instance.Name)
	assert.ElementsMatch(t, []string{"tag-id-1", "tag-id-2"}, instance.TagIds)
}

func TestFindInstance_ReturnsErrorOnRequestFailure(t *testing.T) {
	mockResponse := `invalid`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	searchModel := fptcloud_instance.FindInstanceDTO{VpcId: "vpc_id", Name: "instance-name"}
	instance, err := service.Find(searchModel)
	assert.Error(t, err)
	assert.Nil(t, instance)
}

func TestCreateInstance_ReturnsInstanceIdWhenSuccess(t *testing.T) {
	mockResponse := `{"instance_id": "instance_id"}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	createModel := fptcloud_instance.CreateInstanceDTO{VpcId: "vpc_id", Name: "instance"}
	instanceId, err := service.Create(createModel)
	assert.NoError(t, err)
	assert.Equal(t, "instance_id", instanceId)
}

func TestDeleteInstance_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance": "",
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	response, err := service.Delete("vpc_id", "instance_id")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestRenameInstance_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance": "",
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	response, err := service.Rename("vpc_id", "instance_id", "new-name")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestChangeStatusInstance_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance": "",
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	response, err := service.ChangeStatus("vpc_id", "instance_id", "POWERED_ON")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestResizeInstance_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance": "",
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	response, err := service.Resize("vpc_id", "instance_id", "flavor_id")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestFindRootStorage_ReturnsRootDiskFromThePortalOnVmw(t *testing.T) {
	mockResponse := `{
		"total": 2,
		"data": [
			{"id": "s1", "disk_id": "disk-external", "storage_type": "EXTERNAL", "size": 40960, "status": "ENABLED", "storage_policy_id": "policy-db-2"},
			{"id": "s2", "disk_id": "disk-root", "storage_type": "ROOT", "size": 61440, "status": "ENABLED", "storage_policy_id": "policy-db-1"}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storagesv2": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	rootStorage, err := service.FindRootStorage("vpc_id", "instance_id")
	assert.NoError(t, err)
	assert.Equal(t, "disk-root", rootStorage.DiskId)
	assert.Equal(t, 61440, rootStorage.SizeMb)
	assert.Equal(t, "policy-db-1", rootStorage.StoragePolicyId)
}

// OSP persists the boot disk as LOCAL, never as ROOT
func TestFindRootStorage_ReturnsTheLocalDiskFromThePortalOnOsp(t *testing.T) {
	mockResponse := `{
		"total": 2,
		"data": [
			{"id": "s1", "disk_id": "disk-external", "storage_type": "EXTERNAL", "size": 40960, "status": "ENABLED"},
			{"id": "s2", "disk_id": "disk-boot", "storage_type": "LOCAL", "size": 20480, "status": "ENABLED", "storage_policy_id": "policy-db-1"}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storagesv2": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	rootStorage, err := service.FindRootStorage("vpc_id", "instance_id")
	assert.NoError(t, err)
	assert.Equal(t, "disk-boot", rootStorage.DiskId)
}

// the portal only lists a disk once its row is ENABLED, the infrastructure listing is the fallback
func TestFindRootStorage_FallsBackToTheInfrastructureListing(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storagesv2": `{"total": 0, "data": []}`,
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storages": `{
			"total": 2,
			"data": [
				{"disk_name": null, "disk_id": "disk-boot", "size_mb": 20480, "storage_profile_name": "Premium-SSD_floor5", "is_root": false, "storage_type": "external"},
				{"disk_name": "DISK-CD", "disk_id": "disk-external", "size_mb": 1024, "storage_profile_name": "Premium-SSD_floor5", "is_root": false, "storage_type": "external"}
			]
		}`,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	rootStorage, err := service.FindRootStorage("vpc_id", "instance_id")
	assert.NoError(t, err)
	assert.Equal(t, "disk-boot", rootStorage.DiskId)
	assert.Equal(t, "Premium-SSD_floor5", rootStorage.StoragePolicyName)
	assert.Empty(t, rootStorage.StoragePolicyId)
}

func TestFindRootStorage_FallsBackWhenThePortalRowHasNoDiskIdYet(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storagesv2": `{"total": 1, "data": [{"id": "s1", "disk_id": "", "storage_type": "ROOT", "size": 20480, "status": "CREATING"}]}`,
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storages":   `{"total": 1, "data": [{"disk_name": null, "disk_id": "disk-boot", "size_mb": 20480, "storage_profile_name": "Premium-SSD", "is_root": true, "storage_type": "root"}]}`,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	rootStorage, err := service.FindRootStorage("vpc_id", "instance_id")
	assert.NoError(t, err)
	assert.Equal(t, "disk-boot", rootStorage.DiskId)
}

func TestFindRootStorage_ReturnsErrorWhenBothListingsAreEmpty(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storagesv2": `{"total": 0, "data": []}`,
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storages":   `{"total": 0, "data": []}`,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	rootStorage, err := service.FindRootStorage("vpc_id", "instance_id")
	assert.Error(t, err)
	assert.Nil(t, rootStorage)
}

func TestFindStoragePolicy_ResolvesInfraId(t *testing.T) {
	mockResponse := `{
		"data": [
			{"id": "policy-db-1", "infra_id": "policy-infra-1", "name": "Premium-SSD"},
			{"id": "policy-db-2", "infra_id": "policy-infra-2", "name": "Standard-HDD"}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage-policies": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)

	storagePolicy, err := service.FindStoragePolicy("vpc_id", "policy-db-2")
	assert.NoError(t, err)
	assert.Equal(t, "policy-infra-2", storagePolicy.InfraId)
	assert.Equal(t, "Standard-HDD", storagePolicy.Name)

	// a config holding the infrastructure id is tolerated
	storagePolicy, err = service.FindStoragePolicy("vpc_id", "policy-infra-1")
	assert.NoError(t, err)
	assert.Equal(t, "policy-infra-1", storagePolicy.InfraId)

	_, err = service.FindStoragePolicy("vpc_id", "policy-unknown")
	assert.Error(t, err)
}

func TestResizeRootDisk_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/storages/resize": `{"status": true, "message": "Resize local disk successfully"}`,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	storagePolicyId := "policy-infra-1"
	response, err := service.ResizeRootDisk("vpc_id", "instance_id", fptcloud_instance.ResizeRootDiskDTO{
		DiskId:           "disk-root",
		IncreaseInSizeMb: 81920,
		StoragePolicyId:  &storagePolicyId,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Successfully", response.Data)
}
