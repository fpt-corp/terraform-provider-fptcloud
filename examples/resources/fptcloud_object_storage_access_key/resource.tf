resource "fptcloud_object_storage_access_key" "create_key" {
  vpc_id      = "your_vpc_id"
  region_name = "HN-02"
}

output "access_key_output" {
  value = fptcloud_object_storage_access_key.create_key
}