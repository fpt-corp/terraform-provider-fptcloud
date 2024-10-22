package fptcloud_object_storage

import (
	"context"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSubUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSubUserCreate,
		ReadContext:   resourceSubUserRead,
		UpdateContext: resourceSubUserUpdate,
		DeleteContext: resourceSubUserDelete,
		Schema: map[string]*schema.Schema{
			"role": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"user_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSubUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)

	req := SubUser{
		Role:   d.Get("role").(string),
		UserId: d.Get("user_id").(string),
	}

	subUser, err := objectStorageService.CreateSubUser(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(subUser.UserId)
	d.Set("role", subUser.Role)
	return resourceSubUserRead(ctx, d, m)
}

func resourceSubUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Implement the read logic
	return nil
}

func resourceSubUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Implement the update logic
	return resourceSubUserRead(ctx, d, m)
}

func resourceSubUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Implement the delete logic
	return nil
}
