---
page_title: "fptcloud_storages Data Source - FPT Cloud"
subcategory: "Storage"
description: |-
  Use this data source to list every storage disk in a VPC, including root, local and external disks.
---

# fptcloud_storages (Data Source)

Use this data source to list every storage disk in a VPC: the root disks of instances (type
`ROOT`), `LOCAL` disks and `EXTERNAL` disks. Each item carries the same information as the
[`fptcloud_storage`](storage) data source, plus the disk's status, the name of the instance it is
attached to, and a few more details.

Typical uses are reporting on disks, finding disks that are not attached to any instance, and
listing the disks of one instance. This data source has no `filter` or `sort` block: select the
disks you need with a `for` expression, as shown below. To read a single disk by ID or name, use
the [`fptcloud_storage`](storage) data source instead.

## Example Usage

The examples on this page use the following lookups:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_storages" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}
```

### List every disk in a VPC

Use this to get an overview of the disks in a VPC.

```terraform
output "disks" {
  value = {
    for disk in data.fptcloud_storages.all.storages :
    disk.id => "${disk.name} (${disk.type}, ${disk.size_gb} GB, ${disk.status})"
  }
}
```

### Find data disks that are not attached

Use this to find `EXTERNAL` disks that are not attached to any instance, for example to review
unused capacity.

```terraform
output "detached_disk_ids" {
  value = [
    for disk in data.fptcloud_storages.all.storages : disk.id
    if disk.type == "EXTERNAL" && disk.instance_id == ""
  ]
}
```

### List the disks of one instance

Use this to see every disk of an instance, including its root disk.

```terraform
data "fptcloud_instance" "app" {
  vpc_id = data.fptcloud_vpc.example.id
  name   = "app-server-01"
}

output "app_disks" {
  value = [
    for disk in data.fptcloud_storages.all.storages : {
      name    = disk.name
      type    = disk.type
      size_gb = disk.size_gb
    }
    if disk.instance_id == data.fptcloud_instance.app.id
  ]
}
```

### Pick a single disk

Use `one()` when you expect exactly one match. It returns `null` when nothing matches and fails
when more than one disk matches. Indexing with `[0]` instead fails when the list is empty.

```terraform
locals {
  db_disk = one([
    for disk in data.fptcloud_storages.all.storages : disk
    if disk.name == "db-data-disk" && disk.status == "ENABLED"
  ])
}

output "db_disk_policy" {
  value = local.db_disk == null ? null : local.db_disk.storage_policy
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC whose disks are listed. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source.
* `page_size` - (Optional) How many disks the provider requests from FPT Cloud at a time. Must be
  between `1` and `25`. Defaults to `25`. The provider keeps requesting until it has every disk, so
  this setting does not limit the result; you rarely need to change it.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of this data source. It is the same as `vpc_id`.
* `storages` - The list of disks in the VPC. The list is empty when the VPC has no disks. Each
  item has the following attributes:
    * `id` - The ID of the disk.
    * `vpc_id` - The ID of the VPC the disk belongs to.
    * `name` - The name of the disk. Can be empty for a root or `LOCAL` disk that was created
      together with its instance.
    * `type` - The type of the disk: `ROOT`, `LOCAL` or `EXTERNAL`.
    * `size_gb` - The size of the disk, in GB, rounded up to a whole number.
    * `status` - The current status of the disk. A disk that is ready to use is `ENABLED`.
    * `storage_policy` - The name of the disk's storage policy.
    * `storage_policy_id` - The ID of the disk's storage policy.
    * `instance_id` - The ID of the instance the disk is attached to. Empty when the disk is not
      attached.
    * `instance_name` - The name of the instance the disk is attached to. Empty when the disk is
      not attached.
    * `tag_ids` - The IDs of the tags attached to the disk.
    * `description` - The description of the disk.
    * `disk_id` - The ID of the volume that backs the disk in the underlying infrastructure. You
      do not need it for Terraform; use `id` to refer to the disk.
    * `encrypted` - `true` when the disk is encrypted.
    * `zone_id` - The ID of the availability zone of the disk, when the VPC uses zones.
    * `created_at` - The time the disk was created.
