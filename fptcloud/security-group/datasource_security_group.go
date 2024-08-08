package fptcloud_security_group

import (
	"context"
	"strings"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// DataSourceSecurityGroup function returns a schema.Resource that represents a security group.
// This can be used to query and retrieve details about a specific security group in the infrastructure using its id or name.
func DataSourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Description: strings.Join([]string{
			"Get information on a security group for use in other resources. This data source provides all of the security group properties as configured on your account.",
			"An error will be raised if the provided security group does not exist in your account.",
		}, "\n\n"),
		ReadContext: dataSourceSecurityGroupRead,
		Schema:      dataSourceSecurityGroup,
	}
}

func dataSourceSecurityGroupRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupService := NewSecurityGroupService(apiClient)

	findSecurityGroupModel := FindSecurityGroupDTO{}

	if id, ok := d.GetOk("id"); ok {
		findSecurityGroupModel.ID = id.(string)
	}

	if name, ok := d.GetOk("name"); ok {
		findSecurityGroupModel.Name = name.(string)
	}

	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findSecurityGroupModel.VpcId = vpcId.(string)
	}

	foundSecurityGroup, err := securityGroupService.Find(findSecurityGroupModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve security group: %s", err)
	}

	// Set other attributes
	var setError error
	d.SetId(foundSecurityGroup.ID)
	setError = d.Set("vpc_id", foundSecurityGroup.VpcId)
	setError = d.Set("name", foundSecurityGroup.Name)
	setError = d.Set("edge_gateway_id", foundSecurityGroup.EdgeGatewayId)
	setError = d.Set("type", foundSecurityGroup.Type)
	setError = d.Set("apply_to", foundSecurityGroup.ApplyTo)
	setError = d.Set("created_at", foundSecurityGroup.CreatedAt)

	rules := make([]interface{}, len(foundSecurityGroup.Rules))
	for i, rule := range foundSecurityGroup.Rules {
		ruleMap := map[string]interface{}{
			"id":          rule.ID,
			"direction":   rule.Direction,
			"action":      rule.Action,
			"protocol":    rule.Protocol,
			"port_range":  rule.PortRange,
			"sources":     rule.Sources,
			"description": rule.Description,
		}
		rules[i] = ruleMap
	}

	setError = d.Set("rules", rules)

	if setError != nil {
		return diag.Errorf("[ERR]Security group could not be found")
	}

	return nil
}
