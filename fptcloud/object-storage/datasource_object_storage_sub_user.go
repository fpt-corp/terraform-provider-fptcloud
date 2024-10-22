package fptcloud_object_storage

import (
	"context"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// datasource_object_storage_sub_user.go
func DataSourceSubUser() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSubUserRead,
		Schema: map[string]*schema.Schema{
			"role": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Role of the sub-user",
			},
			"user_id": {
				Type:        schema.TypeString,
				Description: "ID of the sub-user",
				ForceNew:    true,
				Required:    true,
			},
			"vpd_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
		},
	}
}

func dataSourceSubUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))

	subUsers, err := service.ListSubUsers(vpcId, s3ServiceDetail.S3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}

	role := d.Get("role").(string)
	for _, user := range subUsers {
		if user.Role == role {
			d.SetId(user.UserId)
			d.Set("user_id", user.UserId)
			return nil
		}
	}

	return diag.Errorf("sub-user with role %s not found", role)
}
