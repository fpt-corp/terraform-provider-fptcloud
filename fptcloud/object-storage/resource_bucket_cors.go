package fptcloud_object_storage

import (
	"context"
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceBucketCors() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketCorsCreate,
		UpdateContext: nil,
		DeleteContext: resourceBucketCorsDelete,
		ReadContext:   resourceBucketCorsRead,
		Schema: map[string]*schema.Schema{
			"cors_config": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "The bucket lifecycle rule in JSON format, support only one rule",
				ConflictsWith: []string{"cors_config_file"},
				ValidateFunc:  validation.StringIsJSON,
			},
			"cors_config_file": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "Path to the JSON file containing the bucket lifecycle rule, support only one rule",
				ConflictsWith: []string{"cors_config"},
			},
			"status": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Status after bucket cors rule is created",
			},
			"bucket_cors_rules": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
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
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
		},
	}
}

func resourceBucketCorsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	bucketName := d.Get("bucket_name").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := GetServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	var corsConfigData string
	if v, ok := d.GetOk("cors_config"); ok {
		corsConfigData = v.(string)
	} else if v, ok := d.GetOk("cors_config_file"); ok {
		// The actual file reading is handled by Terraform's built-in file() function
		// in the configuration, so we just get the content here
		corsConfigData = v.(string)
	} else {
		return diag.FromErr(fmt.Errorf("either 'cors_config' or 'cors_config_file' must be specified"))
	}
	var jsonMap CorsRule
	err := json.Unmarshal([]byte(corsConfigData), &jsonMap)
	if err != nil {
		return diag.FromErr(err)
	}
	payload := map[string]interface{}{
		"AllowedMethods": jsonMap.AllowedMethods,
		"MaxAgeSeconds":  jsonMap.MaxAgeSeconds,
		"ID":             jsonMap.ID,
		"AllowedOrigins": jsonMap.AllowedOrigins,
	}
	if len(jsonMap.AllowedHeaders) > 0 {
		payload["AllowedHeaders"] = jsonMap.AllowedHeaders
	}
	if len(jsonMap.ExposeHeaders) > 0 {
		payload["ExposeHeaders"] = jsonMap.ExposeHeaders
	}
	r := service.CreateBucketCors(vpcId, s3ServiceDetail.S3ServiceId, bucketName, payload)
	if !r.Status {
		if err := d.Set("status", false); err != nil {
			return diag.FromErr(err)
		}
		return diag.FromErr(fmt.Errorf("%s", r.Message))
	}
	d.SetId(bucketName)
	if err := d.Set("status", true); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
func resourceBucketCorsRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := GetServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}
	page := 1
	pageSize := maxPageSize

	bucketCorsDetails, _ := service.GetBucketCors(vpcId, s3ServiceDetail.S3ServiceId, bucketName, page, pageSize)
	if !bucketCorsDetails.Status {
		return diag.FromErr(fmt.Errorf("failed to fetch life cycle rules for bucket %s", bucketName))
	}
	d.SetId(bucketName)
	var formattedData []interface{}
	if bucketCorsDetails.Total == 0 {
		if err := d.Set("bucket_cors_rules", make([]interface{}, 0)); err != nil {
			d.SetId("")
			return diag.FromErr(err)
		}
	}
	for _, corsRuleDetail := range bucketCorsDetails.CorsRules {
		data := map[string]interface{}{
			"id": corsRuleDetail.ID,
		}
		formattedData = append(formattedData, data)
	}
	if err := d.Set("bucket_cors_rules", formattedData); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	return nil
}

func resourceBucketCorsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := GetServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}
	var corsConfigData string
	if v, ok := d.GetOk("cors_config"); ok {
		corsConfigData = v.(string)
	} else if v, ok := d.GetOk("cors_config_file"); ok {
		// The actual file reading is handled by Terraform's built-in file() function
		// in the configuration, so we just get the content here
		corsConfigData = v.(string)
	} else {
		return diag.FromErr(fmt.Errorf("either 'cors_config' or 'cors_config_file' must be specified"))
	}
	var jsonMap []CorsRule
	err := json.Unmarshal([]byte(corsConfigData), &jsonMap)
	if err != nil {
		return diag.FromErr(err)
	}
	var payload []map[string]interface{}
	for _, corsRule := range jsonMap {
		payload := map[string]interface{}{
			"AllowedOrigins": corsRule.AllowedOrigins,
			"AllowedMethods": corsRule.AllowedMethods,
			"ID":             corsRule.ID,
			"MaxAgeSeconds":  corsRule.MaxAgeSeconds,
		}
		if len(corsRule.AllowedHeaders) > 0 {
			payload["AllowedHeaders"] = corsRule.AllowedHeaders
		}
		if len(corsRule.ExposeHeaders) > 0 {
			payload["ExposeHeaders"] = corsRule.ExposeHeaders
		}
	}
	r := service.UpdateBucketCors(vpcId, s3ServiceDetail.S3ServiceId, bucketName, payload)
	if !r.Status {
		if err := d.Set("status", false); err != nil {
			return diag.FromErr(err)
		}
		return diag.FromErr(fmt.Errorf("%s", r.Message))
	}
	d.SetId(bucketName)
	if err := d.Set("status", true); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
