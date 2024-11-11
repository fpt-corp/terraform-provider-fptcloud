package fptcloud_object_storage

import (
	"context"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSubUserKeys() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSubUserAccessKeyCreate,
		ReadContext:   dataSourceSubUserRead,
		DeleteContext: resourceSubUserAccessKeyDelete,
		Schema: map[string]*schema.Schema{
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
				Required: true,
				ForceNew: true,
			},
		},
	}
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

	return dataSourceSubUserRead(ctx, d, m)
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
