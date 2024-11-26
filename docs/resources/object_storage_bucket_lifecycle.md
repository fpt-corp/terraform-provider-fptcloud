---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fptcloud_object_storage_bucket_lifecycle Resource - terraform-provider-fptcloud"
subcategory: ""
description: |-
  
---

# fptcloud_object_storage_bucket_lifecycle (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `bucket_name` (String) Name of the bucket
- `region_name` (String) The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02
- `vpc_id` (String) The VPC ID

### Optional

- `life_cycle_rule` (String) The bucket lifecycle rule in JSON format, support only one rule
- `life_cycle_rule_file` (String) Path to the JSON file containing the bucket lifecycle rule, support only one rule

### Read-Only

- `id` (String) The ID of this resource.
- `rules` (List of Object) (see [below for nested schema](#nestedatt--rules))
- `state` (Boolean) State after bucket lifecycle rule is created

<a id="nestedatt--rules"></a>
### Nested Schema for `rules`

Read-Only:

- `id` (String)