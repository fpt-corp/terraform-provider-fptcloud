---
page_title: "fptcloud_security_groups Data Source - FPT Cloud"
subcategory: "Networking"
description: |-
  Use this data source to list every security group in a VPC, with the resources each group applies to.
---

# fptcloud_security_groups (Data Source)

Use this data source to list all security groups in a VPC. For each group it returns its name,
firewall type, status, tags, the addresses it applies to and the resources behind those addresses
(`members`).

Typical uses:

* Build a name-to-ID map of your security groups to use in other resources.
* Find which instances or load balancers a security group protects.
* Audit the security groups of a VPC.

The data source has no `filter` or `sort` blocks; it always returns every security group of the VPC.
Select the groups you need with a `for` expression, as shown below.

-> **Note:** The results do not include the rules or the `edge_gateway_id` of each group. To read
them for one group, use the [`fptcloud_security_group`](security_group) data source.

To learn more, see [Security Groups](https://docs.fptcloud.com/docs/fpt-network/security-groups/overview/)
in the FPT Cloud documentation.

## Example Usage

### List the security groups of a VPC

Use this to get the ID of every security group, indexed by name.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_security_groups" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}

output "security_group_ids" {
  value = { for sg in data.fptcloud_security_groups.all.security_groups : sg.name => sg.id }
}
```

### Pick one security group by name

Use this when you need the ID of one group. `one()` returns the single match. If no group has that
name, it returns `null` and any reference to its attributes fails with an error; if several groups
match, `one()` fails. This example reuses the lookups from the previous one.

```terraform
locals {
  web_sg = one([
    for sg in data.fptcloud_security_groups.all.security_groups : sg
    if sg.name == "web-sg"
  ])
}

output "web_sg_id" {
  value = local.web_sg.id
}
```

### Find the instances a security group protects

Use this to check which instances each group applies to. Members of type `instance` are instances;
other types are load balancers or addresses that do not belong to a known resource. This example
reuses the lookups from the first one.

```terraform
output "protected_instances" {
  value = {
    for sg in data.fptcloud_security_groups.all.security_groups :
    sg.name => [for m in sg.members : m.name if m.type == "instance"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC whose security groups are listed. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source.
* `page_size` - (Optional) The number of security groups the provider requests at a time. Valid
  values are `1` to `25`. Defaults to `25`. The provider keeps requesting until it has every
  security group, so this value does not limit the number of results.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of this data source. It is the same as `vpc_id`.
* `security_groups` - The list of security groups in the VPC. The list is empty when the VPC has
  no security group. Each element has the attributes described in
  [security_groups](#security_groups) below.

### security_groups

* `id` - The ID of the security group.
* `name` - The name of the security group, as shown in the FPT Cloud console.
* `type` - The firewall type of the security group: `ACL` (controls traffic to and from the
  Internet) or `DFW` (controls traffic within the local network).
* `apply_to` - The list of IP addresses or CIDR blocks the security group applies to.
* `status` - The current status of the security group, for example `REALIZED`.
* `tag_ids` - The list of IDs of the tags attached to the security group.
* `created_at` - The time the security group was created.
* `updated_at` - The time the security group was last updated.
* `members` - The list of resources matched by the addresses in `apply_to`. Each element has the
  attributes described in [members](#members) below.

### members

* `id` - The ID of the member. For a member of type `other`, this is the address itself.
* `name` - The name of the member. For a member of type `other`, this is the address itself.
* `type` - The kind of member: `instance`, `load_balancer`, or `other` for an address that does
  not belong to a known resource.
* `status` - The status of the member resource. `Unknown` for a member of type `other`.
* `ip_address` - The IP address of the member.
* `ipv6_address` - The IPv6 address of the member. Empty when it has none.
