package fptcloud_object_storage

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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

// Validation functions
func ValidateBucketName(v interface{}, k string) (warnings []string, errors []error) {
	name := v.(string)

	// Bucket name length validation
	if len(name) < 3 || len(name) > 63 {
		errors = append(errors, fmt.Errorf("bucket name must be between 3 and 63 characters long"))
	}

	// Bucket name format validation
	bucketNameRegex := regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)
	if !bucketNameRegex.MatchString(name) {
		errors = append(errors, fmt.Errorf("bucket name must contain only lowercase letters, numbers, dots, and hyphens, and must start and end with a letter or number"))
	}

	// Check for consecutive dots
	if strings.Contains(name, "..") {
		errors = append(errors, fmt.Errorf("bucket name cannot contain consecutive dots"))
	}

	// Check for IP address format
	ipRegex := regexp.MustCompile(`^(\d+\.){3}\d+$`)
	if ipRegex.MatchString(name) {
		errors = append(errors, fmt.Errorf("bucket name cannot be formatted as an IP address"))
	}

	return warnings, errors
}

func ValidateRegionName(v interface{}, k string) (warnings []string, errors []error) {
	region := v.(string)
	validRegions := []string{"HCM-01", "HCM-02", "HN-01", "HN-02"}

	for _, validRegion := range validRegions {
		if region == validRegion {
			return warnings, errors
		}
	}

	errors = append(errors, fmt.Errorf("region must be one of: %s", strings.Join(validRegions, ", ")))
	return warnings, errors
}

func ResourceBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketCreate,
		DeleteContext: resourceBucketDelete,
		ReadContext:   dataSourceBucketRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The name of the bucket. Bucket names must be unique within an account.",
				ValidateFunc: ValidateBucketName,
			},
			"versioning": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultVersioning,
				ForceNew:     true,
				Description:  "The versioning state of the bucket. Accepted values are Enabled or Suspended, default was not set.",
				ValidateFunc: validation.StringInSlice([]string{"Enabled", "Suspended"}, false),
			},
			"object_lock_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultObjectLockEnabled,
				ForceNew:    true,
				Description: "Whether S3 Object Lock is enabled for the bucket.",
			},
			"region_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
				ValidateFunc: ValidateRegionName,
			},
			"acl": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      defaultAcl,
				Description:  "The Access Control List (ACL) status of the bucket.",
				ValidateFunc: validation.StringInSlice([]string{"private", "public-read", "public-read-write", "authenticated-read"}, false),
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

func GetServiceEnableRegion(objectStorageService ObjectStorageService, vpcId, regionName string) S3ServiceDetail {
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
		Name:              d.Get("name").(string),
		Versioning:        d.Get("versioning").(string),
		Acl:               d.Get("acl").(string),
		ObjectLockEnabled: d.Get("object_lock_enabled").(bool),
	}
	s3ServiceDetail := GetServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
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

	s3ServiceDetail := GetServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
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
