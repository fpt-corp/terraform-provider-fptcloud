---
page_title: "fptcloud_storage Resource - FPT Cloud"
subcategory: "Storage"
description: |-
  Manages a storage disk (an additional block storage volume) in an FPT Cloud VPC.
---

# fptcloud_storage (Resource)

Manages a storage disk: a block storage volume that is separate from the root disk an instance
boots from. Use storage disks to keep application data apart from the operating system, to add
capacity to an instance without rebuilding it, or to move data from one instance to another.

With this resource you can create a disk, attach it to an instance, move it to another instance or
detach it, grow it, rename it, change its storage policy, change its tags, and delete it. The
performance tier of a disk comes from its storage policy, which you look up with the
[`fptcloud_storage_policy`](../data-sources/storage_policy) data source.

To learn more, see
[Storage Disks](https://docs.fptcloud.com/docs/fpt-cloud-server/storage-disks/overview/) in the
FPT Cloud documentation.

~> **Important:** Do not use this resource for the root disk of an instance. The root disk belongs
to [`fptcloud_instance`](instance) and is managed with its `storage_size_gb` and
`storage_policy_id` arguments. If both resources managed the same disk, they would overwrite each
other's size and policy. Watch out in a General VPC, where the root disk can be listed with type
`LOCAL` and looks like any other disk.

## Disk types

The `type` argument decides how the disk relates to instances. Choose it carefully: it cannot be
changed after the disk is created.

| | `EXTERNAL` | `LOCAL` |
|---|---|---|
| VPC type | General VPC and Specific VPC | Specific VPC |
| Attach to an instance at creation | Optional | Required (`instance_id`) |
| Move to another instance or detach later | Yes | No |
| Can be imported | Yes | No |

Use `EXTERNAL` unless you have a specific reason to use `LOCAL`. A `LOCAL` disk is created on an
instance and stays on that instance until you delete it.

## Example Usage

The examples on this page share the following lookups and instance. Catalog names such as image,
flavor and storage policy names differ between regions and VPCs, so replace the example values
with names listed in your VPC.

```terraform
variable "ssh_public_key" {
  type = string
}

data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Use a storage policy name listed in your VPC
  }
}

data "fptcloud_subnet" "app" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["app-subnet"]
  }
}

resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = data.fptcloud_subnet.app.subnets[0].id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = var.ssh_public_key
  status            = "POWERED_ON"
}
```

### Standalone data disk

Use this when you want a disk that is not attached to any instance yet, for example to prepare it
before the instance exists. You can attach it later by setting `instance_id`.

```terraform
resource "fptcloud_storage" "shared" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "shared-data-disk"
  type              = "EXTERNAL"
  size_gb           = 100
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
}
```

### Data disk attached to an instance

Use this for the most common case: an extra disk for an instance. Terraform creates the disk and
waits until it is attached to the instance. After that, format and mount the disk from inside the
operating system.

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

### Tagged data disk

Use this to label a disk, for example by environment or cost center. Tags are created with the
[`fptcloud_tagging`](tagging) resource and attached by ID.

```terraform
resource "fptcloud_tagging" "env" {
  key          = "env"
  value        = "production"
  scope_type   = "VPC"
  resource_ids = [data.fptcloud_vpc.example.id]
}

resource "fptcloud_storage" "db" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "db-data-disk"
  type              = "EXTERNAL"
  size_gb           = 500
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  instance_id       = fptcloud_instance.app.id
  tag_ids           = [fptcloud_tagging.env.id]
}
```

### Several disks on one instance

Use this when an instance needs more than one data disk, for example one for data and one for
logs. In a Specific VPC, FPT Cloud attaches only one disk at a time to an instance. If Terraform
creates several disks for the same instance in parallel, the extra attachments fail: the disks are
created but stay detached, and Terraform waits until the provider `timeout` runs out. Chain the
disks with `depends_on` so that they are created one after another. In a General VPC the chain is
not needed, but it does no harm.

```terraform
resource "fptcloud_storage" "app_data" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-data-disk"
  type              = "EXTERNAL"
  size_gb           = 200
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  instance_id       = fptcloud_instance.app.id
}

resource "fptcloud_storage" "app_logs" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-logs-disk"
  type              = "EXTERNAL"
  size_gb           = 50
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  instance_id       = fptcloud_instance.app.id

  # In a Specific VPC, attach the disks of one instance one at a time.
  depends_on = [fptcloud_storage.app_data]
}
```

As an alternative to `depends_on`, run `terraform apply -parallelism=1`. That serializes every
operation in the run, so it is slower.

### Local disk in a Specific VPC

Use this only in a Specific VPC, when you need a disk that always stays on one instance. A `LOCAL`
disk requires `instance_id` and cannot be moved or detached later.

```terraform
resource "fptcloud_storage" "scratch" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "scratch-disk"
  type              = "LOCAL"
  size_gb           = 100
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  instance_id       = fptcloud_instance.app.id
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC to create the disk in. Use the `id` of the
  [`fptcloud_vpc`](../data-sources/vpc) data source. Do not change it after the disk is created:
  a disk cannot be moved to another VPC.
* `name` - (Required) The name of the disk. Must not be empty. For an `EXTERNAL` disk, the name
  must not be used by another existing disk in the VPC; otherwise creation fails with
  `Storage name ... already exists`. Can be updated in place (see
  [Updating a disk](#updating-a-disk) for a Specific VPC limitation).
* `type` - (Required) The type of the disk. Valid values are `EXTERNAL` and `LOCAL`; see
  [Disk types](#disk-types). Any other value is rejected when you apply. Cannot be changed after
  the disk is created: the apply fails with `Storage type can not be changed`. To change the type,
  create a new disk.
* `size_gb` - (Required) The size of the disk, in GB. Must be at least `1`. Can be updated in place
  to grow the disk. FPT Cloud does not support shrinking a disk, so only increase this value; the
  provider does not check it when you run `terraform plan`.
* `storage_policy_id` - (Required) The ID of the storage policy (performance tier) of the disk. Use
  the `id` of a [`fptcloud_storage_policy`](../data-sources/storage_policy) data source. Can be
  updated in place to move the disk to another policy.
* `instance_id` - (Optional) The ID of the instance to attach the disk to. Use the `id` of a
  [`fptcloud_instance`](instance) resource or data source. Required when `type` is `LOCAL`.
  For an `EXTERNAL` disk, it can be updated in place: set another instance ID to move the disk,
  or remove the argument to detach the disk. For a `LOCAL` disk, changing it makes the apply fail.
  When this argument is not set, Terraform keeps the disk detached: if someone attaches it outside
  Terraform, the next apply detaches it.
* `tag_ids` - (Optional) A set of tag IDs to attach to the disk. Use the `id` of
  [`fptcloud_tagging`](tagging) resources. Can be updated in place. If you leave it unset,
  Terraform does not manage the tags and the disk keeps the tags it already has.
* `timeouts` - (Optional) A `timeouts` block as defined in [Timeouts](#timeouts).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the disk.
* `created_at` - The time the disk was created.

## Updating a disk

No argument of this resource forces the disk to be replaced. The table below shows what happens
when you change an argument of an existing disk.

| Argument | Effect of a change |
|---|---|
| `size_gb` | Disk grown in place. It cannot shrink. |
| `storage_policy_id` | Disk moved to the new storage policy in place. |
| `name` | Disk renamed in place. |
| `instance_id` | `EXTERNAL` disk moved to the new instance, or detached if removed. Refused for a `LOCAL` disk. |
| `tag_ids` | Tags updated in place. |
| `type` | Refused: the apply fails. |
| `vpc_id` | Not supported. Do not change it. |

### Growing a disk

Increase `size_gb` and apply. In a General VPC, Terraform waits until the disk reports the new
size. After that, extend the partition and file system from inside the operating system (for
example with `growpart` and `resize2fs` on Linux). The
[Scaling and Resizing](../guides/scaling-and-resizing) guide shows a full walkthrough.

### Changing the storage policy

Set `storage_policy_id` to the ID of another policy and apply. A common pattern is to look the
policy up by name, so that changing tiers is a one-word change:

```terraform
data "fptcloud_storage_policy" "selected" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Change to another policy name listed in your VPC
  }
}

resource "fptcloud_storage" "app_data" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-data-disk"
  type              = "EXTERNAL"
  size_gb           = 200
  storage_policy_id = data.fptcloud_storage_policy.selected.storage_policies[0].id
  instance_id       = fptcloud_instance.app.id
}
```

### Moving or detaching a disk

To move an `EXTERNAL` disk, set `instance_id` to the ID of the other instance and apply. To detach
it, remove `instance_id` and apply. The data on the disk is preserved in both cases.

~> **Important:** Unmount the disk inside the operating system before you move or detach it (for
example `sudo umount /dev/vdb` on Linux). Detaching a disk that is still mounted can cause data
loss.

### Changing an attached disk in a Specific VPC

In a Specific VPC, FPT Cloud does not allow changing the size, name or storage policy of an
`EXTERNAL` disk while it is attached to an instance; the apply fails. Proceed in three steps:

1. Remove `instance_id` and apply, to detach the disk.
2. Change `size_gb`, `name` or `storage_policy_id` and apply.
3. Set `instance_id` again and apply, to attach the disk.

In a General VPC you can change these arguments while the disk is attached.

## Timeouts

Terraform waits for a disk to be created, attached or updated for up to the `timeout` set on the
[provider](../index), in minutes (default `15`).

The `timeouts` block of this resource supports one setting, which is only used for `EXTERNAL`
disks:

* `create` - (Default `15m`) If the request to create the disk gets no answer (for example because
  of a network or gateway timeout), FPT Cloud may still have accepted it. Instead of failing,
  Terraform looks for the new disk by name for up to this long, and never sends the create request
  a second time. Once the disk is found, the wait for it to become ready is still bounded by the
  provider `timeout`.

```terraform
resource "fptcloud_storage" "archive" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "archive-disk"
  type              = "EXTERNAL"
  size_gb           = 1000
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id

  timeouts {
    create = "30m"
  }
}
```

## Import

An existing `EXTERNAL` disk can be imported with an ID in the format
`vpc/<vpc_id>/storage/<storage_id>`. Both IDs are UUIDs.

```shell
terraform import fptcloud_storage.app_data vpc/<vpc_id>/storage/<storage_id>
```

With Terraform 1.5 and later, you can use an `import` block instead:

```terraform
import {
  to = fptcloud_storage.app_data
  id = "vpc/<vpc_id>/storage/<storage_id>"
}
```

To let Terraform write the configuration for you, keep only the `import` block and run
`terraform plan -generate-config-out=generated.tf`.

Please note the following when you import a disk:

* Only a disk of type `EXTERNAL` whose status is `ENABLED` can be imported. A disk that is still
  being created cannot be imported yet; wait and try again.
* `LOCAL` disks and root disks cannot be imported. Manage a root disk through its
  [`fptcloud_instance`](instance).
* If the disk is not found, check the VPC ID, and check that the provider `region` and
  `tenant_name` are the ones the disk belongs to.
* All arguments are read from FPT Cloud, so a configuration that matches the disk plans
  `No changes`.

~> **Important:** If the disk is attached to an instance, set `instance_id` to that instance's ID
in your configuration. Without it, the first plan shows `- instance_id = "..." -> null`, and
applying that plan **detaches** the disk.

## Troubleshooting

* **`Storage name ... already exists`**: another disk in the VPC already uses this name. Choose a
  different `name`.
* **`Storage not found. Please try again later`** on every plan: the disk was deleted outside
  Terraform. Terraform keeps it in the state and reports an error instead of creating it again.
  Remove it from the state with `terraform state rm fptcloud_storage.<name>`, then apply to create
  a new disk if you still need one.
* **Creation runs until the timeout, and the next plan replaces the disk**: when a disk is not
  ready or not attached within the provider `timeout`, Terraform records it as tainted. Check the
  disk in the FPT Cloud console. If it exists and is healthy, run
  `terraform untaint fptcloud_storage.<name>` to keep it; the next apply attaches it if it is still
  detached. In a Specific VPC, see [Several disks on one instance](#several-disks-on-one-instance)
  for the most common cause.
