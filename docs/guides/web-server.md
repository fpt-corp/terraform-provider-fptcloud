---
page_title: "Deploy a Public Web Server"
subcategory: ""
description: |-
  Build a web server that is reachable from the Internet: a subnet, an instance, a data disk, a security group and a floating IP.
---

# Deploy a Public Web Server

This guide builds a small but complete web server setup in one Terraform configuration:

* a **subnet** with outbound Internet access,
* an **instance** that runs the web server,
* a **data disk** for website content, separate from the operating system,
* a **security group** that allows HTTP and HTTPS from anywhere and SSH only from your office,
* a **floating IP** that gives the instance a public address.

It assumes that you have completed the [Getting Started](getting-started) guide and that your
credentials are set as environment variables.

## Architecture

```
            Internet
               │
      floating IP (public)
               │
┌──────────────┼───────────────── example-vpc ─┐
│              │                               │
│   ┌──────── web-subnet 10.30.1.0/24 ──────┐  │
│   │          │                            │  │
│   │   web-sg ┤ 80, 443 from anywhere      │  │
│   │          │ 22 from the office         │  │
│   │   ┌──────┴────────┐                   │  │
│   │   │ web-server-01 │── web-data-disk   │  │
│   │   └───────────────┘                   │  │
│   └───────────────────────────────────────┘  │
└──────────────────────────────────────────────┘
```

## Variables

Create `variables.tf`. Only `office_cidr` must be changed: set it to the public address range of
your office or your own public IP followed by `/32`.

```terraform
variable "vpc_name" {
  description = "Name of the VPC to deploy into."
  type        = string
  default     = "example-vpc"
}

variable "office_cidr" {
  description = "Address range allowed to connect with SSH."
  type        = string
  default     = "203.0.113.0/24"
}

variable "ssh_public_key_path" {
  description = "Path to the SSH public key installed on the instance."
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}
```

## Network and instance

Create `main.tf` with the lookups, the subnet and the instance. Replace the image, flavor and
storage policy names with ones listed in your VPC (see Step 3 of the
[Getting Started](getting-started) guide).

```terraform
data "fptcloud_vpc" "example" {
  name = var.vpc_name
}

data "fptcloud_image" "ubuntu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Ubuntu-22.04"]
  }
}

data "fptcloud_flavor" "web" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["2C4G"]
  }
}

data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"]
  }
}

resource "fptcloud_subnet" "web" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "web-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.30.1.0/24"
  gateway_ip = "10.30.1.1"
}

resource "fptcloud_instance" "web" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "web-server-01"
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = data.fptcloud_flavor.web.flavors[0].name
  subnet_id         = fptcloud_subnet.web.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand(var.ssh_public_key_path))
  status            = "POWERED_ON"
}
```

`NAT_ROUTED` gives the subnet outbound Internet access, so the instance can download packages.
Use `ISOLATED` for subnets that must never reach the Internet, such as database subnets.

## Data disk

Add a disk for the website content and attach it to the instance. Keeping data on its own disk lets
you grow it, or move it to a new instance, without touching the operating system.

```terraform
resource "fptcloud_storage" "web_data" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "web-data-disk"
  type              = "EXTERNAL"
  size_gb           = 50
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  instance_id       = fptcloud_instance.web.id
}
```

## Security group

A security group is a virtual firewall. Traffic that no rule allows is denied. The group is
attached to the instance through `apply_to`, which lists the private IP addresses the group applies
to.

```terraform
# Read the instance once it exists, so that its private IP is known.
# If the IP is still empty right after creation, run `terraform apply` again.
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

locals {
  web_rules = {
    http  = { port = "80", sources = ["0.0.0.0/0"], description = "HTTP from anywhere" }
    https = { port = "443", sources = ["0.0.0.0/0"], description = "HTTPS from anywhere" }
    ssh   = { port = "22", sources = [var.office_cidr], description = "SSH from the office" }
  }
}

resource "fptcloud_security_group_rule" "web" {
  for_each = local.web_rules

  vpc_id            = data.fptcloud_vpc.example.id
  security_group_id = fptcloud_security_group.web.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = each.value.port
  sources           = each.value.sources
  description       = each.value.description
}
```

-> **Note:** Attach security groups with `apply_to` rather than with `security_group_ids` on the
instance. See [Attaching security groups](../resources/instance#attaching-security-groups) for
the reasons.

## Floating IP

Allocate a public IP address and associate it with the instance:

```terraform
resource "fptcloud_floating_ip" "web" {
  vpc_id = data.fptcloud_vpc.example.id
}

resource "fptcloud_floating_ip_association" "web" {
  vpc_id         = data.fptcloud_vpc.example.id
  floating_ip_id = fptcloud_floating_ip.web.id
  instance_id    = fptcloud_instance.web.id
}
```

Because the association is its own resource, you can later move the address to another instance by
changing `instance_id`. Terraform then replaces only the association: the floating IP, and so the
public address, stays the same.

## Outputs

Create `outputs.tf`:

```terraform
output "public_ip" {
  description = "Public address of the web server."
  value       = fptcloud_floating_ip.web.ip_address
}

output "private_ip" {
  description = "Private address of the web server inside the VPC."
  value       = data.fptcloud_instance.web.private_ip
}
```

## Deploy

```shell
terraform init
terraform plan
terraform apply
```

When the apply finishes, connect to the instance with the public IP. The login user depends on the
image (for example `ubuntu` on Ubuntu images):

```shell
ssh ubuntu@$(terraform output -raw public_ip)
```

The instance resource has no option to run a start-up script, so install the web server software
after you connect, or with your configuration management tool. For example, on Ubuntu:

```shell
sudo apt-get update && sudo apt-get install -y nginx
```

Then open `http://<public_ip>` in your browser.

The data disk is attached as a new, empty block device. Format and mount it once from inside the
instance before you use it, for example with `lsblk` to find the device, then `mkfs.ext4` and an
entry in `/etc/fstab`.

## Clean up

FPT Cloud does not delete a security group that is still attached to an instance, so detach it
first. In `main.tf`, change `apply_to` on the security group to an empty list:

```terraform
resource "fptcloud_security_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-sg"
  subnet_id = fptcloud_subnet.web.id
  type      = "ACL"
  apply_to  = []
}
```

Apply the change, then destroy everything:

```shell
terraform apply
terraform destroy
```

Terraform removes the remaining resources in the right order: the floating IP association, the
rules and the security group before the instance, and the instance before the subnet.

## Next steps

* Add more instances behind the same security group by adding their private IPs to `apply_to`.
* Spread several web servers across physical hosts with an
  [instance group](../resources/instance_group).
* Grow the data disk or the instance later with the [Scaling and Resizing](scaling-and-resizing)
  guide.
