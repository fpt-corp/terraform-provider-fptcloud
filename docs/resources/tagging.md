---
page_title: "fptcloud_tagging Resource - FPT Cloud"
subcategory: "Tagging"
description: |-
  Manages a tag, a key:value label that you can attach to FPT Cloud resources.
---

# fptcloud_tagging (Resource)

Manages a tag. A tag is a `key:value` label, such as `environment:production` or `team:backend`,
that you attach to cloud resources to classify them, find them and track their cost.

Working with tags takes two steps:

1. Create the tag with this resource. The tag belongs to your organization, which is the tenant set
   by `tenant_name` in the provider configuration. `scope_type` and `resource_ids` decide where the
   tag can be used: in the whole organization, or only in some projects or VPCs.
2. Attach the tag to resources by adding its `id` to the `tag_ids` argument of those resources.
   The following resources support `tag_ids`: [fptcloud_instance](instance),
   [fptcloud_storage](storage), [fptcloud_subnet](subnet),
   [fptcloud_security_group](security_group) and [fptcloud_floating_ip](floating_ip).

~> **Important:** `resource_ids` does **not** attach the tag to instances, disks or other resources.
It lists the VPCs or projects that make up the tag's scope. To attach a tag, use `tag_ids` on the
resource.

To learn more, see [FPT Cloud Tagging](https://docs.fptcloud.com/docs/category/fpt-cloud-tagging/)
in the FPT Cloud documentation.

## Example Usage

### Tag for the whole organization

Use this for labels that every team and every VPC should share, such as `environment` or
`cost-center`.

```terraform
resource "fptcloud_tagging" "env_production" {
  key        = "environment"
  value      = "production"
  color      = "#057a55" # green
  scope_type = "ORG"
}
```

### Tag limited to one VPC

Use this when the label only makes sense inside one VPC, for example the name of an application
that runs there. List the VPC IDs in `resource_ids`.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

resource "fptcloud_tagging" "app_shop" {
  key          = "app"
  value        = "shop"
  color        = "#5850ec" # indigo
  scope_type   = "VPC"
  resource_ids = [data.fptcloud_vpc.example.id]
}
```

### Attach tags to resources

Use this to label the resources of an application. Each resource takes a set of tag IDs in
`tag_ids`. Terraform creates the tags first because the resources reference them. This example uses
the two tags from the previous examples.

```terraform
data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Use a storage policy name listed in your VPC
  }
}

locals {
  shop_tag_ids = [
    fptcloud_tagging.env_production.id,
    fptcloud_tagging.app_shop.id,
  ]
}

resource "fptcloud_subnet" "web" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "web-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.10.1.0/24"
  gateway_ip = "10.10.1.1"
  tag_ids    = local.shop_tag_ids
}

resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = fptcloud_subnet.web.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
  tag_ids           = local.shop_tag_ids
}

resource "fptcloud_storage" "db_data" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "db-data-disk"
  type              = "EXTERNAL"
  size_gb           = 100
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  instance_id       = fptcloud_instance.app.id
  tag_ids           = local.shop_tag_ids
}

resource "fptcloud_security_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-sg"
  subnet_id = fptcloud_subnet.web.id
  type      = "ACL"
  tag_ids   = local.shop_tag_ids
}

resource "fptcloud_floating_ip" "web" {
  vpc_id  = data.fptcloud_vpc.example.id
  tag_ids = local.shop_tag_ids
}
```

To attach a tag that already exists and is not managed by this configuration, look up its ID with
the [fptcloud_tagging](../data-sources/tagging) data source.

## Argument Reference

The following arguments are supported:

* `key` - (Required) The key of the tag, for example `environment`. It must be 1 to 50 characters
  long and must not contain quotes (`"`, `'`), slashes (`/`), backslashes (`\`) or HTML tags. Can be
  updated in place.
* `value` - (Optional) The value of the tag, for example `production`. The same character rules as
  `key` apply. The combination of `key` and `value` must be unique in your organization. Can be
  updated in place.
* `color` - (Optional) The color that the FPT Cloud console uses to display the tag. Use one of the
  hex values listed in [Colors](#colors). The value is not case-sensitive. Leave it unset, or set it
  to `""`, for a tag without a color. Can be updated in place.
* `scope_type` - (Optional) Where the tag can be used. Valid values, in upper case, are:
  * `ORG` - the whole organization. Leave `resource_ids` empty.
  * `PROJECT` - the projects listed in `resource_ids`.
  * `VPC` - the VPCs listed in `resource_ids`.

  The provider does not enforce it, but FPT Cloud rejects a tag without a valid `scope_type`, so
  always set it. Can be updated in place.
* `resource_ids` - (Optional) The IDs of the projects (when `scope_type` is `PROJECT`) or VPCs (when
  `scope_type` is `VPC`) that make up the tag's scope. Required, with at least one ID, for these two
  scope types. Use the `id` of the [fptcloud_vpc](../data-sources/vpc) data source for a VPC. Not used
  when `scope_type` is `ORG`. Can be updated in place.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the tag. Pass it in the `tag_ids` argument of the resources to tag.

## Colors

`color` takes one of the hex values below, the same palette that the FPT Cloud console offers. The
value is not case-sensitive: `#5850EC` and `#5850ec` are the same color, and FPT Cloud stores it in
lower case. Any other value, including a color name such as `red`, is rejected when Terraform plans.

| Color           | Hex       |
|-----------------|-----------|
| Gray            | `#4b5563` |
| Red             | `#c81e1e` |
| Orange          | `#d03801` |
| Yellow          | `#dc6803` |
| Green           | `#057a55` |
| Teal            | `#047481` |
| Indigo          | `#5850ec` |
| Purple          | `#6c2bd9` |
| Sky blue        | `#00bfff` |
| Green (classic) | `#35994b` |
| Amber (classic) | `#e7a600` |
| Red (classic)   | `#db4646` |

## Import

You can import an existing tag by its ID. To find the ID, look the tag up by key and value with the
[fptcloud_tagging](../data-sources/tagging) data source.

```shell
terraform import fptcloud_tagging.env_production <tag_id>
```

With Terraform 1.5 or later, you can use an `import` block instead:

```terraform
import {
  to = fptcloud_tagging.env_production
  id = "<tag_id>"
}
```

After the import, run `terraform plan` and make the configuration match the values it shows for
`key`, `value`, `color` and `scope_type`.

## Deleting a tag

* Destroying this resource deletes the tag and removes it from every resource it is attached to,
  including resources that Terraform does not manage. The tagged resources themselves are not
  changed otherwise.
* When you remove a tag from your configuration, also remove its ID from the `tag_ids` of the
  resources that reference it. Otherwise Terraform reports a reference to an undeclared resource.
* If the tag is deleted outside Terraform, for example in the FPT Cloud console, the next plan fails
  with an error instead of creating the tag again. Run
  `terraform state rm fptcloud_tagging.<name>` and apply again to create a new tag.
