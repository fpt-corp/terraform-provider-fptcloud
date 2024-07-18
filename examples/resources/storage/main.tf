resource "fptcloud_storage" "example" {
  vpc_id="your_vpc_id"
  name = "storage_name"
  type = "EXTERNAL | LOCAL"
  size_gb = 1
  storage_policy_id = "your_storage_policy_id" # get from storage policy datasource
  instance_id = "your_instance_id_to_attach"
}