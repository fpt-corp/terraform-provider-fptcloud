package fptcloud_floating_ip

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

// ResourceFloatingIp function returns a schema.Resource that represents a floating ip.
// This can be used to create, read and delete operations for a floating ip group in the infrastructure.
func ResourceFloatingIp() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a Fpt cloud instance group which can be attached to an instance in order to provide expanded floating ip.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vpc id of the instance group",
				ForceNew:    true,
			},
			"floating_ip_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The id of the ip address",
				ForceNew:     true,
			},
			"instance_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The id of the instance",
				ForceNew:    true,
			},
			"floating_ip_port": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 65535),
				RequiredWith: []string{"instance_id", "instance_port"},
				Description:  "The port of the ip address",
				ForceNew:     true,
			},
			"instance_port": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 65535),
				RequiredWith: []string{"instance_id", "floating_ip_port"},
				Description:  "The port of the instance",
				ForceNew:     true,
			},
			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		CreateContext: resourceFloatingIpCreate,
		ReadContext:   resourceFloatingIpRead,
		DeleteContext: resourceFloatingIpDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceFloatingIpCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewFloatingIpService(apiClient)

	createModel := CreateFloatingIpDTO{}
	vpcId, okVpcId := d.GetOk("vpc_id")
	if okVpcId {
		createModel.VpcId = vpcId.(string)
	}
	if floatingIpId, ok := d.GetOk("floating_ip_id"); ok {
		createModel.FloatingIpId = floatingIpId.(string)
	}
	if floatingIpPort, ok := d.GetOk("floating_ip_port"); ok {
		createModel.FloatingIpPort = floatingIpPort.(int)
	}
	if instanceId, ok := d.GetOk("instance_id"); ok {
		createModel.InstanceId = instanceId.(string)
	}
	if instancePort, ok := d.GetOk("instance_port"); ok {
		createModel.InstancePort = instancePort.(int)
	}

	result, err := service.CreateFloatingIp(createModel)
	if err != nil || result == nil {
		return diag.Errorf("[ERR] Failed to create a new floating ip: %s", err)
	}

	var setError error
	d.SetId("")
	setError = d.Set("vpc_id", vpcId)
	setError = d.Set("ip_address", result.IpAddress)
	if setError != nil {
		return diag.Errorf("[ERR] Failed to create a new floating ip")
	}

	//Waiting for status active
	createStateConf := &retry.StateChangeConf{
		//Pending: []string{"INACTIVE", "IN_ACTIVE"},
		Pending: []string{"INACTIVE"},
		Target:  []string{"ACTIVE", "IN_ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			findModel := FindFloatingIpDTO{
				IpAddress: result.IpAddress,
				VpcId:     vpcId.(string),
			}
			resp, err := service.FindFloatingIpByAddress(findModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}
			d.SetId(resp.ID)
			return resp, resp.Status, nil
		},
		Timeout:                   5 * time.Minute,
		Delay:                     30 * time.Second,
		MinTimeout:                30 * time.Second,
		ContinuousTargetOccurence: 3,
		NotFoundChecks:            20,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for floating ip (%s) to be created: %s", result.IpAddress, err)
	}

	return resourceFloatingIpRead(ctx, d, m)
}

func resourceFloatingIpRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewFloatingIpService(apiClient)

	log.Printf("[INFO] Retrieving the floating ip %s", d.Id())

	findModel := FindFloatingIpDTO{}
	findModel.FloatingIpID = d.Id()
	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findModel.VpcId = vpcId.(string)
	}

	result, err := service.FindFloatingIp(findModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed retrieving the floating ip: %s", err)
	}
	if result == nil {
		return diag.Errorf("[ERR] Floating ip could not be found")
	}

	var setError error
	d.SetId(result.ID)
	setError = d.Set("ip_address", result.IpAddress)
	setError = d.Set("nat_type", result.NatType)
	setError = d.Set("instance", result.Instance)
	setError = d.Set("status", result.Status)
	setError = d.Set("created_at", result.CreatedAt)
	if setError != nil {
		return diag.Errorf("[ERR] Floating ip could not be found")
	}

	return nil
}

func resourceFloatingIpDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewFloatingIpService(apiClient)

	log.Printf("[INFO] Deleting the floating ip %s", d.Id())

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	_, err := service.DeleteFloatingIp(vpcId.(string), d.Id())

	if err != nil {
		return diag.Errorf("[ERR] An error occurred while trying to delete the floating ip %s", err)
	}
	return nil
}
