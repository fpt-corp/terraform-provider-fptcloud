---
page_title: "fptcloud_flavor Data Source - FPT Cloud"
subcategory: "Compute"
description: |-
  Use this data source to list the flavors (vCPU, RAM and GPU sizes) available in a VPC and pick one for an instance.
---

# fptcloud_flavor (Data Source)

Use this data source to list the flavors available in one of your VPCs. A flavor is a predefined
instance size: a number of vCPUs, an amount of RAM and, for GPU flavors, a GPU. You choose a flavor
by name when you create an instance with [`fptcloud_instance`](../resources/instance) (argument
`flavor_name`).

Flavor names differ between regions and VPCs, so look them up instead of hard-coding them. Use
`filter` blocks to keep only the flavors you want, and `sort` blocks to order the result.

To learn more, see
[Instance (Virtual Server)](https://docs.fptcloud.com/docs/fpt-cloud-server/instance/overview/)
in the FPT Cloud documentation.

~> **Important:** Do not use `gpu_id`, `gpu_name` or `gpu_memory_gb` as a `filter` or `sort` key.
The provider accepts these keys, but the current version stops with an internal error when it
applies them. Select GPU flavors with `type = "GPU_SIZE"` and narrow them further with a `for`
expression, as shown in [Pick a GPU flavor](#pick-a-gpu-flavor).

## Example Usage

All examples use this VPC lookup. Later examples also reuse data sources declared in earlier ones.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

### List every flavor in a VPC

Use this to see which flavor names exist in your VPC and region before you choose one. Without a
`filter`, the data source returns every flavor of the VPC.

```terraform
data "fptcloud_flavor" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}

output "flavor_names" {
  value = [for f in data.fptcloud_flavor.all.flavors : f.name]
}
```

Run `terraform apply` (or `terraform plan` followed by `terraform output`) and copy a name from
`flavor_names` into your configuration.

### Pick a flavor by name

Use this when you already know the flavor name. Matching on a text field with the default
`match_by = "exact"` ignores upper and lower case.

```terraform
data "fptcloud_flavor" "small" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["2C4G"] # Use a flavor name listed in your VPC
  }
}
```

### Pick a flavor by vCPU and RAM

Use this when you care about the size rather than the name. Several `filter` blocks must all match,
so this example keeps standard (non-GPU) flavors with 2 vCPUs and 4 GB of RAM. `memory_mb` is in MB.

```terraform
data "fptcloud_flavor" "two_by_four" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "cpu"
    values = [2]
  }

  filter {
    key    = "memory_mb"
    values = [4 * 1024]
  }

  filter {
    key    = "type"
    values = ["VM_SIZE"]
  }
}
```

### Use the flavor in an instance

Pass the `name` of the selected flavor to `flavor_name`. `flavors[0]` takes the first match and
makes the plan fail with an "Invalid index" error when nothing matched, so you never create an
instance with an unexpected size.

```terraform
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

resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = data.fptcloud_flavor.small.flavors[0].name
  subnet_id         = fptcloud_subnet.app.id # Defined elsewhere in your configuration
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
}
```

### List GPU flavors

Use this to see the GPU flavors of your VPC, with the GPU behind each one. GPU flavors have the
type `GPU_SIZE`.

```terraform
data "fptcloud_flavor" "gpu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "type"
    values = ["GPU_SIZE"]
  }

  sort {
    key       = "cpu"
    direction = "asc"
  }
}

output "gpu_flavors" {
  value = [
    for f in data.fptcloud_flavor.gpu.flavors : {
      name          = f.name
      cpu           = f.cpu
      memory_mb     = f.memory_mb
      gpu_name      = f.gpu_name
      gpu_memory_gb = f.gpu_memory_gb
      is_nvme       = f.is_nvme
    }
  ]
}
```

### Pick a GPU flavor

Use this to create a GPU instance. Select the GPU flavors, keep the ones with the GPU you want with
a `for` expression, and pass both the flavor `name` and its `gpu_name` to the instance:

* `flavor_name` decides the size and the GPU of the instance.
* `gpu_name` is an optional safety check. When it is set, FPT Cloud verifies that it matches the GPU
  of `flavor_name` and rejects the request otherwise. Taking both values from the same flavor keeps
  them consistent.
* `gpu_plan` sets the GPU billing plan: `hold` (reserved) or `detach` (pay-as-you-go). It only
  applies to GPU flavors.

For GPU instances, also pick an image made for GPUs (`is_gpu = true`, see
[`fptcloud_image`](image)).

```terraform
data "fptcloud_image" "gpu_ubuntu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "is_gpu"
    values = [true]
  }

  filter {
    key      = "name"
    values   = ["Ubuntu"] # Use part of an image name listed in your VPC
    match_by = "substring"
  }
}

locals {
  # Use a GPU name shown in the gpu_name field of your VPC's GPU flavors.
  a30_flavors = [
    for f in data.fptcloud_flavor.gpu.flavors : f
    if f.gpu_name == "NVIDIA Tesla A30"
  ]
  gpu_flavor = local.a30_flavors[0]
}

resource "fptcloud_instance" "gpu" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "gpu-server-01"
  image_name        = data.fptcloud_image.gpu_ubuntu.images[0].name
  flavor_name       = local.gpu_flavor.name
  gpu_name          = local.gpu_flavor.gpu_name
  gpu_plan          = "hold"
  subnet_id         = fptcloud_subnet.app.id # Defined elsewhere in your configuration
  storage_size_gb   = 100
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
}
```

-> **Note:** Check `is_nvme` before you choose a GPU flavor. When it is `true`, the instance runs on
a physical NVMe disk that comes with the flavor: the `storage_policy_id` you set is ignored, FPT
Cloud refuses to resize the instance to another flavor later, and the instance cannot be imported
into Terraform. See [NVMe flavors](#nvme-flavors).

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC whose flavors are listed. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source. Must not be empty.
* `filter` - (Optional) One or more `filter` blocks as defined below. When you set several blocks,
  a flavor must match all of them.
* `sort` - (Optional) One or more `sort` blocks as defined below. The first block is the primary
  sort order; the next blocks break ties.

### filter

* `key` - (Required) The flavor field to filter on. Valid values are `cpu`, `gpu_id`,
  `gpu_memory_gb`, `gpu_name`, `id`, `is_nvme`, `memory_mb`, `name` and `type`. Do not use
  `gpu_id`, `gpu_name` or `gpu_memory_gb`: see the note at the top of this page.
* `values` - (Required) The values to match. A flavor matches when its field matches at least one of
  the values (or all of them when `all = true`). Values are written as strings or numbers; for
  number fields (`cpu`, `memory_mb`) they must be whole numbers, and for `is_nvme` they must be
  `true` or `false`.
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

* `key` - (Required) The flavor field to sort on. Valid values are the same as for `filter.key`.
  Do not use `gpu_id`, `gpu_name` or `gpu_memory_gb`.
* `direction` - (Optional) The sort direction. Valid values are `asc` and `desc`. When omitted, the
  result is sorted in ascending order.

Text fields are sorted character by character and are case-sensitive (upper-case letters come before
lower-case letters).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A random identifier generated each time the data source is read. It does not identify a
  flavor; use `flavors[*].id` for that.
* `flavors` - The flavors that match all filters, in the requested sort order. The list is empty when
  nothing matches. Each element has the following attributes:
  * `id` - The ID of the flavor.
  * `name` - The name of the flavor. Pass it to `flavor_name` of
    [`fptcloud_instance`](../resources/instance).
  * `cpu` - The number of vCPUs.
  * `memory_mb` - The amount of RAM, in MB.
  * `type` - The flavor type, for example `VM_SIZE` for standard flavors and `GPU_SIZE` for GPU
    flavors. FPT Cloud can also report the type `OS`.
  * `gpu_id` - The ID of the vGPU that backs a GPU flavor. Empty for flavors without a GPU.
  * `gpu_name` - The name of the GPU of a GPU flavor. Pass it to `gpu_name` of
    [`fptcloud_instance`](../resources/instance) to have FPT Cloud verify the GPU. Empty for
    flavors without a GPU.
  * `gpu_memory_gb` - The GPU memory of a GPU flavor, in GB. `0` for flavors without a GPU.
  * `is_nvme` - `true` when instances of this flavor run on a physical NVMe disk. See
    [NVMe flavors](#nvme-flavors).

## Selecting one flavor

* `data.fptcloud_flavor.example.flavors[0]` takes the first flavor. When nothing matches, the plan
  fails with an "Invalid index" error.
* `one(data.fptcloud_flavor.example.flavors)` returns the only flavor, returns `null` when nothing
  matches, and fails when more than one flavor matches. Because `flavor_name` is optional on
  `fptcloud_instance`, Terraform does not stop a `null` flavor name at plan time, so prefer `[0]`
  or add a check such as the following (requires Terraform 1.2 or later):

```terraform
data "fptcloud_flavor" "checked" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["2C4G"] # Use a flavor name listed in your VPC
  }

  lifecycle {
    postcondition {
      condition     = length(self.flavors) == 1
      error_message = "Expected exactly one flavor named 2C4G in this VPC."
    }
  }
}
```

## NVMe flavors

Some GPU flavors have `is_nvme = true`. An instance created from such a flavor stores its data on a
physical NVMe disk supplied with the flavor:

* FPT Cloud uses the flavor's own disk. The `storage_policy_id` of the instance is still required by
  Terraform but has no effect.
* FPT Cloud refuses to change `flavor_name` of the instance later, so plan the size up front.
* Instances with an NVMe disk cannot be imported with `terraform import`.
