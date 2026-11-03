---
page_title: "Getting Started with the FPT Cloud Provider"
subcategory: ""
description: |-
  Install the FPT Cloud provider, configure your credentials and create your first instance.
---

# Getting Started with the FPT Cloud Provider

This guide takes you from an empty folder to a running FPT Cloud instance. You will:

1. Prepare your credentials.
2. Configure the provider.
3. Find the image, flavor and storage policy to use.
4. Create a subnet and an instance.
5. Clean everything up.

It takes about 15 minutes, most of which is waiting for the instance to start.

## Before you begin

You need:

* [Terraform](https://developer.hashicorp.com/terraform/install) 1.5 or later. Check your version
  with `terraform -version`.
* An FPT Cloud account with access to a VPC. This guide uses a VPC named `example-vpc`; replace
  it with the name of your VPC.
* An SSH key pair on your computer, for example `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub`.
  If you do not have one, create it with `ssh-keygen -t rsa -b 4096`.

## Step 1: Prepare your credentials

The provider authenticates with an API token, and needs to know your tenant and region.

1. Sign in to the FPT Cloud console.
2. Open **Token** from the sidebar and create a new token. Copy the token value and keep it safe:
   it gives access to your resources.
3. Note your **tenant name**, as shown in the console.
4. Note the **region** of your VPC: `VN/HAN` (Hanoi), `VN/SGN` (Ho Chi Minh City) or `JP/JCSI2`
   (Japan).

Export the three values as environment variables in your terminal, so they never appear in your
code:

```shell
export FPTCLOUD_TOKEN="<your API token>"
export FPTCLOUD_TENANT_NAME="example-tenant"
export FPTCLOUD_REGION="VN/HAN"
```

-> **Note:** For the `JP/JCSI2` region, also set
`export FPTCLOUD_API_URL="https://console-api.fptcloud.jp/api"`.

## Step 2: Configure the provider

Create a folder for your project and, inside it, a file named `versions.tf`:

```terraform
terraform {
  required_version = ">= 1.5"

  required_providers {
    fptcloud = {
      source  = "fpt-corp/fptcloud"
      version = "~> 0.3"
    }
  }
}

provider "fptcloud" {}
```

Then download the provider:

```shell
terraform init
```

The output ends with `Terraform has been successfully initialized!`.

## Step 3: Find the image, flavor and storage policy

An instance is built from three catalog items that differ between regions and VPCs:

* an **image** (the operating system),
* a **flavor** (the number of vCPUs and the amount of RAM),
* a **storage policy** (the performance tier of the disk).

Create a file named `catalog.tf` that lists what your VPC offers:

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_image" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}

data "fptcloud_flavor" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}

data "fptcloud_storage_policy" "all" {
  vpc_id = data.fptcloud_vpc.example.id
}

output "image_names" {
  value = data.fptcloud_image.all.images[*].name
}

output "flavors" {
  value = [for f in data.fptcloud_flavor.all.flavors : "${f.name}: ${f.cpu} vCPU, ${f.memory_mb} MB"]
}

output "storage_policy_names" {
  value = data.fptcloud_storage_policy.all.storage_policies[*].name
}
```

Run `terraform apply` and confirm with `yes`. Data sources only read information, so nothing is
created. Pick an image, a flavor and a storage policy from the output. You can then delete
`catalog.tf`, or keep only the `fptcloud_vpc` data source, which the next step uses.

## Step 4: Create a subnet and an instance

Create a file named `main.tf`. Replace the image, flavor and storage policy names with the ones you
picked, and choose a CIDR that does not overlap with the subnets already in your VPC.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_image" "ubuntu" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Ubuntu-22.04"]
  }
}

data "fptcloud_flavor" "small" {
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

resource "fptcloud_subnet" "app" {
  vpc_id     = data.fptcloud_vpc.example.id
  name       = "getting-started-subnet"
  type       = "NAT_ROUTED"
  cidr       = "10.20.1.0/24"
  gateway_ip = "10.20.1.1"
}

resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "getting-started-01"
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = data.fptcloud_flavor.small.flavors[0].name
  subnet_id         = fptcloud_subnet.app.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
}

output "instance_id" {
  value = fptcloud_instance.app.id
}

output "private_ip" {
  value = fptcloud_instance.app.private_ip
}
```

Preview the changes:

```shell
terraform plan
```

The plan should show `2 to add, 0 to change, 0 to destroy`. Create the resources:

```shell
terraform apply
```

Type `yes` to confirm. Terraform waits until the instance is running, which usually takes a few
minutes. When it finishes, it prints the instance ID and its private IP address. You can also see
the new instance in the FPT Cloud console.

The instance only has a private address so far. To reach it from the Internet, add a security group
and a floating IP as shown in the [Web Server](web-server) guide.

## Step 5: Clean up

When you no longer need the resources, delete them:

```shell
terraform destroy
```

Type `yes` to confirm. Terraform deletes the instance first and then the subnet.

## Troubleshooting

| Message or symptom | What to do |
|---|---|
| Authentication or permission errors | Check `FPTCLOUD_TOKEN`, `FPTCLOUD_TENANT_NAME` and `FPTCLOUD_REGION`. The region must be the region of your VPC, and the token must not be expired. |
| `Invalid index` on `images[0]`, `flavors[0]` or `storage_policies[0]` | The filter matched nothing. Check the exact name with the `catalog.tf` outputs from Step 3. Names are case-sensitive. |
| `name cannot contain whitespace` | Names of instances and other resources must not contain spaces. Use `-` instead. |
| The subnet cannot be created | The CIDR probably overlaps an existing subnet in the VPC, or the gateway is outside the CIDR. Choose another range. |
| `timeout while waiting for state` | The operation took longer than the provider `timeout` (15 minutes by default). Check the resource in the console, then raise `timeout` in the `provider` block if needed. |

## Next steps

* [Web Server](web-server) – make the instance reachable from the Internet and add a data disk.
* [Scaling and Resizing](scaling-and-resizing) – change the flavor and grow disks later on.
* [`fptcloud_instance`](../resources/instance) – every option of the instance resource.
