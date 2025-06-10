data "fptcloud_object_storage_bucket_lifecycle" "example_bucket_lifecycle" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
  bucket_name = "your_bucket_name"
  page        = 1
  page_size   = 100
}

output "bucket_lifecycle" {
  value = data.fptcloud_object_storage_bucket_lifecycle.example_bucket_lifecycle
}
