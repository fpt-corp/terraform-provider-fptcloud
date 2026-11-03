---
page_title: "fptcloud_floating_ip_association Resource - FPT Cloud"
subcategory: "Networking"
description: |-
  Associates a floating IP (public IP address) with an instance in an FPT Cloud VPC.
---

# fptcloud_floating_ip_association (Resource)

Associates a floating IP with an instance, so that the instance can be reached from the Internet on
the floating IP's public address. Destroying this resource disassociates the floating IP: the
address stays allocated in your VPC and can be associated with another instance.

Use it together with [`fptcloud_floating_ip`](floating_ip), which allocates the address, and a
security group ([`fptcloud_security_group`](security_group)) that allows the traffic. Because the
association is a separate resource, you can move the public address from one instance to another
without releasing it.

To learn more, see
[Floating IP](https://docs.fptcloud.com/docs/fpt-network/floating-ip/overview/) in the FPT Cloud
documentation.

-> **Note:** Every argument of this resource forces a new association when it changes. In practice
this means "disassociate, then associate again". The floating IP and its address are not affected.

## Example Usage

All examples look up the VPC by name with the [`fptcloud_vpc`](../data-sources/vpc) data source:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

### Associate a floating IP with an instance

Use this to give an instance a public IP address. The `fptcloud_instance.web` resource is not shown
here; see [`fptcloud_instance`](instance) for a complete instance definition.

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

Output the address from `fptcloud_floating_ip.web.ip_address`. The `public_ip` attribute of the
instance also shows it, but only after Terraform has read the instance again, for example on the
next `terraform plan` or `terraform apply`.

### Map one port of the floating IP to a port of the instance

Use `floating_ip_port` and `instance_port` together to map a port of the floating IP to a port of
the instance. This example maps port `2222` of the public address to SSH (port `22`) on the
instance.

```terraform
resource "fptcloud_floating_ip_association" "ssh" {
  vpc_id           = data.fptcloud_vpc.example.id
  floating_ip_id   = fptcloud_floating_ip.web.id
  instance_id      = fptcloud_instance.web.id
  floating_ip_port = 2222
  instance_port    = 22
}
```

When you set one of the two ports, you must also set the other one and `instance_id`; otherwise
`terraform plan` fails.

### Move a floating IP to another instance

Use this pattern to switch the public address between instances, for example from a blue to a
green deployment. Change the value of `active` and apply: Terraform disassociates the floating IP
from the old instance, then associates it with the new one. The public address does not change.

```terraform
variable "active" {
  type        = string
  description = "The instance that receives the public address: blue or green."
  default     = "blue"

  validation {
    condition     = contains(["blue", "green"], var.active)
    error_message = "The active instance must be blue or green."
  }
}

locals {
  instance_ids = {
    blue  = fptcloud_instance.blue.id
    green = fptcloud_instance.green.id
  }
}

resource "fptcloud_floating_ip" "app" {
  vpc_id = data.fptcloud_vpc.example.id
}

resource "fptcloud_floating_ip_association" "app" {
  vpc_id         = data.fptcloud_vpc.example.id
  floating_ip_id = fptcloud_floating_ip.app.id
  instance_id    = local.instance_ids[var.active]
}
```

```shell
terraform apply -var="active=green"
```

The public address is unreachable for the short time between the disassociation and the new
association.

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC that contains the floating IP and the instance. Use the
  `id` of the [`fptcloud_vpc`](../data-sources/vpc) data source. Changing this forces a new
  resource to be created.
* `floating_ip_id` - (Required) The ID of the floating IP to associate. Use the `id` of a
  [`fptcloud_floating_ip`](floating_ip) resource or
  [`fptcloud_floating_ip`](../data-sources/floating_ip) data source. Changing this forces a new
  resource to be created.
* `instance_id` - (Optional) The ID of the instance to associate the floating IP with. Use the `id`
  of a [`fptcloud_instance`](instance) resource or data source. Although the argument is optional,
  you should always set it. Changing this forces a new resource to be created: the floating IP is
  disassociated from the current instance and associated with the new one.
* `floating_ip_port` - (Optional) The port on the floating IP, from `1` to `65535`, that is mapped to
  `instance_port`. Requires `instance_id` and `instance_port`. Changing this forces a new resource
  to be created.
* `instance_port` - (Optional) The port on the instance, from `1` to `65535`, that receives the
  traffic of `floating_ip_port`. Requires `instance_id` and `floating_ip_port`. Changing this forces
  a new resource to be created.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the association. It is the same as the ID of the floating IP.

## Behaviour notes

* **One association per floating IP.** The association is identified by the floating IP. Manage
  each floating IP with at most one `fptcloud_floating_ip_association`: destroying it disassociates
  the floating IP completely.
* **Changes made outside Terraform are detected.** Terraform reads which instance the floating IP is
  associated with. If someone disassociates the floating IP in the console, or associates it with
  another instance, the next plan shows the association as replaced, to bring it back to the
  instance in your configuration.
* **Do not combine with `public_ip` on the instance.** The `public_ip` argument of
  [`fptcloud_instance`](instance) is only used when the instance is created and cannot move or
  detach the address later. Use this resource instead and leave `public_ip` unset.
* **Ordering on destroy.** The association references the floating IP, so Terraform always removes
  the association before it releases the floating IP. No `depends_on` is needed.

## Timeouts

This resource has no `timeouts` block. Terraform waits until the floating IP is associated (on
create) or disassociated (on destroy), for up to the `timeout` set on the [provider](../index), in
minutes (default `15`).

## Import

Importing an existing association is not supported. The import does not record the VPC, so
Terraform cannot read the association back. To manage an association that was made in the console,
disassociate the floating IP there first, then let Terraform create the association.
