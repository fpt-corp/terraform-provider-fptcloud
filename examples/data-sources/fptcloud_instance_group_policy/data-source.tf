data "fptcloud_instance_group_policy" "data" {
  vpc_id = "your_vpc_id"
  filter {
    key = "name"
    values = ["6e1c5151-39cd-4735-ae27-dfd77233630a"]
  }
}

output "show_value" {
  value = data.fptcloud_instance_group_policy.data
}