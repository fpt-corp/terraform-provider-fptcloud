package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceBucketVersioning() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketVersioningCreate,
		ReadContext:   dataSourceBucketVersioningRead,
		DeleteContext: resourceBucketVersioningDelete,
		Schema: map[string]*schema.Schema{
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket",
			},
			"versioning_status": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Status of the versioning, must be Enabled or Suspended",
				ForceNew:    true, // Marking this field as ForceNew to ensure that the resource is recreated when the value is changed
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC ID",
			},
		},
	}
}

func resourceBucketVersioningCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)

	versioningStatus := d.Get("versioning_status").(string)
	if versioningStatus != "Enabled" && versioningStatus != "Suspended" {
		return diag.FromErr(fmt.Errorf("versioning status must be Enabled or Suspended"))
	}
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}
	err := service.PutBucketVersioning(vpcId, s3ServiceDetail.S3ServiceId, bucketName, BucketVersioningRequest{
		Status: versioningStatus,
	})

	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%s:%s", bucketName, versioningStatus))
	d.Set("versioning_status", versioningStatus)
	return nil
}

func resourceBucketVersioningDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	diag.FromErr(fmt.Errorf("deleting bucket versioning is not supported"))
	return nil
}
