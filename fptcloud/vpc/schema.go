package fptcloud_vpc

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	schemaId              = "id"
	schemaName            = "name"
	schemaStatus          = "status"
	schemaTagIds          = "tag_ids"
	schemaOrgId           = "org_id"
	schemaHypervisor      = "hypervisor"
	schemaOwners          = "owners"
	schemaProjectIaasId   = "project_iaas_id"
	schemaSubnetName      = "subnet_name"
	schemaNetworkType     = "network_type"
	schemaCIDR            = "cidr"
	schemaGatewayIp       = "gateway_ip"
	schemaStaticIpPoolFrom = "static_ip_pool_from"
	schemaStaticIpPoolTo   = "static_ip_pool_to"
)

var dataSourceSchema = map[string]*schema.Schema{
	schemaId: {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		ExactlyOneOf: []string{schemaId, schemaName, schemaStatus},
	},
	schemaName: {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		ExactlyOneOf: []string{schemaId, schemaName, schemaStatus},
		Description:  "The name of VPC",
	},
	schemaStatus: {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		ExactlyOneOf: []string{schemaId, schemaName, schemaStatus},
		Description:  "The status of VPC",
	},
}

var resourceSchema = map[string]*schema.Schema{
	schemaOrgId: {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The organization ID (org_id) where the VPC will be created. This is resolved automatically from the tenant configured in the provider.",
	},
	schemaName: {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The name of the VPC (10-50 characters)",
		ForceNew:    true,
	},
	schemaHypervisor: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The hypervisor type",
		ForceNew:    true,
	},
	schemaOwners: {
		Type:        schema.TypeList,
		Required:    true,
		Description: "List of owner user IDs or emails (e.g., 'user@example.com'). Emails will be automatically converted to user IDs.",
		Elem:        &schema.Schema{Type: schema.TypeString},
		ForceNew:    true,
	},
	schemaProjectIaasId: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The project IaaS ID. You can use data.fptcloud_project to get the project ID by name.",
		ForceNew:    true,
	},
	schemaSubnetName: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The name of the subnet (required if network configs are provided)",
		ForceNew:    true,
	},
	schemaNetworkType: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The network type (ROUTED or ISOLATED)",
		ForceNew:    true,
	},
	schemaCIDR: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The CIDR block for the network (required if network configs are provided)",
		ForceNew:    true,
	},
	schemaGatewayIp: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The gateway IP address (required if network configs are provided)",
		ForceNew:    true,
	},
	schemaStaticIpPoolFrom: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The starting IP address for static IP pool",
		ForceNew:    true,
	},
	schemaStaticIpPoolTo: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The ending IP address for static IP pool",
		ForceNew:    true,
	},
	schemaTagIds: {
		Type:        schema.TypeSet,
		Optional:    true,
		Description: "List of tag IDs to associate with the VPC",
		Elem:        &schema.Schema{Type: schema.TypeString},
		ForceNew:    true,
	},
	schemaStatus: {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The status of the VPC",
	},
}
