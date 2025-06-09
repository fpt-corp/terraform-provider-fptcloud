data "fptcloud_object_storage_bucket" "hoanglm32" {
  vpc_id      = "your_vpc_id"
  page        = 1
  page_size   = 100000
  region_name = "your_region_name"
}
// for raw data and all buckets will be listed
output "name" {
  value = data.fptcloud_object_storage_bucket.hoanglm32.list_bucket_result
}
