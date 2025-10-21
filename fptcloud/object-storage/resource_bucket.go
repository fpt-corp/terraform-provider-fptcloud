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
				Default:     "Suspended",
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
			"object_lock": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: "Enable object lock for the bucket. When enabled, objects in the bucket cannot be deleted or overwritten.",
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
	objectLock := d.Get("object_lock").(bool)
	if !objectLock {
		d.Set("object_lock", false)
	} else {
		d.Set("object_lock", true)
	}

	req := BucketRequest{
		Name:       d.Get("name").(string),
		Versioning: d.Get("versioning").(string),
		Acl:        d.Get("acl").(string),
		ObjectLock: objectLock,
	}
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, d.Get("region_name").(string)))
	}

	bucket := objectStorageService.CreateBucket(req, vpcId, s3ServiceDetail.S3ServiceId)
	fmt.Printf("Bucket response: %+v\n", bucket) // Debug
	if !bucket.Status {
		return diag.Errorf("failed to create bucket: %s", bucket.Message)
	}

	d.SetId(req.Name)

	// Set status
	if err := d.Set("status", bucket.Status); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set status for bucket %s: %w", req.Name, err))
	}

	fmt.Println("Bucket created successfully:", req.Name)
	return nil
}

func resourceBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	bucketName := d.Id()

	if bucketName == "" {
		return diag.Errorf("cannot delete bucket: no valid ID found, bucket may not have been created")
	}

	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, d.Get("region_name").(string)))
	}

	status := objectStorageService.DeleteBucket(vpcId, s3ServiceDetail.S3ServiceId, bucketName)
	if !status.Status {
		return diag.Errorf("failed to delete bucket %s: %s", bucketName, status.Message)
	}

	// Set status
	if err := d.Set("status", status.Status); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set status for bucket %s: %w", bucketName, err))
	}

	fmt.Println("Bucket deleted successfully:", bucketName)
	return nil
}
