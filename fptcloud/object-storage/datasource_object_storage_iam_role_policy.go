package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceIamRolePolicy() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIamRolePolicyRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"role_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the IAM role whose inline policy to read",
			},
			"policy": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The attached inline policy document in JSON format, empty when the role has none",
			},
			"policy_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name the platform stores the inline policy under",
			},
			"has_policy": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether an inline policy is attached",
			},
			"template": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "A starter policy document built around a bucket this account owns. Useful as a " +
					"beginning for a policy, and it is guaranteed to pass the API's account-scope check",
			},
		},
	}
}

func dataSourceIamRolePolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	roleName := d.Get("role_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	stored := service.GetIamRolePolicy(vpcId, s3ServiceDetail.S3ServiceId, roleName)
	// nil means the role itself could not be read. A role that exists without a
	// policy still answers, with has_policy false.
	if stored == nil {
		return diag.Errorf("IAM role %s not found in region %s", roleName, regionName)
	}

	policy := ""
	if stored.HasPolicy {
		policy = string(stored.Policy)
	}
	fields := map[string]interface{}{
		"policy":      policy,
		"policy_name": stored.PolicyName,
		"has_policy":  stored.HasPolicy,
		"template":    string(stored.Template),
	}
	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId(roleName)

	return nil
}
