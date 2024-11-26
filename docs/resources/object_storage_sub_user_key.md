---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fptcloud_object_storage_sub_user_key Resource - terraform-provider-fptcloud"
subcategory: ""
description: |-
  
---

# fptcloud_object_storage_sub_user_key (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `region_name` (String) The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02
- `user_id` (String) The sub user id, can retrieve from data source `fptcloud_object_storage_sub_user`
- `vpc_id` (String) The VPC id that the S3 service belongs to

### Read-Only

- `access_key` (String) The access key of the sub user
- `id` (String) The ID of this resource.
- `secret_key` (String) The secret key of the sub user