package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSubUserKeys() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSubUserAccessKeyCreate,
		ReadContext:   resourceReadUserDetail,
		DeleteContext: resourceSubUserAccessKeyDelete,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC id that the S3 service belongs to",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"user_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The sub user id, can retrieve from data source `fptcloud_object_storage_sub_user`",
			},
			"access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key of the sub user",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The secret key of the sub user",
			},
		},
	}
}
func resourceSubUserAccessKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	subUserId := d.Get("user_id").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}
	resp := objectStorageService.CreateSubUserAccessKey(vpcId, s3ServiceDetail.S3ServiceId, subUserId)

	if !resp.Status {
		return diag.FromErr(fmt.Errorf("error creating sub-user access key: %s", resp.Message))
	}

	d.SetId(resp.Credential.AccessKey)
	if err := d.Set("access_key", resp.Credential.AccessKey); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	if err := d.Set("secret_key", resp.Credential.SecretKey); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}

	return nil
}
func resourceReadUserDetail(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}
	subUserId := d.Get("user_id").(string)

	subUser := objectStorageService.DetailSubUser(vpcId, s3ServiceDetail.S3ServiceId, subUserId)
	if subUser.UserID == "" {
		return diag.Errorf("sub-user with ID %s not found", subUserId)
	}
	if err := d.Set("user_id", subUser.UserID); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
func resourceSubUserAccessKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}
	subUserId := d.Get("user_id").(string)
	accessKeyToDelete := d.Get("access_key").(string)

	resp := objectStorageService.DeleteSubUserAccessKey(vpcId, s3ServiceDetail.S3ServiceId, subUserId, accessKeyToDelete)
	if !resp.Status {
		return diag.Errorf("failed to delete sub-user access key: %s", resp.Message)
	}
	d.SetId("")

	return resourceReadUserDetail(ctx, d, m)
}
