package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceSubUser() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSubUserRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"page": {
				Optional:    true,
				Type:        schema.TypeInt,
				Description: "Page number",
			},
			"page_size": {
				Optional:    true,
				Type:        schema.TypeInt,
				Description: "Number of items per page",
			},
			"list_sub_user": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of sub-users",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"role": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"arn": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceSubUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	subUsers, err := service.ListSubUsers(vpcId, s3ServiceDetail.S3ServiceId, page, pageSize)
	if err != nil {
		return diag.FromErr(err)
	}
	if subUsers.Total == 0 {
		return diag.FromErr(fmt.Errorf("no sub-user found"))
	}
	var formattedData []interface{}
	for _, subUser := range subUsers.SubUsers {
		formattedData = append(formattedData, map[string]interface{}{
			"user_id": subUser.UserID,
			"role":    subUser.Role,
			"active":  subUser.Active,
			"arn":     subUser.Arn,
		})
	}
	if err := d.Set("list_sub_user", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting list_sub_user: %s", err))
	}
	d.SetId(vpcId)

	return nil
}
