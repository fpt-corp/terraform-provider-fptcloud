---
page_title: "fptcloud_tagging Data Source - FPT Cloud"
subcategory: "Tagging"
description: |-
  Use this data source to look up tags in your FPT Cloud organization by key and value.
---

# fptcloud_tagging (Data Source)

Use this data source to look up tags that already exist in your organization, for example tags that
an administrator created in the FPT Cloud console. It returns a list of tags with their ID, key,
value, color and scope. Typical uses:

* Attach an existing tag to a resource by passing its `id` in `tag_ids` on
  [fptcloud_instance](../resources/instance), [fptcloud_storage](../resources/storage),
  [fptcloud_subnet](../resources/subnet), [fptcloud_security_group](../resources/security_group) or
  [fptcloud_floating_ip](../resources/floating_ip).
* Find the ID of a tag to import it into an [fptcloud_tagging](../resources/tagging) resource.

The organization is the tenant set by `tenant_name` in the provider configuration. The list only
contains tags that your account is allowed to see. To learn more, see
[FPT Cloud Tagging](https://docs.fptcloud.com/docs/category/fpt-cloud-tagging/) in the FPT Cloud
documentation.

## Example Usage

### Look up one tag by key and value

A key and value pair is unique in an organization, so setting both returns exactly one tag.

```terraform
data "fptcloud_tagging" "env_production" {
  key   = "environment"
  value = "production"
}

output "env_production_tag_id" {
  value = data.fptcloud_tagging.env_production.tags[0].id
}
```

### List every value of a key

Use this to see which values are in use for a key, for example all `environment` tags.

```terraform
data "fptcloud_tagging" "environments" {
  key = "environment"
}

output "environment_tags" {
  value = {
    for tag in data.fptcloud_tagging.environments.tags :
    tag.value => tag.id
  }
}
```

### Attach an existing tag to an instance

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_subnet" "web" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["web-subnet"]
  }
}

data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Use a storage policy name listed in your VPC
  }
}

data "fptcloud_tagging" "env_production" {
  key   = "environment"
  value = "production"
}

resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = data.fptcloud_subnet.web.subnets[0].id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
  tag_ids           = [data.fptcloud_tagging.env_production.tags[0].id]
}
```

## Argument Reference

The following arguments are supported. Both filters are optional and are combined: a tag must match
every filter you set.

* `key` - (Optional) Return only tags whose key is exactly this value.
* `value` - (Optional) Return only tags whose value is exactly this value.
* `scope_type` - (Optional) Accepted, but currently not used: the result is **not** filtered by scope
  type. To keep only some scope types, filter the `tags` list yourself, for example
  `[for t in data.fptcloud_tagging.environments.tags : t if t.scope_type == "ORG"]`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A generated identifier for this data source read. It is not the ID of a tag.
* `tags` - The tags that match the filters, sorted by key and then by value, in descending order.
  Each element has:
  * `id` - The ID of the tag. Pass it in `tag_ids` on the resources to tag.
  * `key` - The key of the tag.
  * `value` - The value of the tag, or an empty string if the tag has no value.
  * `color` - The hex color of the tag in lower case, for example `#5850ec`, or an empty string if
    the tag has no color. See [Colors](../resources/tagging#colors) for the palette.
  * `scope_type` - Where the tag can be used: `ORG`, `PROJECT` or `VPC`.
  * `resource_ids` - The IDs that make up the tag's scope: VPC IDs for a `VPC` tag, project IDs for a
    `PROJECT` tag, or the organization ID for an `ORG` tag. These are not the resources the tag is
    attached to.

## Results and errors

* If no tag matches, the data source fails with the error `Tagging list not found`.
* The list holds at most 25 tags. If you have more tags, set `key`, or `key` and `value`, so that
  the tag you need is in the result.
