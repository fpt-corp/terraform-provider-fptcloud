package fptcloud_security_group

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"terraform-provider-fptcloud/commons/utils"
)

var dataSourceSecurityGroupRule = map[string]*schema.Schema{
	"id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The ID of the security group rule",
	},
	"direction": {
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The direction of the rule can be ingress or egress.",
		ValidateFunc: validation.NoZeroValues,
	},
	"action": {
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The action of the rule can be allow or deny. When we set the `action = 'allow'`, this is going to add a rule to allow traffic. Similarly, setting `action = 'deny'` will deny the traffic.",
		ValidateFunc: validation.StringIsNotEmpty,
	},
	"protocol": {
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The protocol of the security group rule include value `tcp`, `udp` or `icmp`",
		ValidateFunc: validation.NoZeroValues,
	},
	"port_range": {
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The port or port range to open",
		ValidateFunc: validation.NoZeroValues,
	},
	"sources": {
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The sources of the rule, can be a CIDR notation or a IP address",
		ValidateFunc: validation.NoZeroValues,
	},
	"description": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The description of the security group rule",
	},
}
var dataSourceSecurityGroup = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The vpc id of the security group",
	},
	"id": {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The id of the security group",
		ExactlyOneOf: []string{"id", "name"},
	},
	"name": {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: utils.ValidateName,
		Description:  "The name of the security group",
		ExactlyOneOf: []string{"id", "name"},
	},
	"edge_gateway_id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The edge gateway id of the security group",
	},
	"type": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Type of the security group, can be `ACL` (Control traffic through in and through out the internet) or `DFW` (Control traffic through in and through out the local network)",
	},
	"apply_to": {
		Type:        schema.TypeList,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "The list IP apply to of the security group",
	},
	"rules": {
		Type:        schema.TypeList,
		Computed:    true,
		Elem:        &schema.Resource{Schema: dataSourceSecurityGroupRule},
		Description: "The list IP apply to of the security group",
	},
	"created_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The created at of the security group",
	},
	"tag_ids": {
		Type:        schema.TypeList,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "List of tag IDs associated with the security group",
	},
}

var resourceSecurityGroup = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The vpc id of the security group",
	},
	"id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The id of the security group",
	},
	"name": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: utils.ValidateName,
		Description:  "The name of the security group",
	},
	"subnet_id": {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The subnet id of the security group (required when creating)",
	},
	"edge_gateway_id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The edge gateway id of the security group",
	},
	"type": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "Type of the security group, can be `ACL` (Control traffic through in and through out the internet) or `DFW` (Control traffic through in and through out the local network)",
		ValidateFunc: validation.StringInSlice([]string{
			"ACL", "DFW",
		}, false),
	},
	"apply_to": {
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "The list IP apply to of the security group",
	},
	"created_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The created at of the security group",
	},
	"tag_ids": {
		Type:        schema.TypeSet,
		Optional:    true,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "List of tag IDs associated with the security group",
	},
}
