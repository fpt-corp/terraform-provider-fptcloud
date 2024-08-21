data "fptcloud_vpc" "example" {
  name = "vpc-name"
}

output "name" {
  value = data.fptcloud_vpc.example
}