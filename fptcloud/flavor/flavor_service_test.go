package fptcloud_flavor_test

import (
	"terraform-provider-fptcloud/fptcloud/flavor"
	"testing"

	"github.com/stretchr/testify/assert"
	common "terraform-provider-fptcloud/commons"
)

func TestListFlavor_ReturnsFlavors(t *testing.T) {
	mockResponse := `{"data": [{"id": "1", "name": "flavor1", "info": {"vcpu": 2, "memory_mb": 2048, "gpu_memory_gb": 1}, "type": "VM_TYPE"}]}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/flavors": mockResponse,
	})
	defer server.Close()
	service := fptcloud_flavor.NewFlavorService(mockClient)
	flavors, err := service.ListFlavor("vpc_id")
	assert.NoError(t, err)
	assert.NotNil(t, flavors)
	assert.Equal(t, 1, len(*flavors))
	assert.Equal(t, "1", (*flavors)[0].ID)
	assert.Equal(t, "flavor1", (*flavors)[0].Name)
	assert.Equal(t, 2, (*flavors)[0].Cpu)
	assert.Equal(t, 2048, (*flavors)[0].MemoryMb)
	assert.Equal(t, 1, *(*flavors)[0].GpuMemoryGb)
	assert.Equal(t, "VM_TYPE", (*flavors)[0].Type)
}

func TestListFlavor_ReturnsEmptyListOnEmptyResponse(t *testing.T) {
	mockResponse := `{"data": []}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/flavors": mockResponse,
	})
	defer server.Close()
	service := fptcloud_flavor.NewFlavorService(mockClient)
	flavors, err := service.ListFlavor("vpc_id")
	assert.NoError(t, err)
	assert.NotNil(t, flavors)
	assert.Equal(t, 0, len(*flavors))
}
