package fptcloud_object_storage

import (
	"context"
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceBucketPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketPolicyCreate,
		UpdateContext: nil,
		DeleteContext: resourceBucketPolicyDelete,
		ReadContext:   dataSourceBucketPolicyRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC ID",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket",
			},
			"policy": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "The bucket policy in JSON format",
				ConflictsWith: []string{"policy_file"},
				ValidateFunc:  validation.StringIsJSON,
			},
			"policy_file": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "Path to the JSON file containing the bucket policy",
				ConflictsWith: []string{"policy"},
			},
			"status": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Status after bucket policy is created",
			},
		},
	}
}

func resourceBucketPolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)

	// Get policy content either from policy or policy_file
	var policyContent string
	if v, ok := d.GetOk("policy"); ok {
		policyContent = v.(string)
	} else if v, ok := d.GetOk("policy_file"); ok {
		// The actual file reading is handled by Terraform's built-in file() function
		// in the configuration, so we just get the content here
		policyContent = v.(string)
	} else {
		return diag.FromErr(fmt.Errorf("either 'policy' or 'policy_file' must be specified"))
	}

	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", regionName))
	}
	var jsonMap map[string]interface{}
	err := json.Unmarshal([]byte(policyContent), &jsonMap)
	if err != nil {
		return diag.FromErr(err)
	}
	// Reverse from string into json object for matching with the API request
	payload := map[string]interface{}{
		"policy": jsonMap,
	}
	resp := service.PutBucketPolicy(vpcId, s3ServiceDetail.S3ServiceId, bucketName, payload)

	if !resp.Status {
		d.Set("status", false)
		return diag.Errorf(fmt.Sprintf("Error create bucket policy: %s", resp.Message))
	}
	d.SetId(bucketName)
	if err := d.Set("status", true); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}

	return nil
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
