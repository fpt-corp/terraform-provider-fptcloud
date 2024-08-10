package fptcloud_floating_ip_association

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

// ResourceFloatingIpAssociation function returns a schema.Resource that represents a floating ip association.
// This can be used to create, read and release operations for a floating ip association in the infrastructure.
func ResourceFloatingIpAssociation() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a FPT cloud floating ip which can be associate and disassociate floating ip from instances.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vpc id of the floating ip",
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
				Description:  "The port of the floating ip",
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
		},
		CreateContext: resourceFloatingIpAssociationCreate,
		ReadContext:   resourceFloatingIpAssociationRead,
		DeleteContext: resourceFloatingIpAssociationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceFloatingIpAssociationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewFloatingIpAssociationService(apiClient)

	createModel := AssociateFloatingIpDTO{}
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

	result, err := service.Associate(createModel)
	if err != nil || result == nil {
		return diag.Errorf("[ERR] Failed to associate floating ip: %s", err)
	}

	var setError error
	d.SetId(createModel.FloatingIpId)
	setError = d.Set("vpc_id", createModel.VpcId)
	setError = d.Set("floating_ip_id", createModel.FloatingIpId)
	setError = d.Set("instance_id", createModel.InstanceId)
	if setError != nil {
		return diag.Errorf("[ERR] Failed to associate floating ip")
	}

	//Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"IN_ACTIVE", "PENDING"},
		Target:  []string{"ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			findModel := FindFloatingIpDTO{
				FloatingIpID: createModel.FloatingIpId,
				VpcId:        vpcId.(string),
			}
			resp, err := service.FindFloatingIp(findModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}
			return resp, resp.Status, nil
		},
		Timeout:        5 * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for associate floating ip (%s): %s", createModel.FloatingIpId, err)
	}

	return resourceFloatingIpAssociationRead(ctx, d, m)
}

func resourceFloatingIpAssociationRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewFloatingIpAssociationService(apiClient)

	log.Printf("[INFO] Retrieving the floating ip %s", d.Id())

	findModel := FindFloatingIpDTO{}
	findModel.FloatingIpID = d.Id()
	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findModel.VpcId = vpcId.(string)
	}

	result, err := service.FindFloatingIp(findModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed retrieving the floating ip association: %s", err)
	}
	if result == nil {
		return diag.Errorf("[ERR] Floating ip could not be found")
	}

	var setError error
	d.SetId(result.ID)
	setError = d.Set("vpc_id", findModel.VpcId)
	setError = d.Set("floating_ip_id", result.ID)
	setError = d.Set("instance_id", result.Instance.ID)
	if setError != nil {
		return diag.Errorf("[ERR] Floating ip could not be found")
	}

	return nil
}

func resourceFloatingIpAssociationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewFloatingIpAssociationService(apiClient)

	log.Printf("[INFO] Disassociate the floating ip %s", d.Id())

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	_, err := service.Disassociate(vpcId.(string), d.Id())

	if err != nil {
		return diag.Errorf("[ERR] An error occurred while disassociate the floating ip %s", err)
	}

	createStateConf := &retry.StateChangeConf{
		Pending: []string{"ACTIVE", "PENDING"},
		Target:  []string{"IN_ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			findModel := FindFloatingIpDTO{
				FloatingIpID: d.Id(),
				VpcId:        vpcId.(string),
			}
			resp, err := service.FindFloatingIp(findModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}
			return resp, resp.Status, nil
		},
		Timeout:        5 * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for disassociate floating ip (%s): %s", d.Id(), err)
	}

	return nil
}
