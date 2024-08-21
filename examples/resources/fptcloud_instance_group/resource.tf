resource "fptcloud_instance_group" "example" {
  vpc_id    = "your_vpc_id"
  name = "your_instance_group_name"
  policy_id = "your_policy_id"
  vm_ids = "your_instance_id"
}