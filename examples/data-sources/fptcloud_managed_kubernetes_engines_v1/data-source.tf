data "fptcloud_managed_kubernetes_engines_v1" "example" {
  vpc_id = "your_vpc_id"
}

# Names of all clusters in the VPC
output "cluster_names" {
  value = [for cluster in data.fptcloud_managed_kubernetes_engines_v1.example.clusters : cluster.id]
}
