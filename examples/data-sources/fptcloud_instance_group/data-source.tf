data "fptcloud_instance_group" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "id"
    values = ["your_instance_group_id"]
  }
  filter {
    key = "name"
    values = ["your_instance_group_name"]
  }
}

output "show_value" {
  value = data.fptcloud_instance_group.example
}