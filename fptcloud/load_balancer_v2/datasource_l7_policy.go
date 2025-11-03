package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceL7Policies() *schema.Resource {
	return &schema.Resource{
		ReadContext: listL7Policies,
		Schema:      dataSourceL7Policies,
	}
}

func DataSourceL7Policy() *schema.Resource {
	return &schema.Resource{
		ReadContext: getL7Policy,
		Schema:      dataSourceL7Policy,
	}
}

func listL7Policies(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	response, err := service.ListL7Policies(vpcId, listenerId)
	if err != nil {
		return diag.FromErr(err)
	}
	policies := response.L7Policies
	var formattedData []interface{}
	for _, policy := range policies {
		redirect_pool := []interface{}{
			map[string]interface{}{
				"id":       policy.RedirectPool.Id,
				"name":     policy.RedirectPool.Name,
				"protocol": policy.RedirectPool.Protocol,
			},
		}
		formattedData = append(formattedData, map[string]interface{}{
			"id":                  policy.Id,
			"name":                policy.Name,
			"action":              policy.Action,
			"provisioning_status": policy.ProvisioningStatus,
			"redirect_pool":       redirect_pool,
			"redirect_url":        policy.RedirectUrl,
			"redirect_prefix":     policy.RedirectPrefix,
			"redirect_http_code":  policy.RedirectHttpCode,
			"position":            policy.Position,
			"created_at":          policy.CreatedAt,
			"updated_at":          policy.UpdatedAt,
		})
	}
	if err := d.Set("l7policies", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy list: %v", err))
	}
	d.SetId(listenerId)

	return nil
}

func getL7Policy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Get("l7_policy_id").(string)
	response, err := service.GetL7Policy(vpcId, listenerId, l7PolicyId)
	if err != nil {
		return diag.FromErr(err)
	}
	policy := response.L7Policy
	redirect_pool := []interface{}{
		map[string]interface{}{
			"id":       policy.RedirectPool.Id,
			"name":     policy.RedirectPool.Name,
			"protocol": policy.RedirectPool.Protocol,
		},
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
	if err := d.Set("redirect_pool", redirect_pool); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy redirect pool: %v", err))
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
	if err := d.Set("position", policy.Position); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy position: %v", err))
	}
	if err := d.Set("created_at", policy.CreatedAt); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy create date: %v", err))
	}
	if err := d.Set("updated_at", policy.UpdatedAt); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 policy update date: %v", err))
	}
	d.SetId(policy.Id)

	return nil
}
