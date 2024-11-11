package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceBucketPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketPolicyCreate,
		UpdateContext: resourceBucketPolicyUpdate,
		DeleteContext: resourceBucketPolicyDelete,
		ReadContext:   dataSourceBucketPolicyRead,
		Schema: map[string]*schema.Schema{
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket",
			},
			"policy": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The bucket policy in JSON format",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
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

func resourceBucketPolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	policy := d.Get("policy").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}

	resp := service.PutBucketPolicy(vpcId, s3ServiceDetail.S3ServiceId, bucketName, BucketPolicyRequest{
		Policy: policy,
	})

	if !resp.Status {
		return diag.Errorf("failed to create bucket policy for bucket %s", bucketName)
	}

	d.SetId(bucketName)
	return nil
}

func resourceBucketPolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceBucketPolicyCreate(ctx, d, m)
}

func resourceBucketPolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}

	resp := service.PutBucketPolicy(vpcId, s3ServiceDetail.S3ServiceId, bucketName, BucketPolicyRequest{
		Policy: "",
	})

	if !resp.Status {
		return diag.Errorf("failed to delete bucket policy for bucket %s", d.Id())
	}

	return nil
}
