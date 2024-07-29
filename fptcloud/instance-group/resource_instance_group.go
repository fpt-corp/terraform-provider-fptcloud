package fptcloud_instance_group

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

// ResourceInstanceGroup function returns a schema.Resource that represents an instance group.
// This can be used to create, read, update, and delete operations for an instance group in the infrastructure.
func ResourceInstanceGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a Fpt cloud instance group which can be attached to an instance in order to provide expanded instance group.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vpc id of the instance group",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the instance group",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The name of the instance group",
			},
			"policy_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The policy of the instance group",
			},
			"vm_ids": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of instances in the instance group",
			},
		},
		CreateContext: resourceInstanceGroupCreate,
		ReadContext:   resourceInstanceGroupRead,
		DeleteContext: resourceInstanceGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

// function to create a new instance group
func resourceInstanceGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	instanceGroupService := NewInstanceGroupService(apiClient)

	createModel := CreateInstanceGroupDTO{}
	vpcId, okVpcId := d.GetOk("vpc_id")
	name, okName := d.GetOk("name")

	if okVpcId {
		createModel.VpcId = vpcId.(string)
	}
	if okName {
		createModel.Name = name.(string)
	}
	if policyId, ok := d.GetOk("policy_id"); ok {
		createModel.PolicyId = policyId.(string)
	}
	if vmIds, ok := d.GetOk("vm_ids"); ok {
		createModel.VmIds = vmIds.([]string)
	}

	isSuccess, err := instanceGroupService.CreateInstanceGroup(createModel)
	if err != nil || !isSuccess {
		return diag.Errorf("[ERR] Failed to create a new instance group: %s", err)
	}

	var setError error
	setError = d.Set("vpc_id", vpcId)
	setError = d.Set("policy_name", name)
	if setError != nil {
		return diag.Errorf("[ERR] Failed to create a new instance group")
	}

	//Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"PENDING"},
		Target:  []string{"COMPLETE"},
		Refresh: func() (interface{}, string, error) {
			findStorageModel := FindInstanceGroupDTO{
				Name:  name.(string),
				VpcId: vpcId.(string),
			}
			resp, err := instanceGroupService.FindInstanceGroup(findStorageModel)
			if err != nil || resp == nil || resp.ID != "" {
				return nil, "PENDING", common.DecodeError(err)
			}
			return resp, "COMPLETE", nil
		},
		Timeout:        5 * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for instance group (%s) to be created: %s", d.Id(), err)
	}

	return resourceInstanceGroupRead(ctx, d, m)
}

// function to read the instance group
func resourceInstanceGroupRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	var setError error
	d.SetId(foundStorage.ID)
	setError = d.Set("name", foundStorage.Name)
	setError = d.Set("size_gb", foundStorage.SizeGb)
	setError = d.Set("storage_policy_id", foundStorage.StoragePolicyId)
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
