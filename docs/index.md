---
page_title: "Provider: FPT Cloud"
description: |-
  Use the FPT Cloud provider to manage FPT Cloud infrastructure such as instances, storage disks, subnets, security groups and floating IPs with Terraform.
---

# FPT Cloud Provider

[FPT Cloud](https://fptcloud.com) is the cloud platform built and operated by FPT Corporation, with
data centers in Vietnam (Hanoi, Ho Chi Minh City) and Japan (Tokyo). The FPT Cloud provider lets you
describe your FPT Cloud infrastructure as code and manage its full lifecycle with Terraform.

With this provider you can manage, among others:

* **Compute** – instances (virtual servers), instance groups, SSH keys and snapshots.
* **Storage** – storage disks (block storage) and their storage policies.
* **Networking** – subnets, security groups and rules, and floating IPs.
* **Tagging** – tags that you attach to resources to organize and track them.

New to the provider? Start with the [Getting Started](guides/getting-started) guide, which takes
you from an empty folder to a running instance in a few minutes. For the concepts behind each
service, see the [FPT Cloud documentation](https://docs.fptcloud.com/docs/intro/).

## Example Usage

```terraform
terraform {
  required_providers {
    fptcloud = {
      source  = "fpt-corp/fptcloud"
      version = "~> 0.3"
    }
  }
}

# Credentials are read from the FPTCLOUD_TOKEN, FPTCLOUD_TENANT_NAME
# and FPTCLOUD_REGION environment variables.
provider "fptcloud" {}

# Look up the VPC that your resources will be created in.
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}
```

-> **Note:** We recommend that you pin the provider version (for example `version = "~> 0.3"`
or an exact version) so that an upgrade never happens without your review. The list of versions
and their changes is on the [Terraform Registry](https://registry.terraform.io/providers/fpt-corp/fptcloud/latest).

## Authentication and Configuration

The provider needs three values to connect to FPT Cloud:

| Value | Where to find it |
|---|---|
| **API token** | In the FPT Cloud console, open **Token** from the sidebar and create a new token. Copy it when it is shown. |
| **Tenant name** | The name of your tenant (organization), as shown in the FPT Cloud console. |
| **Region** | The region your VPC is in: `VN/HAN` (Hanoi), `VN/SGN` (Ho Chi Minh City) or `JP/JCSI2` (Japan). |

You can provide them in two ways. When both are used, the values in the `provider` block take
precedence over the environment variables.

### Environment variables (recommended)

Environment variables keep your token out of your code and out of version control.

```shell
export FPTCLOUD_TOKEN="<your API token>"
export FPTCLOUD_TENANT_NAME="example-tenant"
export FPTCLOUD_REGION="VN/HAN"
```

```terraform
provider "fptcloud" {}
```

### Provider arguments

You can also set the values in the `provider` block. Pass the token through a sensitive variable,
never as a literal string.

```terraform
variable "fptcloud_token" {
  type      = string
  sensitive = true
}

provider "fptcloud" {
  token       = var.fptcloud_token
  tenant_name = "example-tenant"
  region      = "VN/SGN"
}
```

### Japan region

Resources in Japan are served by a separate API endpoint. Set `api_endpoint` together with the
`JP/JCSI2` region:

```terraform
provider "fptcloud" {
  region       = "JP/JCSI2"
  tenant_name  = "example-tenant"
  api_endpoint = "https://console-api.fptcloud.jp/api"
}
```

### Several regions in one configuration

Declare one `provider` block per region and give each extra block an `alias`. Then select the
provider on each resource or data source with the `provider` meta-argument.

```terraform
provider "fptcloud" {
  region      = "VN/HAN"
  tenant_name = "example-tenant"
}

provider "fptcloud" {
  alias       = "hcm"
  region      = "VN/SGN"
  tenant_name = "example-tenant"
}

data "fptcloud_vpc" "hanoi" {
  name = "example-vpc-hanoi"
}

data "fptcloud_vpc" "hcm" {
  provider = fptcloud.hcm
  name     = "example-vpc-hcm"
}
```

### Longer operations

Creating, resizing and deleting resources is asynchronous in FPT Cloud: Terraform sends the request
and then waits for the operation to finish. The `timeout` argument sets how long Terraform waits, in
minutes. If large instances or disks time out in your environment, raise it:

```terraform
provider "fptcloud" {
  region      = "VN/HAN"
  tenant_name = "example-tenant"
  timeout     = 30
}
```

## Argument Reference

All arguments are optional in the `provider` block when the matching environment variable is set.

* `token` - (Required) The FPT Cloud API token. Can also be set with the `FPTCLOUD_TOKEN`
  environment variable.
* `tenant_name` - (Required) The name of your FPT Cloud tenant. Can also be set with the
  `FPTCLOUD_TENANT_NAME` environment variable.
* `region` - (Required) The region to manage resources in. Valid values are `VN/HAN`, `VN/SGN` and
  `JP/JCSI2`. Can also be set with the `FPTCLOUD_REGION` environment variable.
* `api_endpoint` - (Optional) The URL of the FPT Cloud API. Defaults to
  `https://console-api.fptcloud.com/api`. Set it to `https://console-api.fptcloud.jp/api` for the
  `JP/JCSI2` region. Can also be set with the `FPTCLOUD_API_URL` environment variable.
* `timeout` - (Optional) How long, in minutes, Terraform waits for an operation to complete.
  Defaults to `15` for Compute, Storage, Networking and Tagging resources. For these resources it can
  also be set with the `FPTCLOUD_TIMEOUT` environment variable.

## General VPC and Specific VPC

FPT Cloud offers two VPC types, shown in the console as **General** and **Specific**. They run on
different infrastructure, so a few features, such as some storage disk types and instance
snapshots, are only available in one of them. Where a resource or argument only works in one VPC
type, its page says so. For the
complete comparison, see
[Features by VPC Type](https://docs.fptcloud.com/docs/fpt-cloud-server/instance/overview/#features-by-vpc-type).

## Guides

* [Getting Started](guides/getting-started) – install the provider, configure credentials and
  create your first instance.
* [Web Server](guides/web-server) – a public web server with a subnet, a security group, a
  floating IP and a data disk.
* [Scaling and Resizing](guides/scaling-and-resizing) – change the flavor, grow disks and control
  the power state of instances safely.
