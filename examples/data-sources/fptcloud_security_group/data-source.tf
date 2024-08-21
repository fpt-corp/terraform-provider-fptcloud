data "fptcloud_security_group" "example" {
    name = "your_security_group_name"
    vpc_id = "your_vpc_id"
}

output "output-example" {
  value = data.fptcloud_security_group.example
}