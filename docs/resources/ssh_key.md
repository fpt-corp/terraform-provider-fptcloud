---
page_title: "fptcloud_ssh_key Resource - FPT Cloud"
subcategory: "Compute"
description: |-
  Manages an SSH public key saved in your FPT Cloud account.
---

# fptcloud_ssh_key (Resource)

Manages an SSH public key saved in your FPT Cloud account. Saved keys appear under **SSH Keys** in
the FPT Cloud console, where you and your team can select them when you create instances. An SSH key
belongs to your account, not to a VPC, so this resource has no `vpc_id`.

This resource only stores the **public** key. Keep the private key on your own machine; Terraform
never needs it.

To install the key on an instance, pass the public key text to `ssh_key` on
[fptcloud_instance](instance). The simplest way is to reference this resource's `public_key`
attribute, as shown in the examples.

To learn more, see [SSH Keys](https://docs.fptcloud.com/docs/fpt-cloud-server/ssh-keys/overview/)
in the FPT Cloud documentation.

~> **Important:** Deleting an SSH key from FPT Cloud does not revoke access to instances that already
use it. The key stays in `~/.ssh/authorized_keys` inside each instance until you remove it there.

## Example Usage

### Register a public key from a file

Use this when the key pair already exists on your machine, for example one created with
`ssh-keygen -t rsa -b 4096`.

```terraform
resource "fptcloud_ssh_key" "deployer" {
  name       = "deployer-key"
  public_key = file(pathexpand("~/.ssh/id_rsa.pub"))
}
```

### Register a public key from a variable

Use this in a CI pipeline or a shared module, where the key is passed in instead of read from a
local file.

```terraform
variable "ssh_public_key" {
  type        = string
  description = "OpenSSH public key, for example the content of id_rsa.pub."
}

resource "fptcloud_ssh_key" "deployer" {
  name       = "deployer-key"
  public_key = var.ssh_public_key
}
```

### Use the key on an instance

`ssh_key` on [fptcloud_instance](instance) expects the public key text itself, not the key name or
ID. Reference `public_key` so that the instance always gets the same key you registered.

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

resource "fptcloud_ssh_key" "deployer" {
  name       = "deployer-key"
  public_key = file(pathexpand("~/.ssh/id_rsa.pub"))
}

resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = data.fptcloud_subnet.web.subnets[0].id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = fptcloud_ssh_key.deployer.public_key
  status            = "POWERED_ON"
}
```

After the instance starts, connect with the matching private key, for example
`ssh -i ~/.ssh/id_rsa <user>@<instance IP>`. The user name depends on the image.

To use a key that is already saved in FPT Cloud and not managed by this configuration, read it with
the [fptcloud_ssh_key](../data-sources/ssh_key) data source and pass its `public_key` instead.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the key, as shown in the FPT Cloud console. Must not contain spaces
  or other whitespace. Changing this forces a new resource to be created.
* `public_key` - (Required) The SSH public key in OpenSSH format, for example the content of
  `~/.ssh/id_rsa.pub` (`ssh-rsa AAAA... comment`). Never put a private key here. Changing this
  forces a new resource to be created.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the SSH key.

## Rotating a key

Both arguments force a new key, so changing `public_key` deletes the saved key and registers the new
one. Plan carefully when instances use the key:

* Changing `ssh_key` on an existing [fptcloud_instance](instance) replaces the instance. If an
  instance references `fptcloud_ssh_key.deployer.public_key`, rotating the key therefore also
  re-creates that instance, and its root disk data is lost.
* To rotate the key on running instances without re-creating them, add the new public key to
  `~/.ssh/authorized_keys` inside each instance yourself, then remove the old one. If the instances
  reference this resource, add `ignore_changes = [ssh_key]` in their `lifecycle` block first.

If someone deletes the key in the FPT Cloud console, Terraform registers it again on the next
`terraform apply`.

## Import

You can import an existing SSH key by its ID. The name does not work as an import ID. To find the ID
of a key, read it by name with the [fptcloud_ssh_key](../data-sources/ssh_key) data source and output
its `id`.

```shell
terraform import fptcloud_ssh_key.deployer <ssh_key_id>
```

With Terraform 1.5 or later, you can use an `import` block instead:

```terraform
import {
  to = fptcloud_ssh_key.deployer
  id = "<ssh_key_id>"
}

resource "fptcloud_ssh_key" "deployer" {
  name       = "deployer-key"
  public_key = file(pathexpand("~/.ssh/id_rsa.pub"))

  lifecycle {
    # The public key is not read back after an import. Without this, the
    # first plan would replace the imported key.
    ignore_changes = [public_key]
  }
}
```

Terraform does not read `public_key` back from FPT Cloud. After an import, the first plan therefore
shows the key as replaced unless you add `ignore_changes = [public_key]` as shown above.
