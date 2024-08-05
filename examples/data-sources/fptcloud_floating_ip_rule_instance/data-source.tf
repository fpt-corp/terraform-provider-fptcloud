data "fptcloud_floating_ip_rule_instance" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "id"
    values = ["your_instance_id"]
  }
  filter {
    key = "ip_address"
    values = ["your_ip_address"]
  }
  filter {
    key = "name"
    values = ["your_instance_name"]
  }
}

output "show_value" {
  value = data.fptcloud_floating_ip_rule_instance.example
}