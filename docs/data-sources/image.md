---
page_title: "fptcloud_image Data Source - FPT Cloud"
subcategory: "Compute"
description: |-
  Use this data source to list the operating system images available in a VPC and pick one for an instance.
---

# fptcloud_image (Data Source)

Use this data source to list the images available in one of your VPCs. An image is the operating
system an instance boots from, for example a Linux distribution or Windows Server. Images are
grouped in catalogs (such as all Ubuntu versions), and some images are made for GPU instances. You
choose an image by name when you create an instance with
[`fptcloud_instance`](../resources/instance) (argument `image_name`).

Image names differ between regions and VPCs, so look them up instead of hard-coding them. Use
`filter` blocks to keep only the images you want, and `sort` blocks to order the result.

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

### List every image in a VPC

Use this to see which image names and catalogs exist in your VPC before you choose one. Without a
`filter`, the data source returns every image of the VPC.

```terraform
data "fptcloud_image" "all" {
  vpc_id = data.fptcloud_vpc.example.id

  sort {
    key = "name"
  }
}

output "images" {
  value = [for i in data.fptcloud_image.all.images : "${i.catalog}: ${i.name}"]
}
```

### Find an image by name

Use this when you know the exact image name. Matching on a text field with the default
`match_by = "exact"` ignores upper and lower case.

```terraform
data "fptcloud_image" "ubuntu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Ubuntu-22.04"] # Use an image name listed in your VPC
  }
}
```

### Find the latest image of a catalog

Use this to follow the newest version of an operating system without editing your configuration.
The example keeps the images of one catalog and sorts them by name in descending order, so the
first element has the highest version number.

```terraform
data "fptcloud_image" "latest_ubuntu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "catalog"
    values = ["Ubuntu"] # Use a catalog name listed in your VPC
  }

  filter {
    key    = "is_gpu"
    values = [false]
  }

  sort {
    key       = "name"
    direction = "desc"
  }
}

output "latest_ubuntu" {
  value = data.fptcloud_image.latest_ubuntu.images[0].name
}
```

-> **Note:** Images can only be sorted by `name`, `catalog`, `id` or `is_gpu`; there is no creation
date. Sorting by name compares text character by character, which gives the right order for names
such as `Ubuntu-20.04`, `Ubuntu-22.04` and `Ubuntu-24.04`, but not when version numbers have a
different number of digits (`Debian-9` sorts after `Debian-12`). Check the result with an `output`
before you rely on it.

~> **Important:** When a newer image appears in the catalog, the selected name changes. Because
changing `image_name` of an existing [`fptcloud_instance`](../resources/instance) replaces the
instance, pin the image name (see [Find an image by name](#find-an-image-by-name)) for instances
you do not want to rebuild, or add `image_name` to `lifecycle { ignore_changes = [...] }`.

### Match image names with a regular expression

Use `match_by = "re"` when names follow a pattern. This example keeps the Ubuntu 22.04 and 24.04
images, including names that carry a suffix after the version.

```terraform
data "fptcloud_image" "ubuntu_lts" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key      = "name"
    values   = ["(?i)^ubuntu-2[24]\\.04"]
    match_by = "re"
  }

  sort {
    key       = "name"
    direction = "desc"
  }
}
```

In HCL strings, write a backslash as `\\`. `(?i)` makes the expression ignore upper and lower case.

### Find images for GPU instances

GPU instances need an image made for GPUs. The FPT Cloud console labels these images "GPU", and
the data source reports them with `is_gpu = true`.

```terraform
data "fptcloud_image" "gpu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "is_gpu"
    values = [true]
  }
}

output "gpu_images" {
  value = [for i in data.fptcloud_image.gpu.images : i.name]
}
```

### Use the image in an instance

Pass the `name` of the selected image to `image_name`. `images[0]` takes the first match and makes
the plan fail with an "Invalid index" error when nothing matched.

```terraform
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

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC whose images are listed. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source. Must not be empty.
* `filter` - (Optional) One or more `filter` blocks as defined below. When you set several blocks,
  an image must match all of them.
* `sort` - (Optional) One or more `sort` blocks as defined below. The first block is the primary
  sort order; the next blocks break ties.

### filter

* `key` - (Required) The image field to filter on. Valid values are `catalog`, `id`, `is_gpu` and
  `name`.
* `values` - (Required) The values to match. An image matches when its field matches at least one of
  the values (or all of them when `all = true`). For `is_gpu`, the values must be `true` or `false`.
* `match_by` - (Optional) How text fields are compared. Valid values are:
  * `exact` (default) - the whole value must be equal, ignoring upper and lower case.
  * `substring` - the field must contain the value. This comparison is case-sensitive.
  * `re` - the value is a regular expression ([RE2 syntax](https://github.com/google/re2/wiki/Syntax))
    that must match somewhere in the field. Use `^` and `$` to match the whole field. Case-sensitive
    unless you start the expression with `(?i)`.

  `match_by` has no effect on `is_gpu`, which is always compared exactly.
* `all` - (Optional) Set to `true` to require that the field matches every value in `values`
  instead of at least one. Defaults to `false`.

### sort

* `key` - (Required) The image field to sort on. Valid values are `catalog`, `id`, `is_gpu` and
  `name`.
* `direction` - (Optional) The sort direction. Valid values are `asc` and `desc`. When omitted, the
  result is sorted in ascending order.

Text fields are sorted character by character and are case-sensitive (upper-case letters come before
lower-case letters).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A random identifier generated each time the data source is read. It does not identify an
  image; use `images[*].id` for that.
* `images` - The images that match all filters, in the requested sort order. The list is empty when
  nothing matches. Each element has the following attributes:
  * `id` - The ID of the image.
  * `name` - The name of the image. Pass it to `image_name` of
    [`fptcloud_instance`](../resources/instance).
  * `catalog` - The catalog the image belongs to, for example the operating system family.
  * `is_gpu` - `true` for images made for GPU instances.

## Selecting one image

* `data.fptcloud_image.example.images[0]` takes the first image. When nothing matches, the plan
  fails with an "Invalid index" error.
* `one(data.fptcloud_image.example.images)` returns the only image, returns `null` when nothing
  matches, and fails when more than one image matches.
