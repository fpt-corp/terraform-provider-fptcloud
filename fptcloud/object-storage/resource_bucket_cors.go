package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceBucketCors() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketCorsCreate,
		UpdateContext: resourceBucketCorsUpdate,
		DeleteContext: resourceBucketCorsDelete,
		ReadContext:   dataSourceBucketCorsRead,
		Schema: map[string]*schema.Schema{
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC ID",
			}, "cors_rule": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The bucket cors rule",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_headers": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"allowed_methods": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"allowed_origins": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"expose_headers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"max_age_seconds": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceBucketCorsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}

	bucketName := d.Get("bucket_name").(string)
	corsRule := d.Get("cors_rule").([]interface{})

	cors := make([]CorsRule, 0)
	for _, rule := range corsRule {
		r := rule.(map[string]interface{})
		cors = append(cors, CorsRule{
			AllowedHeaders: r["allowed_headers"].([]string),
			AllowedMethods: r["allowed_methods"].([]string),
			AllowedOrigins: r["allowed_origins"].([]string),
			ExposeHeaders:  r["expose_headers"].([]string),
			MaxAgeSeconds:  r["max_age_seconds"].(int),
			ID:             "", // should implement later
		})
	}

	_, err := service.PutBucketCors(bucketName, vpcId, s3ServiceDetail.S3ServiceId, CorsRule{
		AllowedHeaders: cors[0].AllowedHeaders,
		AllowedMethods: cors[0].AllowedMethods,
		AllowedOrigins: cors[0].AllowedOrigins,
		ExposeHeaders:  cors[0].ExposeHeaders,
	})
	if err != nil {
		return diag.Errorf("failed to create bucket cors for bucket %s", bucketName)
	}

	d.SetId(bucketName)
	return nil
}

func resourceBucketCorsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}

	corsRule := d.Get("cors_rule").([]interface{})

	cors := make([]CorsRule, 0)
	for _, rule := range corsRule {
		r := rule.(map[string]interface{})
		cors = append(cors, CorsRule{
			AllowedHeaders: r["allowed_headers"].([]string),
			AllowedMethods: r["allowed_methods"].([]string),
			AllowedOrigins: r["allowed_origins"].([]string),
			ExposeHeaders:  r["expose_headers"].([]string),
			MaxAgeSeconds:  r["max_age_seconds"].(int),
			ID:             "random-string-id", // should implement later
		})
	}

	_, err := service.UpdateBucketCors(vpcId, s3ServiceDetail.S3ServiceId, bucketName, BucketCors{
		CorsRules: cors,
	})
	if err != nil {
		return diag.Errorf("failed to update bucket cors for bucket %s", bucketName)
	}

	d.SetId(bucketName)
	return nil
}

func resourceBucketCorsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceBucketCorsUpdate(ctx, d, m)
}
