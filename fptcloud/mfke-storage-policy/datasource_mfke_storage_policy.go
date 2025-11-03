package fptcloud_mfke_storage_policy

import (
	"fmt"
	common "terraform-provider-fptcloud/commons"
	data_list "terraform-provider-fptcloud/commons/data-list"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceMfkeStoragePolicy function returns a schema.Resource that represents MFKE storage policies.
// This can be used to query and retrieve details about MFKE storage policies in the infrastructure.
func DataSourceMfkeStoragePolicy() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description:         "Retrieves information about the MFKE storage policies that fpt cloud supports, with the ability to filter the results.",
		RecordSchema:        mfkeStoragePolicySchema(),
		ResultAttributeName: "storage_policies",
		FlattenRecord:       flattenMfkeStoragePolicy,
		GetRecords:          getMfkeStoragePolicies,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the MFKE storage policy",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func mfkeStoragePolicySchema() map[string]*schema.Schema {
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
		"is_default": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether this is the default storage policy",
		},
		"id_db": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The database id of the storage policy",
		},
		"zone": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The zone of the storage policy",
		},
	}
}

func flattenMfkeStoragePolicy(storagePolicy, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := storagePolicy.(MfkeStoragePolicy)

	flattened := map[string]interface{}{}
	flattened["id"] = s.ID
	flattened["name"] = s.Name
	flattened["is_default"] = s.IsDefault
	flattened["id_db"] = s.IDDb
	flattened["zone"] = s.Zone

	return flattened, nil
}

func getMfkeStoragePolicies(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	storageService := NewMfkeStoragePolicyService(apiClient)

	vpcId, ok := extra["vpc_id"].(string)
	if !ok {
		return nil, fmt.Errorf("[ERR] vpc id is required")
	}

	storagePolicies, err := storageService.ListMfkeStoragePolicy(vpcId)
	if err != nil {
		return nil, fmt.Errorf("[ERR] Failed to retrieve MFKE storage policy: %s", err)
	}

	var templates []interface{}
	for _, storagePolicy := range *storagePolicies {
		templates = append(templates, storagePolicy)
	}

	return templates, nil
}
