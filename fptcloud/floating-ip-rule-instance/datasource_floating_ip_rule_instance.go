package fptcloud_floating_ip_rule_instance

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strings"
	common "terraform-provider-fptcloud/commons"
	data_list "terraform-provider-fptcloud/commons/data-list"
)

// DataSourceFloatingIpRuleInstance function returns a schema.Resource that represents a floating ip rule instance.
// This can be used to query and retrieve details about a specific floating ip rule instance in the infrastructure using its id or name.
func DataSourceFloatingIpRuleInstance() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description: strings.Join([]string{
			"Get information on a floating ip rule instance for use in other resources. This data source provides all of the floating ip rule instance properties as configured on your account.",
			"An error will be raised if the provided floating ip rule instance does not exist in your account.",
		}, "\n\n"),
		RecordSchema:        floatingIpRuleInstanceSchema(),
		ResultAttributeName: "floating_ip_rule_instance_att",
		FlattenRecord:       flattenFloatingIpRuleInstance,
		GetRecords:          getFloatingIpRuleInstances,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the floating ip rule instance",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func floatingIpRuleInstanceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The vpc id of the instance rule",
		},
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the instance rule",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the instance rule",
		},
		"ip_address": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The ip address of the instance rule",
		},
		"type": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The type of the instance rule",
		},
	}
}

func flattenFloatingIpRuleInstance(instanceRuleFloatingIp, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := instanceRuleFloatingIp.(InstanceRuleFloatingIp)

	flattened := map[string]interface{}{}
	flattened["id"] = s.ID
	flattened["name"] = s.Name
	flattened["ip_address"] = s.IpAddress
	flattened["type"] = s.Type

	return flattened, nil
}

func getFloatingIpRuleInstances(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	service := NewFloatingIpRuleInstanceService(apiClient)

	vpcId, okVpcId := extra["vpc_id"].(string)
	if !okVpcId {
		return nil, fmt.Errorf("[ERR] Vpc id is required")
	}

	result, err := service.ListExistingInstanceOfFloatingIp(vpcId)
	if err != nil || len(*result) == 0 {
		return nil, fmt.Errorf("[ERR] Failed to retrieve instance: %s", err)
	}

	var templates []interface{}
	for _, item := range *result {
		templates = append(templates, item)
	}

	return templates, nil
}
