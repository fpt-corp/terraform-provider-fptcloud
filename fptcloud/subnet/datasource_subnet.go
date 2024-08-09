package fptcloud_subnet

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strings"
	common "terraform-provider-fptcloud/commons"
	data_list "terraform-provider-fptcloud/commons/data-list"
)

// DataSourceSubnet function returns a schema.Resource that represents a subnet.
// This can be used to query and retrieve details about a specific subnet in the infrastructure using its id or name.
func DataSourceSubnet() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description: strings.Join([]string{
			"Get information on a subnet for use in other resources. This data source provides all of the subnet properties as configured on your account.",
			"An error will be raised if the provided subnet name does not exist in your account.",
		}, "\n\n"),
		RecordSchema:        subnetSchema(),
		ResultAttributeName: "subnets",
		FlattenRecord:       flattenSubnet,
		GetRecords:          getSubnets,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the subnet",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func subnetSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The vpc id of the subnet",
		},
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the subnet",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the subnet",
		},
		"network_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The network name of the subnet",
		},
		"gateway": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The gateway of the subnet",
		},
		"created_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The created at of the subnet",
		},
	}
}

func flattenSubnet(subnet, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := subnet.(Subnet)

	flattened := map[string]interface{}{}
	flattened["id"] = s.ID
	flattened["name"] = s.Name
	flattened["network_name"] = s.NetworkName
	flattened["gateway"] = s.Gateway
	flattened["edge_gateway"] = s.EdgeGateway
	flattened["created_at"] = s.CreatedAt
	return flattened, nil
}

func getSubnets(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	service := NewSubnetService(apiClient)

	vpcId, okVpcId := extra["vpc_id"].(string)
	if !okVpcId {
		return nil, fmt.Errorf("[ERR] Vpc id is required")
	}

	result, err := service.ListSubnet(vpcId)
	if err != nil || len(*result) == 0 {
		return nil, fmt.Errorf("[ERR] Failed to retrieve subnet: %s", err)
	}

	var templates []interface{}
	for _, item := range *result {
		templates = append(templates, item)
	}

	return templates, nil
}
