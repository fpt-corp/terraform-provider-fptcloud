package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceAccessKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAccessKeyCreate,
		ReadContext:   resourceAccessKeyRead,
		DeleteContext: resourceAccessKeyDelete,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC ID",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Required:    false,
				ForceNew:    true,
				Optional:    true,
				Description: "The access key ID",
			},
			"secret_access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "The secret access key",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"status": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "The status after creating the access key",
			},
			"message": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The message after creating the access key",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}
func resourceAccessKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	resp := service.CreateAccessKey(vpcId, s3ServiceDetail.S3ServiceId)

	if !resp.Status {
		return diag.Errorf("failed to delete sub-user access key: %s", resp.Message)
	}

	if resp.Credential.AccessKey != "" {
		d.SetId(resp.Credential.AccessKey)
		if err := d.Set("access_key_id", resp.Credential.AccessKey); err != nil {
			d.SetId("")
			return diag.FromErr(err)
		}
		if err := d.Set("secret_access_key", resp.Credential.SecretKey); err != nil {
			d.SetId("")
			return diag.FromErr(err)
		}
	}

	if err := d.Set("status", resp.Status); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	if resp.Message != "" {
		if err := d.Set("message", resp.Message); err != nil {
			d.SetId("")
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceAccessKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceId := getServiceEnableRegion(service, vpcId, regionName).S3ServiceId
	resp, err := service.ListAccessKeys(vpcId, s3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}
	secretAccessKey := d.Get("secret_access_key").(string)
	accessKeyId := d.Get("access_key_id").(string)
	for _, accessKey := range resp.Credentials {
		for _, key := range accessKey.Credentials {
			if key.AccessKey == accessKeyId {
				if err := d.Set("access_key_id", key.AccessKey); err != nil {
					d.SetId("")
					return diag.FromErr(err)
				}
				if err := d.Set("secret_access_key", secretAccessKey); err != nil {
					d.SetId("")
					return diag.FromErr(err)
				}
				break
			}
		}
	}
	return nil
}

func resourceAccessKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	accessKeyId := d.Id()
	if accessKeyId == "" {
		// If the access key ID is not set, try to get it from the data source
		accessKeyId = d.Get("access_key_id").(string)
	}

	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	if accessKeyId == "" {
		return diag.Errorf("access_key_id is required for deletion")
	}

	data := service.DeleteAccessKey(vpcId, s3ServiceDetail.S3ServiceId, accessKeyId)
	if !data.Status {
		return diag.Errorf("failed to delete access key %s: %s", accessKeyId, data.Message)
	}
	if err := d.Set("status", true); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
