package fptcloud_subnet

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	common "terraform-provider-fptcloud/commons"
	"time"
)

// ResourceSubnet function returns a schema.Resource that represents a subnet.
// This can be used to create, read and delete operations for a subnet in the infrastructure.
func ResourceSubnet() *schema.Resource {
	return &schema.Resource{
		Description:   "Provides a FPT cloud instance group which can be attached to an instance in order to provide expanded subnet.",
		Schema:        resourceSubnet,
		CreateContext: resourceSubnetCreate,
		ReadContext:   resourceSubnetRead,
		UpdateContext: resourceSubnetUpdate,
		DeleteContext: resourceSubnetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceSubnetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewSubnetService(apiClient)

	vpcId, okVpcId := d.GetOk("vpc_id")
	ipRange := d.Get("static_ip_pool").(string)
	ipRangeStart, ipRangeEnd := parseIPRange(ipRange)

	primaryDns, okPrimaryDns := d.GetOk("primary_dns_ip")
	secondaryDns, okSecondaryDns := d.GetOk("secondary_dns_ip")
	tagNames, okTagNames := d.GetOk("tag_names")
	createModel := CreateSubnetDTO{
		Name:         d.Get("name").(string),
		CIDR:         d.Get("cidr").(string),
		Type:         d.Get("type").(string),
		GatewayIp:    d.Get("gateway_ip").(string),
		IpRangeStart: ipRangeStart,
		IpRangeEnd:   ipRangeEnd,
	}
	if okVpcId {
		createModel.VpcId = vpcId.(string)
	}

	if okPrimaryDns {
		createModel.PrimaryDnsIp = primaryDns.(string)
	}
	if okSecondaryDns {
		createModel.SecondaryDnsIp = secondaryDns.(string)
	}
	if okTagNames {
		createModel.TagNames = expandStringList(tagNames.([]interface{}))
	}

	result, err := service.CreateSubnet(createModel)
	if err != nil || result == nil {
		return diag.Errorf("[ERR] Failed to create a new subnet: %s", err)
	}

	d.SetId("")

	if err := d.Set("vpc_id", vpcId); err != nil {
		return diag.Errorf("[ERR] Failed to set 'vpc_id': %s", err)
	}

	if err := d.Set("network_name", result.NetworkName); err != nil {
		return diag.Errorf("[ERR] Failed to set 'network_name': %s", err)
	}

	//Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"PENDING"},
		Target:  []string{"COMPLETE"},
		Refresh: func() (interface{}, string, error) {
			findModel := FindSubnetDTO{
				NetworkName: result.NetworkName,
				VpcId:       vpcId.(string),
			}
			resp, err := service.FindSubnetByName(findModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}

			d.SetId(resp.ID)
			return resp, "COMPLETE", nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     30 * time.Second,
		NotFoundChecks: 20,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for subnet (%s) to be created: %s", createModel.Name, err)
	}

	return resourceSubnetRead(ctx, d, m)
}

func resourceSubnetRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewSubnetService(apiClient)

	log.Printf("[INFO] Retrieving the subnet %s", d.Id())

	findModel := FindSubnetDTO{}
	findModel.NetworkID = d.Id()
	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findModel.VpcId = vpcId.(string)
	}

	result, err := service.FindSubnet(findModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed retrieving the subnet: %s", err)
	}
	if result == nil {
		return diag.Errorf("[ERR] Subnet could not be found")
	}

	d.SetId(result.ID)

	if err := d.Set("name", result.Name); err != nil {
		return diag.Errorf("[ERR] Failed to set 'name': %s", err)
	}

	if err := d.Set("network_name", result.NetworkName); err != nil {
		return diag.Errorf("[ERR] Failed to set 'network_name': %s", err)
	}

	if err := d.Set("gateway", result.Gateway); err != nil {
		return diag.Errorf("[ERR] Failed to set 'gateway': %s", err)
	}

	if err := d.Set("primary_dns_ip", result.PrimaryDnsIp); err != nil {
		return diag.Errorf("[ERR] Failed to set 'primary_dns_ip': %s", err)
	}

	if err := d.Set("secondary_dns_ip", result.SecondaryDnsIp); err != nil {
		return diag.Errorf("[ERR] Failed to set 'secondary_dns_ip': %s", err)
	}

	if err := d.Set("created_at", result.CreatedAt); err != nil {
		return diag.Errorf("[ERR] Failed to set 'created_at': %s", err)
	}

	return nil
}

func resourceSubnetDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewSubnetService(apiClient)

	log.Printf("[INFO] Deleting the subnet %s", d.Id())

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	_, err := service.DeleteSubnet(vpcId.(string), d.Id())

	if err != nil {
		return diag.Errorf("[ERR] An error occurred while trying to delete the subnet %s", err)
	}
	return nil
}

// waitForSubnetState waits for the subnet to reach a target state
func waitForSubnetState(
	ctx context.Context,
	service SubnetService,
	vpcId string,
	subnetId string,
	pending []string,
	target []string,
	timeoutMinutes int,
	waitMsg string,
) error {
	stateConf := &retry.StateChangeConf{
		Pending: pending,
		Target:  target,
		Refresh: func() (interface{}, string, error) {
			findModel := FindSubnetDTO{
				NetworkID: subnetId,
				VpcId:     vpcId,
			}
			resp, err := service.FindSubnet(findModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}
			return resp, "COMPLETE", nil // TODO: thay "COMPLETE" bằng resp.Status nếu có
		},
		Timeout:        time.Duration(timeoutMinutes) * time.Minute,
		Delay:          5 * time.Second,
		MinTimeout:     10 * time.Second,
		NotFoundChecks: 30,
	}
	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return fmt.Errorf("[Error] Waiting for subnet %s (%s): %w", waitMsg, subnetId, err)
	}
	return nil
}

func expandStringList(list []interface{}) []string {
	result := make([]string, len(list))
	for i, v := range list {
		result[i] = v.(string)
	}
	return result
}

func resourceSubnetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewSubnetService(apiClient)

	vpcId, okVpcId := d.GetOk("vpc_id")
	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	subnetId := d.Id()
	if subnetId == "" {
		return diag.Errorf("[ERR] Subnet id is required")
	}

	if d.HasChange("tag_names") {
		tags := expandStringList(d.Get("tag_names").([]interface{}))
		updateTagsDto := UpdateTagsSubnetDTO{
			VpcId:    vpcId.(string),
			SubnetId: subnetId,
			TagNames: tags,
		}
		if _, err := service.UpdateTags(updateTagsDto); err != nil {
			return diag.Errorf("[ERR] Failed to update subnet tags: %s", err)
		}
	}

	if d.HasChange("primary_dns_ip") || d.HasChange("secondary_dns_ip") {
		updateDnsDto := UpdateDnsSubnetDTO{
			VpcId:    vpcId.(string),
			SubnetId: subnetId,
		}
		if d.HasChange("primary_dns_ip") {
			updateDnsDto.PrimaryDnsIp = d.Get("primary_dns_ip").(string)
		}
		if d.HasChange("secondary_dns_ip") {
			updateDnsDto.SecondaryDnsIp = d.Get("secondary_dns_ip").(string)
		}
		_, err := service.UpdateDns(updateDnsDto)
		if err != nil {
			return diag.Errorf("[ERR] Failed to update subnet DNS: %s", err)
		}
		if err := waitForSubnetState(ctx, apiClient, service, vpcId.(string), subnetId, []string{"UPDATING"}, []string{"COMPLETE"}, apiClient.Timeout, "DNS update"); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceSubnetRead(ctx, d, m)
}
