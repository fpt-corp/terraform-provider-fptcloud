data "fptcloud_managed_kubernetes_engine_v1" "example" {
  vpc_id       = "your_vpc_id"
  cluster_name = "my-cluster-ab12cd34" # the full name, as fptcloud_managed_kubernetes_engines_v1 reports it
}

output "k8s_version" {
  value = data.fptcloud_managed_kubernetes_engine_v1.example.k8s_version
}

# Names of the pools that have a GPU
output "gpu_pools" {
  value = [for pool in data.fptcloud_managed_kubernetes_engine_v1.example.pools : pool.name if pool.vgpu_id != ""]
}
