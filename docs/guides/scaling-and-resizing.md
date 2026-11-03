---
page_title: "Scaling and Resizing Instances"
subcategory: ""
description: |-
  Change the vCPU and RAM of an instance, grow its root and data disks, control its power state and add more instances, safely and without data loss.
---

# Scaling and Resizing Instances

Workloads change over time. This guide shows how to adapt existing FPT Cloud resources with
Terraform, without recreating them and without losing data:

* [Scale up or down: change the flavor](#change-the-flavor)
* [Grow the root disk](#grow-the-root-disk)
* [Grow a data disk](#grow-a-data-disk)
* [Power instances on and off](#power-instances-on-and-off)
* [Scale out: add more instances](#scale-out-add-more-instances)
* [Protect important instances](#protect-important-instances)

The examples start from the instance below. Only the arguments that change are discussed in each
section.

```terraform
resource "fptcloud_instance" "app" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-server-01"
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = "2C4G"
  subnet_id         = fptcloud_subnet.app.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
}

resource "fptcloud_storage" "app_data" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "app-data-disk"
  type              = "EXTERNAL"
  size_gb           = 100
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  instance_id       = fptcloud_instance.app.id
}
```

## Before you change anything

Always run `terraform plan` and read it before `terraform apply`:

* `~ update in-place` means the resource is modified and keeps its data.
* `-/+ destroy and then create replacement` and `# forces replacement` mean the resource is
  **deleted and recreated**. For an instance, the data on its root disk is lost.

The arguments in this guide are all updated in place. The
[Updating an instance](../resources/instance#updating-an-instance) table lists the behaviour of
every argument.

## Change the flavor

The flavor sets the number of vCPUs and the amount of RAM. To scale up, change `flavor_name` to a
larger flavor and apply:

```terraform
resource "fptcloud_instance" "app" {
  # ...
  flavor_name = "4C8G" # was "2C4G"
}
```

```shell
terraform plan   # shows: ~ flavor_name = "2C4G" -> "4C8G"
terraform apply
```

Terraform waits until FPT Cloud reports that the resize is complete. Scaling down works the same
way with a smaller flavor. Use the [`fptcloud_flavor`](../data-sources/flavor) data source to list
the flavors of your VPC.

Some resizes require the instance to be powered off, and instances with a physical NVMe disk
(`is_nvme = true`) cannot be resized. If FPT Cloud refuses the resize because of the power state:

1. Set `status = "POWERED_OFF"` and apply.
2. Change `flavor_name` and apply.
3. Set `status = "POWERED_ON"` and apply.

-> **Note:** Plan a resize in a maintenance window, because it can interrupt the applications
running on the instance.

## Grow the root disk

Increase `storage_size_gb`:

```terraform
resource "fptcloud_instance" "app" {
  # ...
  storage_size_gb = 80 # was 40
}
```

The root disk can only grow. If you set a smaller value, `terraform plan` stops with an error
before anything is changed. FPT Cloud also refuses to grow the root disk while the instance has a
snapshot, is suspended, or has a GPU attached.

After the apply, the disk is larger but the partition and file system inside the instance are not.
Extend them from the operating system. On a typical Ubuntu image with an ext4 root file system:

```shell
lsblk                     # find the root disk and partition, for example vda and vda1
sudo growpart /dev/vda 1  # grow partition 1 to the end of the disk
sudo resize2fs /dev/vda1  # grow the ext4 file system
df -h /                   # check the new size
```

For an XFS file system, use `sudo xfs_growfs /` instead of `resize2fs`.

You can also move the root disk to another performance tier by changing `storage_policy_id`.

## Grow a data disk

Increase `size_gb` on the [`fptcloud_storage`](../resources/storage) resource:

```terraform
resource "fptcloud_storage" "app_data" {
  # ...
  size_gb = 200 # was 100
}
```

Then extend the file system inside the instance. For an ext4 file system created directly on the
disk (without a partition), for example on `/dev/vdb`:

```shell
sudo resize2fs /dev/vdb
```

Storage disks can only grow; FPT Cloud does not reduce the size of a disk.

~> **Important:** In a Specific VPC, FPT Cloud does not resize a disk while it is attached to an
instance. Detach it first, grow it, then attach it again, as described in
[Changing an attached disk in a Specific VPC](../resources/storage#changing-an-attached-disk-in-a-specific-vpc).

## Power instances on and off

The `status` argument controls the power state. A common use is to stop development instances
outside working hours. Drive it with a variable:

```terraform
variable "dev_running" {
  type    = bool
  default = true
}

resource "fptcloud_instance" "dev" {
  # ...
  status = var.dev_running ? "POWERED_ON" : "POWERED_OFF"
}
```

```shell
terraform apply -var="dev_running=false"   # in the evening
terraform apply -var="dev_running=true"    # in the morning
```

If people also start and stop the instance from the FPT Cloud console, and you do not want
Terraform to undo their changes, tell Terraform to ignore the power state:

```terraform
resource "fptcloud_instance" "dev" {
  # ...
  status = "POWERED_ON"

  lifecycle {
    ignore_changes = [status]
  }
}
```

## Scale out: add more instances

To run more copies of a server, use `count` (or `for_each`) and increase the number. Attach the
security group to all copies at once with a `for` expression on `apply_to`:

```terraform
variable "web_count" {
  type    = number
  default = 2
}

resource "fptcloud_instance" "web" {
  count = var.web_count

  vpc_id            = data.fptcloud_vpc.example.id
  name              = format("web-server-%02d", count.index + 1)
  image_name        = data.fptcloud_image.ubuntu.images[0].name
  flavor_name       = "2C4G"
  subnet_id         = fptcloud_subnet.app.id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
}

data "fptcloud_instance" "web" {
  count = var.web_count

  vpc_id = data.fptcloud_vpc.example.id
  id     = fptcloud_instance.web[count.index].id
}

resource "fptcloud_security_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-sg"
  subnet_id = fptcloud_subnet.app.id
  type      = "ACL"
  apply_to  = [for i in data.fptcloud_instance.web : i.private_ip]
}
```

Changing `web_count` from `2` to `3` adds `web-server-03` and adds its IP address to the security
group. Existing instances are not touched. Lowering the number deletes the instances with the
highest indexes, so check the plan carefully.

To spread the copies across physical hosts, place them in an
[instance group](../resources/instance_group) with the **Soft Anti Affinity** policy.

## Protect important instances

Add `prevent_destroy` to instances whose loss would hurt. Terraform then refuses any plan that
would delete them, including a replacement caused by a changed `image_name`, `subnet_id` or
`ssh_key`:

```terraform
resource "fptcloud_instance" "db" {
  # ...

  lifecycle {
    prevent_destroy = true
  }
}
```

To delete such an instance on purpose, remove the `lifecycle` block first.
