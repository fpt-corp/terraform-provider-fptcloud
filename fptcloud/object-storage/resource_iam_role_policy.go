package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// The role's inline policy is what the role may do once assumed - distinct from
// its trusted_users, which is who may assume it. Like the user policy it is a
// single portal-managed document, replaced as a whole on every write.
func ResourceIamRolePolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIamRolePolicyCreate,
		ReadContext:   resourceIamRolePolicyRead,
		UpdateContext: resourceIamRolePolicyUpdate,
		DeleteContext: resourceIamRolePolicyDelete,
		Schema: map[string]*schema.Schema{
			"role_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the IAM role the policy is attached to",
			},
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
			"policy": {
				Type:     schema.TypeString,
				Required: true,
				Description: "The inline policy document in JSON format. Every resource it names must be a bucket " +
					"this account owns - the API rejects a policy that reaches outside it",
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: structure.SuppressJsonDiff,
			},
			"policy_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name the platform stores the inline policy under",
			},
		},
	}
}

func resourceIamRolePolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	roleName := d.Get("role_name").(string)
	policy := d.Get("policy").(string)

	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	resp := service.PutIamRolePolicy(vpcId, s3ServiceDetail.S3ServiceId, roleName, policy)
	if !resp.Status {
		switch reconcileIamPolicy(service.GetIamRolePolicy(vpcId, s3ServiceDetail.S3ServiceId, roleName), policy) {
		case createAdopted:
			// The document is in place after all: an earlier attempt committed and
			// only its response was lost.
		case createConflict:
			return diag.Errorf("IAM role %s already carries a different inline policy: %s", roleName, resp.Message)
		default:
			return diag.FromErr(fmt.Errorf("error applying IAM role policy: %s", resp.Message))
		}
	}

	d.SetId(roleName)
	return resourceIamRolePolicyRead(ctx, d, m)
}

func resourceIamRolePolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	stored := service.GetIamRolePolicy(vpcId, s3ServiceDetail.S3ServiceId, d.Id())
	// A role without a policy is a normal state, so it is drift here rather than
	// an error: the role may still exist with its document removed.
	if stored == nil || !stored.HasPolicy {
		d.SetId("")
		return nil
	}

	if err := d.Set("policy", string(stored.Policy)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("policy_name", stored.PolicyName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("role_name", d.Id()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceIamRolePolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	resp := service.PutIamRolePolicy(vpcId, s3ServiceDetail.S3ServiceId, d.Id(), d.Get("policy").(string))
	if !resp.Status {
		return diag.FromErr(fmt.Errorf("error updating IAM role policy: %s", resp.Message))
	}

	return resourceIamRolePolicyRead(ctx, d, m)
}

func resourceIamRolePolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	// Removing the policy leaves the role and its trust policy in place; the
	// role stays assumable but is permitted nothing.
	resp := service.DeleteIamRolePolicy(vpcId, s3ServiceDetail.S3ServiceId, d.Id())
	if !resp.Status {
		return diag.FromErr(fmt.Errorf("error deleting IAM role policy: %s", resp.Message))
	}

	return nil
}
