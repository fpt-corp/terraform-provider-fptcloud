resource "fptcloud_security_group" "example" {
  vpc_id    = "your_vpc_id"
  name      = "example"
  subnet_id = "your_subnet_id"
  type      = "ACL"
  apply_to = ["ip_or_cidr"]
}