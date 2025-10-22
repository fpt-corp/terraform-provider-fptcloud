package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceSubUserDetail() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSubUserDetailRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"user_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The sub-user ID to query",
			},
			"arn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The sub-user ARN",
			},
			"active": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the sub-user is active",
			},
			"role": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The sub-user's role",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The sub-user's creation date",
			},
			"access_keys": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "The sub-user's access keys",
			},
		},
	}
}

func dataSourceSubUserDetailRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)

	// Get all available regions and find the one with the sub-user
	serviceEnable := objectStorageService.CheckServiceEnable(vpcId)
	if serviceEnable.Total == 0 {
		return diag.FromErr(fmt.Errorf("no S3 services found for VPC %s", vpcId))
	}

	subUserId := d.Get("user_id").(string)
	var subUser *DetailSubUser

	// Try to find the sub-user in any available region
	for _, service := range serviceEnable.Data {
		subUser = objectStorageService.DetailSubUser(vpcId, service.S3ServiceID, subUserId)
		if subUser != nil && subUser.UserID != "" {
			break
		}
	}

	if subUser == nil {
		d.SetId("")
		return diag.Errorf("sub-user with ID %s not found", subUserId)
	}

	d.SetId(subUser.UserID)
	if subUser.UserID == "" {
		d.SetId("")
		return diag.Errorf("sub-user with ID %s not found", subUserId)
	}

	if err := d.Set("user_id", subUser.UserID); err != nil {
		return diag.FromErr(err)
	}
	if subUser.Arn != "" {
		if err := d.Set("arn", subUser.Arn); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("role", subUser.Role); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("active", subUser.Active); err != nil {
		return diag.FromErr(err)
	}
	if subUser.CreatedAt != "" {
		if err := d.Set("created_at", subUser.CreatedAt); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("access_keys", subUser.AccessKeys); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
