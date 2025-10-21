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
				Default:     1,
			},
			"page_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of items per page",
				Default:     25,
			},
			"list_bucket_result": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"endpoint": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The endpoint of the bucket",
					},
					"is_enabled_logging": {
						Type:     schema.TypeBool,
						Required: true,
					},
					"bucket_name": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of the bucket",
					},
					"creation_date": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The creation date of the bucket",
					},
					"s3_service_id": {
						Type:     schema.TypeString,
						Required: true,
					},
					"is_empty": {
						Type:        schema.TypeBool,
						Required:    true,
						Description: "The bucket is empty or not",
					},
				}},
			},
		},
	}
}

func dataSourceBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	page := 1
	val, ok := d.GetOk("page")
	if ok {
		page = val.(int)
	}
	pageSize := 25
	valSize, ok := d.GetOk("page_size")
	if ok {
		pageSize = valSize.(int)
	}
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := GetServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}
	buckets := service.ListBuckets(vpcId, s3ServiceDetail.S3ServiceId, page, pageSize)
	if buckets.Total == 0 {
		return diag.Errorf("no buckets found")
	}
	var formattedData []interface{}
	for _, bucket := range buckets.Buckets {
		formattedData = append(formattedData, map[string]interface{}{
			"endpoint":           bucket.Endpoint,
			"is_enabled_logging": bucket.IsEnabledLogging,
			"bucket_name":        bucket.Name,
			"creation_date":      bucket.CreationDate,
			"s3_service_id":      bucket.S3ServiceID,
			"is_empty":           bucket.IsEmpty,
		})
	}
	if err := d.Set("list_bucket_result", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting data: %v", err))
	}
	d.SetId(vpcId)

	return nil
}
