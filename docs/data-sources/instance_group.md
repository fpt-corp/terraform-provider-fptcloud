---
page_title: "fptcloud_instance_group Data Source - FPT Cloud"
subcategory: "Compute"
description: |-
  Use this data source to list the instance groups in a VPC, with their placement policy and member instances.
---

# fptcloud_instance_group (Data Source)

Use this data source to list the instance groups in a VPC. For each group it returns the placement
policy and the instances in it. Typical uses:

* Find the ID of a group that already exists, to pass it to `instance_group_id` when you create an
  [fptcloud_instance](../resources/instance).
* Check which instances are in a group and which policy it uses.

To create a group, use the [fptcloud_instance_group](../resources/instance_group) resource. To learn
more, see [Instance Groups](https://docs.fptcloud.com/docs/fpt-cloud-server/instance-group/overview/)
in the FPT Cloud documentation.

## Example Usage

### Look up a group by name

Use this when you need the ID of one group. Group names are unique in a VPC, so the result usually
has one element.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_instance_group" "web" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["web-group"]
  }
}

output "web_group_id" {
  value = data.fptcloud_instance_group.web.instance_groups[0].id
}
```

### List the members and policy of every group

Use this to review the groups in a VPC. The newest group comes first unless you add a `sort` block.

```terraform
data "fptcloud_instance_group" "all" {
  vpc_id = data.fptcloud_vpc.example.id

  sort {
    key       = "name"
    direction = "asc"
  }
}

output "group_members" {
  value = {
    for group in data.fptcloud_instance_group.all.instance_groups :
    group.name => {
      policy    = group.policy[0].name
      instances = group.vms[*].name
    }
  }
}
```

### Put a new instance in an existing group

See [Add a new instance to an existing group](../resources/instance_group#add-a-new-instance-to-an-existing-group)
on the resource page. Reference the ID like this:

```terraform
variable "subnet_id" {
  type        = string
  description = "ID of the subnet for the new instance."
}

variable "storage_policy_id" {
  type        = string
  description = "ID of the storage policy for the root disk."
}

resource "fptcloud_instance" "web_extra" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "web-server-04"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = var.subnet_id
  storage_size_gb   = 40
  storage_policy_id = var.storage_policy_id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
  instance_group_id = data.fptcloud_instance_group.web.instance_groups[0].id
}
```

`instance_group_id` is only used when the instance is created. Setting it on an existing instance
does not move that instance into the group.

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC to list instance groups from. Use the `id` of the
  [fptcloud_vpc](vpc) data source.
* `filter` - (Optional) One or more `filter` blocks as defined below. A group is returned only if it
  matches every `filter` block.
* `sort` - (Optional) One or more `sort` blocks as defined below. They are applied in order.

### filter

* `key` - (Required) The field to filter on. Use `id`, `name` or `created_at`. The schema also
  accepts `policy`, `vms` and `vpc_id`, but filtering on them fails, so do not use them.
* `values` - (Required) A list of values. A group matches when its field matches at least one of
  them (or all of them, see `all`).
* `match_by` - (Optional) How to compare the values. Valid values are:
  * `exact` (default) - the whole value must match, ignoring upper and lower case.
  * `substring` - the field must contain the value. This comparison is case-sensitive.
  * `re` - the value is a regular expression that the field must match.
* `all` - (Optional) Set to `true` to require that the field matches all of the `values` instead of
  at least one. Defaults to `false`.

### sort

* `key` - (Required) The field to sort by. Use `id`, `name` or `created_at`. Do not use `vpc_id`;
  sorting on it fails.
* `direction` - (Optional) The sort direction. Valid values are `asc` and `desc`. Defaults to `asc`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A generated identifier for this data source read. It is not the ID of a group.
* `instance_groups` - The list of groups that match the filters. Each element has the attributes
  listed below.

### instance_groups

* `id` - The ID of the instance group.
* `name` - The name of the instance group.
* `policy` - A list with one element that describes the placement policy:
  * `id` - The ID of the policy.
  * `name` - The name of the policy: `Soft Affinity` or `Soft Anti Affinity`.
  * `is_active` - Whether the policy is currently offered for new groups.
* `vms` - The instances in the group. Each element has:
  * `id` - The ID of the instance.
  * `name` - The name of the instance.
* `created_at` - The time the group was created.
* `vpc_id` - Not filled in; it is always empty. Use the top-level `vpc_id` instead.

## Results and errors

* If the VPC has no instance groups at all, the data source returns an error.
* If the VPC has groups but none match your filters, `instance_groups` is an empty list. An
  expression such as `instance_groups[0].id` then fails with an "Invalid index" error, which is
  usually what you want when the group must exist.
