data "fptcloud_instance" "example" {
  vpc_id    = "your_vpc_id"
  name      = "your_instance_name"
}

output "output-example" {
  value = data.fptcloud_instance.example
}