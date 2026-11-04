---
page_title: "fptcloud_instance Data Source - FPT Cloud"
subcategory: "Compute"
description: |-
  Use this data source to read the details of one instance, found by name or by ID.
---

# fptcloud_instance (Data Source)

Use this data source to read the details of one existing instance in a VPC: its private and public
IP addresses, size, status, subnet, security groups and more. You find the instance either by its
name or by its ID.

Typical uses:

* Read the private IP of an instance that is managed outside this Terraform configuration (in the
  console or in another configuration), for example to allow it in a security group.
* Read an instance created in the same configuration again, to pass its current values to other
  resources.

To read many instances at once, use [`fptcloud_instances`](instances) instead.

The data source fails with an error when no instance with that name or ID exists in the VPC.

To learn more, see
[Instance (Virtual Server)](https://docs.fptcloud.com/docs/fpt-cloud-server/instance/overview/)
in the FPT Cloud documentation.

## Example Usage

All examples use this VPC lookup:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

### Look up an instance by name

Use this for an instance that was created outside this configuration.

```terraform
data "fptcloud_instance" "db" {
  vpc_id = data.fptcloud_vpc.example.id
  name   = "db-server-01"
}

output "db_private_ip" {
  value = data.fptcloud_instance.db.private_ip
}
```

### Look up an instance by ID

Use this when you know the instance ID, for example from the console or from another Terraform
configuration. The ID always identifies exactly one instance.

```terraform
variable "db_instance_id" {
  type        = string
  description = "ID of the database instance."
}

data "fptcloud_instance" "db" {
  vpc_id = data.fptcloud_vpc.example.id
  id     = var.db_instance_id
}
```

### Allow an instance in a security group

Use this to apply a security group to an instance by its private IP. Because the data source
references the instance `id`, Terraform reads it after the instance exists, and reads it again on
every plan.

```terraform
# fptcloud_instance.app and fptcloud_subnet.app are defined elsewhere in your configuration.
data "fptcloud_instance" "app" {
  vpc_id = data.fptcloud_vpc.example.id
  id     = fptcloud_instance.app.id
}

resource "fptcloud_security_group" "app" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "app-sg"
  subnet_id = fptcloud_subnet.app.id
  type      = "DFW"
  apply_to  = [data.fptcloud_instance.app.private_ip]
}
```

-> **Note:** Right after creation, an instance can report an empty private IP for a short time.
If `private_ip` was still empty when Terraform read it, run `terraform apply` again: the data
source is read again and the security group is updated with the address.

### Reuse the settings of an existing instance

Use this to create a new instance with the same size and subnet as an existing one.

```terraform
data "fptcloud_instance" "template" {
  vpc_id = data.fptcloud_vpc.example.id
  name   = "app-server-01"
}

data "fptcloud_image" "ubuntu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Ubuntu-22.04"] # Use an image name listed in your VPC
  }
}

data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Use a storage policy name listed in your VPC
  }
}

resource "fptcloud_instance" "app_02" {
  vpc_id             = data.fptcloud_vpc.example.id
  name               = "app-server-02"
  image_name         = data.fptcloud_image.ubuntu.images[0].name
  flavor_name        = data.fptcloud_instance.template.flavor_name
  subnet_id          = data.fptcloud_instance.template.subnet_id
  security_group_ids = data.fptcloud_instance.template.security_group_ids
  storage_size_gb    = 40
  storage_policy_id  = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key            = file(pathexpand("~/.ssh/id_rsa.pub"))
  status             = "POWERED_ON"
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC that contains the instance. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source.
* `id` - (Optional) The ID of the instance. Must not be empty. Exactly one of `id` or `name` must be
  set.
* `name` - (Optional) The name of the instance. Must not contain spaces or other whitespace. Exactly
  one of `id` or `name` must be set.

Only `vpc_id`, `id` and `name` are used to find the instance. The data source also accepts
`flavor_name`, `private_ip`, `public_ip`, `storage_policy`, `security_group_ids`,
`instance_group_id` and `tag_ids` in the configuration, but they do not narrow the lookup: their
values are replaced with the values of the instance that was found. Do not set them.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the instance.
* `name` - The name of the instance.
* `vpc_id` - The ID of the VPC that contains the instance.
* `status` - The current status of the instance, for example `POWERED_ON` or `POWERED_OFF`. While an
  operation is running, other values can appear, such as `CREATING` or `RESIZING`.
* `private_ip` - The private IPv4 address of the instance in its subnet. Can be empty for a short
  time after the instance is created.
* `public_ip` - The floating IP address associated with the instance. Empty when the instance has no
  floating IP.
* `flavor_name` - The name of the flavor of the instance. See [`fptcloud_flavor`](flavor).
* `cpu_number` - The number of vCPUs of the instance.
* `memory_mb` - The amount of RAM of the instance, in MB.
* `guest_os` - The operating system of the instance.
* `host_name` - The host name of the instance.
* `subnet_id` - The ID of the subnet the instance is connected to.
* `storage_size_gb` - The size of the root disk, in GB. Can be `0` for instances in a General VPC,
  where the root disk is not reported through this data source.
* `storage_policy` - The storage policy of the root disk. Can be empty for instances in a General
  VPC, for the same reason.
* `security_group_ids` - The IDs of the security groups applied to the instance.
* `instance_group_id` - The ID of the instance group the instance belongs to. Empty when the
  instance is not in a group.
* `tag_ids` - The IDs of the tags attached to the instance.
* `created_at` - The time the instance was created.

This data source does not report the image, GPU or NVMe disk of the instance. The
[`fptcloud_instances`](instances) data source reports `gpu_name` and `is_nvme` for every instance of
a VPC.
