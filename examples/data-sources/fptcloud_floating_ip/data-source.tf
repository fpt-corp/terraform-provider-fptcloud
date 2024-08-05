data "fptcloud_floating_ip" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "ip"
    values = ["your_id"]
  }
  filter {
    key = "ip_address"
    values = ["your_ip_address"]
  }
}

output "show_value" {
  value = data.fptcloud_floating_ip.example
}