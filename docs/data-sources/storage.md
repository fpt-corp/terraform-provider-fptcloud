---
page_title: "fptcloud_storage Data Source - FPT Cloud"
subcategory: "Storage"
description: |-
  Use this data source to get information about one storage disk in a VPC, by ID or by name.
---

# fptcloud_storage (Data Source)

Use this data source to read one existing storage disk, by its ID or by its name, without managing
it with Terraform. It returns the disk's type, size, storage policy, the instance it is attached
to, and its tags. Typical uses are reading a disk that another team or another Terraform
configuration manages, and reusing its storage policy for new disks.

The data source can return any disk of the VPC, including root disks (type `ROOT`) and `LOCAL`
disks. To list every disk of a VPC, use the [`fptcloud_storages`](storages) data source. To create
and manage a disk, use the [`fptcloud_storage`](../resources/storage) resource.

If no disk matches, Terraform reports an error.

## Example Usage

The examples on this page use the following VPC lookup:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

### Look up a disk by name

Use this when you know the disk's name.

```terraform
data "fptcloud_storage" "db" {
  vpc_id = data.fptcloud_vpc.example.id
  name   = "db-data-disk"
}

output "db_disk_attached_to" {
  value = data.fptcloud_storage.db.instance_id
}
```

### Look up a disk by ID

Use this when you have the disk's ID, for example from another configuration's output. Looking a
disk up by ID always returns that exact disk.

```terraform
variable "db_disk_id" {
  type = string
}

data "fptcloud_storage" "db" {
  vpc_id = data.fptcloud_vpc.example.id
  id     = var.db_disk_id
}

output "db_disk_size_gb" {
  value = data.fptcloud_storage.db.size_gb
}
```

### Create a new disk with the same storage policy

Use this to give a new disk the same performance tier as an existing one.

```terraform
data "fptcloud_storage" "db" {
  vpc_id = data.fptcloud_vpc.example.id
  name   = "db-data-disk"
}

resource "fptcloud_storage" "db_backup" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "db-backup-disk"
  type              = "EXTERNAL"
  size_gb           = data.fptcloud_storage.db.size_gb
  storage_policy_id = data.fptcloud_storage.db.storage_policy_id
}
```

## Argument Reference

The following arguments are supported. Exactly one of `id` or `name` must be set.

* `vpc_id` - (Required) The ID of the VPC the disk belongs to. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source.
* `id` - (Optional) The ID of the disk to look up. Must not be empty.
* `name` - (Optional) The name of the disk to look up. Must not be empty.

-> **Note:** If several disks with the same name have existed in the VPC, including disks that were
deleted, a lookup by name returns the most recent one, which can be a deleted disk. Keep disk names
unique, or look the disk up by `id`. You can also use the [`fptcloud_storages`](storages) data
source and filter on `status`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the disk.
* `name` - The name of the disk.
* `vpc_id` - The ID of the VPC the disk belongs to.
* `type` - The type of the disk: `ROOT`, `LOCAL` or `EXTERNAL`.
* `size_gb` - The size of the disk, in GB.
* `storage_policy` - The name of the disk's storage policy.
* `storage_policy_id` - The ID of the disk's storage policy. You can pass it to the
  `storage_policy_id` argument of [`fptcloud_storage`](../resources/storage) or
  [`fptcloud_instance`](../resources/instance).
* `instance_id` - The ID of the instance the disk is attached to. Empty when the disk is not
  attached.
* `tag_ids` - The IDs of the tags attached to the disk.
* `created_at` - The time the disk was created.
