---
page_title: "fptcloud_storage_policy Data Source - FPT Cloud"
subcategory: "Storage"
description: |-
  Use this data source to list the storage policies available in a VPC and get their IDs.
---

# fptcloud_storage_policy (Data Source)

Use this data source to list the storage policies available in a VPC. A storage policy is the
performance tier of a disk, for example a premium SSD tier or a standard HDD tier. Every disk has
one: the root disk of an instance (the `storage_policy_id` argument of
[`fptcloud_instance`](../resources/instance)) and every data disk (the `storage_policy_id`
argument of [`fptcloud_storage`](../resources/storage)).

The policies offered differ between regions and VPCs. Look a policy up by name with this data
source instead of copying its ID into your configuration.

## Example Usage

The examples on this page use the following VPC lookup:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

### List all storage policies of a VPC

Use this to see which policies you can choose from.

```terraform
data "fptcloud_storage_policy" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}

output "storage_policy_names" {
  value = data.fptcloud_storage_policy.all.storage_policies[*].name
}
```

### Look up a policy by name and use it for a disk

Use this in most configurations: find the policy by its name and pass its `id` to a disk or an
instance.

```terraform
data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Use a storage policy name listed in your VPC
  }
}

resource "fptcloud_storage" "app_data" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-data-disk"
  type              = "EXTERNAL"
  size_gb           = 200
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
}
```

`storage_policies[0]` fails with an `Invalid index` error when no policy matches the filter, which
tells you early that the name is wrong. If you prefer to handle a missing policy yourself, use
`one(data.fptcloud_storage_policy.ssd.storage_policies)`, which returns `null` when nothing matches
and fails when more than one policy matches.

### Filter by part of the name and sort

Use `match_by = "substring"` or `match_by = "re"` when you know only part of the name, and `sort`
to make the order of the results predictable.

```terraform
data "fptcloud_storage_policy" "ssd_tiers" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key      = "name"
    values   = ["SSD"]
    match_by = "substring"
  }

  sort {
    key       = "name"
    direction = "asc"
  }
}

output "ssd_policies" {
  value = data.fptcloud_storage_policy.ssd_tiers.storage_policies
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC whose storage policies are listed. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source. Must not be empty.
* `filter` - (Optional) One or more `filter` blocks as defined below. A policy is returned only
  when it matches every `filter` block.
* `sort` - (Optional) One or more `sort` blocks as defined below. Results are sorted by the first
  block, then by the next one for equal values.

### filter

* `key` - (Required) The attribute to filter on. Valid values are `id` and `name`.
* `values` - (Required) A list of values. A policy matches when its attribute matches at least one
  of them (or all of them, when `all` is `true`).
* `match_by` - (Optional) How values are compared. Valid values are:
    * `exact` (default) - the whole value must be equal. The comparison ignores upper and lower
      case.
    * `substring` - the attribute must contain the value. The comparison is case-sensitive.
    * `re` - the value is a regular expression that the attribute must match. The comparison is
      case-sensitive.
* `all` - (Optional) Set to `true` to require that the attribute matches all of `values` instead of
  at least one. Defaults to `false`.

### sort

* `key` - (Required) The attribute to sort on. Valid values are `id` and `name`.
* `direction` - (Optional) The sort direction. Valid values are `asc` and `desc`. Defaults to
  ascending order.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A unique ID generated for this data source read. It does not identify a storage policy.
* `storage_policies` - The list of storage policies that match the filters. The list is empty
  when nothing matches. Each item has the following attributes:
    * `id` - The ID of the storage policy. Pass it to the `storage_policy_id` argument of
      [`fptcloud_storage`](../resources/storage) or [`fptcloud_instance`](../resources/instance).
    * `name` - The name of the storage policy.
