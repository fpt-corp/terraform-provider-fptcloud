package fptcloud_object_storage

import (
	"context"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSubUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSubUserCreate,
		ReadContext:   dataSourceSubUserRead,
		DeleteContext: resourceSubUserDelete,
		Schema: map[string]*schema.Schema{
			"role": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"user_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region_name": {
				Type:     schema.TypeString,
				Required: false,
				Default:  "HCM-02",
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSubUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	req := SubUser{
		Role:   d.Get("role").(string),
		UserId: d.Get("user_id").(string),
	}
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))

	subUser, err := objectStorageService.CreateSubUser(req, vpcId, s3ServiceDetail.S3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(subUser.UserId)
	d.Set("role", subUser.Role)
	return nil
}

func resourceSubUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	err := objectStorageService.DeleteSubUser(d.Id(), vpcId, s3ServiceDetail.S3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSubUserAccessKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	s3ServiceId := d.Get("s3_service_id").(string)
	subUserId := d.Get("sub_user_id").(string)

	accessKey := objectStorageService.CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId)
	if accessKey == nil {
		return diag.Errorf("failed to create sub-user access key")
	}

	d.SetId(accessKey.Credential.AccessKey)
	d.Set("access_key", accessKey.Credential.AccessKey)
	d.Set("secret_key", accessKey.Credential.SecretKey)

	return nil
}

func resourceSubUserAccessKeyDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	s3ServiceId := d.Get("s3_service_id").(string)
	subUserId := d.Get("sub_user_id").(string)
	accessKeyId := d.Id()

	resp := objectStorageService.DeleteSubUserAccessKey(vpcId, s3ServiceId, subUserId, accessKeyId)
	if !resp.Status {
		return diag.Errorf("failed to delete sub-user access key")
	}

	return nil
}
