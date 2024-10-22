package fptcloud_object_storage

import (
	"context"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucketStaticWebsite() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketStaticWebsite,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the bucket to fetch policy for",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    false,
				Default:     "HCM-02",
				Optional:    true,
				Description: "The region name of the bucket",
			},
			"status": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Status of the bucket website configuration",
			},
			"request_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Request ID of the operation",
			},
			"host_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Host ID of the operation",
			},
			"http_status_code": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "HTTP status code of the operation",
			},
			"http_headers": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "HTTP headers of the response",
			},
			"retry_attempts": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of retry attempts",
			},
			"index_document": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Suffix for index document",
				ForceNew:    true,
			},
			"error_document": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Key for error document",
				ForceNew:    true,
			},
		},
	}
}

func dataSourceBucketStaticWebsite(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))

	staticWebsiteResponse := service.GetBucketWebsite(vpcId, bucketName, s3ServiceDetail.S3ServiceId)
	if !staticWebsiteResponse.Status {
		return diag.Errorf("failed to get bucket policy for bucket %s", bucketName)
	}

	d.SetId(bucketName)

	// Set the computed values
	if err := d.Set("status", staticWebsiteResponse.Status); err != nil {
		return diag.FromErr(err)
	}

	if staticWebsiteResponse.Config.ResponseMetadata.RequestID != "" {
		if err := d.Set("request_id", staticWebsiteResponse.Config.ResponseMetadata.RequestID); err != nil {
			return diag.FromErr(err)
		}
	}

	if staticWebsiteResponse.Config.ResponseMetadata.HostID != "" {
		if err := d.Set("host_id", staticWebsiteResponse.Config.ResponseMetadata.HostID); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("http_status_code", staticWebsiteResponse.Config.ResponseMetadata.HTTPStatusCode); err != nil {
		return diag.FromErr(err)
	}

	headers := map[string]string{
		"x-amz-request-id": staticWebsiteResponse.Config.ResponseMetadata.HTTPHeaders.XAmzRequestID,
		"content-type":     staticWebsiteResponse.Config.ResponseMetadata.HTTPHeaders.ContentType,
		"content-length":   staticWebsiteResponse.Config.ResponseMetadata.HTTPHeaders.ContentLength,
		"date":             staticWebsiteResponse.Config.ResponseMetadata.HTTPHeaders.Date,
	}
	if err := d.Set("http_headers", headers); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("retry_attempts", staticWebsiteResponse.Config.ResponseMetadata.RetryAttempts); err != nil {
		return diag.FromErr(err)
	}

	if staticWebsiteResponse.Config.IndexDocument.Suffix != "" {
		if err := d.Set("index_document", staticWebsiteResponse.Config.IndexDocument.Suffix); err != nil {
			return diag.FromErr(err)
		}
	}

	if staticWebsiteResponse.Config.ErrorDocument.Key != "" {
		if err := d.Set("error_document", staticWebsiteResponse.Config.ErrorDocument.Key); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
