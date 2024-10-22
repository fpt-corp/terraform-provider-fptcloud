package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceBucketStaticWebsite() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketStaticWebsiteCreate,
		ReadContext:   dataSourceBucketStaticWebsite,
		DeleteContext: resourceDeleteBucketStaticWebsite,
		//UpdateContext: nil,
		Schema: map[string]*schema.Schema{
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket",
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
			"index_document_suffix": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "index.html",
				ForceNew:    true,
				Description: "Suffix that is appended to a request that is for a directory",
			},
			"error_document_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "error.html",
				ForceNew:    true,
				Description: "The object key name to use when a 4XX class error occurs",
			},
			"status": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "The status after configuring the bucket website",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceBucketStaticWebsiteCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	indexDocument := d.Get("index_document_suffix").(string)
	errorDocument := d.Get("error_document_key").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	putBucketWebsite := service.PutBucketWebsite(vpcId, s3ServiceDetail.S3ServiceId, bucketName, BucketWebsiteRequest{
		Bucket: bucketName,
		Suffix: indexDocument,
		Key:    errorDocument,
	})
	fmt.Println("\n Put bucket website response: \n", putBucketWebsite)

	if !putBucketWebsite.Status {
		diag.Errorf("failed to create bucket website for bucket %s", bucketName)
		d.Set("status", false)
		return nil
	}
	d.Set("status", true)
	d.SetId(bucketName)
	return nil
}

func resourceDeleteBucketStaticWebsite(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)

	resp := service.DeleteBucketStaticWebsite(vpcId, s3ServiceDetail.S3ServiceId, bucketName)
	if !resp.Status {
		return diag.Errorf("failed to delete bucket website for bucket %s", bucketName)
	}

	return nil
}
