---
page_title: "fptcloud_vgpu Data Source - FPT Cloud"
subcategory: "Compute"
description: |-
  Use this data source to list the vGPU types available in a VPC.
---

# fptcloud_vgpu (Data Source)

Use this data source to list the vGPU types available in one of your VPCs. A vGPU type describes a
GPU model that FPT Cloud can attach to workloads, with its memory size and status.

How vGPUs relate to other resources:

* **Instances** do not take a vGPU directly. An instance gets a GPU from its flavor: choose a GPU
  flavor (type `GPU_SIZE`) with the [`fptcloud_flavor`](flavor) data source and pass its name to
  `flavor_name` of [`fptcloud_instance`](../resources/instance). The `gpu_id` and `gpu_name` fields
  of a GPU flavor describe the vGPU behind it.
* **Managed Kubernetes** worker pools take a vGPU ID: pass the `id` of a vGPU from this data source
  to `vgpu_id` of a worker pool in
  [`fptcloud_managed_kubernetes_engine_v1`](../resources/managed_kubernetes_engine_v1).

The vGPU types you can use depend on your region and VPC, so look them up instead of hard-coding
their IDs.

## Example Usage

All examples use this VPC lookup:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

### List every vGPU type in a VPC

Use this to see which vGPU types exist in your VPC. Without a `filter`, the data source returns every
vGPU type of the VPC.

```terraform
data "fptcloud_vgpu" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}

output "vgpus" {
  value = [
    for g in data.fptcloud_vgpu.all.vgpus : {
      name         = g.name
      display_name = g.display_name
      memory_gb    = g.memory
      status       = g.status
    }
  ]
}
```

### Find a vGPU type by name

Use this when you know the vGPU name. Matching on a text field with the default
`match_by = "exact"` ignores upper and lower case.

```terraform
data "fptcloud_vgpu" "a30" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["nvidia_a30"] # Use a vGPU name listed in your VPC
  }
}

output "a30_vgpu_id" {
  value = data.fptcloud_vgpu.a30.vgpus[0].id
}
```

Pass `data.fptcloud_vgpu.a30.vgpus[0].id` to `vgpu_id` of a worker pool in
[`fptcloud_managed_kubernetes_engine_v1`](../resources/managed_kubernetes_engine_v1).

### Find vGPU types with enough memory

Use several values to accept more than one vGPU type, and sort to put the largest first.

```terraform
data "fptcloud_vgpu" "large" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "memory"
    values = [40, 80] # GPU memory in GB
  }

  sort {
    key       = "memory"
    direction = "desc"
  }
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC whose vGPU types are listed. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source. Must not be empty.
* `filter` - (Optional) One or more `filter` blocks as defined below. When you set several blocks,
  a vGPU must match all of them.
* `sort` - (Optional) One or more `sort` blocks as defined below. The first block is the primary
  sort order; the next blocks break ties.

### filter

* `key` - (Required) The vGPU field to filter on. Valid values are `created_at`, `display_name`,
  `enable_nvme`, `id`, `is_dedicated`, `memory`, `name`, `parent_id`, `platform`,
  `service_type_id` and `status`. Avoid `service_type_id`: when a vGPU has no service type, the
  current provider version stops with an internal error.
* `values` - (Required) The values to match. A vGPU matches when its field matches at least one of
  the values (or all of them when `all = true`). For `memory`, the values must be whole numbers; for
  `enable_nvme` and `is_dedicated`, they must be `true` or `false`.
* `match_by` - (Optional) How text fields are compared. Valid values are:
  * `exact` (default) - the whole value must be equal, ignoring upper and lower case.
  * `substring` - the field must contain the value. This comparison is case-sensitive.
  * `re` - the value is a regular expression ([RE2 syntax](https://github.com/google/re2/wiki/Syntax))
    that must match somewhere in the field. Use `^` and `$` to match the whole field. Case-sensitive
    unless you start the expression with `(?i)`.

  `match_by` has no effect on number and boolean fields, which are always compared exactly.
* `all` - (Optional) Set to `true` to require that the field matches every value in `values`
  instead of at least one. Defaults to `false`.

### sort

* `key` - (Required) The vGPU field to sort on. Valid values are the same as for `filter.key`. Avoid
  `service_type_id` for the same reason.
* `direction` - (Optional) The sort direction. Valid values are `asc` and `desc`. When omitted, the
  result is sorted in ascending order.

Text fields are sorted character by character and are case-sensitive (upper-case letters come before
lower-case letters).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A random identifier generated each time the data source is read. It does not identify a
  vGPU; use `vgpus[*].id` for that.
* `vgpus` - The vGPU types that match all filters, in the requested sort order. The list is empty
  when nothing matches. Each element has the following attributes:
  * `id` - The ID of the vGPU type. Pass it to `vgpu_id` of a managed Kubernetes worker pool.
  * `name` - The name of the vGPU type, for example `nvidia_a30`.
  * `display_name` - The display name of the vGPU type, for example `NVIDIA Tesla A30`.
  * `memory` - The GPU memory, in GB.
  * `status` - The status of the vGPU type.
  * `is_dedicated` - Whether the vGPU type is dedicated.
  * `platform` - The platform of the vGPU type.
  * `parent_id` - The ID of the parent of the vGPU type.
  * `service_type_id` - The ID of the service type of the vGPU type. Empty when it has none.
  * `enable_nvme` - Whether NVMe is enabled for the vGPU type.
  * `created_at` - The time the vGPU type was created.

## Selecting one vGPU type

* `data.fptcloud_vgpu.example.vgpus[0]` takes the first vGPU type. When nothing matches, the plan
  fails with an "Invalid index" error.
* `one(data.fptcloud_vgpu.example.vgpus)` returns the only vGPU type, returns `null` when nothing
  matches, and fails when more than one vGPU type matches.
