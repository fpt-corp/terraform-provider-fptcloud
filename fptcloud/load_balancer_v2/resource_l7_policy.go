package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceL7Policy() *schema.Resource {
	return &schema.Resource{
		CreateContext: createL7Policy,
		ReadContext:   readL7Policy,
		UpdateContext: updateL7Policy,
		DeleteContext: deleteL7Policy,
		Schema:        resourceL7Policy,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 6 || parts[0] != "vpc" || parts[2] != "listener" || parts[4] != "l7policy" {
					return nil, fmt.Errorf("invalid import id format, expected vpc/<vpc_id>/listener/<listener_id>/l7policy/<l7_policy_id>")
				}
				vpcId := parts[1]
				l7PolicyId := parts[5]

				if err := d.Set("vpc_id", vpcId); err != nil {
					return nil, fmt.Errorf("error setting vpc id: %s", err)
				}
				d.SetId(l7PolicyId)

				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func readL7Policy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Id()

	response, err := service.GetL7Policy(vpcId, listenerId, l7PolicyId)
	if err != nil {
		return diag.FromErr(err)
	}

	policy := response.L7Policy

	pos := policy.Position
	position, err := strconv.Atoi(pos)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid position value: %s", pos))
	}

	if err := d.Set("name", policy.Name); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy name: %v", err))
	}
	if err := d.Set("action", policy.Action); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy action: %v", err))
	}
	if err := d.Set("provisioning_status", policy.ProvisioningStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy provisioning status: %v", err))
	}
	if err := d.Set("redirect_url", policy.RedirectUrl); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy redirect url: %v", err))
	}
	if err := d.Set("redirect_prefix", policy.RedirectPrefix); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy redirect prefix: %v", err))
	}
	if err := d.Set("redirect_http_code", policy.RedirectHttpCode); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy redirect http code: %v", err))
	}
	if err := d.Set("redirect_pool", policy.RedirectPool.Id); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy redirect pool: %v", err))
	}
	if err := d.Set("position", position); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy position: %v", err))
	}
	if err := d.Set("created_at", policy.CreatedAt); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy create date: %v", err))
	}
	if err := d.Set("updated_at", policy.UpdatedAt); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy update date: %v", err))
	}
	return nil
}

func createL7Policy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)

	var payload L7PolicyInput
	payload.Name = d.Get("name").(string)
	payload.Action = d.Get("action").(string)
	payload.RedirectPool = d.Get("redirect_pool").(string)
	payload.RedirectUrl = d.Get("redirect_url").(string)
	payload.RedirectPrefix = d.Get("redirect_prefix").(string)
	payload.RedirectHttpCode = d.Get("redirect_http_code").(int)
	payload.Position = d.Get("position").(int)

	response, err := service.CreateL7Policy(vpcId, listenerId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(response.Data.Id)
	return nil
}

func updateL7Policy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Id()

	var payload L7PolicyInput
	payload.Name = d.Get("name").(string)
	payload.Action = d.Get("action").(string)
	payload.RedirectPool = d.Get("redirect_pool").(string)
	payload.RedirectUrl = d.Get("redirect_url").(string)
	payload.RedirectPrefix = d.Get("redirect_prefix").(string)
	payload.RedirectHttpCode = d.Get("redirect_http_code").(int)
	payload.Position = d.Get("position").(int)

	_, err := service.UpdateL7Policy(vpcId, listenerId, l7PolicyId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	return readL7Policy(ctx, d, m)
}

func deleteL7Policy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Id()

	_, err := service.DeleteL7Policy(vpcId, listenerId, l7PolicyId)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
