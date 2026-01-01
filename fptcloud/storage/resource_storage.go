package fptcloud_storage

import (
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	"strings"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
	"time"
)

// ResourceStorage function returns a schema.Resource that represents a storage.
// This can be used to create, read, update, and delete operations for a storage in the infrastructure.
func ResourceStorage() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a Fpt cloud storage which can be attached to an instance in order to provide expanded storage.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vpc id of the storage",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the storage",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The name of the storage",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The type of the storage (EXTERNAL | LOCAL)",
			},
			"size_gb": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "The size of the storage (in GB)",
			},
			"storage_policy_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The policy id of the storage",
			},
			"instance_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The instance attached the storage (require if storage type is local)",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The created at of the storage",
			},
			"tag_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of tag IDs associated with the storage",
			},
		},
		CreateContext: resourceStorageCreate,
		ReadContext:   resourceStorageRead,
		UpdateContext: resourceStorageUpdate,
		DeleteContext: resourceStorageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

// function to create the new Storage
func resourceStorageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	storageService := NewStorageService(apiClient)

	storageModel := StorageDTO{}
	vpcId, okVpcId := d.GetOk("vpc_id")
	storageType, okStorageType := d.GetOk("type")
	instanceId, okInstanceId := d.GetOk("instance_id")

	if name, ok := d.GetOk("name"); ok {
		storageModel.Name = name.(string)
	}

	if size, ok := d.GetOk("size_gb"); ok {
		storageModel.SizeGb = size.(int)
	}

	if storagePolicyId, ok := d.GetOk("storage_policy_id"); ok {
		storageModel.StoragePolicyId = storagePolicyId.(string)
	}

	if okVpcId {
		storageModel.VpcId = vpcId.(string)
	}

	if okStorageType {
		storageModel.Type = storageType.(string)
	}

	if !(storageModel.Type == Local || storageModel.Type == External) {
		return diag.Errorf("[ERR] Storage type %s not supported", storageModel.Type)
	}

	if okInstanceId {
		instanceIdValue := instanceId.(string)
		storageModel.InstanceId = &instanceIdValue
	}

	if tags, ok := d.GetOk("tag_ids"); ok {
		storageModel.TagIds = utils.ExpandTagIDs(tags.(*schema.Set))
	}

	if storageType == Local && !okInstanceId {
		return diag.Errorf("[ERR] Instance id is required with storage type LOCAL")
	}

	storageId, err := storageService.CreateStorage(storageModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to create storage: %s", err)
	}

	d.SetId(storageId)
	setError := d.Set("vpc_id", vpcId.(string))
	if setError != nil {
		return diag.Errorf("[ERR] Failed to create storage")
	}

	//Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"DISABLE", "PENDING", "DISABLED"},
		Target:  []string{"ENABLED"},
		Refresh: func() (interface{}, string, error) {
			findStorageModel := FindStorageDTO{
				ID:    storageId,
				VpcId: vpcId.(string),
			}
			resp, err := storageService.FindStorage(findStorageModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}
			return resp, resp.Status, nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for storage (%s) to be created: %s", d.Id(), err)
	}

	return resourceStorageRead(ctx, d, m)
}

// function to read the Storage
func resourceStorageRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	storageService := NewStorageService(apiClient)

	log.Printf("[INFO] Retrieving the storage %s", d.Id())

	findStorageModel := FindStorageDTO{}
	findStorageModel.ID = d.Id()

	vpcId, ok := d.GetOk("vpc_id")
	if !ok {
		return diag.Errorf("[ERR] vpc_id is required but not found in state. This may indicate a state corruption issue.")
	}
	vpcIdStr := vpcId.(string)
	if vpcIdStr == "" {
		return diag.Errorf("[ERR] vpc_id is required but is empty in state")
	}
	findStorageModel.VpcId = vpcIdStr

	foundStorage, err := storageService.FindStorage(findStorageModel)
	if err != nil {
		if errors.Is(err, common.TimeoutError) {
			log.Printf("[WARN] Timeout while retrieving storage %s, keeping in state for retry", d.Id())
			return diag.Errorf("[ERR] timeout while retrieving the storage: %s", err)
		}

		var httpErr common.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.Code == 404 {
				log.Printf("[WARN] Storage %s not found (404), keeping in state for retry", d.Id())
				return diag.Errorf("[ERR] Storage not found (404). Please try again later: %s", err)
			}
			log.Printf("[WARN] HTTP error %d while retrieving storage %s, keeping in state", httpErr.Code, d.Id())
			return diag.Errorf("[ERR] HTTP error while retrieving the storage: %s", err)
		}

		if errors.Is(err, common.ZeroMatchesError) {
			log.Printf("[WARN] Storage %s not found (ZeroMatchesError), keeping in state for retry", d.Id())
			return diag.Errorf("[ERR] Storage not found. Please try again later: %s", err)
		}

		errStr := err.Error()
		if strings.Contains(errStr, "ZeroMatchesError") ||
			(strings.Contains(errStr, "not found") && !strings.Contains(errStr, "timeout")) {
			log.Printf("[WARN] Storage %s not found (%s), keeping in state for retry", d.Id(), errStr)
			return diag.Errorf("[ERR] Storage not found. Please try again later: %s", err)
		}

		log.Printf("[WARN] Error retrieving storage %s: %s. Keeping in state for retry.", d.Id(), err)
		return diag.Errorf("[ERR] failed retrieving the storage: %s", err)
	}

	if foundStorage == nil {
		log.Printf("[WARN] FindStorage returned nil for storage %s, but no error. Keeping in state for retry.", d.Id())
		return diag.Errorf("[ERR] Storage API returned empty response. Please try again later: storage %s", d.Id())
	}

	expectedId := d.Id()
	if foundStorage.ID == "" {
		log.Printf("[WARN] Storage %s returned empty ID, keeping in state for retry", expectedId)
		return diag.Errorf("[ERR] Storage returned empty ID. Please try again later: storage %s", d.Id())
	}

	if foundStorage.ID != expectedId {
		log.Printf("[ERR] Storage ID mismatch: expected %s, got %s. This may indicate API query issue.", expectedId, foundStorage.ID)
		return diag.Errorf("[ERR] storage ID mismatch: expected %s but API returned %s. This may indicate a query parameter issue.", expectedId, foundStorage.ID)
	}

	if foundStorage.Status != "ENABLED" {
		log.Printf("[WARN] Storage %s status is %s (not ENABLED), keeping in state", d.Id(), foundStorage.Status)
	}

	d.SetId(foundStorage.ID)

	if err := d.Set("name", foundStorage.Name); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("size_gb", foundStorage.SizeGb); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("storage_policy_id", foundStorage.StoragePolicyId); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("type", foundStorage.Type); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("instance_id", foundStorage.InstanceId); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("vpc_id", foundStorage.VpcId); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("created_at", foundStorage.CreatedAt); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("tag_ids", foundStorage.TagIds); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// function to update the Storage
func resourceStorageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	storageService := NewStorageService(apiClient)

	if d.HasChange("type") {
		return diag.Errorf("[ERR] Storage type can not be changed")
	}

	updateStorageModel := UpdateStorageDTO{}

	vpcId := d.Get("vpc_id").(string)
	hasChangedSize := d.HasChange("size_gb")
	hasChangedStoragePolicy := d.HasChange("storage_policy_id")
	hasChangedName := d.HasChange("name")
	hasChangeAttachedInstance := d.HasChange("instance_id")
	hasChangeTags := d.HasChange("tag_ids")

	if hasChangedSize || hasChangedName || hasChangedStoragePolicy {
		updateStorageModel.Name = d.Get("name").(string)
		updateStorageModel.SizeGb = d.Get("size_gb").(int)
		updateStorageModel.StoragePolicyId = d.Get("storage_policy_id").(string)
		_, err := storageService.UpdateStorage(vpcId, d.Id(), updateStorageModel)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while update storage %s", err)
		}
	}

	if hasChangeAttachedInstance {
		instanceId := d.Get("instance_id")
		storageType := d.Get("type")
		if storageType == Local {
			return diag.Errorf("[ERR] Can not update attached when storage type is LOCAL")
		}

		var instanceIdStr *string
		if instanceId != nil {
			if id, ok := instanceId.(string); ok && id != "" {
				instanceIdStr = &id
			}
		}

		_, err := storageService.UpdateAttachedInstance(vpcId, d.Id(), instanceIdStr)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while change attached instance from storage %s", d.Id())
		}
	}

	if hasChangeTags {
		tagIds := utils.ExpandTagIDs(d.Get("tag_ids").(*schema.Set))
		_, err := storageService.UpdateTags(vpcId, d.Id(), tagIds)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while updating storage tags %s", err)
		}
	}

	//Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"DISABLE", "PENDING", "UPDATING"},
		Target:  []string{"ENABLED"},
		Refresh: func() (interface{}, string, error) {
			findStorageModel := FindStorageDTO{
				ID:    d.Id(),
				VpcId: vpcId,
			}
			resp, err := storageService.FindStorage(findStorageModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}
			return resp, resp.Status, nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err := createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for storage (%s) to be updated: %s", d.Id(), err)
	}
	return resourceStorageRead(ctx, d, m)
}

// function to delete the Storage
func resourceStorageDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	storageService := NewStorageService(apiClient)

	log.Printf("[INFO] Deleting the volume %s", d.Id())

	vpcId, okVpcId := d.GetOk("vpc_id")

	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	_, err := storageService.DeleteStorage(vpcId.(string), d.Id())

	if err != nil {
		return diag.Errorf("[ERR] An error occurred while trying to delete the storage %s", err)
	}
	return nil
}
