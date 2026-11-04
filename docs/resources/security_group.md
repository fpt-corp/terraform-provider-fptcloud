---
page_title: "fptcloud_security_group Resource - FPT Cloud"
subcategory: "Networking"
description: |-
  Manages a security group (virtual firewall) in an FPT Cloud VPC and the resources it applies to.
---

# fptcloud_security_group (Resource)

Manages a security group: a virtual firewall that controls the network traffic going into and out
of resources in your VPC. A security group holds a set of rules, and it protects every resource
whose IP address you list in its `apply_to` argument.

You add rules to a group with the [`fptcloud_security_group_rule`](security_group_rule) resource.
When FPT Cloud creates a group, it also adds default rules that allow all outbound traffic (one for
IPv4 and one for IPv6). Inbound traffic that does not match any rule is denied, so a new group
blocks all inbound traffic until you add rules that allow it.

To learn more, see [Security Groups](https://docs.fptcloud.com/docs/fpt-network/security-groups/overview/)
in the FPT Cloud documentation.

~> **Important:** Attach a security group to instances with the `apply_to` argument of this
resource, not with the `security_group_ids` argument of [`fptcloud_instance`](instance). See
[Attaching a security group to instances](#attaching-a-security-group-to-instances).

## Example Usage

The examples on this page share the following lookups, subnet and variable. Catalog names such as
image, flavor and storage policy names differ between regions and VPCs, so replace the example
values with names listed in your VPC.

```terraform
variable "ssh_public_key" {
  description = "Public SSH key, in OpenSSH format, used to log in to the instances."
  type        = string
}

data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Use a storage policy name listed in your VPC
  }
}

resource "fptcloud_subnet" "web" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "web-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.10.1.0/24"
  gateway_ip = "10.10.1.1"
}
```

### Basic security group

Use this when you want to create an empty group first and attach it to resources later. The group
only has the default outbound rules created by FPT Cloud.

```terraform
resource "fptcloud_security_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-sg"
  subnet_id = fptcloud_subnet.web.id
  type      = "ACL"
}
```

### Web server open to HTTP and HTTPS

Use this when an instance serves a website: HTTP and HTTPS are open to everyone, and SSH is open
only to your office network. The private IP of the instance is read through the
[`fptcloud_instance`](../data-sources/instance) data source, because a newly created instance can
report an empty `private_ip` for a short time.

```terraform
resource "fptcloud_instance" "web" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "web-server-01"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = fptcloud_subnet.web.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = var.ssh_public_key
  status            = "POWERED_ON"
}

data "fptcloud_instance" "web" {
  vpc_id = data.fptcloud_vpc.example.id
  id     = fptcloud_instance.web.id
}

resource "fptcloud_security_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-sg"
  subnet_id = fptcloud_subnet.web.id
  type      = "ACL"
  apply_to  = [data.fptcloud_instance.web.private_ip]
}

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

### One security group for several instances

Use this when several instances share the same firewall rules. List the private IP of every
instance in `apply_to`; adding or removing an instance updates the group in place.

```terraform
resource "fptcloud_instance" "app" {
  count = 3

  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-0${count.index + 1}"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = fptcloud_subnet.web.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = var.ssh_public_key
  status            = "POWERED_ON"
}

data "fptcloud_instance" "app" {
  count = length(fptcloud_instance.app)

  vpc_id = data.fptcloud_vpc.example.id
  id     = fptcloud_instance.app[count.index].id
}

resource "fptcloud_security_group" "app" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "app-sg"
  subnet_id = fptcloud_subnet.web.id
  type      = "ACL"
  apply_to  = data.fptcloud_instance.app[*].private_ip
}
```

### Attachments managed outside Terraform

Use this when your team attaches the group to resources in the FPT Cloud console and you want
Terraform to manage only the group itself. Without `ignore_changes`, Terraform would detach the
group from those resources on the next apply, because `apply_to` is not set in the configuration.

```terraform
resource "fptcloud_security_group" "shared" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "shared-sg"
  subnet_id = fptcloud_subnet.web.id
  type      = "ACL"

  lifecycle {
    ignore_changes = [apply_to]
  }
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC to create the security group in. Use the `id` of the
  [`fptcloud_vpc`](../data-sources/vpc) data source. Changing it after the group is created is not
  supported; to use another VPC, create a new security group.
* `name` - (Required) The name of the security group. Must not contain spaces. Can be updated in
  place.
* `type` - (Required) The firewall type of the security group. Valid values are:
  * `ACL` - controls traffic to and from the Internet.
  * `DFW` - controls traffic within the local network.

  Cannot be changed after the group is created: `terraform apply` stops with an error. To use
  another type, create a new security group.
* `subnet_id` - (Optional) The ID of the subnet the security group belongs to. Use the `id` of a
  [`fptcloud_subnet`](subnet) resource or data source. You **must set it when you create the
  group**: creation fails without it. Cannot be changed after the group is created:
  `terraform apply` stops with an error.
* `apply_to` - (Optional) A set of IP addresses of the resources the security group protects,
  usually the private IP addresses of instances, for example
  `[data.fptcloud_instance.web.private_ip]`. CIDR blocks are also accepted. Every element must be a
  non-empty address. Can be updated in place: Terraform sends the full list, so addresses you
  remove are detached from the group. If you leave it unset, the group applies to no resource;
  see [Attachments managed outside Terraform](#attachments-managed-outside-terraform) to keep
  attachments made in the console.
* `tag_ids` - (Optional) A set of tag IDs to attach to the security group. Use the `id` of
  [`fptcloud_tagging`](tagging) resources or data sources. Can be updated in place. If you leave it
  unset, Terraform keeps the tags the group already has.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the security group. Use it as the `security_group_id` of
  [`fptcloud_security_group_rule`](security_group_rule).
* `edge_gateway_id` - The ID of the edge gateway the security group is associated with. It can be
  empty.
* `created_at` - The time the security group was created.

## Attaching a security group to instances

Attach a security group to instances by listing their private IP addresses in `apply_to`, as in the
[web server example](#web-server-open-to-http-and-https). We recommend this over the
`security_group_ids` argument of [`fptcloud_instance`](instance), for two reasons:

* `security_group_ids` is only used when the instance is created. Later changes to it are not sent
  to FPT Cloud.
* FPT Cloud reports every security group that applies to an instance, including the groups attached
  through `apply_to`. When the same groups are also managed through `security_group_ids`, Terraform
  shows a change on every plan.

Read the private IP through the [`fptcloud_instance`](../data-sources/instance) data source
(`data.fptcloud_instance.web.private_ip`) rather than from the resource
(`fptcloud_instance.web.private_ip`). A newly created instance can report an empty private IP for a
short time, and FPT Cloud rejects an empty address in `apply_to`.

## Default rules

FPT Cloud creates default rules in every new security group: two outbound (`EGRESS`) rules that
allow all protocols, all ports and all destinations, one for IPv4 and one for IPv6. These rules are
not managed by Terraform and do not appear in your configuration. You can see them in the `rules`
attribute of the [`fptcloud_security_group`](../data-sources/security_group) data source.

Do not declare a [`fptcloud_security_group_rule`](security_group_rule) identical to a default rule:
FPT Cloud rejects duplicate rules.

## Deleting a security group

FPT Cloud deletes a security group only when it no longer applies to any resource. Because the
group references the instances in `apply_to`, Terraform deletes the group **before** those
instances. If the group still lists them in `apply_to` at that moment, the deletion cannot complete
and `terraform destroy` ends with an error.

To delete a security group together with the instances it protects, detach it first:

1. Set `apply_to = []` on the security group and run `terraform apply`.
2. Run `terraform destroy`.

To delete only the security group, remove it and its rules from your configuration after step 1.
Terraform deletes the rules before the group, because each rule references the group. If you added
rules to the group in the FPT Cloud console, delete them there as well: FPT Cloud does not delete a
group that still has rules you added. The default rules are deleted together with the group.

## Timeouts

This resource has no `timeouts` block. Terraform waits for the security group to be ready after it
is created, after `apply_to` changes, and while it is deleted, for up to the `timeout` set on the
[provider](../index), in minutes (default `15`).

## Troubleshooting

* **`Failed to retrieve security group` on plan or refresh.** The security group was probably
  deleted outside Terraform. Terraform does not recreate it automatically; remove it from the state
  with `terraform state rm fptcloud_security_group.web` (use your resource address), then run
  `terraform apply` to create it again.
* **`Security group type can not be changed` or `Security group subnet can not be changed`.** You
  changed `type` or `subnet_id` of an existing group. Restore the previous value, or create a new
  security group with the value you need.
