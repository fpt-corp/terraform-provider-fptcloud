resource "fptcloud_object_storage_bucket" "test_create_new_bucket" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
  name        = "test_bucket"
  acl         = "private"
}
