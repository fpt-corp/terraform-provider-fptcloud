data "fptcloud_database_restored_clusters" "db" {
  cluster_id = "owiiqtii"
}

output "restored" {
  value = [for c in data.fptcloud_database_restored_clusters.db.clusters : "${c.cluster_name} (${c.status})" if c.status != "deleted"]
}
