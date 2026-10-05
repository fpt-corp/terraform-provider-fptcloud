package fptcloud_security_group

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	common "terraform-provider-fptcloud/commons"
)

// dataSourceSecurityGroupsSchema describes the schema of the fptcloud_security_groups data source.
// It requires a vpc_id and optionally a page_size (max 100); every security group in the VPC is
// returned in the `security_groups` list. Item attributes keep the names used by the
// fptcloud_security_group data source wherever the list endpoint returns the same value.
var dataSourceSecurityGroupsSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the VPC whose security groups are retrieved.",
	},
	"page_size": {
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      common.ListPageSize,
		ValidateFunc: validation.IntBetween(1, common.ListPageSize),
		Description:  "The number of security groups requested per API call. Must be between `1` and `100`; defaults to `100`. The provider requests successive pages until every security group has been retrieved.",
	},
	"security_groups": {
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The security groups in the VPC.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the security group.",
				},
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the security group.",
				},
				"type": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The type of the security group: `ACL` (controls traffic to and from the internet) or `DFW` (controls traffic within the local network).",
				},
				"apply_to": {
					Type:        schema.TypeList,
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "The IP addresses or CIDR blocks the security group is applied to.",
				},
				"status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The current status of the security group, for example `REALIZED`.",
				},
				"tag_ids": {
					Type:        schema.TypeList,
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "The IDs of the tags attached to the security group.",
				},
				"created_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The time the security group was created.",
				},
				"updated_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The time the security group was last updated.",
				},
				"members": {
					Type:        schema.TypeList,
					Computed:    true,
					Description: "The resources matched by the addresses in `apply_to`.",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The ID of the member. For a member of type `other`, this is the address itself.",
							},
							"name": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The name of the member. For a member of type `other`, this is the address itself.",
							},
							"type": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The kind of member: `instance`, `load_balancer`, or `other` for an address that does not belong to a known resource.",
							},
							"status": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The status of the member resource. `Unknown` for a member of type `other`.",
							},
							"ip_address": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The IP address of the member.",
							},
							"ipv6_address": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The IPv6 address of the member, when it has one.",
							},
						},
					},
				},
			},
		},
	},
}
