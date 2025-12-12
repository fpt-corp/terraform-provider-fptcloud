package fptcloud_subnet

import (
	"context"
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
	createModel := CreateSubnetDTO{
		Name:         d.Get("name").(string),
		CIDR:         d.Get("cidr").(string),
		Type:         d.Get("type").(string),
		GatewayIp:    d.Get("gateway_ip").(string),
		IpRangeStart: ipRangeStart,
		IpRangeEnd:   ipRangeEnd,
	}
	//if tags, ok := d.GetOk("tag_ids"); ok {
	//	createModel.TagIds = expandTagIDs(tags.(*schema.Set))
	//}
	if okVpcId {
		createModel.VpcId = vpcId.(string)
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

	if err := d.Set("network_id", result.NetworkID); err != nil {
		return diag.Errorf("[ERR] Failed to set 'network_id': %s", err)
	}

	if err := d.Set("network_name", result.NetworkName); err != nil {
		return diag.Errorf("[ERR] Failed to set 'network_name': %s", err)
	}

	if err := d.Set("gateway", result.Gateway); err != nil {
		return diag.Errorf("[ERR] Failed to set 'gateway': %s", err)
	}

	if err := d.Set("created_at", result.CreatedAt); err != nil {
		return diag.Errorf("[ERR] Failed to set 'created_at': %s", err)
	}

	if err := d.Set("tag_ids", result.TagIds); err != nil {
		return diag.Errorf("[ERR] Failed to set 'tag_ids': %s", err)
	}

	return nil
}

func resourceSubnetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Update operation is not yet implemented for tagging resource")
	//apiClient := m.(*common.Client)
	//service := NewSubnetService(apiClient)
	//
	//if !d.HasChange("tag_ids") {
	//	return resourceSubnetRead(ctx, d, m)
	//}
	//
	//vpcId := d.Get("vpc_id").(string)
	//tagIds := expandTagIDs(d.Get("tag_ids").(*schema.Set))
	//_, err := service.UpdateTags(vpcId, d.Id(), tagIds)
	//if err != nil {
	//	return diag.Errorf("[ERR] An error occurred while updating subnet tags %s", err)
	//}
	//
	//return resourceSubnetRead(ctx, d, m)
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

func expandTagIDs(tagSet *schema.Set) []string {
	tagIds := make([]string, 0, tagSet.Len())
	for _, tag := range tagSet.List() {
		tagIds = append(tagIds, tag.(string))
	}
	return tagIds
}
