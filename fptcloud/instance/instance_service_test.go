package fptcloud_instance_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	common "terraform-provider-fptcloud/commons"

	"github.com/stretchr/testify/assert"
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
	response, err := service.Resize("vpc_id", "instance_id", "flavor_id", "")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestChangeBillingTypeInstance_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instance/instance_id/billing-type": "",
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	response, err := service.ChangeBillingType("vpc_id", "instance_id", "RESERVED")
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

func TestListAllInstances_UsesListShape(t *testing.T) {
	mockResponse := `{
		"total": 2,
		"data": [
			{
				"id": "11111111-aaaa-1111-bbbb-111111111111",
				"vpc_id": "vpc_id",
				"name": "vm-1",
				"name_infra": "vm-1-infra",
				"status": "POWERED_ON",
				"guest_os": "Ubuntu",
				"host_name": null,
				"ip_address": "10.0.0.1",
				"ipv6_address": null,
				"number_of_cpus": 2,
				"memory_mb": 2048,
				"network_name": "net-1",
				"platform": "VMW",
				"created_at": "2024-01-01T00:00:00",
				"updated_at": "2024-01-01T00:00:00",
				"vGpuIds": [],
				"ip_public": "1.2.3.4",
				"enabled_allocate_ip": true,
				"storage_size_gb": 20,
				"billing_type": null,
				"gpu_name": null,
				"is_multi_storage": false,
				"is_nvme": false,
				"flavor": null,
				"vm_group_id": "group-1",
				"flavor_id": "flavor-1",
				"vm_tags": [{"id": "tag-1", "key": "env", "value": "prod", "color": "red"}]
			},
			{
				"id": "22222222-bbbb-2222-cccc-222222222222",
				"vpc_id": "vpc_id",
				"name": "vm-2",
				"name_infra": "vm-2",
				"status": "POWERED_OFF",
				"guest_os": "Windows",
				"host_name": null,
				"ip_address": "10.0.0.2",
				"ipv6_address": null,
				"number_of_cpus": 4,
				"memory_mb": 4096,
				"network_name": "net-1",
				"platform": "OSP",
				"created_at": "2024-01-01T00:00:00",
				"updated_at": "2024-01-01T00:00:00",
				"vGpuIds": [],
				"ip_public": "",
				"enabled_allocate_ip": false,
				"storage_size_gb": 40,
				"billing_type": null,
				"gpu_name": null,
				"is_multi_storage": false,
				"is_nvme": true,
				"flavor": null,
				"vm_tags": []
			}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instances": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)

	instances, err := service.ListAll(fptcloud_instance.InstanceListDTO{VpcId: "vpc_id", PageSize: 25})
	assert.NoError(t, err)
	assert.Len(t, instances, 2)
	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", *instances[0].ID)
	assert.Equal(t, "1.2.3.4", *instances[0].IpPublic)
	assert.Equal(t, 2, instances[0].CpuNumber)
	assert.Equal(t, "group-1", *instances[0].VmGroupId)
	assert.Equal(t, "tag-1", instances[0].VmTags[0].ID)
	assert.True(t, instances[1].IsNvme)
}

func TestListAllInstances_EmptyList(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instances": `{"total": 0, "data": []}`,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)

	instances, err := service.ListAll(fptcloud_instance.InstanceListDTO{VpcId: "vpc_id", PageSize: 25})
	assert.NoError(t, err)
	assert.Empty(t, instances)
}

func TestListAllInstances_WalksMultiplePages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		page := req.URL.Query().Get("page")
		if page == "1" {
			data := "["
			for i := 0; i < 25; i++ {
				if i > 0 {
					data += ","
				}
				data += fmt.Sprintf(`{"id":"vm-p1-%d","name":"vm-%d","flavor":null}`, i, i)
			}
			data += "]"
			_, _ = rw.Write([]byte(fmt.Sprintf(`{"total":30,"data":%s}`, data)))
			return
		}
		_, _ = rw.Write([]byte(`{"total":30,"data":[{"id":"vm-p2-0","name":"last","flavor":null}]}`))
	}))
	defer server.Close()
	mockClient, _ := common.NewClientForTestingWithServer(server)
	service := fptcloud_instance.NewInstanceService(mockClient)

	instances, err := service.ListAll(fptcloud_instance.InstanceListDTO{VpcId: "vpc_id", PageSize: 25})
	assert.NoError(t, err)
	assert.Len(t, instances, 26)
	assert.Equal(t, "vm-p1-0", *instances[0].ID)
	assert.Equal(t, "vm-p2-0", *instances[25].ID)
}

func TestListAllInstances_RejectsOutOfRangePageSize(t *testing.T) {
	service := fptcloud_instance.NewInstanceService(nil)

	_, err := service.ListAll(fptcloud_instance.InstanceListDTO{VpcId: "vpc_id", PageSize: 0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "page_size")

	_, err = service.ListAll(fptcloud_instance.InstanceListDTO{VpcId: "vpc_id", PageSize: 26})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "page_size")
}
func TestFindStoragePolicyByProfile_ResolvesTheMostSpecificPolicy(t *testing.T) {
	mockResponse := `{
		"data": [
			{"id": "policy-ssd", "infra_id": "infra-ssd", "name": "SSD"},
			{"id": "policy-ssd-premium", "infra_id": "infra-ssd-premium", "name": "SSD_Premium"},
			{"id": "policy-hdd", "infra_id": "infra-hdd", "name": "HDD"}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage-policies": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)

	// the longest policy name prefixing the profile wins, in whatever order the API lists them
	for profile, want := range map[string]string{
		"SSD_Premium_HN1": "policy-ssd-premium",
		"SSD_Premium":     "policy-ssd-premium",
		"SSD_HN1":         "policy-ssd",
		"SSD":             "policy-ssd",
		"HDD_SGN":         "policy-hdd",
	} {
		storagePolicy, err := service.FindStoragePolicyByProfile("vpc_id", profile)
		if assert.NoError(t, err, profile) {
			assert.Equal(t, want, storagePolicy.ID, profile)
		}
	}
}

func TestFindStoragePolicyByProfile_NeverMatchesAnotherPolicyByPrefix(t *testing.T) {
	mockResponse := `{"data": [{"id": "policy-ssd", "infra_id": "infra-ssd", "name": "SSD"}]}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage-policies": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)

	// only the policy name itself, or followed by "_<zone>", belongs to the policy
	for _, profile := range []string{"SSDX", "SSDX_HN1", "SS", "SSD-Premium", "NVME", "", "HDD_SSD"} {
		_, err := service.FindStoragePolicyByProfile("vpc_id", profile)
		assert.Error(t, err, profile)
	}
}

func TestFindStoragePolicyByProfile_RejectsEqualMatches(t *testing.T) {
	mockResponse := `{
		"data": [
			{"id": "policy-1", "infra_id": "infra-1", "name": "SSD"},
			{"id": "policy-2", "infra_id": "infra-2", "name": "SSD"}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/storage-policies": mockResponse,
	})
	defer server.Close()

	_, err := fptcloud_instance.NewInstanceService(mockClient).FindStoragePolicyByProfile("vpc_id", "SSD_HN1")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "several storage policies")
	}
}

func TestGetInstanceByName_QueriesByNameOnly(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance?name=team%2Fweb-x7k2p": `{"data": {"id": "11111111-aaaa-1111-bbbb-111111111111", "name": "team/web-x7k2p"}}`,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	instance, err := service.GetByName("vpc_id", "team/web-x7k2p")
	assert.NoError(t, err)
	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", instance.ID)
}

func TestGetInstanceByName_ReturnsErrorWithoutInstance(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/instance": `{"data": {}}`,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	instance, err := service.GetByName("vpc_id", "web")
	assert.Error(t, err)
	assert.Nil(t, instance)
}