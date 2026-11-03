---
page_title: "fptcloud_vpc Data Source - FPT Cloud"
subcategory: "Networking"
description: |-
  Use this data source to look up an FPT Cloud VPC by name and get its ID.
---

# fptcloud_vpc (Data Source)

Use this data source to get the ID of one of your VPCs from its name. A VPC (Virtual Private Cloud)
is an isolated network on FPT Cloud that holds your instances, subnets, security groups, floating
IPs and storage disks. Almost every resource of this provider needs a `vpc_id`, so this lookup is
usually the first block of a configuration.

The VPC must exist already, either a General VPC or a Specific VPC, and belong to the tenant set in
the `tenant_name` argument of the [provider](../index). To learn more, see
[VPC](https://docs.fptcloud.com/docs/fpt-network/vpc/overview/) in the FPT Cloud documentation.

## Example Usage

### Look up a VPC by name

Use this to get the VPC ID from the name shown in the FPT Cloud console.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

output "vpc_id" {
  value = data.fptcloud_vpc.example.id
}
```

### Use the VPC ID in other resources

Use this to pass the VPC to the resources you create in it.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

resource "fptcloud_subnet" "app" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "app-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.10.1.0/24"
  gateway_ip = "10.10.1.1"
}

resource "fptcloud_floating_ip" "web" {
  vpc_id = data.fptcloud_vpc.example.id
}
```

### Choose the VPC per environment

Use this to deploy the same configuration into different VPCs, for example one per environment.

```terraform
variable "vpc_name" {
  type        = string
  description = "The name of the VPC to deploy into, as shown in the FPT Cloud console."
}

data "fptcloud_vpc" "selected" {
  name = var.vpc_name
}
```

```shell
terraform apply -var="vpc_name=example-vpc"
```

## Argument Reference

The following arguments are supported:

* `name` - (Required in practice) The name of the VPC, as shown in the FPT Cloud console. It must
  not be empty.
* `id` - (Optional) Accepted by the schema, but a lookup by ID is not supported. See the note below.
* `status` - (Optional) Accepted by the schema, but a lookup by status is not supported. See the
  note below.

Exactly one of `name`, `id` or `status` can be set.

~> **Important:** Only the lookup by `name` works. If you set `id` or `status` instead of `name`,
the lookup fails with the error `please enter vpc name`. If you already know the VPC ID, you do not
need this data source: pass the ID directly, for example through a variable.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the VPC. Pass it to the `vpc_id` argument of other resources and data sources.
* `name` - The name of the VPC, as FPT Cloud reports it.
* `status` - The current status of the VPC, as FPT Cloud reports it.

The type of the VPC (General VPC or Specific VPC) is not exported.

## Errors

The lookup fails, and Terraform stops, when no VPC with that name exists in the tenant set on the
provider. Check the name in the FPT Cloud console and the `tenant_name` of the provider.
