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
				Description: "The region name where the bucket is located, e.g., HCM-02, can be retrieved when creating the bucket",
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
	fmt.Println("applyObjects", applyObjects)
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
		d.Set("status", false)
		return diag.Errorf("failed to create bucket ACL for bucket %s", bucketName)
	}
	d.Set("status", true)
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
	d.Set("canned_acl", r.CannedACL)
	d.Set("status", r.Status)
	return nil
}

func resourceBucketAclDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Remove the resource from the state
	d.SetId("")
	fmt.Println("Delete operation is not supported for bucket ACLs. This is a no-op.")
	return nil
}
