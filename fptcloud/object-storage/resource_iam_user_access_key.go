package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// An IAM user may hold at most two access keys, which is what makes a rotation
// possible: create the second, move the consumers over, destroy the first.
//
// Unlike every other resource here this one is never adopted after a failed
// create. The secret is returned exactly once and no endpoint can read it back,
// so a key adopted from the server would be recorded without a usable secret -
// worse than a clean failure.
func ResourceIamUserAccessKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIamUserAccessKeyCreate,
		ReadContext:   resourceIamUserAccessKeyRead,
		DeleteContext: resourceIamUserAccessKeyDelete,
		Schema: map[string]*schema.Schema{
			"user_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the IAM user the access key belongs to",
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
			"access_key_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key ID",
			},
			"secret_access_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
				Description: "The secret access key. The API returns it only when the key is created, so it is " +
					"recorded in state at that moment and can never be recovered afterwards - a lost secret means a new key",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the access key was created",
			},
			"created_by": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The portal user that created the access key",
			},
		},
	}
}

func resourceIamUserAccessKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	userName := d.Get("user_name").(string)

	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	created := service.CreateIamUserAccessKey(vpcId, s3ServiceDetail.S3ServiceId, userName)
	if !created.Status {
		return diag.FromErr(fmt.Errorf("error creating access key for IAM user %s: %s", userName, created.Message))
	}
	if created.AccessKey.AccessKeyID == "" {
		return diag.FromErr(fmt.Errorf("the API reported success but returned no access key for IAM user %s", userName))
	}

	d.SetId(created.AccessKey.AccessKeyID)

	// Written here and nowhere else: the read cannot recover the secret, so a
	// later refresh must not be the first thing to record it.
	fields := map[string]interface{}{
		"access_key_id":     created.AccessKey.AccessKeyID,
		"secret_access_key": created.AccessKey.SecretAccessKey,
		"created_at":        created.AccessKey.CreatedAt,
	}
	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			d.SetId("")
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceIamUserAccessKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	userName := d.Get("user_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	list, err := service.ListIamUserAccessKeys(vpcId, s3ServiceDetail.S3ServiceId, userName)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, key := range list.AccessKeys {
		if key.AccessKeyID != d.Id() {
			continue
		}
		// secret_access_key is deliberately left alone. The listing never carries
		// it, so writing it here would blank the only copy in state.
		fields := map[string]interface{}{
			"access_key_id": key.AccessKeyID,
			"created_at":    key.CreatedAt,
			"created_by":    key.CreatedBy,
		}
		for name, value := range fields {
			if err := d.Set(name, value); err != nil {
				return diag.FromErr(err)
			}
		}
		return nil
	}

	// Revoked out of band, or the user itself is gone. Either way the key is not
	// there and Terraform should plan a new one.
	d.SetId("")
	return nil
}

func resourceIamUserAccessKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	userName := d.Get("user_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	resp := service.DeleteIamUserAccessKey(vpcId, s3ServiceDetail.S3ServiceId, userName, d.Id())
	if !resp.Status {
		return diag.FromErr(fmt.Errorf("error revoking access key %s: %s", d.Id(), resp.Message))
	}

	return nil
}
