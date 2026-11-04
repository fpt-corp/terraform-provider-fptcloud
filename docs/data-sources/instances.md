---
page_title: "fptcloud_instances Data Source - FPT Cloud"
subcategory: "Compute"
description: |-
  Use this data source to list all instances in a VPC.
---

# fptcloud_instances (Data Source)

Use this data source to list every instance in one of your VPCs, with its name, status, size, IP
addresses, GPU and tags. Typical uses are building an inventory, finding the instances that match a
naming convention, and passing their private IP addresses to other resources.

The data source returns all instances of the VPC. It has no `filter` or `sort` blocks: select
instances with Terraform `for` expressions, as shown in the examples below.

To read one instance, including its subnet ID, root disk and security groups, use
[`fptcloud_instance`](instance) instead.

To learn more, see
[Instance (Virtual Server)](https://docs.fptcloud.com/docs/fpt-cloud-server/instance/overview/)
in the FPT Cloud documentation.

## Example Usage

All examples use this VPC lookup and list:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_instances" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}
```

### List instance names

Use this to see which instances exist in the VPC.

```terraform
output "instance_names" {
  value = [for i in data.fptcloud_instances.all.instances : i.name]
}
```

### Select instances by name, status or flavor

Use `for` expressions with conditions to keep only the instances you need. `regex` matches part of
the name; `can(...)` turns "no match" into `false`.

```terraform
locals {
  # Instances whose name starts with "web-".
  web_instances = [
    for i in data.fptcloud_instances.all.instances : i
    if can(regex("^web-", i.name))
  ]

  # Instances that are currently running.
  running_instances = [
    for i in data.fptcloud_instances.all.instances : i
    if i.status == "POWERED_ON"
  ]

  # Instances of one flavor.
  small_instances = [
    for i in data.fptcloud_instances.all.instances : i
    if i.flavor_name == "2C4G" # Use a flavor name listed in your VPC
  ]

  # Instances with a GPU.
  gpu_instances = [
    for i in data.fptcloud_instances.all.instances : i
    if i.gpu_name != ""
  ]
}
```

### Build an inventory

Use this to publish a summary of the VPC, for example for a monitoring or configuration management
tool. The map is keyed by instance ID because instance IDs are always unique.

```terraform
output "inventory" {
  value = {
    for i in data.fptcloud_instances.all.instances : i.id => {
      name        = i.name
      status      = i.status
      flavor      = i.flavor_name
      cpu         = i.cpu_number
      memory_mb   = i.memory_mb
      private_ip  = i.private_ip
      public_ip   = i.public_ip
      subnet_name = i.subnet_name
    }
  }
}
```

### Apply a security group to a group of instances

Use this to apply a security group to every web server of the VPC by private IP. Instances that do
not have a private IP yet are skipped; run `terraform apply` again later to include them.

```terraform
resource "fptcloud_security_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-sg"
  subnet_id = fptcloud_subnet.web.id # Defined elsewhere in your configuration
  type      = "DFW"
  apply_to = [
    for i in data.fptcloud_instances.all.instances : i.private_ip
    if can(regex("^web-", i.name)) && i.private_ip != ""
  ]
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC whose instances are listed. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source.
* `page_size` - (Optional) How many instances the provider requests from FPT Cloud at a time. The
  provider keeps requesting until it has every instance, so this setting does not limit the result.
  Must be between `1` and `25`. Defaults to `25`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the VPC (same value as `vpc_id`).
* `instances` - All instances in the VPC. The list is empty when the VPC has no instances. Each
  element has the following attributes:
  * `id` - The ID of the instance.
  * `vpc_id` - The ID of the VPC the instance belongs to.
  * `name` - The name of the instance.
  * `status` - The current status of the instance, for example `POWERED_ON` or `POWERED_OFF`.
  * `private_ip` - The private IPv4 address of the instance. Can be empty for a short time after the
    instance is created.
  * `ipv6_address` - The private IPv6 address of the instance. Empty when the instance has no IPv6
    address.
  * `public_ip` - The floating IP address associated with the instance. Empty when the instance has
    no floating IP.
  * `flavor_id` - The ID of the flavor of the instance.
  * `flavor_name` - The name of the flavor of the instance. See [`fptcloud_flavor`](flavor).
  * `cpu_number` - The number of vCPUs of the instance.
  * `memory_mb` - The amount of RAM of the instance, in MB.
  * `gpu_name` - The name of the GPU attached to the instance. Empty for instances without a GPU.
  * `is_nvme` - `true` when the instance runs on a physical NVMe disk.
  * `guest_os` - The operating system of the instance.
  * `host_name` - The host name of the instance.
  * `subnet_name` - The name of the subnet the instance is connected to.
  * `instance_group_id` - The ID of the instance group the instance belongs to. Empty when the
    instance is not in a group.
  * `tag_ids` - The IDs of the tags attached to the instance.
  * `created_at` - The time the instance was created.
  * `updated_at` - The time the instance was last updated.

-> **Note:** The list does not include the subnet ID, root disk size, root disk storage policy or
security groups of the instances. Read them for a specific instance with the
[`fptcloud_instance`](instance) data source by passing the instance `id` from this list to its `id`
argument.
