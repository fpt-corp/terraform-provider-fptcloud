data "fptcloud_image" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "name"
    values = ["debian-10"]
  }
}

data "fptcloud_image" "example_filter" {
  vpc_id = "your_vpc_id"
  filter {
    key = "catalog"
    values = ["Debian"]
  }
  filter {
    key = "is_gpu"
    values = [false]
  }
}

output "show_value" {
  value = element(data.fptcloud_image.example.images, 0)
}