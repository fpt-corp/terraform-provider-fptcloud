package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucketCors() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketCorsRead,
		Schema: map[string]*schema.Schema{
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the bucket",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"page_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The number of items to return in each page",
			},
			"page": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The page number",
			},
			"cors_rule": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
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
				Description: "The bucket cors rule",
			},
		},
	}
}

func dataSourceBucketCorsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", d.Get("region_name").(string)))
	}
	bucketName := d.Get("bucket_name").(string)
	page := 1
	if d.Get("page").(int) > 0 {
		page = d.Get("page").(int)
	}
	pageSize := 25
	if d.Get("page_size").(int) > 0 {
		pageSize = d.Get("page_size").(int)
	}
	corsRule, err := service.GetBucketCors(vpcId, s3ServiceDetail.S3ServiceId, bucketName, page, pageSize)
	if err != nil {
		return diag.FromErr(err)
	}

	if corsRule.Total == 0 {
		return diag.Errorf("bucket %s does not have cors rule", bucketName)
	}
	var formattedData []interface{}
	for _, rule := range corsRule.CorsRules {
		formattedData = append(formattedData, map[string]interface{}{
			"id":              rule.ID,
			"allowed_headers": rule.AllowedHeaders,
			"allowed_methods": rule.AllowedMethods,
			"allowed_origins": rule.AllowedOrigins,
			"expose_headers":  rule.ExposeHeaders,
			"max_age_seconds": rule.MaxAgeSeconds,
		})
	}
	d.SetId(bucketName)
	if err := d.Set("cors_rule", formattedData); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	return nil

}
