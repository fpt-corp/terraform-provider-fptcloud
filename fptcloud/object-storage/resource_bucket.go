package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
		ReadContext:   resourceBucketRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the bucket. Bucket names must be unique within an account.",
			},
			"versioning": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "Suspended",
				ForceNew:     true,
				Description:  "The versioning state of the bucket. Accepted values are Enabled or Suspended, default was not set.",
				ValidateFunc: validation.StringInSlice([]string{"Enabled", "Suspended"}, false),
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"acl": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "private",
				ValidateFunc: validation.StringInSlice([]string{"private", "public-read"}, false),
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
	versioning := d.Get("versioning").(string)

	// Validate object lock and versioning constraints
	if objectLock && versioning == "Suspended" {
		return diag.FromErr(fmt.Errorf("object lock cannot be enabled when versioning is suspended. Object lock requires versioning to be enabled"))
	}

	if err := d.Set("object_lock", objectLock); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set object lock for bucket %s: %w", d.Get("name").(string), err))
	}

	req := BucketRequest{
		Name:       d.Get("name").(string),
		Versioning: versioning,
		Acl:        d.Get("acl").(string),
		ObjectLock: objectLock,
	}
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, d.Get("region_name").(string)))
	}

	bucket := objectStorageService.CreateBucket(req, vpcId, s3ServiceDetail.S3ServiceId)
	if !bucket.Status {
		return diag.Errorf("failed to create bucket: %s", bucket.Message)
	}

	d.SetId(req.Name)

	// Set status
	if err := d.Set("status", bucket.Status); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set status for bucket %s: %w", req.Name, err))
	}

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

	return nil
}

func resourceBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	bucketName := d.Get("name").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)

	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	// List buckets to find the specific bucket
	buckets := objectStorageService.ListBuckets(vpcId, s3ServiceDetail.S3ServiceId, 1, 1000)
	if buckets.Total == 0 {
		d.SetId("")
		return diag.Errorf("no buckets found")
	}

	// Find the specific bucket
	var foundBucket *struct {
		Name             string `json:"Name"`
		CreationDate     string `json:"CreationDate"`
		IsEmpty          bool   `json:"isEmpty"`
		S3ServiceID      string `json:"s3_service_id"`
		IsEnabledLogging bool   `json:"isEnabledLogging"`
		Endpoint         string `json:"endpoint"`
	}
	for _, bucket := range buckets.Buckets {
		if bucket.Name == bucketName {
			foundBucket = &bucket
			break
		}
	}

	if foundBucket == nil {
		d.SetId("")
		return diag.Errorf("bucket %s not found", bucketName)
	}

	// Set the basic attributes
	if err := d.Set("name", foundBucket.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("vpc_id", vpcId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region_name", regionName); err != nil {
		return diag.FromErr(err)
	}
	// Note: ListBuckets API doesn't return acl, versioning, object_lock details
	// These would need to be retrieved from state or other APIs
	if err := d.Set("status", true); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
