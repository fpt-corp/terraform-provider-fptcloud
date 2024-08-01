data "fptcloud_storage" "example" {
    name = "your_storage_name"
    vpc_id = "your_vpc_id"
}

output "output-example" {
  value = data.fptcloud_storage.example
}