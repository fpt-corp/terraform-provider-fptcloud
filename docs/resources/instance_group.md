---
page_title: "fptcloud_instance_group Resource - FPT Cloud"
subcategory: "Compute"
description: |-
  Manages an instance group, which controls how FPT Cloud places a set of instances on physical hosts.
---

# fptcloud_instance_group (Resource)

Manages an instance group in a VPC. An instance group applies a placement policy to the instances
in it:

* **Soft Anti Affinity** places the instances on different physical hosts when possible. Use it for
  availability: if one host fails, the other instances in the group keep running. If no other host
  has enough capacity, FPT Cloud still creates the instance, on a host the group already uses.
* **Soft Affinity** places the instances on the same physical host when possible. Use it when the
  instances talk to each other a lot and you want low network latency. If the host is full, FPT
  Cloud still creates the instance on another host.

Look up the policy with the [fptcloud_instance_group_policy](../data-sources/instance_group_policy)
data source. An instance joins a group in one of two ways: you set `instance_group_id` on the
[fptcloud_instance](instance) when you create it, or you list existing instances in `vm_ids` when
you create the group (Specific VPC only, see below).

To learn more, see [Instance Groups](https://docs.fptcloud.com/docs/fpt-cloud-server/instance-group/overview/)
in the FPT Cloud documentation.

~> **Important:** All arguments of this resource force a new group when they change, including
`vm_ids`. Destroying a group does not delete its instances, but they are no longer in a group. See
[Changing a group](#changing-a-group).

## Example Usage

### Spread new instances across hosts (General VPC)

Use this when you build a set of identical servers, such as web servers behind a load balancer, and
want a host failure to affect as few of them as possible. You create an empty group, then create
each instance with `instance_group_id`. FPT Cloud picks the host when the instance is created.

```terraform
data "fptcloud_vpc" "example" {
  name = "example-vpc"
}

data "fptcloud_instance_group_policy" "anti_affinity" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Soft Anti Affinity"]
  }
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

resource "fptcloud_instance_group" "web" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "web-group"
  policy_id = data.fptcloud_instance_group_policy.anti_affinity.instance_group_policies[0].id
}

resource "fptcloud_instance" "web" {
  count = 3

  vpc_id            = data.fptcloud_vpc.example.id
  name              = "web-server-0${count.index + 1}"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = data.fptcloud_subnet.web.subnets[0].id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
  instance_group_id = fptcloud_instance_group.web.id
}
```

### Group existing instances (Specific VPC)

In a Specific VPC, FPT Cloud creates a group only together with its first members: `vm_ids` must
list 2 to 10 existing instances that are not in another group. This example keeps an application
server and its cache on the same host with the **Soft Affinity** policy. It reuses the VPC, subnet
and storage policy lookups from the previous example.

```terraform
data "fptcloud_instance_group_policy" "affinity" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["Soft Affinity"]
  }
}

resource "fptcloud_instance" "app" {
  for_each = toset(["app-server-01", "app-cache-01"])

  vpc_id            = data.fptcloud_vpc.example.id
  name              = each.key
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = data.fptcloud_subnet.web.subnets[0].id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"

  lifecycle {
    # The group sets instance_group_id on these instances after they exist.
    # Ignore it so that Terraform does not try to clear it on every plan.
    ignore_changes = [instance_group_id]
  }
}

resource "fptcloud_instance_group" "app" {
  vpc_id    = data.fptcloud_vpc.example.id
  name      = "app-group"
  policy_id = data.fptcloud_instance_group_policy.affinity.instance_group_policies[0].id
  vm_ids    = [for instance in fptcloud_instance.app : instance.id]
}
```

### Add a new instance to an existing group

Use this when the group already exists, for example because another team or another Terraform
configuration manages it. Look the group up by name with the
[fptcloud_instance_group](../data-sources/instance_group) data source and pass its ID to the new
instance. This works in both VPC types.

```terraform
data "fptcloud_instance_group" "web" {
  vpc_id = data.fptcloud_vpc.example.id

  filter {
    key    = "name"
    values = ["web-group"]
  }
}

resource "fptcloud_instance" "web_extra" {
  vpc_id            = data.fptcloud_vpc.example.id
  name              = "web-server-04"
  image_name        = "Ubuntu-22.04" # Use an image name listed in your VPC
  flavor_name       = "2C4G"         # Use a flavor name listed in your VPC
  subnet_id         = data.fptcloud_subnet.web.subnets[0].id
  storage_size_gb   = 40
  storage_policy_id = data.fptcloud_storage_policy.ssd.storage_policies[0].id
  ssh_key           = file(pathexpand("~/.ssh/id_rsa.pub"))
  status            = "POWERED_ON"
  instance_group_id = data.fptcloud_instance_group.web.instance_groups[0].id
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC to create the group in. Use the `id` of the
  [fptcloud_vpc](../data-sources/vpc) data source. Changing this forces a new resource to be created.
* `name` - (Required) The name of the group. It must be unique in the VPC and at most 40 characters
  long. Use letters, digits, spaces, underscores (`_`), hyphens (`-`) and periods (`.`). Changing this
  forces a new resource to be created.
* `policy_id` - (Required) The ID of the placement policy: **Soft Affinity** or **Soft Anti
  Affinity**. Use the `id` of an element of `instance_group_policies` from the
  [fptcloud_instance_group_policy](../data-sources/instance_group_policy) data source. Policy IDs are
  different in a General VPC and a Specific VPC, so always look them up in the group's VPC. Changing
  this forces a new resource to be created.
* `vm_ids` - (Optional) The IDs of existing instances to put in the group when it is created. Use
  the `id` of [fptcloud_instance](instance) resources.
  * In a **Specific VPC**, this argument is required in practice: list 2 to 10 instances that are
    in the same VPC and not already in another group.
  * In a **General VPC**, this argument is ignored. Instances join the group only when you create
    them with `instance_group_id`.

  Terraform does not read the members back into `vm_ids`; use the `vms` attribute to see them.
  Changing this forces a new resource to be created.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the instance group. Pass it to `instance_group_id` on
  [fptcloud_instance](instance).
* `policy` - The placement policy, as a JSON-encoded object with the fields `id`, `name` and
  `is_active`. Read a field with `jsondecode()`, for example
  `jsondecode(fptcloud_instance_group.web.policy).name`.
* `vms` - The instances in the group, as a JSON-encoded list of objects with the fields `id` and
  `name`. It is updated on every refresh, so it also shows instances that joined the group through
  `instance_group_id`.
* `created_at` - The time the group was created.
* `policy_name` - Holds the group's `name`, not the policy name. Read the policy name from `policy`
  instead.

## Changing a group

FPT Cloud does not let you change a group's policy, and this resource does not update groups in
place. Any change to `vpc_id`, `name`, `policy_id` or `vm_ids` destroys the group and creates a new
one:

* The instances are not deleted. They are no longer in a group until the new group exists.
* In a Specific VPC, the instances listed in `vm_ids` join the new group.
* Instances that joined through `instance_group_id` do **not** join the new group. `instance_group_id`
  on [fptcloud_instance](instance) is only used when the instance is created; changing it later does
  not move the instance, and Terraform keeps showing the change in the plan. To put such an
  instance in the new group, the instance must be replaced, for example with
  `terraform apply -replace="fptcloud_instance.web[0]"`.

The same rule applies when you set or change `instance_group_id` on an instance that already exists:
it has no effect until the instance is re-created.

## Limitations

* FPT Cloud allows at most 10 instance groups per VPC.
* A group holds at most 10 instances. In a Specific VPC, a group also needs at least 2 instances
  when it is created.
* An instance belongs to at most one group.
* The policies are preferences ("soft"): when hosts are short of capacity, FPT Cloud still creates
  the instance even if it cannot honor the policy.
* Importing an existing instance group into Terraform is not supported. Look it up with the
  [fptcloud_instance_group](../data-sources/instance_group) data source instead.
