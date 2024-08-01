package fptcloud_flavor

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/data-list"
)

// DataSourceFlavor function returns a schema.Resource that represents a Storage.
// This can be used to query and retrieve details about a specific Flavor in the infrastructure.
func DataSourceFlavor() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description:         "Retrieves information about the flavor that fpt cloud supports, with the ability to filter the results.",
		RecordSchema:        flavorSchema(),
		ResultAttributeName: "flavors",
		FlattenRecord:       flattenFlavor,
		GetRecords:          getFlavors,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the flavor",
			},
		},
	}

	return data_list.NewResource(dataListConfig)

}
func flavorSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the flavor",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the flavor",
		},
		"cpu": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The cpu number of the flavor",
		},
		"memory_mb": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The memory size (mb) of the flavor",
		},
		"gpu_memory_gb": {
			Type:        schema.TypeInt,
			Computed:    true,
			Optional:    true,
			Description: "The memory size (mb) of the gpu",
		},
		"type": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Flavor type (VM_SIZE | GPU_SIZE | OS)",
		},
	}
}

func flattenFlavor(flavor, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {

	s := flavor.(Flavor)

	flattened := map[string]interface{}{}
	flattened["name"] = s.Name
	flattened["id"] = s.ID
	flattened["cpu"] = s.Cpu
	flattened["memory_mb"] = s.MemoryMb
	flattened["gpu_memory_gb"] = s.GpuMemoryGb
	flattened["type"] = s.Type

	return flattened, nil
}

func getFlavors(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	flavorService := NewFlavorService(apiClient)

	vpcId, ok := extra["vpc_id"].(string)
	if !ok {
		return nil, fmt.Errorf("[ERR] vpc id is required")
	}

	flavors, err := flavorService.ListFlavor(vpcId)
	if err != nil {
		return nil, fmt.Errorf("[ERR] Failed to retrieve flavors: %s", err)
	}

	var templates []interface{}
	for _, flavor := range *flavors {
		templates = append(templates, flavor)
	}

	return templates, nil
}
