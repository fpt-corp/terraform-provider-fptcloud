package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucketStaticWebsite() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketStaticWebsite,
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
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"index_document_suffix": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"error_document_key": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceBucketStaticWebsite(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, d.Get("region_name").(string)))
	}

	staticWebsiteResponse := service.GetBucketWebsite(vpcId, s3ServiceDetail.S3ServiceId, bucketName)
	if !staticWebsiteResponse.Status {
		return diag.Errorf("failed to get bucket static website config for bucket %s", bucketName)
	}
	if staticWebsiteResponse.Config.IndexDocument.Suffix == "" && staticWebsiteResponse.Config.ErrorDocument.Key == "" {
		return diag.Errorf("bucket %s does not have static website configuration", bucketName)
	}
	if err := d.Set("index_document_suffix", staticWebsiteResponse.Config.IndexDocument.Suffix); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	if err := d.Set("error_document_key", staticWebsiteResponse.Config.ErrorDocument.Key); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	d.SetId(bucketName)
	return nil
}
