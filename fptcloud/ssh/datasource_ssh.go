package fptcloud_ssh

import (
	"context"
	"log"
	"strings"
	common "terraform-provider-fptcloud/commons"

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
	apiClient := m.(*common.Client)
	sshService := NewSSHKeyService(apiClient)
	var searchBy string

	if id, ok := d.GetOk("id"); ok {
		log.Printf("[INFO] Getting the ssh key by id")
		searchBy = id.(string)
	} else if name, ok := d.GetOk("name"); ok {
		log.Printf("[INFO] Getting the ssh key by name")
		searchBy = name.(string)
	}

	sshKey, err := sshService.FindSSHKey(searchBy)
	log.Printf("[INFO] search by : %s", searchBy)

	if err != nil {
		return diag.Errorf("[ERR] SSH key not found")
	}

	d.SetId(sshKey.ID)

	var setError error
	setError = d.Set("name", sshKey.Name)
	setError = d.Set("public_key", sshKey.PublicKey)
	setError = d.Set("created_at", sshKey.CreatedAt)
	if setError != nil {
		return diag.Errorf("[ERR] SSH key could not be found")
	}

	return nil
}
