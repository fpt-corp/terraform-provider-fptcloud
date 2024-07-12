package ssh

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceSSHKey function returns a schema.Resource that represents an SSH Key.
// This can be used to query and retrieve details about a specific SSH Key in the infrastructure using its id or name.
func DataSourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Description: strings.Join([]string{
			"Get information on a SSH key. This data source provides the name as configured on your account.",
			"An error will be raised if the provided SSH key name does not exist in your account.",
		}, "\n\n"),
		ReadContext: dataSourceSSHKeyRead,
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
				Description:  "The name of the SSH key",
			},
		},
	}
}

func dataSourceSSHKeyRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}
