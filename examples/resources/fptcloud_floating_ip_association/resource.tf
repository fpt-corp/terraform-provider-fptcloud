resource "fptcloud_floating_ip" "example" {
  vpc_id = "your_vpc_id"
}

resource "fptcloud_floating_ip_association" "association_example" {
  vpc_id         = "your_vpc_id"
  floating_ip_id = fptcloud_floating_ip.example.id
  instance_id    = "your_instance_id"
  depends_on = [fptcloud_floating_ip.example]
}