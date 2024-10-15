package fptcloud_mfke

import (
	"context"
	"github.com/stretchr/testify/assert"
	"terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"
	"testing"
)

func TestGetNetworkIdByPlatform(t *testing.T) {
	vpcId := "a659e537-f231-4c2b-93d7-1bb9b19af35c"
	mockResponse := `{
		"status": true,
		"message": "ok",
        "data": {
			"id": "f0e7d6fc-0f44-4e78-b6f1-d9447ad3138f",
			"name": "test-net",
			"network_name": "test-net",
			"gateway": "10.10.200.1",
			"vpc_id": "a659e537-f231-4c2b-93d7-1bb9b19af35c",
			"is_vdc_group": false,
			"edge_gateway": {
				"name": "XPLAT-EG",
				"id": "ba37b5a5-a9bb-401b-bbe3-6f0b0ff14f8b",
				"edge_gateway_id": "urn:vcloud:gateway:099110cc-4b4e-4c21-be40-d6c88d2f2036"
			},
			"prefix_length": 24,
			"created_at": "2023-10-02T08:35:28",
			"network_id": "urn:vcloud:network:10642b67-cf52-4076-bba0-0255786e2a6a",
			"dns_servers": "1.1.1.1 - 8.8.8.8",
			"tags": []
		}
	}`

	dto := fptcloud_subnet.FindSubnetDTO{
		NetworkName: "test-net",
		NetworkID:   "",
		VpcId:       vpcId,
	}
	apiPath := commons.ApiPath.FindSubnetByName(vpcId) + utils.ToQueryParams(dto)

	mockClient, server, _ := commons.NewClientForTesting(map[string]string{
		apiPath: mockResponse,
	})

	defer server.Close()

	service := fptcloud_subnet.NewSubnetService(mockClient)
	n, err := getNetworkId(context.Background(), service, vpcId, dto.NetworkName, dto.NetworkID)
	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.Equal(t, "f0e7d6fc-0f44-4e78-b6f1-d9447ad3138f", n)
}
