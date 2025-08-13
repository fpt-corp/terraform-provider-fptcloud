package fptcloud_instance

import (
	"terraform-provider-fptcloud/commons/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var dataSourceInstanceSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The vpc id of the instance",
	},
	"id": {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The id of the instance",
		ExactlyOneOf: []string{"id", "name"},
	},
	"name": {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: utils.ValidateName,
		Description:  "The name of the instance",
		ExactlyOneOf: []string{"id", "name"},
	},
	"guest_os": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The guest os of the instance",
	},
	"host_name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The host name of the instance",
	},
	"status": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The status of the instance",
	},
	"private_ip": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The private ip of the instance",
	},
	"public_ip": {
		Type:        schema.TypeString,
		Computed:    true,
		Optional:    true,
		Description: "The public ip (floating ip) of the instance",
	},
	"memory_mb": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The memory (mb) number of the instance",
	},
	"cpu_number": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The cpu number of the instance",
	},
	"flavor_name": {
		Type:        schema.TypeString,
		Computed:    true,
		Optional:    true,
		Description: "The flavor name of the instance",
	},
	"subnet_id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The subnet id of the instance",
	},
	"storage_size_gb": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The root storage size of the instance",
	},
	"storage_policy": {
		Type:        schema.TypeString,
		Computed:    true,
		Optional:    true,
		Description: "The root storage policy of the instance",
	},
	"security_group_ids": {
		Type:        schema.TypeList,
		Computed:    true,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "The security group associated with the instance",
	},
	"instance_group_id": {
		Type:        schema.TypeString,
		Computed:    true,
		Optional:    true,
		Description: "The instance group id of the instance",
	},
	"created_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The created at of the security group",
	},
}

var resourceInstanceSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The vpc id of the instance",
		ForceNew:     true,
	},
	"id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The id of the instance",
		ForceNew:    true,
	},
	"name": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: utils.ValidateName,
		Description:  "The name of the instance",
	},
	"status": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The status of the instance (`POWERED_ON` or `POWERED_OFF`)",
		ValidateFunc: validation.StringInSlice([]string{
			"POWERED_ON", "POWERED_OFF",
		}, false),
	},
	"private_ip": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The private ip of the instance.",
	},
	"public_ip": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The public ip (floating ip) of the instance.",
	},
	"flavor_name": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The flavor name of the instance (get from API or data source)",
	},
	"image_name": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The image name of the instance (get from API or data source)",
		ForceNew:    true,
	},
	"subnet_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The subnet id of the instance",
		ForceNew:    true,
	},
	"storage_size_gb": {
		Type:        schema.TypeInt,
		Required:    true,
		Description: "The root storage size of the instance",
		ForceNew:    true,
	},
	"storage_policy_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The root storage policy of the instance",
		ForceNew:    true,
	},
	"security_group_ids": {
		Type:        schema.TypeSet,
		Computed:    true,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "The security group associated with the instance",
	},
	"instance_group_id": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The instance group id of the instance",
	},
	"ssh_key": {
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The ssh key of the instance",
		ForceNew:     true,
		ExactlyOneOf: []string{"ssh_key", "password"},
	},
	"password": {
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The password of the instance",
		ForceNew:     true,
		ExactlyOneOf: []string{"ssh_key", "password"},
	},
	"created_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The created at of the security group",
		ForceNew:    true,
	},
	"vm_action": {
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "VM action configuration",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Type of VM action (e.g., REBOOT)",
					ValidateFunc: validation.StringInSlice([]string{
						"REBOOT", "POWER_ON", "POWER_OFF", "SUSPEND", "RESET",
					}, false),
				},
			},
		},
	},
	"snapshot_action": {
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Snapshot action configuration",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Type of snapshot action (e.g., CREATE)",
					ValidateFunc: validation.StringInSlice([]string{
						"CREATE", "DELETE", "UPDATE",
					}, false),
				},
				"include_ram": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Whether to include RAM in the snapshot",
				},
			},
		},
	},
	"template_action": {
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Template action configuration",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Type of template action (e.g., CREATE)",
					ValidateFunc: validation.StringInSlice([]string{
						"CREATE",
					}, false),
				},
				"name": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Name of the template",
				},
				"description": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Description of the template",
				},
			},
		},
	},
	"reset_password": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "The reset password of the instance",
	},
	"block_deletion": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "The block deletion of the instance",
	},
	"resize_config": {
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Configuration for resizing disk",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"increase_in_size_mb": {
					Type:        schema.TypeInt,
					Required:    true,
					Description: "The increase in size in MB",
				},
				"disk_id": {
					Type:        schema.TypeInt,
					Required:    true,
					Description: "The disk ID to resize",
				},
				"name": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The name of the disk",
				},
				"storage_policy_id": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The storage policy ID for the disk",
				},
			},
		},
	},
}
