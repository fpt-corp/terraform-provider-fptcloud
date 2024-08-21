resource "fptcloud_instance" "example" {
  name              = "example"
  vpc_id            = "your_vpc_id"
  ssh_key           = "your_public_key"
  image_id          = "your_image_id"
  flavor_id         = "your_flavor_id"
  public_ip         = "your_ip_public"
  subnet_id         = "f25b15f5-9098-429e-887d-1c9562b648ae"
  storage_size_gb   = 50
  storage_policy_id = "your_storage_policy_id"
  security_group_ids = []
  status            = "POWERED_ON"
}