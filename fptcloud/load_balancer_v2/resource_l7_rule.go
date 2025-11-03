package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceL7Rule() *schema.Resource {
	return &schema.Resource{
		CreateContext: createL7Rule,
		ReadContext:   readL7Rule,
		UpdateContext: updateL7Rule,
		DeleteContext: deleteL7Rule,
		Schema:        resourceL7Rule,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 8 || parts[0] != "vpc" || parts[2] != "listener" || parts[4] != "l7policy" || parts[6] != "l7rule" {
					return nil, fmt.Errorf("invalid import id format, expected vpc/<vpc_id>/listener/<listener_id>/l7policy/<l7_policy_id>/l7rule/<l7_rule_id>")
				}
				vpcId := parts[1]
				l7RuleId := parts[7]

				if err := d.Set("vpc_id", vpcId); err != nil {
					return nil, fmt.Errorf("error setting vpc id: %s", err)
				}
				d.SetId(l7RuleId)

				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func readL7Rule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Get("l7_policy_id").(string)
	l7RuleId := d.Id()
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
	return nil
}

func createL7Rule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Get("l7_policy_id").(string)

	var payload L7RuleInput
	payload.Type = d.Get("type").(string)
	payload.CompareType = d.Get("compare_type").(string)
	payload.Key = d.Get("key").(string)
	payload.Value = d.Get("value").(string)
	payload.Invert = d.Get("invert").(bool)

	response, err := service.CreateL7Rule(vpcId, listenerId, l7PolicyId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(response.Data.Id)
	return nil
}

func updateL7Rule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Get("l7_policy_id").(string)
	l7RuleId := d.Id()

	var payload L7RuleInput
	payload.Type = d.Get("type").(string)
	payload.CompareType = d.Get("compare_type").(string)
	payload.Key = d.Get("key").(string)
	payload.Value = d.Get("value").(string)
	payload.Invert = d.Get("invert").(bool)

	_, err := service.UpdateL7Rule(vpcId, listenerId, l7PolicyId, l7RuleId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	return readL7Rule(ctx, d, m)
}

func deleteL7Rule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	l7PolicyId := d.Get("l7_policy_id").(string)
	l7RuleId := d.Id()

	_, err := service.DeleteL7Rule(vpcId, listenerId, l7PolicyId, l7RuleId)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
