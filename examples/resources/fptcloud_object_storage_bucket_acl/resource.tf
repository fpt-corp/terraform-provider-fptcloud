resource "fptcloud_object_storage_bucket_acl" "bucket_acl" {
  vpc_id      = "your_vpc_id"
  region_name = "your_bucket_region"
  bucket_name = "your_bucket_name"
  canned_acl  = "private"
}
