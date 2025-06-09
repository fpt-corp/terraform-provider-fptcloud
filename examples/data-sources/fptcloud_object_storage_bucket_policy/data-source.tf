data "fptcloud_object_storage_bucket_policy" "example_bucket_policy" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
  bucket_name = "your_bucket_name"
}

output "bucket_policy" {
  value = data.fptcloud_object_storage_bucket_policy.example_bucket_policy.policy
}
