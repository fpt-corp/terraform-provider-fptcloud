package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type S3ServiceDetail struct {
	S3ServiceName string
	S3ServiceId   string
	S3Platform    string
}

func ResourceBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketCreate,
		DeleteContext: resourceBucketDelete,
		ReadContext:   dataSourceBucketRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the bucket. Bucket names must be unique within an account.",
			},
			"versioning": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				ForceNew:    true,
				Description: "The versioning state of the bucket. Accepted values are Enabled or Suspended, default was not set.",
			},
			"region_name": {
				Type:     schema.TypeString,
				Required: false,
				// Default:  "HCM-02" if not provided
				Default:     "HCM-02",
				Optional:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service.",
			},
			"acl": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "private",
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func getServiceEnableRegion(objectStorageService ObjectStorageService, vpcId, regionName string) S3ServiceDetail {
	serviceEnable := objectStorageService.CheckServiceEnable(vpcId)
	if serviceEnable.Total == 0 {
		return S3ServiceDetail{}
	}

	var s3ServiceDetail S3ServiceDetail
	for _, service := range serviceEnable.Data {
		if service.S3ServiceName == regionName {
			s3ServiceDetail.S3ServiceName = service.S3ServiceName
			s3ServiceDetail.S3ServiceId = service.S3ServiceID
			s3ServiceDetail.S3Platform = service.S3Platform
			break
		}
	}
	return s3ServiceDetail
}

func resourceBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)

	req := BucketRequest{
		Name:       d.Get("name").(string),
		Versioning: d.Get("versioning").(string),
		Acl:        d.Get("acl").(string),
	}
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}

	bucket := objectStorageService.CreateBucket(req, vpcId, s3ServiceDetail.S3ServiceId)
	if !bucket.Status {
		return diag.Errorf(bucket.Message)
	}
	return resourceBucketRead(ctx, d, m)
}
func resourceBucketRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))

	bucket := objectStorageService.ListBuckets(vpcId, s3ServiceDetail.S3ServiceId, 1, 99999)
	if bucket.Total == 0 {
		return diag.Errorf("no buckets found")
	}
	for _, b := range bucket.Buckets {
		if b.Name == d.Get("name").(string) {
			d.SetId(b.Name)
			d.Set("name", b.Name)
			return nil
		}
	}
	return diag.Errorf("bucket with name %s not found", d.Get("name").(string))
}
func resourceBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	bucketName := d.Get("name").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))

	err := objectStorageService.DeleteBucket(bucketName, vpcId, s3ServiceDetail.S3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
