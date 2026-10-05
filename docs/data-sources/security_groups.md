---
page_title: "fptcloud_security_groups Data Source - terraform-provider-fptcloud"
subcategory: ""
description: |-
  Use this data source to retrieve all security groups in a VPC.
  Results are retrieved page by page, up to page_size security groups per API call, so listing a large VPC never requires one request per security group. The attributes of each item match those of the fptcloud_security_group data source wherever the list API provides the same information.
  ~> Note: Security group rules and edge_gateway_id are not included in the list results. Use the fptcloud_security_group data source to read them for a specific security group.
---

# fptcloud_security_groups (Data Source)

Use this data source to retrieve all security groups in a VPC.

Results are retrieved page by page, up to `page_size` security groups per API call, so listing a large VPC never requires one request per security group. The attributes of each item match those of the `fptcloud_security_group` data source wherever the list API provides the same information.

~> **Note:** Security group rules and `edge_gateway_id` are not included in the list results. Use the `fptcloud_security_group` data source to read them for a specific security group.

## Example Usage

```terraform
data "fptcloud_security_groups" "example" {
  vpc_id = "your_vpc_id"
}

# Security group IDs indexed by name
output "security_group_ids" {
  value = { for sg in data.fptcloud_security_groups.example.security_groups : sg.name => sg.id }
}
```

## Schema

### Required

- `vpc_id` (String) The ID of the VPC whose security groups are retrieved.

### Optional

- `page_size` (Number) The number of security groups requested per API call. Must be between `1` and `100`; defaults to `100`. The provider requests successive pages until every security group has been retrieved.

### Read-Only

- `id` (String) The ID of this data source. It is set to the value of `vpc_id`.
- `security_groups` (List of Object) The security groups in the VPC. (see [below for nested schema](#nestedatt--security_groups))

<a id="nestedatt--security_groups"></a>
### Nested Schema for `security_groups`

Read-Only:

- `apply_to` (List of String) The IP addresses or CIDR blocks the security group is applied to.
- `created_at` (String) The time the security group was created.
- `id` (String) The ID of the security group.
- `members` (List of Object) The resources matched by the addresses in `apply_to`. (see [below for nested schema](#nestedatt--security_groups--members))
- `name` (String) The name of the security group.
- `status` (String) The current status of the security group, for example `REALIZED`.
- `tag_ids` (List of String) The IDs of the tags attached to the security group.
- `type` (String) The type of the security group: `ACL` (controls traffic to and from the internet) or `DFW` (controls traffic within the local network).
- `updated_at` (String) The time the security group was last updated.

<a id="nestedatt--security_groups--members"></a>
### Nested Schema for `security_groups.members`

Read-Only:

- `id` (String) The ID of the member. For a member of type `other`, this is the address itself.
- `ip_address` (String) The IP address of the member.
- `ipv6_address` (String) The IPv6 address of the member, when it has one.
- `name` (String) The name of the member. For a member of type `other`, this is the address itself.
- `status` (String) The status of the member resource. `Unknown` for a member of type `other`.
- `type` (String) The kind of member: `instance`, `load_balancer`, or `other` for an address that does not belong to a known resource.
