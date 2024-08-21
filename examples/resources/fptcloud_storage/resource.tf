resource "fptcloud_storage" "example" {
  vpc_id="your_vpc_id"
  name = "storage_name"
  type = "EXTERNAL | LOCAL" # Type Local allow only VPC specific
  size_gb = 1
  storage_policy_id = "your_storage_policy_id" # get from storage policy datasource
  instance_id = "your_instance_id_to_attach"
  depends_on = [fptcloud_storage.example_before] # if you creation multi storage at once you need fill resource name to depends_on
}