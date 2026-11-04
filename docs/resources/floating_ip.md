---
page_title: "fptcloud_floating_ip Resource - FPT Cloud"
subcategory: "Networking"
description: |-
  Allocates a floating IP (a public IP address) in an FPT Cloud VPC.
---

# fptcloud_floating_ip (Resource)

Allocates a floating IP: a public IP address reserved in one of your FPT Cloud VPCs. FPT Cloud maps
the floating IP to the private IP address of an instance, so that the instance can be reached from
the Internet. A floating IP exists independently of any instance: you can move it from one instance
to another without changing the public address, so your DNS records and firewall allowlists stay
valid.

This resource only allocates the address. To connect it to an instance, use the
[`fptcloud_floating_ip_association`](floating_ip_association) resource. Traffic from the Internet
must also be allowed by a security group ([`fptcloud_security_group`](security_group)).
Destroying this resource releases the address back to FPT Cloud.

Floating IPs work in both General VPC and Specific VPC. To learn more, see
[Floating IP](https://docs.fptcloud.com/docs/fpt-network/floating-ip/overview/) in the FPT Cloud
documentation.

!> **Warning:** Destroying a floating IP releases the public address. You cannot get the same
address back afterwards. If other systems depend on the address, protect the resource with
`lifecycle { prevent_destroy = true }`.

## Example Usage

All examples look up the VPC by name with the [`fptcloud_vpc`](../data-sources/vpc) data source:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

### Allocate a public IP address

Use this to reserve a public IP address in your VPC. The allocated address is available in
`ip_address`.

```terraform
resource "fptcloud_floating_ip" "web" {
  vpc_id = data.fptcloud_vpc.example.id
}

output "web_public_ip" {
  value = fptcloud_floating_ip.web.ip_address
}
```

### Give an instance a public IP address

Use this to make an instance reachable from the Internet. The association resource connects the
floating IP to the instance. The [Web Server](../guides/web-server) guide shows the complete setup,
including the security group.

```terraform
resource "fptcloud_floating_ip" "web" {
  vpc_id = data.fptcloud_vpc.example.id
}

resource "fptcloud_floating_ip_association" "web" {
  vpc_id         = data.fptcloud_vpc.example.id
  floating_ip_id = fptcloud_floating_ip.web.id
  instance_id    = fptcloud_instance.web.id
}

output "web_public_ip" {
  value = fptcloud_floating_ip.web.ip_address
}
```

The `fptcloud_instance.web` resource is not shown here; see
[`fptcloud_instance`](instance) for a complete instance definition.

### Floating IP with tags

Use this to attach FPT Cloud tags to the floating IP, for example to track costs per environment.
Tags can be changed in place later.

```terraform
resource "fptcloud_tagging" "env" {
  key   = "env"
  value = "production"
}

resource "fptcloud_floating_ip" "web" {
  vpc_id  = data.fptcloud_vpc.example.id
  tag_ids = [fptcloud_tagging.env.id]
}
```

### Keep the address when you rebuild an instance

Use this when the public address must survive changes to the instance. Because the floating IP is
a separate resource, replacing the instance only replaces the association; the floating IP and its
address are kept. `prevent_destroy` stops Terraform from releasing the address by mistake.

```terraform
resource "fptcloud_floating_ip" "api" {
  vpc_id = data.fptcloud_vpc.example.id

  lifecycle {
    prevent_destroy = true
  }
}

resource "fptcloud_floating_ip_association" "api" {
  vpc_id         = data.fptcloud_vpc.example.id
  floating_ip_id = fptcloud_floating_ip.api.id
  instance_id    = fptcloud_instance.api.id
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC to allocate the floating IP in. Use the `id` of the
  [`fptcloud_vpc`](../data-sources/vpc) data source. Changing this forces a new resource to be
  created, which releases the current address and allocates a new one.
* `tag_ids` - (Optional) A set of tag IDs to attach to the floating IP. Use the `id` of
  [`fptcloud_tagging`](tagging) resources or data sources. Can be updated in place. When omitted,
  Terraform keeps the tags that FPT Cloud reports.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the floating IP. Pass it to the `floating_ip_id` argument of
  [`fptcloud_floating_ip_association`](floating_ip_association).
* `ip_address` - The public IP address that was allocated.
* `created_at` - The time the floating IP was allocated.

## Floating IP on an instance: `public_ip` or the association resource

There are two ways to give an instance a public address:

* The `public_ip` argument of [`fptcloud_instance`](instance). It is only used when the instance is
  created. Changing or removing it later has no effect on FPT Cloud, so you cannot move or detach
  the address this way.
* The [`fptcloud_floating_ip_association`](floating_ip_association) resource. Changing its
  `instance_id` moves the address to another instance, and destroying it detaches the address.

We recommend the association resource for anything that may change over time. Leave `public_ip`
on the instance unset in that case: it is still filled in with the associated address the next time
Terraform reads the instance. To output the address, use `fptcloud_floating_ip.<name>.ip_address`,
which is known as soon as the floating IP is allocated.

## Timeouts

This resource has no `timeouts` block. After it allocates the address, Terraform waits until the
floating IP is ready, for up to the `timeout` set on the [provider](../index), in minutes
(default `15`).

## Import

Importing an existing floating IP is not supported. The import does not record the VPC of the
floating IP, so Terraform cannot read it back. To use an existing floating IP in your
configuration, read it with the [`fptcloud_floating_ip`](../data-sources/floating_ip) data source
and pass its `id` to [`fptcloud_floating_ip_association`](floating_ip_association).

## Troubleshooting

* **The instance does not answer on the public address.** Check that a security group attached to
  the instance allows the traffic, for example TCP port 443 from `0.0.0.0/0`.
* **Destroying the floating IP fails.** FPT Cloud requires a floating IP to be disassociated from
  its instance before it is released. When the association is managed by
  `fptcloud_floating_ip_association`, Terraform removes it first. If you associated the floating IP
  outside Terraform, for example in the console, disassociate it there first.
* **The plan fails because the floating IP no longer exists.** If the floating IP was released
  outside Terraform, reading it fails instead of planning a new one. Remove it from the state with
  `terraform state rm fptcloud_floating_ip.<name>`, then run `terraform apply` to allocate a new
  address.
