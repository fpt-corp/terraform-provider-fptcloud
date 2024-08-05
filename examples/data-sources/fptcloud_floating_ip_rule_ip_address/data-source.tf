data "fptcloud_floating_ip_rule_ip_address" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "id"
    values = ["your_ip_address_id"]
  }
  filter {
    key = "name"
    values = ["your_ip_address_name"]
  }
}

output "show_value" {
  value = data.fptcloud_floating_ip_rule_ip_address.example
}