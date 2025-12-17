package fptcloud_database_flavors

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/data-list"
)

// DataSourceDatabaseFlavor function returns a schema.Resource that represents a Database Flavor.
// This can be used to query and retrieve details about specific Database Flavors in the infrastructure.
func DataSourceDatabaseFlavor() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description:         "Retrieves information about the database flavor that fpt cloud supports, with the ability to filter the results.",
		RecordSchema:        databaseFlavorSchema(),
		ResultAttributeName: "database_flavors",
		FlattenRecord:       flattenDatabaseFlavor,
		GetRecords:          getDatabaseFlavors,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the database flavor",
			},
			"is_ops": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"no", "yes"}, false),
				Description:  "Whether the flavor is for OPS database (no: false, yes: true)",
			},
		},
	}
	return data_list.NewResource(dataListConfig)
}

func databaseFlavorSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the database flavor",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the database flavor",
		},
		"cpu": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The cpu number (vcpu) of the database flavor",
		},
		"memory_mb": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The memory size (mb) of the database flavor",
		},
		"is_scale": {
			Type:        schema.TypeInt,
			Computed:    true,
			Optional:    true,
			Description: "Whether the database flavor supports scaling (1: true, 0/null: false)",
		},
		"flavor_site": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The site URL of the database flavor",
		},
	}
}

func flattenDatabaseFlavor(flavor, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := flavor.(DatabaseFlavor)

	flattened := map[string]interface{}{}
	flattened["name"] = s.Name
	flattened["id"] = s.ID
	flattened["cpu"] = s.Cpu
	flattened["memory_mb"] = s.MemoryMb
	flattened["is_scale"] = s.IsScale
	flattened["flavor_site"] = s.FlavorSite

	return flattened, nil
}

func getDatabaseFlavors(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	if m == nil {
		return nil, fmt.Errorf("[ERR] provider configuration is nil")
	}

	apiClient, ok := m.(*common.Client)
	if !ok {
		return nil, fmt.Errorf("[ERR] invalid provider configuration type")
	}
	if apiClient == nil {
		return nil, fmt.Errorf("[ERR] API client is nil")
	}
	databaseFlavorService := NewDatabaseFlavorService(apiClient)

	vpcId, ok := extra["vpc_id"].(string)
	if !ok {
		return nil, fmt.Errorf("[ERR] vpc id is required")
	}

	isOps, ok := extra["is_ops"].(string)
	if !ok {
		return nil, fmt.Errorf("[ERR] is_ops parameter is required")
	}

	flavors, err := databaseFlavorService.ListDatabaseFlavor(vpcId, isOps)
	if err != nil {
		return nil, fmt.Errorf("[ERR] Failed to retrieve database flavors: %s", err)
	}
	if flavors == nil {
		return []interface{}{}, nil
	}

	var result []interface{}
	for _, flavor := range *flavors {
		result = append(result, flavor)
	}

	return result, nil
}
