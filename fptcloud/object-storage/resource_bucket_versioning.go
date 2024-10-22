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
		DeleteContext: resourceBucketVersioningCreate,
		Schema: map[string]*schema.Schema{
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Enable or suspend versioning",
				ForceNew:    true, // Marking this field as ForceNew to ensure that the resource is recreated when the value is changed
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    false,
				Default:     "HCM-02",
				Optional:    true,
				ForceNew:    true,
				Description: "The region name of the bucket",
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
	enabled := d.Get("enabled").(bool)

	status := "Suspended"
	if enabled {
		status = "Enabled"
	}
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)

	err := service.PutBucketVersioning(vpcId, s3ServiceDetail.S3ServiceId, bucketName, BucketVersioningRequest{
		Status: status,
	})

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(bucketName)
	fmt.Println("Bucket versioning is updated for bucket", bucketName)
	return nil
}
