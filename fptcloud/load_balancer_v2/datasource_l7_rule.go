package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceL7Rules() *schema.Resource {
	return &schema.Resource{
		ReadContext: listL7Rules,
		Schema:      dataSourceL7Rules,
	}
}

func DataSourceL7Rule() *schema.Resource {
	return &schema.Resource{
		ReadContext: getL7Rule,
		Schema:      dataSourceL7Rule,
	}
}

func listL7Rules(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Get("l7_policy_id").(string)
	response, err := service.ListL7Rules(vpcId, listenerId, l7PolicyId)
	if err != nil {
		return diag.FromErr(err)
	}
	rules := response.Data.L7Rules
	var formattedData []interface{}
	for _, rule := range rules {
		formattedData = append(formattedData, map[string]interface{}{
			"id":                  rule.Id,
			"type":                rule.Type,
			"compare_type":        rule.CompareType,
			"key":                 rule.Key,
			"value":               rule.Value,
			"invert":              rule.Invert,
			"operating_status":    rule.OperatingStatus,
			"provisioning_status": rule.ProvisioningStatus,
		})
	}
	if err := d.Set("l7rules", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 rule list: %v", err))
	}
	d.SetId(l7PolicyId)

	return nil
}

func getL7Rule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Get("l7_policy_id").(string)
	l7RuleId := d.Get("l7_rule_id").(string)
	response, err := service.GetL7Rule(vpcId, listenerId, l7PolicyId, l7RuleId)
	if err != nil {
		return diag.FromErr(err)
	}
	rule := response.L7Rule
	if err := d.Set("type", rule.Type); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 rule type: %v", err))
	}
	if err := d.Set("compare_type", rule.CompareType); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 rule compare type: %v", err))
	}
	if err := d.Set("key", rule.Key); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 rule key: %v", err))
	}
	if err := d.Set("value", rule.Value); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 rule value: %v", err))
	}
	if err := d.Set("invert", rule.Invert); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 rule invert: %v", err))
	}
	if err := d.Set("operating_status", rule.OperatingStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 rule operating status: %v", err))
	}
	if err := d.Set("provisioning_status", rule.ProvisioningStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting l7 rule provisioning status: %v", err))
	}
	d.SetId(l7RuleId)

	return nil
}
