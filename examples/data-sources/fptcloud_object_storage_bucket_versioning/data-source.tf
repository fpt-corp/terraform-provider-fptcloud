data "fptcloud_object_storage_bucket_versioning" "example_bucket_versioning" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
  bucket_name = "your_bucket_name"
}

output "bucket_versioning" {
  value = data.fptcloud_object_storage_bucket_versioning.example_bucket_versioning
}
