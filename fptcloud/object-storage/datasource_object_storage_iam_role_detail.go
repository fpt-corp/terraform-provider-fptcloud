package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceIamRoleDetail() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIamRoleDetailRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"role_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the IAM role to query",
			},
			"role_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM role ID assigned by the storage platform",
			},
			"arn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM role ARN",
			},
			"path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM role path",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the IAM role was created",
			},
			"trusted_users": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Names of the IAM users allowed to assume this role",
			},
		},
	}
}

func dataSourceIamRoleDetailRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	roleName := d.Get("role_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	role := service.GetIamRole(vpcId, s3ServiceDetail.S3ServiceId, roleName)
	if role == nil || role.RoleName == "" {
		return diag.Errorf("IAM role %s not found in region %s", roleName, regionName)
	}

	fields := map[string]interface{}{
		"role_id":       role.RoleID,
		"arn":           role.Arn,
		"path":          role.Path,
		"created_at":    role.CreatedAt,
		"trusted_users": role.TrustedUsers,
	}
	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId(role.RoleName)

	return nil
}
