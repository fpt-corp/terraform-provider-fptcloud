package fptcloud_vpc_test

import (
	"context"
	"terraform-provider-fptcloud/fptcloud/vpc"
	"testing"

	"github.com/stretchr/testify/assert"
	common "terraform-provider-fptcloud/commons"
)

func TestGetTenant_ReturnsTenant(t *testing.T) {
	mockResponse := `{
		"status": true,
		"data": {
			"id": "11111111-aaaa-1111-bbbb-111111111111",
			"name": "tenant-name"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/tenant": mockResponse,
	})
	defer server.Close()
	service := fptcloud_vpc.NewService(mockClient)
	tenant, err := service.GetTenant(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", tenant.Id)
	assert.Equal(t, "tenant-name", tenant.Name)
}

func TestFindVPC_ReturnsVPC(t *testing.T) {
	mockResponse := `{
		"status": true,
		"data": {
			"id": "11111111-aaaa-1111-bbbb-111111111111",
			"name": "vpc-name",
			"status": "ACTIVE"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/org/tenant_id/vpc": mockResponse,
	})
	defer server.Close()
	service := fptcloud_vpc.NewService(mockClient)
	searchModel := fptcloud_vpc.FindVPCParam{Name: "vpc-name"}
	vpc, err := service.FindVPC(context.Background(), "tenant_id", searchModel)
	assert.NoError(t, err)
	assert.NotNil(t, vpc)
	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", vpc.Id)
	assert.Equal(t, "vpc-name", vpc.Name)
}
