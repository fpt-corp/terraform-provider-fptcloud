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
				ForceNew:    true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The name of the instance group",
				ForceNew:     true,
			},
			"policy_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The policy of the instance group",
				ForceNew:     true,
			},
			"vm_ids": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of instances in the instance group",
				Elem:        &schema.Schema{Type: schema.TypeString},
				ForceNew:    true,
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
		DeleteContext: resourceInstanceGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
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
		createModel.VmIds = vmIds.([]string)
	}

	isSuccess, err := service.CreateInstanceGroup(createModel)
	if err != nil || !isSuccess {
		return diag.Errorf("[ERR] Failed to create a new instance group: %s", err)
	}

	var setError error
	d.SetId("")
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
		Timeout:        5 * time.Minute,
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
		if result == nil {
			d.SetId("")
			return nil
		}
		return diag.Errorf("[ERR] Failed retrieving the instance group: %s", err)
	}
	if result == nil || len(*result) == 0 {
		d.SetId("")
		return nil
	}

	var setError error
	data := (*result)[0]

	d.SetId(data.ID)
	setError = d.Set("name", data.Name)
	setError = d.Set("policy", data.Policy)
	setError = d.Set("vms", data.Vms)
	setError = d.Set("vpc_id", data.VpcId)
	setError = d.Set("created_at", data.CreatedAt)

	if setError != nil {
		return diag.Errorf("[ERR] Instance group could not be found")
	}

	return nil
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
