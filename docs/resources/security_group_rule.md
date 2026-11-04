---
page_title: "fptcloud_security_group_rule Resource - FPT Cloud"
subcategory: "Networking"
description: |-
  Manages one inbound or outbound rule in an FPT Cloud security group.
---

# fptcloud_security_group_rule (Resource)

Manages one rule in a [`fptcloud_security_group`](security_group). A rule allows or blocks traffic
in one direction (inbound or outbound), for one protocol, on a port or port range, from or to a list
of addresses. Declare one `fptcloud_security_group_rule` per rule you need.

FPT Cloud also creates default rules in every new security group that allow all outbound traffic.
Inbound traffic that does not match any rule is denied, so you only need inbound rules for the
traffic you want to let in. See [Default rules](#default-rules).

A rule cannot be modified: changing any of its arguments deletes the rule and creates a new one.

To learn more, see
[Manage Inbound and Outbound Rules](https://docs.fptcloud.com/docs/fpt-network/security-groups/guides/guide-manage-rules/)
in the FPT Cloud documentation.

## Example Usage

The examples on this page share the following lookup, subnet and security group.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

resource "fptcloud_subnet" "web" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "web-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.10.1.0/24"
  gateway_ip = "10.10.1.1"
}

resource "fptcloud_security_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-sg"
  subnet_id = fptcloud_subnet.web.id
  type      = "ACL"
}
```

### Allow SSH from your office

Use this when you log in to your instances over SSH and want to accept connections only from your
office network.

```terraform
resource "fptcloud_security_group_rule" "ssh" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "22"
  sources           = ["203.0.113.0/24"] # Your office network
  description       = "SSH from the office"
}
```

### Web server: HTTP and HTTPS from anywhere

Use this for a public website. `sources = ["ALL"]` accepts traffic from any address.

```terraform
resource "fptcloud_security_group_rule" "http" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "80"
  sources           = ["ALL"]
  description       = "HTTP from anywhere"
}

resource "fptcloud_security_group_rule" "https" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "443"
  sources           = ["ALL"]
  description       = "HTTPS from anywhere"
}
```

### Rules generated from a map

Use this when a group has many similar rules. Each map key becomes part of the resource address
(`fptcloud_security_group_rule.web["https"]`), so adding or removing an entry only creates or
deletes that rule. Prefer a map with stable keys over a list: with a list, removing an element
shifts the indexes and replaces every rule after it.

```terraform
locals {
  web_ingress_rules = {
    http = {
      protocol    = "TCP"
      port_range  = "80"
      sources     = ["ALL"]
      description = "HTTP from anywhere"
    }
    https = {
      protocol    = "TCP"
      port_range  = "443"
      sources     = ["ALL"]
      description = "HTTPS from anywhere"
    }
    ssh = {
      protocol    = "TCP"
      port_range  = "22"
      sources     = ["203.0.113.0/24", "198.51.100.10"]
      description = "SSH from the office and the VPN gateway"
    }
    app_ports = {
      protocol    = "TCP"
      port_range  = "8000-8100"
      sources     = ["10.10.0.0/16"]
      description = "Application ports from inside the VPC"
    }
  }
}

resource "fptcloud_security_group_rule" "web" {
  for_each = local.web_ingress_rules

  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = each.value.protocol
  port_range        = each.value.port_range
  sources           = each.value.sources
  description       = each.value.description
}
```

### Database reachable only from the application subnet

Use this to protect a database server: PostgreSQL (port 5432) accepts connections only from the
subnet where your application servers run. Referencing the subnet's `cidr` keeps the rule correct if
the subnet changes.

```terraform
resource "fptcloud_subnet" "app" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "app-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.10.2.0/24"
  gateway_ip = "10.10.2.1"
}

resource "fptcloud_subnet" "db" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "db-subnet"
  type       = "ISOLATED"
  cidr       = "10.10.3.0/24"
  gateway_ip = "10.10.3.1"
}

resource "fptcloud_security_group" "db" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "db-sg"
  subnet_id = fptcloud_subnet.db.id
  type      = "DFW"
}

resource "fptcloud_security_group_rule" "postgres" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.db.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "5432"
  sources           = [fptcloud_subnet.app.cidr]
  description       = "PostgreSQL from the application subnet"
}
```

### Allow ping (ICMP)

Use this to let monitoring tools or administrators ping your instances. ICMP has no ports, so set
`port_range` to `ALL`.

```terraform
resource "fptcloud_security_group_rule" "ping" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "ICMP"
  port_range        = "ALL"
  sources           = ["10.10.0.0/16"]
  description       = "Ping from inside the VPC"
}
```

### Restrict outbound traffic

Use this when instances should only reach a few destinations, for example HTTPS servers and your
DNS resolvers. For an outbound (`EGRESS`) rule, `sources` lists the **destination** addresses.

```terraform
resource "fptcloud_security_group_rule" "egress_https" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "EGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "443"
  sources           = ["ALL"]
  description       = "HTTPS to anywhere"
}

resource "fptcloud_security_group_rule" "egress_dns" {
  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "EGRESS"
  action            = "ALLOW"
  protocol          = "UDP"
  port_range        = "53"
  sources           = ["10.10.0.2", "10.10.0.3"] # Your DNS resolvers
  description       = "DNS to the internal resolvers"
}
```

~> **Important:** The default rules of the group already allow all outbound traffic, so these
rules restrict nothing while the default outbound rules exist. The default rules are not managed by
Terraform. To actually restrict outbound traffic, remove the default outbound rules of the group in
the FPT Cloud console once your own outbound rules are in place.

## Argument Reference

The following arguments are supported. Every argument forces a new rule to be created when it
changes; see [Changing a rule](#changing-a-rule).

* `vpc_id` - (Required) The ID of the VPC that contains the security group. Use the same value as
  the `vpc_id` of the [`fptcloud_security_group`](security_group). Changing this forces a new
  resource to be created.
* `security_group_id` - (Required) The ID of the security group the rule belongs to. Use the `id`
  of a [`fptcloud_security_group`](security_group) resource or data source. Changing this forces a
  new resource to be created.
* `direction` - (Required) The direction of the traffic the rule matches. Valid values are:
  * `INGRESS` - inbound traffic, coming into the protected resources.
  * `EGRESS` - outbound traffic, leaving the protected resources.

  Changing this forces a new resource to be created.
* `action` - (Required) What happens to matching traffic. Valid values are `ALLOW` (let the traffic
  through) and `DROP` (block the traffic). Changing this forces a new resource to be created.
* `protocol` - (Required) The protocol the rule matches. Valid values are `TCP`, `UDP`, `ICMP` and
  `ALL` (all protocols). Changing this forces a new resource to be created.
* `port_range` - (Required) The ports the rule matches, as a string. Use a single port (`"22"`), a
  range (`"8000-8100"`), or `"ALL"` for all ports. Must be `"ALL"` when `protocol` is `ALL`. For
  `ICMP`, which has no ports, use `"ALL"`. Changing this forces a new resource to be created.
* `sources` - (Required) A set of addresses the rule matches. For an `INGRESS` rule these are the
  addresses the traffic comes from; for an `EGRESS` rule, the addresses it goes to. Each element is
  a CIDR block (`"203.0.113.0/24"`), a single IP address (`"198.51.100.10"`), or `"ALL"` for any
  address. Changing this forces a new resource to be created.
* `description` - (Optional) A note that explains the purpose of the rule. It is shown in the FPT
  Cloud console. Changing this forces a new resource to be created.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the security group rule.
* `ip_type` - The IP version of the rule, for example `IPv4`.

## Default rules

FPT Cloud creates two default rules in every new security group: outbound (`EGRESS`) rules that
allow all protocols, all ports and all destinations, one for IPv4 and one for IPv6. They are not
managed by Terraform. You can list them with the `rules` attribute of the
[`fptcloud_security_group`](../data-sources/security_group) data source.

Do not declare a rule identical to a default rule, such as `direction = "EGRESS"`,
`protocol = "ALL"`, `port_range = "ALL"` and `sources = ["ALL"]`: FPT Cloud rejects a rule that
duplicates an existing rule of the same group.

## Changing a rule

A rule cannot be modified in place. When you change any argument, even only `description`,
Terraform deletes the rule and then creates the new one. Between the two steps, traffic that only
the old rule allowed is blocked for a short time. Plan such changes for a quiet period.

When you delete a security group with Terraform, its rules are deleted first, because each rule
references the group.

## Timeouts

This resource has no `timeouts` block. Terraform waits for the rule to be active after it is
created, and to disappear when it is deleted, for up to the `timeout` set on the
[provider](../index), in minutes (default `15`).

## Troubleshooting

* **`Failed to retrieve security group rule` on plan or refresh.** The rule was probably deleted
  outside Terraform, for example in the FPT Cloud console. Terraform does not recreate it
  automatically; remove it from the state with
  `terraform state rm 'fptcloud_security_group_rule.ssh'` (use your resource address), then run
  `terraform apply` to create it again.
* **`Failed to create security group rule`.** Check that the rule does not duplicate a default rule
  or another rule of the same group, and that `port_range` is `"ALL"` when `protocol` is `ALL`.
