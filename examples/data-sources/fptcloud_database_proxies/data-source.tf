data "fptcloud_database_proxies" "db" {
  cluster_id = fptcloud_database.db.id
}

output "proxy_names" {
  value = [for p in data.fptcloud_database_proxies.db.proxies : p.name]
}
