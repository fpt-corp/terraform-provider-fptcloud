---
page_title: "fptcloud_subnet Resource - FPT Cloud"
subcategory: "Networking"
description: |-
  Manages a subnet (an IP address range) inside an FPT Cloud VPC.
---

# fptcloud_subnet (Resource)

Manages a subnet: a range of private IP addresses inside one of your FPT Cloud VPCs. Every instance
is placed in a subnet and receives its private IP address from the subnet's range. A subnet belongs
to exactly one VPC.

FPT Cloud offers two subnet types:

* **NAT_ROUTED** (shown as *Routed* in the console): instances can reach the Internet (outbound)
  through the VPC's NAT gateway. Use it for web and application servers that download updates or
  call external services.
* **ISOLATED**: no route to the Internet. Instances can only talk to other resources inside the
  VPC. Use it for databases, caches and other internal workloads.

Subnets work in both General VPC and Specific VPC. To receive traffic from the Internet, an instance
also needs a floating IP ([`fptcloud_floating_ip`](floating_ip) and
[`fptcloud_floating_ip_association`](floating_ip_association)) and a security group that allows the
traffic ([`fptcloud_security_group`](security_group)).

To learn more, see [Subnets](https://docs.fptcloud.com/docs/fpt-network/subnets/overview/) in the
FPT Cloud documentation.

~> **Important:** Almost every argument of a subnet is fixed once it is created. Changing `name`,
`type`, `cidr`, `gateway_ip`, `static_ip_pool` or `vpc_id` deletes the subnet and creates a new one,
which also replaces every instance placed in it. See [Updating a subnet](#updating-a-subnet).

## Example Usage

All examples look up the VPC by name with the [`fptcloud_vpc`](../data-sources/vpc) data source:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

### Application subnet with Internet access

Use this for application servers that need outbound Internet access, for example to install
packages. The gateway is usually the first address of the range.

```terraform
resource "fptcloud_subnet" "app" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "app-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.10.1.0/24"
  gateway_ip = "10.10.1.1"
}
```

### Isolated subnet for databases

Use this for workloads that must not reach the Internet, such as database servers. Instances in an
`ISOLATED` subnet can still talk to instances in the other subnets of the same VPC.

```terraform
resource "fptcloud_subnet" "db" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "db-subnet"
  type       = "ISOLATED"
  cidr       = "10.10.2.0/24"
  gateway_ip = "10.10.2.1"
}
```

### Subnet with custom DNS servers

Use this when instances must resolve names with specific DNS servers, for example your own internal
DNS servers. You can change the DNS servers later without recreating the subnet.

```terraform
resource "fptcloud_subnet" "web" {
  vpc_id           = data.fptcloud_vpc.example.id
  name             = "web-subnet"
  type             = "NAT_ROUTED"
  cidr             = "10.10.3.0/24"
  gateway_ip       = "10.10.3.1"
  primary_dns_ip   = "8.8.8.8"
  secondary_dns_ip = "8.8.4.4"
}
```

### Subnet with a static IP pool

Use this to reserve a part of the subnet for static allocation. Write the pool as
`<first address>-<last address>`, without spaces. Both addresses must be inside `cidr`.

```terraform
resource "fptcloud_subnet" "backend" {
  vpc_id         = data.fptcloud_vpc.example.id
  name           = "backend-subnet"
  type           = "NAT_ROUTED"
  cidr           = "10.10.4.0/24"
  gateway_ip     = "10.10.4.1"
  static_ip_pool = "10.10.4.100-10.10.4.200"
}
```

### Several subnets from one address plan

Use this to create a set of subnets (for example web, app and database tiers) from one address
plan. `cidrsubnet()` cuts a `/24` out of the `/16` plan for each tier, and `cidrhost()` picks its
first address as the gateway. The `/16` plan is a value you choose; it is not read from the VPC.

```terraform
locals {
  address_plan = "10.20.0.0/16"

  tiers = {
    web = { netnum = 1, type = "NAT_ROUTED" }
    app = { netnum = 2, type = "NAT_ROUTED" }
    db  = { netnum = 3, type = "ISOLATED" }
  }
}

resource "fptcloud_subnet" "tier" {
  for_each = local.tiers

  vpc_id     = data.fptcloud_vpc.example.id
  name       = "${each.key}-subnet"
  type       = each.value.type
  cidr       = cidrsubnet(local.address_plan, 8, each.value.netnum)
  gateway_ip = cidrhost(cidrsubnet(local.address_plan, 8, each.value.netnum), 1)
}

output "subnet_ids" {
  value = { for tier, subnet in fptcloud_subnet.tier : tier => subnet.id }
}
```

The CIDR of each subnet comes from its fixed `netnum`, so you can add or remove a tier without
touching the others. Prefer `for_each` over `count` here: with `count`, removing an item from the
middle of a list shifts the index of the following subnets, which changes their CIDR and forces
Terraform to recreate them.

### Subnet with tags

Use this to attach FPT Cloud tags to the subnet. Tags can be changed in place later.

```terraform
resource "fptcloud_tagging" "env" {
  key   = "env"
  value = "production"
}

resource "fptcloud_subnet" "tagged" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "tagged-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.10.5.0/24"
  gateway_ip = "10.10.5.1"
  tag_ids    = [fptcloud_tagging.env.id]
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC to create the subnet in. Use the `id` of the
  [`fptcloud_vpc`](../data-sources/vpc) data source. Changing this forces a new resource to be
  created.
* `name` - (Required) The name of the subnet. It must not be empty and must be unique within the
  VPC. Changing this forces a new resource to be created.
* `type` - (Required) The subnet type. Valid values are `NAT_ROUTED` (outbound Internet access
  through the NAT gateway) and `ISOLATED` (no Internet access). The values are case-sensitive.
  Changing this forces a new resource to be created.
* `cidr` - (Required) The IP address range of the subnet, in CIDR notation, for example
  `10.10.1.0/24`. It must not overlap the range of any other subnet in the same VPC. Changing this
  forces a new resource to be created.
* `gateway_ip` - (Required) The IPv4 address of the subnet's gateway. It must be inside `cidr`;
  the first address of the range (for example `10.10.1.1`) is the usual choice. Changing this
  forces a new resource to be created.
* `static_ip_pool` - (Optional) A range of addresses inside `cidr` reserved for static allocation,
  written as `<first address>-<last address>` without spaces, for example
  `10.10.1.100-10.10.1.200`. The first address must not be greater than the last one. When
  omitted, FPT Cloud manages the whole range automatically. Changing this forces a new resource to
  be created.
* `primary_dns_ip` - (Optional) The IPv4 address of the primary DNS server used by resources in
  the subnet. Can be updated in place. When omitted, Terraform keeps the value that FPT Cloud
  reports, if any.
* `secondary_dns_ip` - (Optional) The IPv4 address of the backup DNS server, used when the primary
  server does not respond. Can be updated in place. When omitted, Terraform keeps the value that FPT
  Cloud reports, if any.
* `tag_ids` - (Optional) A set of tag IDs to attach to the subnet. Use the `id` of
  [`fptcloud_tagging`](tagging) resources or data sources. Can be updated in place.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the subnet. Pass it to the `subnet_id` argument of
  [`fptcloud_instance`](instance) and other resources.
* `network_id` - The network ID of the subnet, as used by some other FPT Cloud services.
* `network_name` - The network name of the subnet, as FPT Cloud reports it.
* `gateway` - The gateway address of the subnet, as FPT Cloud reports it.
* `created_at` - The time the subnet was created.

## Updating a subnet

The table below shows what happens when you change an argument of an existing subnet.

| Argument | Effect of a change |
|---|---|
| `primary_dns_ip`, `secondary_dns_ip` | DNS servers updated in place. |
| `tag_ids` | Tags updated in place. |
| `vpc_id`, `name`, `type`, `cidr`, `gateway_ip`, `static_ip_pool` | The subnet is **destroyed and recreated**. |

!> **Warning:** The `subnet_id` of an instance cannot change either. When a subnet is recreated,
every instance that references it through `subnet_id` is destroyed and recreated too, and the data
on its root disk is lost. Always read the plan before you apply, and look for
`# forces replacement`.

Please also note:

* Removing `primary_dns_ip` or `secondary_dns_ip` from your configuration does not clear the DNS
  server. Terraform keeps the current value. To change it, set a new address.
* Setting `tag_ids = []` does not remove tags that are already attached to the subnet. Remove them
  in the FPT Cloud console.

## Timeouts

This resource has no `timeouts` block. After it sends the create request, Terraform waits until the
new subnet is available, for up to the `timeout` set on the [provider](../index), in minutes
(default `15`).

## Import

Importing an existing subnet is not supported. The import does not record the VPC of the subnet,
so Terraform cannot read the subnet back. To use an existing subnet in your configuration, read it
with the [`fptcloud_subnet`](../data-sources/subnet) data source instead.

## Troubleshooting

* **The apply fails on the CIDR or the gateway.** Check that `cidr` does not overlap another subnet
  in the same VPC, including subnets created in the console, and that `gateway_ip` is inside
  `cidr`. The [`fptcloud_subnet`](../data-sources/subnet) data source lists the subnets that
  already exist.
* **Errors when you create many subnets at once.** Terraform creates independent resources in
  parallel. If you create many subnets in the same VPC in one apply and some of them fail, run the
  apply again with less parallelism, for example `terraform apply -parallelism=1`. Subnets that were
  created successfully are kept in the state and are not created again.
* **The plan fails because a subnet no longer exists.** If a subnet was deleted outside Terraform,
  reading it fails instead of planning a new subnet. Remove it from the state with
  `terraform state rm fptcloud_subnet.<name>`, then run `terraform apply` to create it again.
