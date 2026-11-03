---
page_title: "fptcloud_snapshot Resource - FPT Cloud"
subcategory: "Compute"
description: |-
  Manages an instance snapshot: a point-in-time copy of every disk attached to an instance.
---

# fptcloud_snapshot (Resource)

Manages an instance snapshot. A snapshot captures every disk attached to an instance, the root
disk and the data disks, at one point in time. Take one before a risky change, such as an operating
system upgrade, so that you can go back to the saved state.

The snapshot is crash-consistent: the disks are captured as they are at that moment, without
pausing or flushing the file systems inside the instance. For the most reliable result, stop
writes in your applications, or power the instance off, before you take the snapshot.

Each `fptcloud_snapshot` resource is one snapshot. Terraform takes it when the resource is created
and deletes it when the resource is destroyed. A snapshot cannot be renamed or moved: changing
`vpc_id`, `instance_id`, `name` or `include_ram` deletes the snapshot and takes a new one. Only
`tag_ids` can be changed in place.

To learn more, see
[Snapshots](https://docs.fptcloud.com/docs/fpt-cloud-server/snapshots/overview/) in the FPT Cloud
documentation.

## Differences between VPC types

| | General VPC | Specific VPC |
|---|---|---|
| Snapshots per instance | Several | One. Taking a new snapshot replaces the instance's previous one. |
| `name` | Used. Set it. | Not stored by FPT Cloud. |
| `include_ram` | Ignored | Used |
| `infra_snapshot_id` | Set once the snapshot is ready | Empty |

~> **Important:** In a Specific VPC, declare at most one `fptcloud_snapshot` resource per instance.
A second one replaces the snapshot the first one manages.

## Example Usage

The examples on this page use the following VPC lookup and an instance named `app-server-01`.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_instance" "app" {
  vpc_id = data.fptcloud_vpc.example.id
  name   = "app-server-01"
}
```

### Snapshot before an upgrade

Use this to save the state of an instance before you change it.

```terraform
resource "fptcloud_snapshot" "before_upgrade" {
  vpc_id      = data.fptcloud_vpc.example.id
  instance_id = data.fptcloud_instance.app.id
  name        = "app-before-upgrade"
}

output "snapshot_status" {
  value = fptcloud_snapshot.before_upgrade.status
}
```

### Snapshot with tags

Use this to label snapshots, for example to find them by owner or purpose in the console. Tags are
created with the [`fptcloud_tagging`](tagging) resource and can be changed later without taking a
new snapshot.

```terraform
resource "fptcloud_tagging" "purpose" {
  key          = "purpose"
  value        = "pre-upgrade"
  scope_type   = "VPC"
  resource_ids = [data.fptcloud_vpc.example.id]
}

resource "fptcloud_snapshot" "before_upgrade" {
  vpc_id      = data.fptcloud_vpc.example.id
  instance_id = data.fptcloud_instance.app.id
  name        = "app-before-upgrade"
  tag_ids     = [fptcloud_tagging.purpose.id]
}
```

### Snapshot with memory in a Specific VPC

Use this in a Specific VPC to capture the instance's memory together with its disks.

```terraform
resource "fptcloud_snapshot" "with_memory" {
  vpc_id      = data.fptcloud_vpc.example.id
  instance_id = data.fptcloud_instance.app.id
  include_ram = true
}
```

### Taking a new snapshot

Editing an argument other than `tag_ids` does not add a snapshot: it replaces the one the resource
manages. To keep several snapshots of an instance in a General VPC, declare several resources. To
take a fresh snapshot without changing the configuration, replace the resource:

```shell
terraform apply -replace="fptcloud_snapshot.before_upgrade"
```

Terraform first deletes the old snapshot, then takes the new one.

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC that contains the instance. Use the `id` of the
  [`fptcloud_vpc`](../data-sources/vpc) data source. Must not be empty. Changing this forces a new
  resource to be created.
* `instance_id` - (Required) The ID of the instance to snapshot. Use the `id` of a
  [`fptcloud_instance`](instance) resource or data source. Must not be empty. Changing this forces
  a new resource to be created.
* `name` - (Optional) The name of the snapshot. It must be unique within the VPC and contain 1 to
  30 characters: letters, digits, underscores (`_`), dashes (`-`), dots (`.`) and spaces. Leading
  and trailing spaces are removed. Set it when the instance is in a General VPC. In a Specific VPC,
  FPT Cloud does not store a snapshot name. If you leave it unset, Terraform records the name
  FPT Cloud reports, if any. Changing this forces a new resource to be created.
* `include_ram` - (Optional) Set to `true` to capture the instance's memory together with its
  disks. Only used in a Specific VPC; ignored in a General VPC. Defaults to `false`. Changing this
  forces a new resource to be created.
* `tag_ids` - (Optional) A set of tag IDs to attach to the snapshot. Use the `id` of
  [`fptcloud_tagging`](tagging) resources. Can be updated in place. To remove every tag, set
  `tag_ids = []` or remove the argument.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the snapshot.
* `status` - The current status of the snapshot. A ready snapshot is `ACTIVE` in a General VPC and
  `AVAILABLE` in a Specific VPC.
* `size_gb` - The total size of the snapshot, in GB. This is the size of the saved data, not the
  size of the instance's disks.
* `snapshot_type` - The type of the snapshot, as reported by FPT Cloud.
* `created_at` - The time the snapshot was taken.
* `created_by` - The user who created the snapshot.
* `infra_snapshot_id` - The ID of the snapshot in the underlying infrastructure. Empty in a
  Specific VPC. You do not need it for Terraform; use `id` to refer to the snapshot.
* `tags` - The tags attached to the snapshot. Each item has the following attributes:
    * `id` - The ID of the tag.
    * `key` - The key of the tag.
    * `value` - The value of the tag.

## Timeouts

This resource has no `timeouts` block. Terraform waits for a snapshot to be ready, or to be
deleted, for up to the `timeout` set on the [provider](../index), in minutes (default `15`).
Snapshots of instances with large disks can take longer; in that case raise the provider timeout,
for example `timeout = 30`.

If FPT Cloud reports that the snapshot failed, the apply stops at once with the reported status.
If the snapshot is not ready when the timeout runs out, Terraform still records it in the state and
marks it as tainted, so that you can refresh it, or destroy it, instead of losing track of it. The
next apply replaces a tainted snapshot.

## Import

An existing snapshot can be imported with an ID in the format
`vpc/<vpc_id>/snapshot/<snapshot_id>`.

```shell
terraform import fptcloud_snapshot.before_upgrade vpc/<vpc_id>/snapshot/<snapshot_id>
```

With Terraform 1.5 and later, you can use an `import` block instead:

```terraform
import {
  to = fptcloud_snapshot.before_upgrade
  id = "vpc/<vpc_id>/snapshot/<snapshot_id>"
}
```

Please note the following when you import a snapshot:

* `name`, `instance_id` and `tag_ids` are read from FPT Cloud. Make your configuration match them.
* FPT Cloud does not always report the instance a snapshot was taken from, or (in a Specific VPC)
  its name. When the imported state has an empty `instance_id` or `name`, the first plan shows
  `forces replacement`. In that case, add
  `lifecycle { ignore_changes = [instance_id, name] }` to the resource.

~> **Important:** Do not apply a plan that shows `forces replacement` after an import: it deletes
the imported snapshot. Adjust your configuration until the plan shows no replacement.

## Snapshots deleted outside Terraform

If a snapshot is deleted outside Terraform, for example in the FPT Cloud console, Terraform removes
it from the state on the next refresh, and the next apply takes a new snapshot. Destroying a
resource whose snapshot is already gone succeeds.
