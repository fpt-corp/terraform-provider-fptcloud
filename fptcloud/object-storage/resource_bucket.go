package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type S3ServiceDetail struct {
	S3ServiceName string
	S3ServiceId   string
	S3Platform    string
}

func ResourceBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketCreate,
		DeleteContext: resourceBucketDelete,
		ReadContext:   dataSourceBucketRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"versioning": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Suspended",
				ForceNew: true,
			},
			"region_name": {
				Type:     schema.TypeString,
				Required: false,
				// Default:  "HCM-02" if not provided
				Default:  "HCM-02",
				Optional: true,
				ForceNew: true,
			},
			"acl": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "private",
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func getServiceEnableRegion(objectStorageService ObjectStorageService, vpcId, regionName string) S3ServiceDetail {
	serviceEnable := objectStorageService.CheckServiceEnable(vpcId)
	if serviceEnable.Total == 0 {
		return S3ServiceDetail{}
	}

	var s3ServiceDetail S3ServiceDetail
	for _, service := range serviceEnable.Data {
		if service.S3ServiceName == regionName {
			s3ServiceDetail.S3ServiceName = service.S3ServiceName
			s3ServiceDetail.S3ServiceId = service.S3ServiceID
			s3ServiceDetail.S3Platform = service.S3Platform
			break
		}
	}
	return s3ServiceDetail
}

func resourceBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)

	req := BucketRequest{
		Name:       d.Get("name").(string),
		Versioning: d.Get("versioning").(string),
		Acl:        d.Get("acl").(string),
	}
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))

	bucket, err := objectStorageService.CreateBucket(req, vpcId, s3ServiceDetail.S3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}
	fmt.Println("Bucket created with ID: ", bucket.Name)

	d.SetId(bucket.Name)
	return nil

}

func resourceBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))

	err := objectStorageService.DeleteBucket(d.Id(), vpcId, s3ServiceDetail.S3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
