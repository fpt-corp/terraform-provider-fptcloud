package fptcloud_instance_group

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	common "terraform-provider-fptcloud/commons"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceInstanceGroup function returns a schema. Resource that represents an instance group.
// This can be used to create, read, update, and delete operations for an instance group in the infrastructure.
func ResourceInstanceGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a FPT cloud instance group that can be attached to an instance in order to provide an expanded instance group.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vpc id of the instance group",
				ForceNew:    true,
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
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The list of instances in the instance group",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"policy": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"policy_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vms": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		CreateContext: resourceInstanceGroupCreate,
		ReadContext:   resourceInstanceGroupRead,
		UpdateContext: resourceInstanceGroupUpdate,
		DeleteContext: resourceInstanceGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceInstanceGroupImportState,
		},
	}
}

// function to create a new instance group
func resourceInstanceGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewInstanceGroupService(apiClient)

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
		vmIdsSet := vmIds.(*schema.Set)
		vmIdsList := make([]string, 0, len(vmIdsSet.List()))
		for _, v := range vmIdsSet.List() {
			vmIdsList = append(vmIdsList, v.(string))
		}
		createModel.VmIds = vmIdsList
	}

	isSuccess, err := service.CreateInstanceGroup(createModel)
	if err != nil || !isSuccess {
		return diag.Errorf("[ERR] Failed to create a new instance group: %s", err)
	}

	d.SetId("")

	if err := d.Set("vpc_id", vpcId); err != nil {
		return diag.Errorf("[ERR] Failed to set 'vpc_id': %s", err)
	}

	if err := d.Set("policy_name", name); err != nil {
		return diag.Errorf("[ERR] Failed to set 'policy_name': %s", err)
	}

	//Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"PENDING"},
		Target:  []string{"COMPLETE"},
		Refresh: func() (interface{}, string, error) {
			findModel := FindInstanceGroupDTO{
				Name:  name.(string),
				VpcId: vpcId.(string),
			}
			resp, err := service.FindInstanceGroup(findModel)
			if err != nil || len(*resp) == 0 {
				return nil, "", common.DecodeError(err)
			}

			rsInstanceGroup := (*resp)[0]
			d.SetId(rsInstanceGroup.ID)
			return (*resp)[0], "COMPLETE", nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     30 * time.Second,
		NotFoundChecks: 20,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for instance group (%s) to be created: %s", d.Id(), err)
	}

	return resourceInstanceGroupRead(ctx, d, m)
}

// resourceInstanceGroupImportState supports import id format vpc_id/instance_group_id.
func resourceInstanceGroupImportState(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("[ERR] Invalid import format: expected format vpc_id/instance_group_id, got %q", d.Id())
	}
	d.Set("vpc_id", parts[0])
	d.SetId(parts[1])
	return []*schema.ResourceData{d}, nil
}

// function to read the instance group
func resourceInstanceGroupRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewInstanceGroupService(apiClient)

	log.Printf("[INFO] Retrieving the instance group %s", d.Id())

	findModel := FindInstanceGroupDTO{}
	findModel.ID = d.Id()
	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findModel.VpcId = vpcId.(string)
	}
	if policyName, ok := d.GetOk("name"); ok {
		findModel.Name = policyName.(string)
	}

	result, err := service.FindInstanceGroup(findModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed retrieving the instance group (%s): %s", d.Id(), err)
	}

	if result == nil || len(*result) == 0 {
		return nil
	}

	data := (*result)[0]

	d.SetId(data.ID)

	if err := d.Set("name", data.Name); err != nil {
		return diag.Errorf("[ERR] Failed to set 'name': %s", err)
	}

	// Schema "policy" is TypeString; API returns object or slice — convert to string before set
	if data.Policy != nil {
		if s, ok := data.Policy.(string); ok {
			_ = d.Set("policy", s)
		} else if b, err := json.Marshal(data.Policy); err == nil {
			_ = d.Set("policy", string(b))
		}
	}

	// Schema "vms" is TypeString; API returns list — convert to string before set
	if data.Vms != nil {
		if b, err := json.Marshal(data.Vms); err == nil {
			_ = d.Set("vms", string(b))
		}
	}

	if data.VpcId != "" {
		if err := d.Set("vpc_id", data.VpcId); err != nil {
			return diag.Errorf("[ERR] Failed to set 'vpc_id': %s", err)
		}
	}

	if err := d.Set("created_at", data.CreatedAt); err != nil {
		return diag.Errorf("[ERR] Failed to set 'created_at': %s", err)
	}

	if data.Policy != nil {
		if policyMap, ok := data.Policy.(map[string]interface{}); ok && policyMap["id"] != nil {
			_ = d.Set("policy_id", policyMap["id"])
		} else if list, ok := data.Policy.([]interface{}); ok && len(list) > 0 {
			if m, ok := list[0].(map[string]interface{}); ok && m["id"] != nil {
				_ = d.Set("policy_id", m["id"])
			}
		}
	}
	if data.Vms != nil {
		vmIds := make([]interface{}, 0, len(data.Vms))
		for _, v := range data.Vms {
			if m, ok := v.(map[string]interface{}); ok && m["id"] != nil {
				vmIds = append(vmIds, m["id"])
			}
		}
		if err := d.Set("vm_ids", vmIds); err != nil {
			return diag.Errorf("[ERR] Failed to set 'vm_ids': %s", err)
		}
	}

	return nil
}

// function to update the instance group (name, vm_ids)
func resourceInstanceGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewInstanceGroupService(apiClient)

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] vpc_id is required")
	}

	oldName, _ := d.GetChange("name")
	oldVmIds, _ := d.GetChange("vm_ids")

	payload := UpdateInstanceGroupDTO{
		Name:  d.Get("name").(string),
		VmIds: []string{},
	}
	if vmIds, ok := d.GetOk("vm_ids"); ok {
		for _, v := range vmIds.(*schema.Set).List() {
			payload.VmIds = append(payload.VmIds, v.(string))
		}
	}

	ok, err := service.UpdateInstanceGroup(vpcId.(string), d.Id(), payload)
	if err != nil || !ok {
		_ = d.Set("name", oldName)
		if oldSet, ok := oldVmIds.(*schema.Set); ok {
			_ = d.Set("vm_ids", oldSet)
		}
		return diag.Errorf("[ERR] Failed to update instance group: %s", err)
	}

	return resourceInstanceGroupRead(ctx, d, m)
}

// function to delete the instance group
func resourceInstanceGroupDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewInstanceGroupService(apiClient)

	log.Printf("[INFO] Deleting the instance group %s", d.Id())

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	_, err := service.DeleteInstanceGroup(vpcId.(string), d.Id())

	if err != nil {
		return diag.Errorf("[ERR] An error occurred while trying to delete the instance group %s", err)
	}
	return nil
}
