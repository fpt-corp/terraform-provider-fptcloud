package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceIamUserDetail() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIamUserDetailRead,
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
			"user_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the IAM user to query",
			},
			"user_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM user ID assigned by the storage platform",
			},
			"arn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM user ARN",
			},
			"path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM user path",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the IAM user was created",
			},
			"access_key_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "How many access keys the IAM user currently holds",
			},
			"has_policy": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether an inline policy is attached to the IAM user",
			},
			"policy_names": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The names of the inline policies attached to the IAM user",
			},
		},
	}
}

func dataSourceIamUserDetailRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	userName := d.Get("user_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	detail := service.GetIamUser(vpcId, s3ServiceDetail.S3ServiceId, userName)
	// A data source names something the configuration depends on, so a missing
	// user is an error here - unlike in the resource, where it is drift.
	if detail == nil || detail.UserName == "" {
		return diag.Errorf("IAM user %s not found in region %s", userName, regionName)
	}

	fields := map[string]interface{}{
		"user_id":          detail.UserID,
		"arn":              detail.Arn,
		"path":             detail.Path,
		"created_at":       detail.CreatedAt,
		"access_key_count": detail.AccessKeyCount,
		"has_policy":       detail.HasPolicy,
		"policy_names":     detail.PolicyNames,
	}
	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId(detail.UserName)

	return nil
}
