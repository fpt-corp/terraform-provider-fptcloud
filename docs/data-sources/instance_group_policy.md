---
page_title: "fptcloud_instance_group_policy Data Source - FPT Cloud"
subcategory: "Compute"
description: |-
  Use this data source to look up the placement policies you can use for an instance group in a VPC.
---

# fptcloud_instance_group_policy (Data Source)

Use this data source to look up the placement policies that you can choose when you create an
[fptcloud_instance_group](../resources/instance_group). It returns the ID and name of each policy
offered in the VPC:

* **Soft Anti Affinity** places the instances of a group on different physical hosts when possible.
  Choose it for availability.
* **Soft Affinity** places the instances of a group on the same physical host when possible. Choose
  it for low network latency between the instances.

Policy IDs are different in a General VPC and a Specific VPC. Always look the policy up by name in
the VPC of the group instead of copying an ID.

To learn more, see [Instance Groups](https://docs.fptcloud.com/docs/fpt-cloud-server/instance-group/overview/)
in the FPT Cloud documentation.

## Example Usage

### Look up a policy by name

Use this to get the ID to pass to `policy_id`.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_instance_group_policy" "anti_affinity" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Soft Anti Affinity"]
  }
}

resource "fptcloud_instance_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-group"
  policy_id = data.fptcloud_instance_group_policy.anti_affinity.instance_group_policies[0].id
}
```

See the [fptcloud_instance_group](../resources/instance_group) resource for complete examples,
including how to put instances in the group.

### List all policies offered in a VPC

Use this to see which policies a VPC offers.

```terraform
data "fptcloud_instance_group_policy" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}

output "instance_group_policies" {
  value = {
    for policy in data.fptcloud_instance_group_policy.all.instance_group_policies :
    policy.name => policy.id
  }
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC. Use the `id` of the [fptcloud_vpc](vpc) data source.
* `filter` - (Optional) One or more `filter` blocks as defined below. A policy is returned only if it
  matches every `filter` block.
* `sort` - (Optional) One or more `sort` blocks as defined below. They are applied in order.

### filter

* `key` - (Required) The field to filter on. Valid values are `id` and `name`.
* `values` - (Required) A list of values. A policy matches when its field matches at least one of
  them (or all of them, see `all`).
* `match_by` - (Optional) How to compare the values. Valid values are:
  * `exact` (default) - the whole value must match, ignoring upper and lower case.
  * `substring` - the field must contain the value. This comparison is case-sensitive.
  * `re` - the value is a regular expression that the field must match.
* `all` - (Optional) Set to `true` to require that the field matches all of the `values` instead of
  at least one. Defaults to `false`.

### sort

* `key` - (Required) The field to sort by. Valid values are `id` and `name`.
* `direction` - (Optional) The sort direction. Valid values are `asc` and `desc`. Defaults to `asc`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A generated identifier for this data source read. It is not the ID of a policy.
* `instance_group_policies` - The list of policies that match the filters. Each element has:
  * `id` - The ID of the policy. Pass it to `policy_id` on
    [fptcloud_instance_group](../resources/instance_group).
  * `name` - The name of the policy, for example `Soft Anti Affinity`.

If no policy matches your filters, `instance_group_policies` is an empty list, and an expression
such as `instance_group_policies[0].id` fails with an "Invalid index" error. Check the spelling of
the policy name.
