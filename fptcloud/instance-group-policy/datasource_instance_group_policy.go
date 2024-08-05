package fptcloud_instance_group_policy

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/data-list"
)

// DataSourceInstanceGroupPolicy function returns a schema.Resource that represents an instance group policy.
// This can be used to query and retrieve details about a specific instance group policy in the infrastructure using its id or name.
func DataSourceInstanceGroupPolicy() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description:         "Retrieves information about the instance group policy that fpt cloud supports, with the ability to filter the results.",
		RecordSchema:        instanceGroupPolicySchema(),
		ResultAttributeName: "instance_group_policies",
		FlattenRecord:       flattenInstanceGroupPolicy,
		GetRecords:          getInstanceGroupPolicy,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the instance group policy",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func instanceGroupPolicySchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the instance group policy",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the instance group policy",
		},
	}
}

func flattenInstanceGroupPolicy(instanceGroupPolicy, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := instanceGroupPolicy.(InstanceGroupPolicy)

	flattened := map[string]interface{}{}
	flattened["name"] = s.Name
	flattened["id"] = s.ID

	return flattened, nil
}

func getInstanceGroupPolicy(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	instanceGroupPolicyService := NewInstanceGroupPolicyService(apiClient)

	vpcId, ok := extra["vpc_id"].(string)
	if !ok {
		return nil, fmt.Errorf("[ERR] VPC id is required")
	}

	instanceGroupPolicies, err := instanceGroupPolicyService.ListInstanceGroupPolicies(vpcId)
	if err != nil {
		return nil, fmt.Errorf("[ERR] Failed to retrieve instance group policies: %s", err)
	}

	var templates []interface{}
	for _, instanceGroupPolicy := range *instanceGroupPolicies {
		templates = append(templates, instanceGroupPolicy)
	}

	return templates, nil
}
