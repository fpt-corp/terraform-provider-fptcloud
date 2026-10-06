package fptcloud_instance

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	common "terraform-provider-fptcloud/commons"
)

// dataSourceInstancesSchema describes the schema of the fptcloud_instances data source.
// It requires a vpc_id and optionally a page_size (max 25); every instance in the VPC is
// returned in the `instances` list. Item attributes keep the names used by the
// fptcloud_instance data source wherever the list endpoint returns the same value.
var dataSourceInstancesSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the VPC whose instances are retrieved.",
	},
	"page_size": {
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      common.ListPageSize,
		ValidateFunc: validation.IntBetween(1, common.ListPageSize),
		Description:  "The number of instances requested per API call. Must be between `1` and `25`; defaults to `25`. The provider requests successive pages until every instance has been retrieved.",
	},
	"instances": {
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The instances in the VPC.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the instance.",
				},
				"vpc_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the VPC the instance belongs to.",
				},
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the instance.",
				},
				"guest_os": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The guest operating system of the instance.",
				},
				"host_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The host name of the instance.",
				},
				"status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The current status of the instance, for example `POWERED_ON` or `POWERED_OFF`.",
				},
				"private_ip": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The private IPv4 address of the instance.",
				},
				"public_ip": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The public (floating) IP address of the instance. Empty when no floating IP is assigned.",
				},
				"memory_mb": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The amount of memory of the instance, in MB.",
				},
				"cpu_number": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The number of vCPUs of the instance.",
				},
				"flavor_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the flavor of the instance.",
				},
				"flavor_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the flavor of the instance.",
				},
				"instance_group_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the instance group the instance belongs to. Empty when the instance is not in a group.",
				},
				"tag_ids": {
					Type:        schema.TypeList,
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "The IDs of the tags attached to the instance.",
				},
				"gpu_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the GPU attached to the instance. Empty for instances without a GPU.",
				},
				"is_nvme": {
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Whether the instance runs on a physical NVMe disk.",
				},
				"created_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The time the instance was created.",
				},
				"ipv6_address": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The private IPv6 address of the instance. Empty when the instance has no IPv6 address.",
				},
				"subnet_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the subnet the instance is connected to.",
				},
				"updated_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The time the instance was last updated.",
				},
			},
		},
	},
}
