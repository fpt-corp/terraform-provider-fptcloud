package fptcloud_instance_group

import (
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"
	data_list "terraform-provider-fptcloud/commons/data-list"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceInstanceGroup function returns a schema.Resource that represents an instance group.
// This can be used to query and retrieve details about a specific instance group in the infrastructure using its id or name.
func DataSourceInstanceGroup() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description: strings.Join([]string{
			"Get information on an instance group for use in other resources. This data source provides all of the instance group properties as configured on your account.",
			"An error will be raised if the provided instance group name does not exist in your account.",
		}, "\n\n"),
		RecordSchema:        instanceGroupSchema(),
		ResultAttributeName: "instance_groups",
		FlattenRecord:       flattenInstanceGroup,
		GetRecords:          getInstanceGroups,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the instance group",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func instanceGroupSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The vpc id of the instance group",
		},
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the instance group",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the instance group",
		},
		"policy": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"name": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"is_active": {
						Type:     schema.TypeBool,
						Computed: true,
					},
				},
			},
		},
		"vms": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"name": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
		"created_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The created at of the instance group",
		},
	}
}

func flattenInstanceGroup(instanceGroup, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := instanceGroup.(InstanceGroup)

	flattened := map[string]interface{}{}
	flattened["id"] = s.ID
	flattened["name"] = s.Name
	flattened["policy"] = []interface{}{s.Policy}
	flattened["vms"] = s.Vms
	flattened["created_at"] = s.CreatedAt

	return flattened, nil
}

func getInstanceGroups(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	service := NewInstanceGroupService(apiClient)

	findModel := FindInstanceGroupDTO{}

	if id, ok := extra["id"].(string); ok {
		findModel.ID = id
	}

	if name, ok := extra["name"].(string); ok {
		findModel.Name = name
	}

	if vpcId, ok := extra["vpc_id"].(string); ok {
		findModel.VpcId = vpcId
	}

	result, err := service.FindInstanceGroup(findModel)
	if err != nil || len(*result) == 0 {
		return nil, fmt.Errorf("[ERR] Failed to retrieve instance group: %s", err)
	}

	var templates []interface{}
	for _, item := range *result {
		templates = append(templates, item)
	}

	return templates, nil
}
