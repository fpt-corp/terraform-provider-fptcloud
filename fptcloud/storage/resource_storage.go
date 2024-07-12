package storage

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceStorage function returns a schema.Resource that represents a storage.
// This can be used to create, read, update, and delete operations for a storage in the infrastructure.
func ResourceStorage() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a Fpt cloud storage which can be attached to an instance in order to provide expanded storage.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A name that you wish to use to refer to this storage",
			},
			"size_gb": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "A minimum of 1 and a maximum of your available disk space from your quota specifies the size of the storage in gigabytes ",
			},
			"storage_policy": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The policy name of the storage",
			},
		},
		CreateContext: resourceStorageCreate,
		ReadContext:   resourceStorageRead,
		UpdateContext: resourceStorageUpdate,
		DeleteContext: resourceStorageDelete,
		Importer: &schema.ResourceImporter{
			State: resourceStorageImport,
		},
	}
}

// function to create the new Storage
func resourceStorageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceStorageRead(ctx, d, m)
}

// function to read the Storage
func resourceStorageRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

// function to update the Storage
func resourceStorageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceStorageRead(ctx, d, m)
}

// function to delete the Storage
func resourceStorageDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceStorageImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	return []*schema.ResourceData{d}, nil
}
