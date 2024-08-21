package fptcloud_ssh

import (
	"context"
	"log"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceSSHKey function returns a schema.Resource that represents an SSH Key.
// This can be used to create, read, and delete operations for an SSH Key in the infrastructure.
func ResourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a SSH key resource to allow you to manage SSH keys for instance access. Keys created with this resource can be referenced in your instance configuration via their ID.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "a string that will be the reference for the SSH key.",
				ValidateFunc: utils.ValidateName,
				ForceNew:     true,
			},
			"public_key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "a string containing the SSH public key.",
				ForceNew:    true,
			},
		},
		CreateContext: resourceSSHKeyCreate,
		ReadContext:   resourceSSHKeyRead,
		DeleteContext: resourceSSHKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

// function to create a new ssh key
func resourceSSHKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	sshService := NewSSHKeyService(apiClient)

	log.Printf("[INFO] creating the new ssh key %s", d.Get("name").(string))
	sshKey, err := sshService.NewSSHKey(d.Get("name").(string), d.Get("public_key").(string))
	if err != nil {
		return diag.Errorf("[ERR] failed to create a new ssh key: %s", err)
	}

	d.SetId(sshKey.ID)

	return resourceSSHKeyRead(ctx, d, m)
}

// function to read a ssh key
func resourceSSHKeyRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	sshService := NewSSHKeyService(apiClient)

	log.Printf("[INFO] retrieving the new ssh key %s", d.Get("name").(string))
	sshKey, err := sshService.FindSSHKey(d.Id())
	if err != nil {
		if sshKey == nil {
			d.SetId("")
			return nil
		}

		return diag.Errorf("[ERR] error retrieving ssh key: %s", err)
	}

	var setError error
	setError = d.Set("name", sshKey.Name)
	if setError != nil {
		return diag.Errorf("[ERR] error retrieving ssh key: %s", setError)
	}
	return nil
}

// function to delete the ssh key
func resourceSSHKeyDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	sshService := NewSSHKeyService(apiClient)

	log.Printf("[INFO] deleting the ssh key %s", d.Id())
	_, err := sshService.DeleteSSHKey(d.Id())
	if err != nil {
		return diag.Errorf("[ERR] an error occurred while trying to delete the ssh key %s", d.Id())
	}
	return nil
}
