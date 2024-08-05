resource "fptcloud_floating_ip" "example" {
  vpc_id = "934a79d8-8de9-40a2-a5e6-cca500132f15"
  floating_ip_id = "new"
  instance_id = "95ef6f8f-1f73-4b81-8ad5-ad4953d70e28"
}