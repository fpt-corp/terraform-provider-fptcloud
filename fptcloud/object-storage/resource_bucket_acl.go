package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceBucketAcl() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketAclCreate,
		ReadContext:   resourceBucketAclRead,
		DeleteContext: resourceBucketAclDelete,
		UpdateContext: nil,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC ID",
			},
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket to config the ACL",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"canned_acl": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Access Control List (ACL) status of the bucket which can be one of the following values: private, public-read, default is private",
				ForceNew:    true,
			},
			"apply_objects": {
				Type:        schema.TypeBool,
				Default:     false,
				ForceNew:    true,
				Optional:    true,
				Description: "Apply the ACL to all objects in the bucket",
			},
			"status": {
				Type:        schema.TypeBool,
				Computed:    true,
				ForceNew:    true,
				Description: "The status after configuring the bucket ACL",
			},
		},
	}
}

func resourceBucketAclCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	bucketName := d.Get("bucket_name").(string)
	regionName := d.Get("region_name").(string)
	cannedAcl := d.Get("canned_acl").(string)
	applyObjects := d.Get("apply_objects").(bool)
	if cannedAcl != "private" && cannedAcl != "public-read" {
		return diag.Errorf("canned_acl must be either private or public-read, got %s", cannedAcl)
	}
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", regionName))
	}
	var bucketAclRequest BucketAclRequest
	bucketAclRequest.CannedAcl = cannedAcl
	bucketAclRequest.ApplyObjects = applyObjects

	r := service.PutBucketAcl(vpcId, s3ServiceDetail.S3ServiceId, bucketName, bucketAclRequest)
	if !r.Status {
		if err := d.Set("status", false); err != nil {
			return diag.Errorf("failed to create bucket ACL for bucket %s", bucketName)
		}
		return diag.Errorf("failed to create bucket ACL for bucket %s", bucketName)
	}
	if err := d.Set("status", true); err != nil {
		return diag.FromErr(err)
	}
	return resourceBucketAclRead(ctx, d, m)
}

func resourceBucketAclRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	bucketName := d.Get("bucket_name").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", regionName))
	}
	r := service.GetBucketAcl(vpcId, s3ServiceDetail.S3ServiceId, bucketName)
	if !r.Status {
		return diag.Errorf("failed to get bucket ACL for bucket %s", bucketName)
	}
	if err := d.Set("canned_acl", r.CannedACL); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", r.Status); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceBucketAclDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Remove the resource from the state
	d.SetId("")
	return diag.Errorf("Delete operation is not supported for bucket ACLs. This is a no-op.")
}
