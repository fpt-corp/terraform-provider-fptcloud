resource "fptcloud_object_storage_bucket_lifecycle" "example_bucket_lifecycle" {
  bucket_name = "your_bucket_name"
  region_name = "your_region_name"
  vpc_id      = "your_vpc_id"

  # Option 1: Load policy from file
  life_cycle_rule_file = file("${path.module}/your_bucket_lifecycle.json")

  # Option 2: Inline policy
  # life_cycle_rule = jsonencode({
  #     "ID": "FCI",
  #     "Filter": {
  #         "Prefix": ""
  #     },
  #     "Expiration": {
  #         "Days": 98
  #     },
  #     "NoncurrentVersionExpiration": {
  #         "NoncurrentDays": 83
  #     },
  #     "AbortIncompleteMultipartUpload": {
  #         "DaysAfterInitiation": 68
  #     }
  # })
}

output "bucket_lifecycle" {
  value = fptcloud_object_storage_bucket_lifecycle.example_bucket_lifecycle
}
