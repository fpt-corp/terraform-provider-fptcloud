package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// An IAM user is created bare: no access key, no policy. Both are separate
// resources, which is what lets a key be rotated or a policy rewritten without
// destroying the identity that owns them.
func ResourceIamUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIamUserCreate,
		ReadContext:   resourceIamUserRead,
		DeleteContext: resourceIamUserDelete,
		Schema: map[string]*schema.Schema{
			"user_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the IAM user. The API has no rename, so changing it replaces the user",
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
			"user_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM user ID assigned by the storage platform",
			},
			"arn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM user ARN",
			},
			"path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM user path",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the IAM user was created",
			},
			"access_key_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "How many access keys the IAM user currently holds",
			},
			"has_policy": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether an inline policy is attached to the IAM user",
			},
		},
	}
}

func resourceIamUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	userName := d.Get("user_name").(string)

	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	resp := service.CreateIamUser(vpcId, s3ServiceDetail.S3ServiceId, userName)
	if !resp.Status {
		switch reconcileIamUser(service, vpcId, s3ServiceDetail.S3ServiceId, userName) {
		case createAdopted:
			// The user exists: an earlier attempt committed and only its response
			// was lost. Record it instead of failing forever.
		default:
			return diag.FromErr(fmt.Errorf("error creating IAM user: %s", resp.Message))
		}
	}

	d.SetId(userName)
	return resourceIamUserRead(ctx, d, m)
}

func resourceIamUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	detail := service.GetIamUser(vpcId, s3ServiceDetail.S3ServiceId, d.Id())
	// A user that is no longer there is drift, not a failure: clearing the ID
	// lets Terraform plan a recreate, whereas returning an error makes every
	// subsequent plan fail until the state entry is removed by hand.
	if detail == nil || detail.UserName == "" {
		d.SetId("")
		return nil
	}

	fields := map[string]interface{}{
		"user_name":        detail.UserName,
		"user_id":          detail.UserID,
		"arn":              detail.Arn,
		"path":             detail.Path,
		"created_at":       detail.CreatedAt,
		"access_key_count": detail.AccessKeyCount,
		"has_policy":       detail.HasPolicy,
	}
	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceIamUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	// The API revokes the user's access keys and removes its policies before it
	// deletes the user, so a destroy here takes those with it even when they are
	// separate resources in the configuration.
	if err := service.DeleteIamUser(vpcId, s3ServiceDetail.S3ServiceId, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
