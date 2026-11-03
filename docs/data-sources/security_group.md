---
page_title: "fptcloud_security_group Data Source - FPT Cloud"
subcategory: "Networking"
description: |-
  Use this data source to get the details and rules of one security group in a VPC, by name or by ID.
---

# fptcloud_security_group (Data Source)

Use this data source to read one existing security group in a VPC, found by its name or its ID. It
returns the group's firewall type, the addresses it applies to, its tags and all of its rules,
including the default rules created by FPT Cloud.

Typical uses:

* Add rules with [`fptcloud_security_group_rule`](../resources/security_group_rule) to a group
  that was created in the FPT Cloud console or by another Terraform configuration.
* Check which rules and attachments a group has.

If no security group matches, Terraform stops with an error. To list every security group in a VPC
instead, use the [`fptcloud_security_groups`](security_groups) data source.

To learn more, see [Security Groups](https://docs.fptcloud.com/docs/fpt-network/security-groups/overview/)
in the FPT Cloud documentation.

## Example Usage

### Look up a security group by name

Use this when you know the name of the group, as shown in the FPT Cloud console.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_security_group" "web" {
  vpc_id = data.fptcloud_vpc.example.id
  name   = "web-sg"
}

output "web_sg_type" {
  value = data.fptcloud_security_group.web.type
}
```

### Add a rule to an existing security group

Use this when the group itself is managed elsewhere and you only want to add a rule to it. This
example reuses the lookups from the previous one.

```terraform
resource "fptcloud_security_group_rule" "https" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = data.fptcloud_security_group.web.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "443"
  sources           = ["ALL"]
  description       = "HTTPS from anywhere"
}
```

### List the inbound rules of a group

Use this to review which inbound traffic a group allows. The `rules` attribute contains every rule
of the group, so filter it with a `for` expression. This example reuses the lookups from the first
one.

```terraform
output "web_sg_inbound_rules" {
  value = [
    for rule in data.fptcloud_security_group.web.rules :
    "${rule.action} ${rule.protocol} ${rule.port_range} from ${rule.sources}"
    if rule.direction == "INGRESS"
  ]
}
```

## Argument Reference

The following arguments are supported. Exactly one of `id` or `name` must be set.

* `vpc_id` - (Required) The ID of the VPC that contains the security group. Use the `id` of the
  [`fptcloud_vpc`](vpc) data source.
* `id` - (Optional) The ID of the security group to read.
* `name` - (Optional) The name of the security group to read. Must not contain spaces.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `type` - The firewall type of the security group: `ACL` (controls traffic to and from the
  Internet) or `DFW` (controls traffic within the local network).
* `apply_to` - The list of IP addresses (or CIDR blocks) the security group applies to.
* `rules` - The list of rules of the security group, including the default rules created by FPT
  Cloud. Each element has the attributes described in [rules](#rules) below.
* `tag_ids` - The list of IDs of the tags attached to the security group.
* `edge_gateway_id` - The ID of the edge gateway the security group is associated with. It can be
  empty.
* `created_at` - The time the security group was created.

### rules

* `id` - The ID of the rule.
* `direction` - `INGRESS` (inbound traffic) or `EGRESS` (outbound traffic).
* `action` - `ALLOW` or `DROP`.
* `protocol` - `TCP`, `UDP`, `ICMP` or `ALL`.
* `port_range` - The port, port range or `ALL`.
* `sources` - The addresses the rule matches, as a single string (for example `ALL` or
  `203.0.113.0/24`). For an `EGRESS` rule these are the destination addresses.
* `description` - The description of the rule.
