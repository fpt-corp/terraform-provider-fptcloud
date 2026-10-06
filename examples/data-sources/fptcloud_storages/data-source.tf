data "fptcloud_storages" "example" {
  vpc_id = "your_vpc_id"
}

# Storages that are not attached to any instance
output "unattached_storage_ids" {
  value = [for storage in data.fptcloud_storages.example.storages : storage.id if storage.instance_id == ""]
}
