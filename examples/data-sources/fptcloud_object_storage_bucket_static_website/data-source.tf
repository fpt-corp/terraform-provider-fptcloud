data "fptcloud_object_storage_bucket_static_website" "example_bucket_static_website" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
  bucket_name = "your_bucket_name"
}

output "bucket_static_website" {
  value = data.fptcloud_object_storage_bucket_static_website.example_bucket_static_website
}
