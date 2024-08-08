data "fptcloud_instance_group_policy" "data" {
  vpc_id = "your_vpc_id"
  filter {
    key = "id"
    values = ["your_id"]
  }
  filter {
    key = "name"
    values = ["Soft Affinity", "Soft Anti Affinity"]
  }
}

output "show_value" {
  value = data.fptcloud_instance_group_policy.data
}