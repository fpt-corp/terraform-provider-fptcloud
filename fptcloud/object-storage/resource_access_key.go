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
				Computed:    true,
				ForceNew:    true,
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
				Description: "The region name to create the access key",
			},
			"create_access_key_response": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The create access key response",
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
	resp := service.CreateAccessKey(vpcId, s3ServiceDetail.S3ServiceId)
	var createAccessKeyResponse CreateAccessKeyResponse
	if resp.Credential.AccessKey != "" && resp.Credential.SecretKey != "" {
		createAccessKeyResponse.Credential.AccessKey = resp.Credential.AccessKey
		createAccessKeyResponse.Credential.SecretKey = resp.Credential.SecretKey
	}
	if resp.Message != "" {
		createAccessKeyResponse.Message = resp.Message
	}
	createAccessKeyResponse.Status = resp.Status
	fmt.Println("Create access key response: ", createAccessKeyResponse)

	p := fmt.Sprintf("%v", createAccessKeyResponse)
	d.Set("access_key_id", createAccessKeyResponse.Credential.AccessKey)
	d.Set("secret_access_key", createAccessKeyResponse.Credential.SecretKey)
	d.SetId(resp.Credential.AccessKey)
	d.Set("create_access_key_response", p)

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
	accessKeyId := d.Get("access_key_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)

	service.DeleteAccessKey(vpcId, s3ServiceDetail.S3ServiceId, accessKeyId)
	return nil
}
