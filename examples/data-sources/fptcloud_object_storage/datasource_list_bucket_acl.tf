data "fptcloud_object_storage_bucket_acl" "example_bucket_acl" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
  bucket_name = "your_bucket_name"
}

output "bucket_acl" {
  value = data.fptcloud_object_storage_bucket_acl.example_bucket_acl
}
