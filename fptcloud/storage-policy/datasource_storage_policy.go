package fptcloud_storage_policy

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/data-list"
)

// DataSourceStoragePolicy function returns a schema.Resource that represents a Storage.
// This can be used to query and retrieve details about a specific Storage in the infrastructure using its id or name.
func DataSourceStoragePolicy() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description:         "Retrieves information about the storage policy that fpt cloud supports, with the ability to filter the results.",
		RecordSchema:        storagePolicySchema(),
		ResultAttributeName: "storage_policies",
		FlattenRecord:       flattenStoragePolicy,
		GetRecords:          getStoragePolicies,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the storage policy",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func storagePolicySchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the storage policy",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the storage policy",
		},
	}
}

func flattenStoragePolicy(storagePolicy, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := storagePolicy.(StoragePolicy)

	flattened := map[string]interface{}{}
	flattened["name"] = s.Name
	flattened["id"] = s.ID

	return flattened, nil
}

func getStoragePolicies(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	storageService := NewStoragePolicyService(apiClient)

	vpcId, ok := extra["vpc_id"].(string)
	if !ok {
		return nil, fmt.Errorf("[ERR] vpc id is required")
	}

	storagePolicies, err := storageService.ListStoragePolicy(vpcId)
	if err != nil {
		return nil, fmt.Errorf("[ERR] Failed to retrieve storage policy: %s", err)
	}

	var templates []interface{}
	for _, storagePolicy := range *storagePolicies {
		templates = append(templates, storagePolicy)
	}

	return templates, nil
}
