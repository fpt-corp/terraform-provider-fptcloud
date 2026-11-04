---
page_title: "fptcloud_ssh_key Data Source - FPT Cloud"
subcategory: "Compute"
description: |-
  Use this data source to look up an SSH public key saved in your FPT Cloud account by name or ID.
---

# fptcloud_ssh_key (Data Source)

Use this data source to look up an SSH public key that is already saved in your FPT Cloud account,
for example a key that a colleague uploaded in the console. It returns the key's ID, name and public
key text. Typical uses:

* Install a saved key on a new [fptcloud_instance](../resources/instance) by passing its
  `public_key` to `ssh_key`.
* Find the ID of a key so that you can import it into an
  [fptcloud_ssh_key](../resources/ssh_key) resource.

SSH keys belong to your account, not to a VPC, so this data source has no `vpc_id`. To learn more,
see [SSH Keys](https://docs.fptcloud.com/docs/fpt-cloud-server/ssh-keys/overview/) in the FPT Cloud
documentation.

## Example Usage

### Look up a key by name

```terraform
data "fptcloud_ssh_key" "deployer" {
  name = "deployer-key"
}

output "deployer_key_id" {
  value = data.fptcloud_ssh_key.deployer.id
}
```

### Install a saved key on an instance

`ssh_key` on [fptcloud_instance](../resources/instance) expects the public key text, so pass
`public_key`, not the name or the ID.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_subnet" "web" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["web-subnet"]
  }
}

data "fptcloud_storage_policy" "ssd" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Premium-SSD"] # Use a storage policy name listed in your VPC
  }
}

data "fptcloud_ssh_key" "deployer" {
  name = "deployer-key"
}

resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = data.fptcloud_subnet.web.subnets[0].id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = data.fptcloud_ssh_key.deployer.public_key
  status            = "POWERED_ON"
}
```

~> **Important:** Changing `ssh_key` on an existing instance replaces the instance. If someone
deletes the saved key and uploads a different one with the same name, the next plan replaces every
instance that reads it through this data source.

## Argument Reference

Exactly one of `name` or `id` must be set.

* `name` - (Optional) The name of the SSH key.
* `id` - (Optional) The ID of the SSH key.

The value is matched against both the name and the ID of every key in your account:

1. A key whose name or ID is exactly the value is returned.
2. Otherwise, if exactly one key's name or ID contains the value, that key is returned.
3. Otherwise the data source fails with the error `SSH key not found`. This happens when no key
   matches, and also when the value is part of several key names. Use the full name to avoid it.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the SSH key.
* `name` - The name of the SSH key.
* `public_key` - The SSH public key text in OpenSSH format. Pass it to `ssh_key` on
  [fptcloud_instance](../resources/instance).
