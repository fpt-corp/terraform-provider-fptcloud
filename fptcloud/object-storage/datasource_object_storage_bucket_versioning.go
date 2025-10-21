package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucketVersioning() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketVersioningRead,
		Schema: map[string]*schema.Schema{
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"versioning_status": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Status of the versioning, must be Enabled or Suspended",
				ForceNew:    true, // Marking this field as ForceNew to ensure that the resource is recreated when the value is changed
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
		},
	}
}

func dataSourceBucketVersioningRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := GetServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, d.Get("region_name").(string)))
	}
	bucketName := d.Get("bucket_name").(string)

	versioning := service.GetBucketVersioning(vpcId, s3ServiceDetail.S3ServiceId, bucketName)
	if !versioning.Status {
		return diag.Errorf("Could not get versioning status for bucket %s", bucketName)
	}

	if err := d.Set("versioning_status", versioning.Config); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	d.SetId(bucketName)

	return nil
}
