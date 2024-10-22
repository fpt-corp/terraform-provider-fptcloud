package fptcloud_edge_gateway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"terraform-provider-fptcloud/commons"
	"testing"
)

func TestReadEdgeGatewayDataSource(t *testing.T) {
	vpcId := "123bbb12-12f3-1234-abcd-a12345678a0b"

	mockResponse := `{
		"data": [
			{
				"id": "123456ab-1234-5ab6-7891-011abc12345ab",
				"vpc_id": "123bbb12-12f3-1234-abcd-a12345678a0b",
				"name": "edge_gateway_1",
				"description": null,
				"edge_gateway_id": "56789ab-1234-12ab-34cd-12345668",
				"vdc_group_id": null,
				"vdc_group_name": null,
				"created_at": "2020-01-01 00:00:00",
				"ip_quotas": "1000",
				"networks": []
			},
			{
				"id": "123456ab-1234-5ab6-7891-011abc12346ab",
				"vpc_id": "123bbb12-12f3-1234-abcd-a12345678a0b",
				"name": "edge_gateway_2",
				"description": null,
				"edge_gateway_id": "56789ab-1234-12ab-34cd-12345669",
				"vdc_group_id": null,
				"vdc_group_name": null,
				"created_at": "2020-01-01 00:00:00",
				"ip_quotas": "1000",
				"networks": []
			}
		]
	}`

	// Mock client and server setup
	apiPath := commons.ApiPath.EdgeGatewayList(vpcId)
	mockClient, server, _ := commons.NewClientForTesting(map[string]string{
		apiPath: mockResponse,
	})
	defer server.Close()

	state := edge_gateway{
		VpcId: types.StringValue(vpcId),
	}
	d := datasourceEdgeGateway{
		client: mockClient,
	}

	edgeGatewayList, err := d.internalRead(context.Background(), &state)

	// Check if the edgeGatewayList is correct
	assert.NoError(t, err)
	assert.NotNil(t, edgeGatewayList)

	assert.Equal(t, "123456ab-1234-5ab6-7891-011abc12345ab", (*edgeGatewayList)[0].Id)
	assert.Equal(t, "edge_gateway_1", (*edgeGatewayList)[0].Name)
	assert.Equal(t, "56789ab-1234-12ab-34cd-12345668", (*edgeGatewayList)[0].EdgeGatewayId)

	assert.Equal(t, "123456ab-1234-5ab6-7891-011abc12346ab", (*edgeGatewayList)[1].Id)
	assert.Equal(t, "edge_gateway_2", (*edgeGatewayList)[1].Name)
	assert.Equal(t, "56789ab-1234-12ab-34cd-12345669", (*edgeGatewayList)[1].EdgeGatewayId)
}
