---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fptcloud_object_storage_bucket_cors Resource - terraform-provider-fptcloud"
subcategory: ""
description: |-
  
---

# fptcloud_object_storage_bucket_cors (Resource)



## Example Usage

```terraform
resource "fptcloud_object_storage_bucket_cors" "example_bucket_cors" {
  vpc_id      = "1b413c55-b752-4183-abad-06c4b5aca6ad"
  region_name = "HCM-02"
  bucket_name = "a-hoanglm32-test"

  # Option 1: Load cors config from file
  cors_config_file = file("${path.module}/your_bucket_cors_config.json")

  # Option 2: Inline cors_config
  # cors_config = jsonencode({
  #  {
  #     "ID": "a9099",
  #     "AllowedOrigins": ["http://www.example.com", "http://www.example2.com"],
  #     "AllowedMethods": ["GET", "PUT", "DELETE"],
  #     "MaxAgeSeconds": 3000,
  #     "ExposeHeaders": ["Etag", "x-amz"],
  #     "AllowedHeaders": ["*", "demo"]
  #   }
  # })
}
output "bucket_cors" {
  value = fptcloud_object_storage_bucket_cors.example_bucket_cors.status
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `bucket_name` (String) Name of the bucket
- `region_name` (String) The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02
- `vpc_id` (String) The VPC ID

### Optional

- `cors_config` (String) The bucket lifecycle rule in JSON format, support only one rule
- `cors_config_file` (String) Path to the JSON file containing the bucket lifecycle rule, support only one rule

### Read-Only

- `bucket_cors_rules` (List of Object) (see [below for nested schema](#nestedatt--bucket_cors_rules))
- `id` (String) The ID of this resource.
- `status` (Boolean) Status after bucket cors rule is created

<a id="nestedatt--bucket_cors_rules"></a>
### Nested Schema for `bucket_cors_rules`

Read-Only:

- `id` (String)
