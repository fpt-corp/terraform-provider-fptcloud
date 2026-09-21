package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceIamRole() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIamRoleRead,
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
			"page": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Page number",
			},
			"page_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of items per page",
			},
			"list_iam_role": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of IAM roles",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"role_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"arn": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"path": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"created_at": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"trusted_users": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func dataSourceIamRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	page := 1
	pageSize := 100
	if d.Get("page").(int) > 0 {
		page = d.Get("page").(int)
	}
	if d.Get("page_size").(int) > 0 {
		pageSize = d.Get("page_size").(int)
	}

	list, err := service.ListIamRoles(vpcId, s3ServiceDetail.S3ServiceId, page, pageSize)
	if err != nil {
		return diag.FromErr(err)
	}

	formattedData := make([]interface{}, 0, len(list.IamRoles))
	for _, iamRole := range list.IamRoles {
		formattedData = append(formattedData, map[string]interface{}{
			"role_name":     iamRole.RoleName,
			"role_id":       iamRole.RoleID,
			"arn":           iamRole.Arn,
			"path":          iamRole.Path,
			"created_at":    iamRole.CreatedAt,
			"trusted_users": iamRole.TrustedUsers,
		})
	}
	if err := d.Set("list_iam_role", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting list_iam_role: %s", err))
	}
	d.SetId(vpcId)

	return nil
}
