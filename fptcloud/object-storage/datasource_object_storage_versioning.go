package fptcloud_object_storage

import (
	"context"
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
			"vpd_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable or suspend versioning",
			},
		},
	}
}

func dataSourceBucketVersioningRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	bucketName := d.Get("bucket_name").(string)

	versioning := service.GetBucketVersioning(vpcId, bucketName, s3ServiceDetail.S3ServiceId)
	if versioning == nil {
		return diag.Errorf("failed to get bucket versioning for bucket %s", bucketName)
	}

	d.SetId(bucketName)
	d.Set("enabled", versioning.Status == "Enabled")

	return nil
}
