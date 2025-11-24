package fptcloud_nic

import (
	"context"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceNic() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a FPT cloud NIC which attached to an instance in order to provide network connectivity.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The vpc id of the NIC",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the NIC",
			},
			"instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The instance id of the NIC",
			},
			"subnet_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The subnet id of the NIC",
			},
			"mac_address": {
				Type:     schema.TypeString,
				Computed: true,
				//Optional:    true,
				Description: "The mac address of the NIC",
			},
			"private_ip": {
				Type:     schema.TypeString,
				Computed: true,
				//Optional:    true,
				Description: "The private ip of the NIC",
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
				//Optional:    true,
				Description: "The status of the NIC",
			},
			"subnet_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the subnet",
			},
			"is_primary": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Whether the NIC is the primary NIC of the instance",
			},
		},
		CreateContext: resourceNicCreate,
		ReadContext:   resourceNicRead,
		UpdateContext: resourceNicUpdate,
		DeleteContext: resourceNicDelete,
	}
}

// func resourceNicImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
//  // Call Read to populate other fields
//  diags := resourceNicRead(ctx, d, m)
//  if diags.HasError() {
//   return nil, fmt.Errorf("failed to read NIC: %v", diags)
//  }

//  return []*schema.ResourceData{d}, nil
// }

func resourceNicCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	nicService := NewNicService(apiClient)

	createdModel := CreateNicDto{}
	vpcId, okVpcId := d.GetOk("vpc_id")
	instanceId, okInstanceId := d.GetOk("instance_id")
	subnetId, okSubnetId := d.GetOk("subnet_id")
	isPrimary, okIsPrimary := d.GetOk("is_primary")

	if !okVpcId {
		return diag.Errorf("[ERR] VPC id is required")
	}

	if !okInstanceId {
		return diag.Errorf("[ERR] Instance id is required")
	}

	if !okSubnetId {
		return diag.Errorf("[ERR] Subnet id is required")
	}

	createdModel.SubnetId = subnetId.(string)
	createdModel.InstanceId = instanceId.(string)
	createdModel.VpcId = vpcId.(string)

	createdNic, err := nicService.Create(vpcId.(string), createdModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to create NIC: %s", err)
	}

	d.SetId(createdNic.ID)

	// waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"PENDING", "CREATING"},
		Target:  []string{"ACTIVE", "ERROR", "IN_ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			resp, err := nicService.Find(vpcId.(string), instanceId.(string), createdNic.ID)
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
		return diag.Errorf("[ERR] Failed to create NIC: %s", err)
	}

	// Read the initial state
	diags := resourceNicRead(ctx, d, m)
	if diags.HasError() {
		return diags
	}

	// If is_primary was specified in the config and differs from the API's default, update it
	if okIsPrimary && isPrimary.(bool) == true {
		d.Set("is_primary", isPrimary)
		// Call the update function to handle is_primary changes
		return resourceNicUpdate(ctx, d, m)
	}

	return nil
}

func resourceNicRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	nicService := NewNicService(apiClient)

	nicId := d.Id()
	if nicId == "" {
		return diag.Errorf("[ERR] NIC id is required")
	}

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] VPC id is required")
	}
	instanceId, okInstanceId := d.GetOk("instance_id")
	if !okInstanceId {
		return diag.Errorf("[ERR] Instance id is required")
	}

	foundNic, err := nicService.Find(vpcId.(string), instanceId.(string), nicId)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve NIC: %s", err)
	}

	d.SetId(foundNic.ID)

	if err := d.Set("vpc_id", foundNic.VpcId); err != nil {
		return diag.Errorf("[ERR] Failed to set vpc_id: %s", err)
	}
	if err := d.Set("instance_id", foundNic.InstanceId); err != nil {
		return diag.Errorf("[ERR] Failed to set instance_id: %s", err)
	}
	if err := d.Set("subnet_id", foundNic.SubnetId); err != nil {
		return diag.Errorf("[ERR] Failed to set subnet_id: %s", err)
	}
	if err := d.Set("mac_address", foundNic.MacAddress); err != nil {
		return diag.Errorf("[ERR] Failed to set mac_address: %s", err)
	}
	if err := d.Set("private_ip", foundNic.PrivateIp); err != nil {
		return diag.Errorf("[ERR] Failed to set private_ip: %s", err)
	}
	if err := d.Set("status", foundNic.Status); err != nil {
		return diag.Errorf("[ERR] Failed to set status: %s", err)
	}
	if err := d.Set("subnet_name", foundNic.SubnetName); err != nil {
		return diag.Errorf("[ERR] Failed to set subnet_name: %s", err)
	}
	if err := d.Set("is_primary", foundNic.IsPrimary); err != nil {
		return diag.Errorf("[ERR] Failed to set is_primary: %s", err)
	}

	return nil
}

func resourceNicUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	nicService := NewNicService(apiClient)

	nicId := d.Id()
	if nicId == "" {
		return diag.Errorf("[ERR] NIC id is required")
	}

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] VPC id is required")
	}
	instanceId, okInstanceId := d.GetOk("instance_id")
	if !okInstanceId {
		return diag.Errorf("[ERR] Instance id is required")
	}

	if d.HasChange("is_primary") {
		isPrimary := d.Get("is_primary").(bool)
		if isPrimary == false {
			return diag.Errorf("[ERR] Primary NIC can not be set to false")
		}

		updatedModel := UpdateNicDto{
			IsPrimary:  isPrimary,
			InstanceId: instanceId.(string),
		}

		_, err := nicService.Update(vpcId.(string), nicId, updatedModel)
		if err != nil {
			return diag.Errorf("[ERR] Failed to update NIC: %s", err)
		}

		// Wait for update to complete
		updateStateConf := &retry.StateChangeConf{
			Pending: []string{"PENDING", "UPDATING"},
			Target:  []string{"ACTIVE", "ERROR", "IN_ACTIVE"},
			Refresh: func() (interface{}, string, error) {
				resp, err := nicService.Find(vpcId.(string), instanceId.(string), nicId)
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
		_, err = updateStateConf.WaitForStateContext(ctx)
		if err != nil {
			return diag.Errorf("[ERR] Failed to wait for NIC update: %s", err)
		}
	}

	return resourceNicRead(ctx, d, m)
}

func resourceNicDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	nicService := NewNicService(apiClient)

	nicId := d.Id()
	if nicId == "" {
		return diag.Errorf("[ERR] NIC id is required")
	}

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] VPC id is required")
	}
	instanceId, okInstanceId := d.GetOk("instance_id")
	if !okInstanceId {
		return diag.Errorf("[ERR] Instance id is required")
	}

	_, err := nicService.Delete(vpcId.(string), nicId, DeleteNicDto{
		InstanceId: instanceId.(string),
		SubnetId:   d.Get("subnet_id").(string),
	})
	if err != nil {
		return diag.Errorf("[ERR] Failed to delete NIC: %s", err)
	}
	deleteStateConf := &retry.StateChangeConf{
		Pending: []string{"DELETING"},
		Target:  []string{""},
		Refresh: func() (interface{}, string, error) {
			resp, err := nicService.Find(vpcId.(string), instanceId.(string), nicId)
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
	_, err = deleteStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[ERR] Failed to wait for NIC delete: %s", err)
	}

	d.SetId("")
	return nil
}
