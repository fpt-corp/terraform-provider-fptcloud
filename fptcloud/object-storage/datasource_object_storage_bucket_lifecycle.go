package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucketLifecycle() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketLifecycleRead,
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
				Required:    true,
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
			"life_cycle_rules": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"filter": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"prefix": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"prefix": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"expiration": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"days": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"expired_object_delete_marker": {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},
						"noncurrent_version_expiration": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"noncurrent_days": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"abort_incomplete_multipart_upload": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"days_after_initiation": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
func parseData(lifeCycleResponse BucketLifecycleResponse) []interface{} {
	var formattedData []interface{}

	for _, lifecycleRule := range lifeCycleResponse.Rules {
		data := map[string]interface{}{
			"id":     lifecycleRule.ID,
			"status": lifecycleRule.Status,
			"noncurrent_version_expiration": []interface{}{
				map[string]interface{}{
					"noncurrent_days": lifecycleRule.NoncurrentVersionExpiration.NoncurrentDays,
				},
			},
			"abort_incomplete_multipart_upload": []interface{}{
				map[string]interface{}{
					"days_after_initiation": lifecycleRule.AbortIncompleteMultipartUpload.DaysAfterInitiation,
				},
			},
		}
		// for fully prefix
		if lifecycleRule.Prefix == "" {
			data["prefix"] = lifecycleRule.Prefix
		}
		// for filter
		if lifecycleRule.Filter.Prefix != "" {
			data["filter"] = []interface{}{
				map[string]interface{}{
					"prefix": lifecycleRule.Filter.Prefix,
				},
			}
		}
		if lifecycleRule.Expiration.Days > 0 {
			data["expiration"] = []interface{}{
				map[string]interface{}{
					"days": lifecycleRule.Expiration.Days,
				},
			}
		}
		if lifecycleRule.Expiration.ExpiredObjectDeleteMarker {
			data["expiration"] = []interface{}{
				map[string]interface{}{
					"expired_object_delete_marker": lifecycleRule.Expiration.ExpiredObjectDeleteMarker,
				},
			}
		}
		formattedData = append(formattedData, data)
	}
	return formattedData
}
func dataSourceBucketLifecycleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}
	page := 1
	v, ok := d.GetOk("page")
	if ok {
		page = v.(int)
	}
	pageSize := 25
	v, ok = d.GetOk("page_size")
	if ok {
		pageSize = v.(int)
	}

	lifeCycleResponse := service.GetBucketLifecycle(vpcId, s3ServiceDetail.S3ServiceId, bucketName, page, pageSize)
	if !lifeCycleResponse.Status {
		return diag.FromErr(fmt.Errorf("failed to fetch life cycle rules for bucket %s", bucketName))
	}
	if lifeCycleResponse.Total == 0 {
		if err := d.Set("life_cycle_rules", make([]interface{}, 0)); err != nil {
			d.SetId("")
			return diag.FromErr(err)
		}
	}
	d.SetId(bucketName)
	formattedData := parseData(lifeCycleResponse)
	if err := d.Set("life_cycle_rules", formattedData); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	return nil
}
