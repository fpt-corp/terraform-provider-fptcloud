package fptcloud_subnet

import (
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"
	data_list "terraform-provider-fptcloud/commons/data-list"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the subnet",
		},
		"network_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The network id of the subnet",
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
		"cidr": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The CIDR block of the subnet (e.g. gateway/prefix_length such as 172.28.22.1/24).",
		},
		"gateway": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The gateway of the subnet",
		},
		"gateway_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The gateway ip of the subnet",
		},
		"edge_gateway": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "The edge gateway of the subnet",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"created_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The created at of the subnet",
		},
		"tag_ids": {
			Type:        schema.TypeList,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "List of tag IDs associated with the subnet",
		},
		"prefix_length": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The CIDR length of the subnet",
		},
		"type": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The type of the subnet. One of ISOLATED (no Internet) or NAT_ROUTED (Internet via NAT gateway).",
		},
		"primary_dns_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The primary DNS server IP address used by the subnet (e.g. 8.8.8.8).",
		},
		"secondary_dns_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The secondary DNS server IP address used by the subnet (e.g. 8.8.4.4).",
		},
	}
}

// extractTagIdsFromSubnet returns tag IDs from response. Prefer tags[].id (list API); fallback to tag_ids (get API). Handles tags nil, [], and items without id.
func extractTagIdsFromSubnet(s Subnet) []string {
	if len(s.Tags) > 0 {
		ids := make([]string, 0, len(s.Tags))
		for _, t := range s.Tags {
			if t.ID != "" {
				ids = append(ids, t.ID)
			}
		}
		return ids
	}
	return s.TagIds
}

func flattenSubnet(subnet, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := subnet.(Subnet)

	flattened := map[string]interface{}{}
	flattened["id"] = s.ID
	flattened["network_id"] = s.NetworkID
	flattened["name"] = s.Name
	flattened["network_name"] = s.NetworkName
	flattened["cidr"] = fmt.Sprintf("%s/%d", s.Gateway, s.PrefixLength)
	flattened["prefix_length"] = s.PrefixLength
	flattened["gateway"] = s.Gateway
	flattened["gateway_ip"] = s.Gateway
	edgeGateways := s.EdgeGateway
	mapEdgeGateway := map[string]interface{}{
		"id":              edgeGateways.ID,
		"name":            edgeGateways.Name,
		"edge_gateway_id": edgeGateways.EdgeGatewayId,
	}
	flattened["edge_gateway"] = mapEdgeGateway
	flattened["created_at"] = s.CreatedAt
	flattened["tag_ids"] = extractTagIdsFromSubnet(s)
	flattened["type"] = s.Type
	flattened["primary_dns_ip"] = s.PrimaryDNSIp
	flattened["secondary_dns_ip"] = s.SecondaryDNSIp
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
