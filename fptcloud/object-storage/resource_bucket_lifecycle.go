package fptcloud_object_storage

import (
	"context"
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
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
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Description:      "The bucket lifecycle rule in JSON format, support only one rule",
				ConflictsWith:    []string{"life_cycle_rule_file"},
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: structure.SuppressJsonDiff,
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
	if jsonMap.Status != "" && jsonMap.Status != lifeCycleStatusEnabled && jsonMap.Status != lifeCycleStatusDisabled {
		return fmt.Errorf("Status must be %q or %q, got %q", lifeCycleStatusEnabled, lifeCycleStatusDisabled, jsonMap.Status)
	}
	if jsonMap.Expiration != nil && jsonMap.Expiration.Days != 0 && jsonMap.Expiration.ExpiredObjectDeleteMarker {
		return fmt.Errorf("Expiration.Days and Expiration.ExpiredObjectDeleteMarker cannot be set at the same time")
	}

	return nil
}

const (
	lifeCycleStatusEnabled  = "Enabled"
	lifeCycleStatusDisabled = "Disabled"
)

// lifeCycleRulePayload builds the API request body, carrying only the fields the
// rule actually declares. An omitted object is left out entirely rather than
// sent as a zero value, which the API rejects with InvalidArgument.
func lifeCycleRulePayload(jsonMap S3BucketLifecycleConfig) map[string]interface{} {
	payload := map[string]interface{}{"ID": jsonMap.ID}
	if jsonMap.Status != "" {
		payload["Status"] = jsonMap.Status
	}
	if jsonMap.Filter != nil {
		payload["Filter"] = map[string]interface{}{"Prefix": jsonMap.Filter.Prefix}
	}
	if jsonMap.NoncurrentVersionExpiration != nil {
		payload["NoncurrentVersionExpiration"] = map[string]interface{}{"NoncurrentDays": jsonMap.NoncurrentVersionExpiration.NoncurrentDays}
	}
	if jsonMap.AbortIncompleteMultipartUpload != nil {
		payload["AbortIncompleteMultipartUpload"] = map[string]interface{}{"DaysAfterInitiation": jsonMap.AbortIncompleteMultipartUpload.DaysAfterInitiation}
	}
	if jsonMap.Expiration != nil {
		if jsonMap.Expiration.Days != 0 {
			payload["Expiration"] = map[string]interface{}{"Days": jsonMap.Expiration.Days}
		} else if jsonMap.Expiration.ExpiredObjectDeleteMarker {
			payload["Expiration"] = map[string]interface{}{"ExpiredObjectDeleteMarker": true}
		}
	}
	return payload
}

// lifeCycleRuleID returns the ID of the single rule this resource manages, read
// from whichever of the two config fields is set.
func lifeCycleRuleID(d *schema.ResourceData) (string, error) {
	var lifecycleRuleContent string
	if v, ok := d.GetOk("life_cycle_rule"); ok {
		lifecycleRuleContent = v.(string)
	} else if v, ok := d.GetOk("life_cycle_rule_file"); ok {
		lifecycleRuleContent = v.(string)
	} else {
		return "", fmt.Errorf("either 'life_cycle_rule' or 'life_cycle_rule_file' must be specified")
	}
	jsonMap, err := parseLifeCycleData(lifecycleRuleContent)
	if err != nil {
		return "", err
	}
	return jsonMap.ID, nil
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
	if jsonMap.Expiration != nil && jsonMap.Expiration.Days != 0 && jsonMap.Expiration.ExpiredObjectDeleteMarker {
		return diag.FromErr(fmt.Errorf("Expiration.Days and Expiration.ExpiredObjectDeleteMarker cannot be set at the same time"))
	}
	payload := lifeCycleRulePayload(jsonMap)
	r := service.PutBucketLifecycle(vpcId, s3ServiceDetail.S3ServiceId, bucketName, payload)
	// The ID is only claimed once the rule is known to exist. Returning an error
	// with an ID set makes Terraform record the resource as tainted, so the next
	// apply or destroy issues a delete for a rule that was never created and the
	// backend answers 404.
	if !r.Status {
		switch reconcileLifeCycleRule(service, vpcId, s3ServiceDetail.S3ServiceId, bucketName, jsonMap) {
		case createAdopted:
			// The rule is on the bucket after all: an earlier attempt committed and
			// only its response was lost. Record it instead of failing forever.
		case createConflict:
			return diag.Errorf("lifecycle rule %q already exists on bucket %s with different settings: %s", jsonMap.ID, bucketName, r.Message)
		default:
			if err := d.Set("state", false); err != nil {
				return diag.FromErr(err)
			}
			return diag.FromErr(fmt.Errorf("%s", r.Message))
		}
	}
	d.SetId(bucketName)
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
	ruleID, err := lifeCycleRuleID(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// The resource owns one rule, so it is gone once its own ID is absent from the
	// bucket's rule set - which is what happens when a rule is removed out of band
	// or lost to a concurrent write. Clearing the ID lets Terraform plan a
	// recreate; leaving it set strands the resource in a state no refresh can
	// reconcile and makes the eventual destroy fail with a 404.
	var formattedData []interface{}
	found := false
	for _, lifecycleRule := range lifeCycleResponse.Rules {
		if lifecycleRule.ID == ruleID {
			found = true
		}
		data := map[string]interface{}{
			"id": lifecycleRule.ID,
		}
		formattedData = append(formattedData, data)
	}
	if !found {
		d.SetId("")
		return nil
	}

	d.SetId(bucketName)
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

	if jsonMap.Expiration != nil && jsonMap.Expiration.Days != 0 && jsonMap.Expiration.ExpiredObjectDeleteMarker {
		return diag.FromErr(fmt.Errorf("Expiration.Days and Expiration.ExpiredObjectDeleteMarker cannot be set at the same time"))
	}
	payload := lifeCycleRulePayload(jsonMap)
	payload["OrgID"] = jsonMap.ID // Portal need both ID and OrgID
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
