package storage

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceStorage function returns a schema.Resource that represents a Storage.
// This can be used to query and retrieve details about a specific Storage in the infrastructure using its id or name.
func DataSourceStorage() *schema.Resource {
	return &schema.Resource{
		Description: strings.Join([]string{
			"Get information on a volume for use in other resources. This data source provides all of the storage properties as configured on your account.",
			"An error will be raised if the provided volume name does not exist in your account.",
		}, "\n\n"),
		ReadContext: dataSourceStorageRead,
		Schema: map[string]*schema.Schema{
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
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The vpc id of the storage",
			},
		},
	}
}

func dataSourceStorageRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}
