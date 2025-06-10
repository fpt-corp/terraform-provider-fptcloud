resource "fptcloud_object_storage_sub_user" "example" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
  user_id     = "your_user_id"
  role        = "your_role"
}
