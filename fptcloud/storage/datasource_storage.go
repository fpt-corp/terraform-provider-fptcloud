package fptcloud_storage

import (
	"context"
	"strings"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceStorage function returns a schema.Resource that represents a Storage.
// This can be used to query and retrieve details about a specific Storage in the infrastructure using its id or name.
func DataSourceStorage() *schema.Resource {
	return &schema.Resource{
		Description: strings.Join([]string{
			"Get information on a storage for use in other resources. This data source provides all of the storage properties as configured on your account.",
			"An error will be raised if the provided storage name does not exist in your account.",
		}, "\n\n"),
		ReadContext: dataSourceStorageRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vpc id of the storage",
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.NoZeroValues,
				ExactlyOneOf: []string{"id", "name"},
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.NoZeroValues,
				ExactlyOneOf: []string{"id", "name"},
				Description:  "The name of the storage",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the storage",
			},
			"size_gb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the storage (in GB)",
			},
			"storage_policy": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The policy name of the storage",
			},
			"storage_policy_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The policy id of the storage",
			},
			"instance_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The vpc id of the storage",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The created at of the storage",
			},
		},
	}
}

func dataSourceStorageRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	storageService := NewStorageService(apiClient)

	findStorageModel := FindStorageDTO{}

	if id, ok := d.GetOk("id"); ok {
		findStorageModel.ID = id.(string)
	}

	if name, ok := d.GetOk("name"); ok {
		findStorageModel.Name = name.(string)
	}

	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findStorageModel.VpcId = vpcId.(string)
	}

	foundStorage, err := storageService.FindStorage(findStorageModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve storage: %s", err)
	}

	d.SetId(foundStorage.ID)

	if err := d.Set("name", foundStorage.Name); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("size_gb", foundStorage.SizeGb); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("storage_policy", foundStorage.StoragePolicy); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("storage_policy_id", foundStorage.StoragePolicyId); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("type", foundStorage.Type); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("instance_id", foundStorage.InstanceId); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("vpc_id", foundStorage.VpcId); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("created_at", foundStorage.CreatedAt); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
