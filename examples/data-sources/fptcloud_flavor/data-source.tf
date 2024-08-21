data "fptcloud_flavor" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "name"
    values = ["Extra-8"]
  }
}

data "fptcloud_flavor" "example_filter" {
  vpc_id = "your_vpc_id"
  filter {
    key = "cpu"
    values = [2]
  }
  filter {
    key = "memory_mb"
    values = [4 * 1024]
  }
  filter {
    key = "type"
    values = ["VM_SIZE"]
  }
}

output "show_value" {
  value = element(data.fptcloud_flavor.example.flavors, 0)
}