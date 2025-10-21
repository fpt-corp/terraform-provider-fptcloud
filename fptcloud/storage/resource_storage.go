package fptcloud_storage

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	common "terraform-provider-fptcloud/commons"
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

	if storageModel.Type != Local && storageModel.Type != External {
		return diag.Errorf("[ERR] Storage type %s not supported", storageModel.Type)
	}

	if okInstanceId {
		instanceIdValue := instanceId.(string)
		storageModel.InstanceId = &instanceIdValue
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
	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findStorageModel.VpcId = vpcId.(string)
	}

	foundStorage, err := storageService.FindStorage(findStorageModel)
	if err != nil {
		if foundStorage == nil {
			d.SetId("")
			return nil
		}
		return diag.Errorf("[ERR] failed retrieving the storage: %s", err)
	}
	if foundStorage.Status != "ENABLED" {
		d.SetId("")
		return nil
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
