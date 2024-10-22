package fptcloud_object_storage

import (
	"context"
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
			"s3_service_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The S3 service ID",
			},
			"sub_user_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The sub-user ID",
			},
			"user_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The sub-user ID",
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
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	s3ServiceId := d.Get("s3_service_id").(string)
	subUserId := d.Get("sub_user_id").(string)

	subUser := service.DetailSubUser(vpcId, s3ServiceId, subUserId)
	if subUser == nil {
		return diag.Errorf("sub-user with ID %s not found", subUserId)
	}

	d.SetId(subUser.UserID)
	d.Set("user_id", subUser.UserID)
	if subUser.Arn != nil {
		d.Set("arn", subUser.Arn)
	}
	d.Set("active", subUser.Active)
	d.Set("role", subUser.Role)
	if subUser.CreatedAt != nil {
		d.Set("created_at", subUser.CreatedAt)
	}
	d.Set("access_keys", subUser.AccessKeys)

	return nil
}
