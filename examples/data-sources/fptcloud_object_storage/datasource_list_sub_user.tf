data "fptcloud_object_storage_sub_user" "hoanglm32" {
  vpc_id      = "your_vpc_id"
  page        = 1
  page_size   = 100000
  region_name = "your_region_name"
}
// for raw data and all sub users will be listed
output "list_sub_user" {
  value = data.fptcloud_object_storage_sub_user.hoanglm32.list_sub_user
}
