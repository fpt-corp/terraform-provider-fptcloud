data "fptcloud_object_storage_access_key" "keys" {
  vpc_id      = "your_vpc_id"
  region_name = "your_region_name"
}
// for raw data and all access keys from region_name will be listed
output "access_key" {
  value = data.fptcloud_object_storage_access_key.keys
}
