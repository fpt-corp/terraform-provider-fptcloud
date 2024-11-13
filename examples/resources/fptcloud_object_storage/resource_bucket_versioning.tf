resource "fptcloud_object_storage_bucket_versioning" "versioning" {
  vpc_id            = "your_vpc_id"
  region_name       = "your_bucket_region"
  bucket_name       = "your_bucket_name"
  versioning_status = "Suspended" // or "Enabled"
}

