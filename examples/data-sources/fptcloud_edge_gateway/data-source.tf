data "fptcloud_edge_gateway" "example" {
  vpc_id = "your_vpc_id"
  name = "your_edge_gateway_name"
}
output "show_value" {
  value = data.fptcloud_edge_gateway.example
}