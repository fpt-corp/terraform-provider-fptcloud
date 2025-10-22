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

func ResourceBucketLifeCycle() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketLifeCycleCreate,
		UpdateContext: nil,
		DeleteContext: resourceBucketLifeCycleDelete,
		ReadContext:   resourceBucketLifeCycleRead,
		CustomizeDiff: customizeBucketLifecycleDiff,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC ID",
			},
			"state": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "State after bucket lifecycle rule is created",
			},
			"rules": {
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
			}, "bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"life_cycle_rule": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "The bucket lifecycle rule in JSON format, support only one rule",
				ConflictsWith: []string{"life_cycle_rule_file"},
				ValidateFunc:  validation.StringIsJSON,
			},
			"life_cycle_rule_file": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "Path to the JSON file containing the bucket lifecycle rule, support only one rule",
				ConflictsWith: []string{"life_cycle_rule"},
			},
		},
	}
}

// customizeBucketLifecycleDiff performs plan-time validation for lifecycle rule JSON
// Ensures a valid JSON is provided and contains a non-empty ID. Also validates
// Expiration fields do not conflict (Days vs ExpiredObjectDeleteMarker).
func customizeBucketLifecycleDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	var lifecycleRuleContent string
	if v, ok := d.GetOk("life_cycle_rule"); ok {
		lifecycleRuleContent = v.(string)
	} else if v, ok := d.GetOk("life_cycle_rule_file"); ok {
		lifecycleRuleContent = v.(string)
	} else {
		// Nothing to validate if neither is provided (Terraform may compute later)
		return nil
	}

	// Parse JSON into typed struct for stronger validation
	jsonMap, err := parseLifeCycleData(lifecycleRuleContent)
	if err != nil {
		return fmt.Errorf("life_cycle_rule must be valid JSON: %w", err)
	}
	if jsonMap.ID == "" {
		return fmt.Errorf("life_cycle_rule must include non-empty ID")
	}
	if jsonMap.Expiration.Days != 0 && jsonMap.Expiration.ExpiredObjectDeleteMarker {
		return fmt.Errorf("Expiration.Days and Expiration.ExpiredObjectDeleteMarker cannot be set at the same time")
	}

	return nil
}
func parseLifeCycleData(lifeCycleData string) (S3BucketLifecycleConfig, error) {
	var jsonMap S3BucketLifecycleConfig
	err := json.Unmarshal([]byte(lifeCycleData), &jsonMap)
	if err != nil {
		return S3BucketLifecycleConfig{}, err
	}
	return jsonMap, nil
}
func resourceBucketLifeCycleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	bucketName := d.Get("bucket_name").(string)
	regionName := d.Get("region_name").(string)
	vpcId := d.Get("vpc_id").(string)

	var lifecycleRuleContent string
	if v, ok := d.GetOk("life_cycle_rule"); ok {
		lifecycleRuleContent = v.(string)
	} else if v, ok := d.GetOk("life_cycle_rule_file"); ok {
		// The actual file reading is handled by Terraform's built-in file() function
		// in the configuration, so we just get the content here
		lifecycleRuleContent = v.(string)
	} else {
		return diag.FromErr(fmt.Errorf("either 'life_cycle_rule' or 'life_cycle_rule_file' must be specified"))
	}
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}
	jsonMap, err := parseLifeCycleData(lifecycleRuleContent)
	if err != nil {
		return diag.FromErr(err)
	}
	payload := map[string]interface{}{
		"ID":                             jsonMap.ID,
		"NoncurrentVersionExpiration":    map[string]interface{}{"NoncurrentDays": jsonMap.NoncurrentVersionExpiration.NoncurrentDays},
		"AbortIncompleteMultipartUpload": map[string]interface{}{"DaysAfterInitiation": jsonMap.AbortIncompleteMultipartUpload.DaysAfterInitiation},
		"Filter":                         map[string]interface{}{"Prefix": jsonMap.Filter.Prefix},
	}
	if jsonMap.Expiration.Days != 0 && jsonMap.Expiration.ExpiredObjectDeleteMarker {
		return diag.FromErr(fmt.Errorf("Expiration.Days and Expiration.ExpiredObjectDeleteMarker cannot be set at the same time"))
	}
	if jsonMap.Expiration.Days != 0 {
		payload["Expiration"] = map[string]interface{}{"Days": jsonMap.Expiration.Days}
	}
	if jsonMap.Expiration.ExpiredObjectDeleteMarker {
		payload["Expiration"] = map[string]interface{}{"ExpiredObjectDeleteMarker": jsonMap.Expiration.ExpiredObjectDeleteMarker}
	}
	r := service.PutBucketLifecycle(vpcId, s3ServiceDetail.S3ServiceId, bucketName, payload)
	d.SetId(bucketName)
	if !r.Status {
		if err := d.Set("state", false); err != nil {
			d.SetId("")
			return diag.FromErr(err)
		}
		return diag.FromErr(fmt.Errorf("%s", r.Message))
	}
	if err := d.Set("state", true); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}

	return nil
}
func resourceBucketLifeCycleRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
	pageSize := 999999

	lifeCycleResponse := service.GetBucketLifecycle(vpcId, s3ServiceDetail.S3ServiceId, bucketName, page, pageSize)
	if !lifeCycleResponse.Status {
		return diag.FromErr(fmt.Errorf("failed to fetch life cycle rules for bucket %s", bucketName))
	}
	d.SetId(bucketName)
	var formattedData []interface{}
	if lifeCycleResponse.Total == 0 {
		if err := d.Set("rules", make([]interface{}, 0)); err != nil {
			d.SetId("")
			return diag.FromErr(err)
		}
	}
	for _, lifecycleRule := range lifeCycleResponse.Rules {
		data := map[string]interface{}{
			"id": lifecycleRule.ID,
		}
		formattedData = append(formattedData, data)
	}

	if err := d.Set("rules", formattedData); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	return nil
}
func resourceBucketLifeCycleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	bucketName := d.Get("bucket_name").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}
	var lifecycleRuleContent string
	if v, ok := d.GetOk("life_cycle_rule"); ok {
		lifecycleRuleContent = v.(string)
	} else if v, ok := d.GetOk("life_cycle_rule_file"); ok {
		// The actual file reading is handled by Terraform's built-in file() function
		// in the configuration, so we just get the content here
		lifecycleRuleContent = v.(string)
	} else {
		return diag.FromErr(fmt.Errorf("either 'life_cycle_rule' or 'life_cycle_rule_file' must be specified"))
	}
	jsonMap, err := parseLifeCycleData(lifecycleRuleContent)
	if err != nil {
		return diag.FromErr(err)
	}

	payload := map[string]interface{}{
		"AbortIncompleteMultipartUpload": map[string]interface{}{"DaysAfterInitiation": jsonMap.AbortIncompleteMultipartUpload.DaysAfterInitiation},
		"Status":                         "Enabled",
		"ID":                             jsonMap.ID,
		"OrgID":                          jsonMap.ID, // Portal need both ID and OrgID
		"Filter":                         map[string]interface{}{"Prefix": jsonMap.Filter.Prefix},
		"NoncurrentVersionExpiration":    map[string]interface{}{"NoncurrentDays": jsonMap.NoncurrentVersionExpiration.NoncurrentDays},
	}
	if jsonMap.Expiration.Days != 0 && jsonMap.Expiration.ExpiredObjectDeleteMarker {
		return diag.FromErr(fmt.Errorf("Expiration.Days and Expiration.ExpiredObjectDeleteMarker cannot be set at the same time"))
	}
	if jsonMap.Expiration.Days != 0 {
		payload["Expiration"] = map[string]interface{}{"Days": jsonMap.Expiration.Days}
	}
	if jsonMap.Expiration.ExpiredObjectDeleteMarker {
		payload["Expiration"] = map[string]interface{}{"ExpiredObjectDeleteMarker": jsonMap.Expiration.ExpiredObjectDeleteMarker}
	}
	r := service.DeleteBucketLifecycle(vpcId, s3ServiceDetail.S3ServiceId, bucketName, payload)
	if !r.Status {
		if err := d.Set("state", false); err != nil {
			return diag.FromErr(err)
		}
		return diag.FromErr(fmt.Errorf("%s", r.Message))
	}
	d.SetId(bucketName)
	if err := d.Set("state", true); err != nil {
		return diag.FromErr(err)
	}
	return resourceBucketLifeCycleRead(ctx, d, m)
}
