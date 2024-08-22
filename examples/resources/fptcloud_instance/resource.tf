resource "fptcloud_instance" "example" {
  name              = "example"
  vpc_id            = "your_vpc_id"
  ssh_key           = "your_public_key"
  image_name          = "CentOS-7"
  flavor_name         = "1C1G"
  public_ip         = "your_ip_public"
  subnet_id         = "your_subnet_id"
  storage_size_gb   = 50
  storage_policy_id = "your_storage_policy_id"
  security_group_ids = []
  status            = "POWERED_ON"
}