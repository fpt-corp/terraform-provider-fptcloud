data "fptcloud_database_proxy_types" "db" {
  cluster_id = fptcloud_database.db.id
}

output "proxy_types" {
  value = data.fptcloud_database_proxy_types.db.types
}
