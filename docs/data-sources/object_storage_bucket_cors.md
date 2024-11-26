---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fptcloud_object_storage_bucket_cors Data Source - terraform-provider-fptcloud"
subcategory: ""
description: |-
  
---

# fptcloud_object_storage_bucket_cors (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `bucket_name` (String) Name of the bucket
- `region_name` (String) The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02
- `vpc_id` (String) The VPC ID

### Optional

- `page` (Number) The page number
- `page_size` (Number) The number of items to return in each page

### Read-Only

- `cors_rule` (List of Object) The bucket cors rule (see [below for nested schema](#nestedatt--cors_rule))
- `id` (String) The ID of this resource.

<a id="nestedatt--cors_rule"></a>
### Nested Schema for `cors_rule`

Read-Only:

- `allowed_headers` (List of String)
- `allowed_methods` (List of String)
- `allowed_origins` (List of String)
- `expose_headers` (List of String)
- `id` (String)
- `max_age_seconds` (Number)