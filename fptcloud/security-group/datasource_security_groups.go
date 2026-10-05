package fptcloud_security_group

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	common "terraform-provider-fptcloud/commons"
)

// DataSourceSecurityGroups function returns a schema.Resource that represents a list of security groups.
// Its ReadContext fetches every security group in a VPC, walking the portal's paginated list endpoint
// instead of calling Find once per security group.
func DataSourceSecurityGroups() *schema.Resource {
	return &schema.Resource{
		Description: strings.Join([]string{
			"Use this data source to retrieve all security groups in a VPC.",
			"Results are retrieved page by page, up to `page_size` security groups per API call, so listing a large VPC never requires one request per security group. The attributes of each item match those of the `fptcloud_security_group` data source wherever the list API provides the same information.",
			"~> **Note:** Security group rules and `edge_gateway_id` are not included in the list results. Use the `fptcloud_security_group` data source to read them for a specific security group.",
		}, "\n\n"),
		ReadContext: dataSourceSecurityGroupsRead,
		Schema:      dataSourceSecurityGroupsSchema,
	}
}

func dataSourceSecurityGroupsRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupService := NewSecurityGroupService(apiClient)

	listModel := SecurityGroupListDTO{
		VpcId:    d.Get("vpc_id").(string),
		PageSize: d.Get("page_size").(int),
	}

	securityGroups, err := securityGroupService.ListAll(listModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve security groups: %s", err)
	}

	formatted := make([]interface{}, 0, len(securityGroups))
	for _, sg := range securityGroups {
		// The singular endpoint reports the display name when one is set.
		name := sg.Name
		if sg.DisplayName != nil && *sg.DisplayName != "" {
			name = *sg.DisplayName
		}

		applyTo := make([]interface{}, 0, len(sg.IpAddresses))
		for _, ip := range sg.IpAddresses {
			applyTo = append(applyTo, ip)
		}

		tagIds := make([]interface{}, 0, len(sg.Tags))
		for _, tag := range sg.Tags {
			tagIds = append(tagIds, tag.ID)
		}

		members := make([]interface{}, 0, len(sg.Instances))
		for _, member := range sg.Instances {
			ipv6Address := ""
			if member.Ipv6Address != nil {
				ipv6Address = *member.Ipv6Address
			}
			members = append(members, map[string]interface{}{
				"id":           member.ID,
				"name":         member.Name,
				"type":         member.Type,
				"status":       member.Status,
				"ip_address":   member.IpAddress,
				"ipv6_address": ipv6Address,
			})
		}

		formatted = append(formatted, map[string]interface{}{
			"id":         sg.ID,
			"name":       name,
			"type":       sg.FirewallType,
			"apply_to":   applyTo,
			"status":     sg.Status,
			"tag_ids":    tagIds,
			"created_at": sg.CreatedAt,
			"updated_at": sg.UpdatedAt,
			"members":    members,
		})
	}

	if err := d.Set("security_groups", formatted); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(listModel.VpcId)
	return nil
}
