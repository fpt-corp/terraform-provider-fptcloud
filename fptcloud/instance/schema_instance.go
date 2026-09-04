package fptcloud_instance

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"terraform-provider-fptcloud/commons/utils"
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
	"tag_ids": {
		Type:        schema.TypeList,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "List of tag IDs associated with the instance",
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
		Description: "The flavor name of the instance (get from API or data source). Changing this resizes the instance in place (not ForceNew). For OSP-backed VMs, this is also how GPUs are attached/detached: switching to/from a GPU flavor (see `gpu_id`/`gpu_name` on the `fptcloud_flavor` data source) attaches/detaches the GPU as part of the resize. Resize is blocked by the server when `is_nvme` is `true`.",
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
		Type:             schema.TypeInt,
		Required:         true,
		Description:      "The root storage size of the instance. Changing this updates the storage in place (not ForceNew); the server only allows growing the size, not shrinking it. Ignored for NVMe GPU flavors (see `is_nvme` on the `fptcloud_flavor` data source) — the server always uses that flavor's own NVMe storage size instead.",
		DiffSuppressFunc: suppressDiffForNvme,
	},
	"storage_policy_id": {
		Type:             schema.TypeString,
		Required:         true,
		Description:      "The root storage policy of the instance. Changing this updates the storage in place (not ForceNew). Ignored for NVMe GPU flavors (see `is_nvme` on the `fptcloud_flavor` data source) — the server always uses that flavor's own NVMe storage policy instead.",
		DiffSuppressFunc: suppressDiffForNvme,
	},
	"storage_id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Internal id of the instance's root storage, used to support in-place storage updates.",
	},
	"storage_name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Internal name of the instance's root storage, used to support in-place storage updates.",
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
	"tag_ids": {
		Type:        schema.TypeSet,
		Computed:    true,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "List of tag IDs to associate with the instance",
	},
	"gpu_plan": {
		Type:         schema.TypeString,
		Optional:     true,
		Computed:     true,
		Description:  "Billing plan for GPU instances: `hold` (reserved) or `detach` (payg). Only applicable when flavor_name is a GPU flavor.",
		ValidateFunc: validation.StringInSlice([]string{"hold", "detach"}, false),
	},
	"gpu_name": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Optional verification input: the GPU this instance is expected to get, as reported by the `gpu_name` field of the `fptcloud_flavor` data source. Leave it unset and the flavor alone decides the GPU (see `vm_type` for what the instance actually got). When set, the server checks it against `flavor_name` on create and on resize, and rejects the request if it names a different GPU or if `flavor_name` is not a GPU flavor. Point it at the same data source as `flavor_name` so the two can never drift apart.",
	},
	"vm_type": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Type of the instance (`cpu` or `gpu`), derived from the server based on flavor_name.",
	},
	"is_nvme": {
		Type:        schema.TypeBool,
		Computed:    true,
		Description: "Whether the instance uses a physical NVMe disk instead of the requested storage_policy_id (see the `is_nvme` field on the `fptcloud_flavor` data source).",
	},
}
