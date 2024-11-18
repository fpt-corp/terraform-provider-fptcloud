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
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
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
			"status": {
				Type:        schema.TypeBool,
				Computed:    true,
				ForceNew:    true,
				Description: "The status after create or delete the bucket",
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
	if s3ServiceDetail.S3ServiceId == "" {
		return S3ServiceDetail{}
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
		return diag.Errorf("%s", bucket.Message)
	}
	return resourceBucketRead(ctx, d, m)
}
func resourceBucketRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}

	bucket := objectStorageService.ListBuckets(vpcId, s3ServiceDetail.S3ServiceId, 1, 99999)
	if bucket.Total == 0 {
		return diag.Errorf("no buckets found")
	}
	for _, b := range bucket.Buckets {
		if b.Name == d.Get("name").(string) {
			if err := d.Set("name", b.Name); err != nil {
				return diag.FromErr(err)
			}
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
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}

	satus := objectStorageService.DeleteBucket(vpcId, s3ServiceDetail.S3ServiceId, bucketName)
	if !satus.Status {
		return diag.Errorf("failed to delete bucket %s", bucketName)
	}

	return nil
}
