# Get all edge gateways in a VPC
data "fptcloud_edge_gateways" "all" {
  vpc_id = "your_vpc_id"
}

# Filter edge gateways by name
data "fptcloud_edge_gateways" "filtered" {
  vpc_id = "your_vpc_id"
  name   = "my-edge-gateway"
}

# Output all edge gateways
output "all_edge_gateways" {
  value = data.fptcloud_edge_gateways.all.edge_gateways
}

# Output filtered edge gateways
output "filtered_edge_gateways" {
  value = data.fptcloud_edge_gateways.filtered.edge_gateways
}

# Access first edge gateway id
output "first_edge_gateway_id" {
  value = length(data.fptcloud_edge_gateways.all.edge_gateways) > 0 ? data.fptcloud_edge_gateways.all.edge_gateways[0].id : null
}

