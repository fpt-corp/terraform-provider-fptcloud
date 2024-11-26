---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fptcloud_object_storage_sub_user Data Source - terraform-provider-fptcloud"
subcategory: ""
description: |-
  
---

# fptcloud_object_storage_sub_user (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `region_name` (String) The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02
- `vpc_id` (String) The VPC ID

### Optional

- `page` (Number) Page number
- `page_size` (Number) Number of items per page

### Read-Only

- `id` (String) The ID of this resource.
- `list_sub_user` (List of Object) List of sub-users (see [below for nested schema](#nestedatt--list_sub_user))

<a id="nestedatt--list_sub_user"></a>
### Nested Schema for `list_sub_user`

Read-Only:

- `active` (Boolean)
- `arn` (String)
- `role` (String)
- `user_id` (String)