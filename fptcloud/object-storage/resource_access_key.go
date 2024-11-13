package fptcloud_object_storage

import (
	"context"
	"fmt"
	"log"
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
		return diag.FromErr(fmt.Errorf("region %s is not enabled", regionName))
	}

	resp := service.CreateAccessKey(vpcId, s3ServiceDetail.S3ServiceId)

	if !resp.Status {
		return diag.Errorf("failed to delete sub-user access key: %s", resp.Message)
	}

	if resp.Credential.AccessKey != "" {
		d.SetId(resp.Credential.AccessKey)
		d.Set("access_key_id", resp.Credential.AccessKey)
		d.Set("secret_access_key", resp.Credential.SecretKey)
	}

	d.Set("status", resp.Status)
	if resp.Message != "" {
		d.Set("message", resp.Message)
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
				d.Set("access_key_id", key.AccessKey)
				d.Set("secret_access_key", secretAccessKey)
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
		log.Printf("[ERROR] Region %s is not enabled for VPC %s", regionName, vpcId)
		return diag.FromErr(fmt.Errorf("region %s is not enabled", regionName))
	}

	log.Printf("[DEBUG] Found S3 service ID: %s", s3ServiceDetail.S3ServiceId)

	if accessKeyId == "" {
		log.Printf("[ERROR] access_key_id is empty")
		return diag.Errorf("access_key_id is required for deletion")
	}

	err := service.DeleteAccessKey(vpcId, s3ServiceDetail.S3ServiceId, accessKeyId)
	if err != nil {
		log.Printf("[ERROR] Failed to delete access key %s: %v", accessKeyId, err)
		return diag.FromErr(err)
	}
	d.Set("status", true)
	d.SetId("")
	return nil
}
