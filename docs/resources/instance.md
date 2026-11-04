---
page_title: "fptcloud_instance Resource - FPT Cloud"
subcategory: "Compute"
description: |-
  Manages a virtual server (instance) in an FPT Cloud VPC.
---

# fptcloud_instance (Resource)

Manages an instance: a virtual server running in one of your FPT Cloud VPCs. An instance boots
from an image, gets its vCPU and RAM from a flavor, stores its operating system on a root disk, and
receives a private IP address from the subnet you place it in.

With this resource you can create an instance, power it on or off, rename it, resize its vCPU and
RAM, grow its root disk, change tags, and delete it. Related tasks are handled by dedicated
resources:

* Extra data disks: [`fptcloud_storage`](storage).
* Firewall rules: [`fptcloud_security_group`](security_group) and
  [`fptcloud_security_group_rule`](security_group_rule).
* Public IP addresses: [`fptcloud_floating_ip`](floating_ip) and
  [`fptcloud_floating_ip_association`](floating_ip_association).
* Host placement: [`fptcloud_instance_group`](instance_group).

To learn more about instances, see
[Instance (Virtual Server)](https://docs.fptcloud.com/docs/fpt-cloud-server/instance/overview/)
in the FPT Cloud documentation. If this is your first instance, the
[Getting Started](../guides/getting-started) guide walks you through a complete setup.

## Example Usage

The examples on this page share the following lookups. Catalog names such as image, flavor and
storage policy names differ between regions and VPCs, so replace the example values with names
listed in your VPC.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_image" "ubuntu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Ubuntu-22.04"] # Use an image name listed in your VPC
  }
}

data "fptcloud_flavor" "small" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["2C4G"] # Use a flavor name listed in your VPC
  }
}

data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Use a storage policy name listed in your VPC
  }
}

resource "fptcloud_subnet" "app" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "app-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.10.1.0/24"
  gateway_ip = "10.10.1.1"
}
```

### Linux instance with an SSH key

The most common setup: a Linux instance you log in to with your SSH key. Pass the public key
itself, in OpenSSH format, to `ssh_key`.

```terraform
resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = data.fptcloud_flavor.small.flavors[0].name
  subnet_id         = fptcloud_subnet.app.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
}

output "app_private_ip" {
  value = fptcloud_instance.app.private_ip
}
```

To reuse a key that is already registered in FPT Cloud, read it with the
[`fptcloud_ssh_key`](../data-sources/ssh_key) data source and pass its `public_key`:

```terraform
data "fptcloud_ssh_key" "deployer" {
  name = "deployer-key"
}

resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = data.fptcloud_flavor.small.flavors[0].name
  subnet_id         = fptcloud_subnet.app.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = data.fptcloud_ssh_key.deployer.public_key
  status            = "POWERED_ON"
}
```

### Instance with a password

Use `password` instead of `ssh_key` when you log in with a password, for example on Windows
images. Keep the password out of your code by passing it as a sensitive variable.

```terraform
variable "admin_password" {
  type      = string
  sensitive = true
}

resource "fptcloud_instance" "windows" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "win-server-01"
  image_name        = "Windows-Server-2022" # Use an image name listed in your VPC
  flavor_name       = "4C8G"                # Use a flavor name listed in your VPC
  subnet_id         = fptcloud_subnet.app.id
  storage_size_gb   = 80
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  password          = var.admin_password
  status            = "POWERED_ON"
}
```

~> **Important:** The password is stored in the Terraform state in plain text. Protect your state
file, for example with a remote backend that encrypts data at rest.

### Instance with a fixed private IP address

Set `private_ip` when other systems expect the instance at a known address. The address must be
inside the subnet's CIDR and not used by another resource.

```terraform
resource "fptcloud_instance" "db" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "db-server-01"
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = data.fptcloud_flavor.small.flavors[0].name
  subnet_id         = fptcloud_subnet.app.id
  private_ip        = "10.10.1.50"
  storage_size_gb   = 100
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
}
```

### Several identical instances spread across physical hosts

Create a pool of instances with `count`, and place them in an instance group that uses the
**Soft Anti Affinity** policy so that one host failure does not take down the whole pool.

```terraform
data "fptcloud_instance_group_policy" "anti_affinity" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Soft Anti Affinity"]
  }
}

resource "fptcloud_instance_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-group"
  policy_id = data.fptcloud_instance_group_policy.anti_affinity.instance_group_policies[0].id
}

resource "fptcloud_instance" "web" {
  count = 3

  vpc_id            = data.fptcloud_vpc.example.id
  name              = format("web-server-%02d", count.index + 1)
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = data.fptcloud_flavor.small.flavors[0].name
  subnet_id         = fptcloud_subnet.app.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  instance_group_id = fptcloud_instance_group.web.id
  status            = "POWERED_ON"
}
```

### Instance reachable from the Internet

Allow traffic with a security group and give the instance a public address with a floating IP.
The security group is attached through its `apply_to` argument, using the instance's private IP.
The [Web Server](../guides/web-server) guide explains this setup step by step.

```terraform
data "fptcloud_instance" "app" {
  vpc_id = data.fptcloud_vpc.example.id
  id     = fptcloud_instance.app.id
}

resource "fptcloud_security_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-sg"
  subnet_id = fptcloud_subnet.app.id
  type      = "ACL"
  apply_to  = [data.fptcloud_instance.app.private_ip]
}

resource "fptcloud_security_group_rule" "https" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "443"
  sources           = ["0.0.0.0/0"]
  description       = "HTTPS from anywhere"
}

resource "fptcloud_floating_ip" "app" {
  vpc_id = data.fptcloud_vpc.example.id
}

resource "fptcloud_floating_ip_association" "app" {
  vpc_id         = data.fptcloud_vpc.example.id
  floating_ip_id = fptcloud_floating_ip.app.id
  instance_id    = fptcloud_instance.app.id
}
```

### GPU instance

Choose a GPU flavor to get a GPU instance. Read the flavor with the
[`fptcloud_flavor`](../data-sources/flavor) data source and pass its `gpu_name` as well, so that
FPT Cloud checks that the flavor and the GPU you expect match. `gpu_plan` selects the GPU billing
plan.

```terraform
data "fptcloud_flavor" "gpu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "type"
    values = ["GPU_SIZE"]
  }
}

resource "fptcloud_instance" "training" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "gpu-training-01"
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = data.fptcloud_flavor.gpu.flavors[0].name
  gpu_name          = data.fptcloud_flavor.gpu.flavors[0].gpu_name
  gpu_plan          = "hold"
  subnet_id         = fptcloud_subnet.app.id
  storage_size_gb   = 100
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
}
```

### Instance with an extra data disk

Keep application data on its own disk so you can grow, back up or move it independently of the
operating system. See [`fptcloud_storage`](storage) for all options.

```terraform
resource "fptcloud_storage" "app_data" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-data-disk"
  type              = "EXTERNAL"
  size_gb           = 200
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  instance_id       = fptcloud_instance.app.id
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC to create the instance in. Use the `id` of the
  [`fptcloud_vpc`](../data-sources/vpc) data source. Changing this forces a new resource to be
  created.
* `name` - (Required) The name of the instance. It must not contain spaces and must be unique
  within the VPC. Can be updated in place.
* `image_name` - (Required) The name of the image to boot from, for example the `name` of an
  image returned by the [`fptcloud_image`](../data-sources/image) data source. Changing this forces
  a new resource to be created.
* `subnet_id` - (Required) The ID of the subnet the instance is connected to. The instance gets
  its private IP address from this subnet. Changing this forces a new resource to be created.
* `storage_size_gb` - (Required) The size of the root disk, in GB. Must be at least `1`. Can be
  updated in place, but it can only be **increased**: a smaller value is rejected when you run
  `terraform plan`. See [Updating an instance](#updating-an-instance).
* `storage_policy_id` - (Required) The ID of the storage policy (disk performance tier) of the
  root disk. Use the `id` of a [`fptcloud_storage_policy`](../data-sources/storage_policy) data
  source. Can be updated in place: the root disk is moved to the new policy.
* `status` - (Required) The power state you want the instance to have. Valid values are
  `POWERED_ON` and `POWERED_OFF`. Can be updated in place to power the instance on or off.
* `flavor_name` - (Optional) The name of the flavor, which sets the number of vCPUs and the amount
  of RAM (and the GPU, for GPU flavors). Use the `name` of a flavor returned by the
  [`fptcloud_flavor`](../data-sources/flavor) data source. Although the argument is optional, you
  should always set it. Can be updated in place: changing it resizes the instance. Switching to or
  from a GPU flavor attaches or detaches the GPU as part of the same resize.
* `ssh_key` - (Optional) The SSH public key, in OpenSSH format (for example the content of
  `~/.ssh/id_rsa.pub`), installed on the instance for login. Exactly one of `ssh_key` or
  `password` must be set. Changing this forces a new resource to be created.
* `password` - (Optional) The login password of the instance. Exactly one of `ssh_key` or
  `password` must be set. Changing this forces a new resource to be created. Pass it through a
  variable declared with `sensitive = true` so that Terraform hides it in plan output.
* `private_ip` - (Optional) A specific private IP address for the instance, inside the CIDR of
  `subnet_id`. When omitted, FPT Cloud assigns a free address from the subnet. Only used when the
  instance is created; set it once and do not change it afterwards.
* `public_ip` - (Optional) A floating IP address to attach to the instance when it is created.
  Only used when the instance is created. To attach, move or detach a public IP later, use
  [`fptcloud_floating_ip_association`](floating_ip_association) instead and leave this argument
  unset.
* `security_group_ids` - (Optional) A set of security group IDs to attach when the instance is
  created. Only used when the instance is created. We recommend that you leave this argument unset
  and attach security groups with the `apply_to` argument of
  [`fptcloud_security_group`](security_group) instead; see
  [Attaching security groups](#attaching-security-groups).
* `instance_group_id` - (Optional) The ID of the [instance group](instance_group) to place the
  instance in. Only used when the instance is created.
* `tag_ids` - (Optional) A set of tag IDs to attach to the instance. Use the `id` of
  [`fptcloud_tagging`](tagging) resources or data sources. Can be updated in place.
* `gpu_plan` - (Optional) The billing plan of the GPU. Valid values are `hold` (reserved) and
  `detach` (pay as you go). Only meaningful while the instance has a GPU flavor. Can be updated in
  place.
* `gpu_name` - (Optional) The GPU you expect the flavor to provide, as reported by the `gpu_name`
  attribute of the [`fptcloud_flavor`](../data-sources/flavor) data source. When set, FPT Cloud
  checks it against `flavor_name` on creation and on every resize, and rejects the request if the
  flavor has a different GPU or no GPU at all. Leave it unset to let the flavor alone decide.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the instance.
* `private_ip` - The private IP address of the instance.
* `public_ip` - The floating IP address attached to the instance, if any.
* `gpu_name` - The GPU attached to the instance. Empty for instances without a GPU.
* `vm_type` - The type of the instance: `cpu`, or `gpu` when a GPU is attached.
* `is_nvme` - `true` when the flavor gives the instance a physical NVMe disk. Such an instance does
  not use `storage_policy_id` and cannot be resized.
* `created_at` - The time the instance was created.

## Updating an instance

The table below shows what happens when you change an argument of an existing instance.

| Argument | Effect of a change |
|---|---|
| `name` | Renamed in place. |
| `status` | Powered on or off in place. |
| `flavor_name`, `gpu_name` | Resized in place (vCPU, RAM and GPU). |
| `gpu_plan` | GPU billing plan changed in place. |
| `storage_size_gb` | Root disk grown in place. It cannot shrink. |
| `storage_policy_id` | Root disk moved to the new storage policy in place. |
| `tag_ids` | Tags updated in place. |
| `vpc_id`, `subnet_id`, `image_name`, `ssh_key`, `password` | The instance is **destroyed and recreated**. Data on the root disk is lost. |
| `private_ip`, `public_ip`, `security_group_ids`, `instance_group_id` | Not applied to an existing instance. Use the dedicated resources described above. |

!> **Warning:** Always read the plan before you apply. A line that ends with
`# forces replacement` means the instance and the data on its root disk will be deleted.

### Resizing vCPU and RAM

Change `flavor_name` and apply. Terraform waits until the resize is complete. FPT Cloud may refuse
a resize in some situations, for example when the instance has a physical NVMe disk
(`is_nvme = true`), or when the instance must be powered off for the resize. If the resize is
refused because of the power state, set `status = "POWERED_OFF"` and apply, then change
`flavor_name` and apply again, and finally set `status = "POWERED_ON"`. To learn more, see
[Resize an Instance](https://docs.fptcloud.com/docs/fpt-cloud-server/instance/guides/guide-resize/).

### Growing the root disk

Increase `storage_size_gb` and apply. The disk is grown in place and Terraform waits until it
reports the new size. After that, go into the operating system of the instance to expand the
corresponding partition (see [Manage storage disks](https://docs.fptcloud.com/docs/fpt-cloud-server/instance/guides/guide-storage-disk/) in the FPT Cloud documentation). FPT Cloud refuses a root disk resize
when the instance has a snapshot, is suspended, or has a GPU attached.

The root disk belongs to `fptcloud_instance`. Do not declare a [`fptcloud_storage`](storage)
resource for it: both resources would then manage the same disk and overwrite each other's size and
policy. Use `fptcloud_storage` only for additional data disks.

The [Scaling and Resizing](../guides/scaling-and-resizing) guide shows a full walkthrough.

## Attaching security groups

Attach security groups through the `apply_to` argument of
[`fptcloud_security_group`](security_group), as in the
[Internet example](#instance-reachable-from-the-internet) above, rather than through
`security_group_ids` on the instance. The reasons:

* `security_group_ids` is only applied when the instance is created; later changes are not sent to
  FPT Cloud.
* FPT Cloud reports every security group that applies to the instance, including groups attached
  through `apply_to`. Listing them in `security_group_ids` as well makes Terraform show a change on
  every plan.

Reference the private IP through the [`fptcloud_instance`](../data-sources/instance) data source
(`data.fptcloud_instance.app.private_ip`). The data source reads the instance after it has been
created, and again on every plan. A newly created instance can report an empty `private_ip` for a
short time; if `apply_to` ends up empty, run `terraform apply` again.

~> **Important:** FPT Cloud does not delete a security group that is still attached to instances.
Before you destroy a security group together with its instances, set `apply_to = []` on the group
and run `terraform apply`, then run `terraform destroy`. See
[Deleting a security group](security_group#deleting-a-security-group) for details.

## Timeouts

This resource has no `timeouts` block. Terraform waits for creation, resizing and deletion for up to
the `timeout` set on the [provider](../index), in minutes (default `15`). Large instances or busy
periods can take longer; in that case raise the provider timeout, for example `timeout = 30`.

## Import

An existing instance can be imported by ID or by name, using the VPC ID and either of these
formats:

* `vpc/<vpc_id>/instance/<instance_id>`
* `vpc/<vpc_id>/instance_name/<instance_name>`

```shell
terraform import fptcloud_instance.app vpc/<vpc_id>/instance/<instance_id>
terraform import fptcloud_instance.app vpc/<vpc_id>/instance_name/app-server-01
```

With Terraform 1.5 and later, you can use an `import` block instead:

```terraform
import {
  to = fptcloud_instance.app
  id = "vpc/<vpc_id>/instance/<instance_id>"
}
```

Please note the following when you import an instance:

* If several instances share the same name, import by ID.
* An instance that is busy (for example being created, resized or deleted) cannot be imported.
  Wait until the operation is finished and try again.
* Instances with a physical NVMe disk (`is_nvme = true`) cannot be imported.
* `ssh_key` and `password` cannot be read back from FPT Cloud. Set them in your configuration to
  the values the instance was created with. The first plan shows them as added; applying that plan
  only records them in the state and does not change the instance.
* If `image_name` cannot be read back, set it to the image the instance was created from. The
  first apply records it in the state without changing the instance.

~> **Important:** Do not apply a plan that shows `forces replacement` after an import. Adjust your
configuration until the plan shows no replacement.

To let Terraform write the configuration for you (Terraform 1.5+), keep only the `import` block,
run `terraform plan -generate-config-out=generated.tf`, fill in `ssh_key` or `password` (and
`image_name` if it is `null`) in `generated.tf`, and run `terraform plan` again.
