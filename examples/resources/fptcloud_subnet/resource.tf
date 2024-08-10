resource "fptcloud_subnet" "example" {
  vpc_id="your_vpc_id"
  name = "subnet_name"
  type = "ISOLATED | NAT_ROUTED"
  cidr = "your_cidr"
  gateway_ip = "your_gateway_ip"
  static_ip_pool = "static_ip_pool_of_instance"
}