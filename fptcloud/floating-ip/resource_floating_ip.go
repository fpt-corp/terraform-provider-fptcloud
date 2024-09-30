package fptcloud_floating_ip

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	common "terraform-provider-fptcloud/commons"
	"time"
)

// ResourceFloatingIp function returns a schema.Resource that represents a floating ip.
// This can be used to create, read and delete operations for a floating ip in the infrastructure.
func ResourceFloatingIp() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a FPT cloud floating ip which can be created to public ip address in order to provide expanded floating ip.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vpc id of the floating ip",
				ForceNew:    true,
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

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}
	result, err := service.CreateFloatingIp(vpcId.(string))
	if err != nil || result == nil {
		return diag.Errorf("[ERR] Failed to create a new floating ip: %s", err)
	}

	d.SetId(result.ID)

	if err := d.Set("vpc_id", vpcId); err != nil {
		return diag.Errorf("[ERR] Failed to set 'vpc_id': %s", err)
	}

	if err := d.Set("ip_address", result.IpAddress); err != nil {
		return diag.Errorf("[ERR] Failed to set 'ip_address': %s", err)
	}

	//Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{},
		Target:  []string{"ACTIVE", "IN_ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			findModel := FindFloatingIpDTO{
				FloatingIpID: result.ID,
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

	d.SetId(result.ID)

	if err := d.Set("vpc_id", findModel.VpcId); err != nil {
		return diag.Errorf("[ERR] Failed to set 'vpc_id': %s", err)
	}

	if err := d.Set("ip_address", result.IpAddress); err != nil {
		return diag.Errorf("[ERR] Failed to set 'ip_address': %s", err)
	}

	if err := d.Set("created_at", result.CreatedAt); err != nil {
		return diag.Errorf("[ERR] Failed to set 'created_at': %s", err)
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
