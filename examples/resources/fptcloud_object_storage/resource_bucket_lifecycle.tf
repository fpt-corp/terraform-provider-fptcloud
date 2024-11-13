resource "fptcloud_object_storage_bucket_lifecycle" "example_bucket_lifecycle" {
  bucket_name = "your_bucket_name"
  region_name = "your_region_name"
  vpc_id      = "your_vpc_id"

  # Option 1: Load policy from file
  life_cycle_rule_file = file("${path.module}/your_bucket_lifecycle.json")
}

output "bucket_lifecycle" {
  value = fptcloud_object_storage_bucket_lifecycle.example_bucket_lifecycle
}
