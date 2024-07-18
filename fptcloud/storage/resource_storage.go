package fptcloud_storage

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	common "terraform-provider-fptcloud/commons"
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

	if !(storageModel.Type == Local || storageModel.Type == External) {
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
	setError := d.Set("vpc_id", vpcId)
	if setError != nil {
		return diag.Errorf("[ERR] Failed to create storage")
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

	var setError error
	d.SetId(foundStorage.ID)
	setError = d.Set("name", foundStorage.Name)
	setError = d.Set("size_gb", foundStorage.SizeGb)
	setError = d.Set("storage_policy", foundStorage.StoragePolicy)
	setError = d.Set("type", foundStorage.Type)
	setError = d.Set("instance_id", foundStorage.InstanceId)
	setError = d.Set("vpc_id", foundStorage.VpcId)
	setError = d.Set("created_at", foundStorage.CreatedAt)

	if setError != nil {
		return diag.Errorf("[ERR] Storage could not be found")
	}

	return nil
}

// function to update the Storage
func resourceStorageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
