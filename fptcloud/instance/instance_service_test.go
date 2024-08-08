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
			"created_at": "2024-01-01T00:00:00"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/instance": mockResponse,
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	searchModel := fptcloud_instance.FindInstanceDTO{VpcId: "vpc_id", Name: "vm-12345678901-xyzxyzxyz"}
	instance, err := service.Find(searchModel)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", instance.ID)
	assert.Equal(t, "vm-12345678901-xyzxyzxyz", instance.Name)
}

func TestFindInstance_ReturnsErrorOnRequestFailure(t *testing.T) {
	mockResponse := `invalid`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/instance": mockResponse,
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
		"/v1/terraform/vpc/vpc_id/instance": mockResponse,
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
		"/v1/terraform/vpc/vpc_id/instance": "",
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
		"/v1/terraform/vpc/vpc_id/instance": "",
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
		"/v1/terraform/vpc/vpc_id/instance": "",
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
		"/v1/terraform/vpc/vpc_id/instance": "",
	})
	defer server.Close()
	service := fptcloud_instance.NewInstanceService(mockClient)
	response, err := service.Resize("vpc_id", "instance_id", "flavor_id")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}
