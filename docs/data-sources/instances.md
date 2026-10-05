---
page_title: "fptcloud_instances Data Source - terraform-provider-fptcloud"
subcategory: ""
description: |-
  Use this data source to retrieve all instances in a VPC.
  Results are retrieved page by page, up to page_size instances per API call, so listing a large VPC never requires one request per instance. The attributes of each item match those of the fptcloud_instance data source wherever the list API provides the same information.
  ~> Note: subnet_id, storage_size_gb, storage_policy and security_group_ids are not included in the list results. Use the fptcloud_instance data source to read them for a specific instance.
---

# fptcloud_instances (Data Source)

Use this data source to retrieve all instances in a VPC.

Results are retrieved page by page, up to `page_size` instances per API call, so listing a large VPC never requires one request per instance. The attributes of each item match those of the `fptcloud_instance` data source wherever the list API provides the same information.

~> **Note:** `subnet_id`, `storage_size_gb`, `storage_policy` and `security_group_ids` are not included in the list results. Use the `fptcloud_instance` data source to read them for a specific instance.

## Example Usage

```terraform
data "fptcloud_instances" "example" {
  vpc_id = "your_vpc_id"
}

# Names of all instances that are currently running
output "running_instance_names" {
  value = [for instance in data.fptcloud_instances.example.instances : instance.name if instance.status == "POWERED_ON"]
}
```

## Schema

### Required

- `vpc_id` (String) The ID of the VPC whose instances are retrieved.

### Optional

- `page_size` (Number) The number of instances requested per API call. Must be between `1` and `100`; defaults to `100`. The provider requests successive pages until every instance has been retrieved.

### Read-Only

- `id` (String) The ID of this data source. It is set to the value of `vpc_id`.
- `instances` (List of Object) The instances in the VPC. (see [below for nested schema](#nestedatt--instances))

<a id="nestedatt--instances"></a>
### Nested Schema for `instances`

Read-Only:

- `cpu_number` (Number) The number of vCPUs of the instance.
- `created_at` (String) The time the instance was created.
- `flavor_id` (String) The ID of the flavor of the instance.
- `flavor_name` (String) The name of the flavor of the instance.
- `gpu_name` (String) The name of the GPU attached to the instance. Empty for instances without a GPU.
- `guest_os` (String) The guest operating system of the instance.
- `host_name` (String) The host name of the instance.
- `id` (String) The ID of the instance.
- `instance_group_id` (String) The ID of the instance group the instance belongs to. Empty when the instance is not in a group.
- `ipv6_address` (String) The private IPv6 address of the instance. Empty when the instance has no IPv6 address.
- `is_nvme` (Boolean) Whether the instance runs on a physical NVMe disk.
- `memory_mb` (Number) The amount of memory of the instance, in MB.
- `name` (String) The name of the instance.
- `private_ip` (String) The private IPv4 address of the instance.
- `public_ip` (String) The public (floating) IP address of the instance. Empty when no floating IP is assigned.
- `status` (String) The current status of the instance, for example `POWERED_ON` or `POWERED_OFF`.
- `subnet_name` (String) The name of the subnet the instance is connected to.
- `tag_ids` (List of String) The IDs of the tags attached to the instance.
- `updated_at` (String) The time the instance was last updated.
- `vpc_id` (String) The ID of the VPC the instance belongs to.
