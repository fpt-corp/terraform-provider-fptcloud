package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucketLifecycle() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketLifecycle,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the bucket to fetch policy for",
			},
			"policy": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The bucket policy in JSON format",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    false,
				Default:     "HCM-02",
				Optional:    true,
				Description: "The region name of the bucket",
			},
			"page_size": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "25",
				Description: "The number of items to return in each page",
			},
			"page": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "1",
				Description: "The page number",
			},
		},
	}
}

func dataSourceBucketLifecycle(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	page := d.Get("page").(string)
	pageSize := d.Get("page_size").(string)

	lifeCycleResponse, err := service.GetBucketLifecycle(vpcId, s3ServiceDetail.S3ServiceId, bucketName, page, pageSize)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s-%s", vpcId, bucketName))
	if err := d.Set("policy", lifeCycleResponse.Rules); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
