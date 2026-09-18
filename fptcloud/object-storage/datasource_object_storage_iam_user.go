package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceIamUser() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIamUserRead,
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
			"list_iam_user": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of IAM users",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"user_id": {
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
					},
				},
			},
		},
	}
}

func dataSourceIamUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	list, err := service.ListIamUsers(vpcId, s3ServiceDetail.S3ServiceId, page, pageSize)
	if err != nil {
		return diag.FromErr(err)
	}

	// An account with no IAM users is a legitimate answer, not an error: the
	// list is simply empty and a consumer can branch on its length.
	formattedData := make([]interface{}, 0, len(list.IamUsers))
	for _, iamUser := range list.IamUsers {
		formattedData = append(formattedData, map[string]interface{}{
			"user_name":  iamUser.UserName,
			"user_id":    iamUser.UserID,
			"arn":        iamUser.Arn,
			"path":       iamUser.Path,
			"created_at": iamUser.CreatedAt,
		})
	}
	if err := d.Set("list_iam_user", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting list_iam_user: %s", err))
	}
	d.SetId(vpcId)

	return nil
}
