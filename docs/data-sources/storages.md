---
page_title: "fptcloud_storages Data Source - terraform-provider-fptcloud"
subcategory: ""
description: |-
  Use this data source to retrieve all storages in a VPC, including root, local and external disks.
  Results are retrieved page by page, up to page_size storages per API call, so listing a large VPC never requires one request per storage. The attributes of each item match those of the fptcloud_storage data source.
---

# fptcloud_storages (Data Source)

Use this data source to retrieve all storages in a VPC, including root, local and external disks.

Results are retrieved page by page, up to `page_size` storages per API call, so listing a large VPC never requires one request per storage. The attributes of each item match those of the `fptcloud_storage` data source.

## Example Usage

```terraform
data "fptcloud_storages" "example" {
  vpc_id = "your_vpc_id"
}

# Storages that are not attached to any instance
output "unattached_storage_ids" {
  value = [for storage in data.fptcloud_storages.example.storages : storage.id if storage.instance_id == ""]
}
```

## Schema

### Required

- `vpc_id` (String) The ID of the VPC whose storages are retrieved.

### Optional

- `page_size` (Number) The number of storages requested per API call. Must be between `1` and `25`; defaults to `25`. The provider requests successive pages until every storage has been retrieved.

### Read-Only

- `id` (String) The ID of this data source. It is set to the value of `vpc_id`.
- `storages` (List of Object) The storages in the VPC. (see [below for nested schema](#nestedatt--storages))

<a id="nestedatt--storages"></a>
### Nested Schema for `storages`

Read-Only:

- `created_at` (String) The time the storage was created.
- `description` (String) The description of the storage.
- `disk_id` (String) The ID of the disk backing the storage on the underlying platform.
- `encrypted` (Boolean) Whether the storage is encrypted.
- `id` (String) The ID of the storage.
- `instance_id` (String) The ID of the instance the storage is attached to. Empty when the storage is not attached.
- `instance_name` (String) The name of the instance the storage is attached to. Empty when the storage is not attached.
- `name` (String) The name of the storage. May be empty for a root or local disk created together with its instance.
- `size_gb` (Number) The size of the storage, in GB.
- `status` (String) The current status of the storage.
- `storage_policy` (String) The name of the storage policy.
- `storage_policy_id` (String) The ID of the storage policy.
- `tag_ids` (List of String) The IDs of the tags attached to the storage.
- `type` (String) The type of the storage: `ROOT`, `LOCAL` or `EXTERNAL`.
- `vpc_id` (String) The ID of the VPC the storage belongs to.
- `zone_id` (String) The ID of the availability zone of the storage, when the VPC uses zones.
