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

// An IAM user carries a single portal-managed inline policy, so this resource
// is one per user. The write is a full replace, which is why an update is an
// ordinary in-place change rather than a destroy and recreate.
func ResourceIamUserPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIamUserPolicyCreate,
		ReadContext:   resourceIamUserPolicyRead,
		UpdateContext: resourceIamUserPolicyUpdate,
		DeleteContext: resourceIamUserPolicyDelete,
		Schema: map[string]*schema.Schema{
			"user_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the IAM user the policy is attached to",
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

func resourceIamUserPolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	userName := d.Get("user_name").(string)
	policy := d.Get("policy").(string)

	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	resp := service.PutIamUserPolicy(vpcId, s3ServiceDetail.S3ServiceId, userName, policy)
	if !resp.Status {
		switch reconcileIamPolicy(service.GetIamUserPolicy(vpcId, s3ServiceDetail.S3ServiceId, userName), policy) {
		case createAdopted:
			// The document is in place after all: an earlier attempt committed and
			// only its response was lost.
		case createConflict:
			return diag.Errorf("IAM user %s already carries a different inline policy: %s", userName, resp.Message)
		default:
			return diag.FromErr(fmt.Errorf("error applying IAM user policy: %s", resp.Message))
		}
	}

	d.SetId(userName)
	return resourceIamUserPolicyRead(ctx, d, m)
}

func resourceIamUserPolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	stored := service.GetIamUserPolicy(vpcId, s3ServiceDetail.S3ServiceId, d.Id())
	// No policy is a normal state for an IAM user, so it is drift here rather
	// than an error: the user may still exist with its document removed.
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
	if err := d.Set("user_name", d.Id()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceIamUserPolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	// The write is a full replace, and the API validates before it touches
	// anything - a refused document leaves the previous one in place.
	resp := service.PutIamUserPolicy(vpcId, s3ServiceDetail.S3ServiceId, d.Id(), d.Get("policy").(string))
	if !resp.Status {
		return diag.FromErr(fmt.Errorf("error updating IAM user policy: %s", resp.Message))
	}

	return resourceIamUserPolicyRead(ctx, d, m)
}

func resourceIamUserPolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	// Removing the policy leaves the IAM user and its access keys in place; only
	// what the user is permitted to do goes away.
	resp := service.DeleteIamUserPolicy(vpcId, s3ServiceDetail.S3ServiceId, d.Id())
	if !resp.Status {
		return diag.FromErr(fmt.Errorf("error deleting IAM user policy: %s", resp.Message))
	}

	return nil
}
