package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucketPolicy() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketPolicyRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the bucket to fetch policy for",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"policy": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The bucket policy in JSON format",
			},
		},
	}
}

func dataSourceBucketPolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, d.Get("region_name").(string)))
	}
	policyResponse := service.GetBucketPolicy(vpcId, s3ServiceDetail.S3ServiceId, bucketName)
	if !policyResponse.Status {
		return diag.Errorf("failed to get bucket policy for bucket %s", bucketName)
	}

	// Set the policy field in the schema
	if err := d.Set("policy", policyResponse.Policy); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	// Set the ID to be a combination of bucket name to ensure unique data source
	d.SetId(fmt.Sprintf("bucket_policy_%s", bucketName))

	return nil
}
