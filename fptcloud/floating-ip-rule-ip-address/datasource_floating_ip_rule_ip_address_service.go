package fptcloud_floating_ip_rule_ip_address

import (
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"
	data_list "terraform-provider-fptcloud/commons/data-list"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceFloatingIpRuleIpAddress function returns a schema.Resource that represents a floating ip.
// This can be used to query and retrieve details about a specific floating ip in the infrastructure using its id or name.
func DataSourceFloatingIpRuleIpAddress() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description: strings.Join([]string{
			"Get information on a floating ip for use in other resources. This data source provides all of the floating ip properties as configured on your account.",
			"An error will be raised if the provided floating ip name does not exist in your account.",
		}, "\n\n"),
		RecordSchema:        floatingIpRuleIpAddressSchema(),
		ResultAttributeName: "floating_ip_rule_ip_address_att",
		FlattenRecord:       flattenFloatingIpRuleIpAddress,
		GetRecords:          getFloatingIpRuleIpAddresss,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the floating ip rule ip address",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func floatingIpRuleIpAddressSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The vpc id of the ip address",
		},
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the ip address",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the ip address",
		},
	}
}

func flattenFloatingIpRuleIpAddress(floatingIp, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := floatingIp.(IpAddress)

	flattened := map[string]interface{}{}
	flattened["id"] = s.ID
	flattened["name"] = s.Name

	return flattened, nil
}

func getFloatingIpRuleIpAddresss(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	service := NewFloatingIpRuleIpAddressService(apiClient)

	vpcId, okVpcId := extra["vpc_id"].(string)
	if !okVpcId {
		return nil, fmt.Errorf("[ERR] Vpc id is required")
	}

	result, err := service.ListExistingIpOfFloatingIp(vpcId)
	if err != nil || len(*result) == 0 {
		return nil, fmt.Errorf("[ERR] Failed to retrieve ip address: %s", err)
	}

	var templates []interface{}
	for _, item := range *result {
		templates = append(templates, item)
	}

	return templates, nil
}
