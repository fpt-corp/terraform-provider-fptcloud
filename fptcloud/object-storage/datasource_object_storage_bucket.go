package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucket() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketRead,
		Schema: map[string]*schema.Schema{
			"vpd_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the bucket",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region where the bucket is located",
			},
			"versioning": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether versioning is enabled",
			},
			"acl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Access control list",
			},
		},
	}
}

func dataSourceBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}
	page := 1
	if d.Get("page") != nil {
		page = d.Get("page").(int)
	}
	pageSize := 25
	if d.Get("page_size") != nil {
		pageSize = d.Get("page_size").(int)
	}
	buckets := service.ListBuckets(vpcId, s3ServiceDetail.S3ServiceId, page, pageSize)
	if buckets.Total == 0 {
		return diag.Errorf("no buckets found")
	}

	bucketName := d.Get("name").(string)
	for _, bucket := range buckets.Buckets {
		if bucket.Name == bucketName {
			d.SetId(bucket.Name)
			return nil
		}
	}

	return diag.Errorf("bucket with name %s not found", bucketName)
}
