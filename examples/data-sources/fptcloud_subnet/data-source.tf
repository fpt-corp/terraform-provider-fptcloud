data "fptcloud_subnet" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "name"
    values = ["your_subnet_name"]
  }
}

output "show_value" {
  value = element(data.fptcloud_subnet.example.subnets,0)
}