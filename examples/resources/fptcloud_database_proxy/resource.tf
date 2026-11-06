# The proxy types the cluster takes: `maxscale` for MariaDB, `mysqlrouter` for MySQL.
data "fptcloud_database_proxy_types" "db" {
  cluster_id = fptcloud_database.db.id
}

# A proxy with two nodes. Changing node_cpu and node_ram later resizes it in place.
resource "fptcloud_database_proxy" "db" {
  db_cluster_id   = fptcloud_database.db.id
  vpc_id          = var.vpc_id
  name            = "db-proxy-01"
  lb_type         = data.fptcloud_database_proxy_types.db.types[0]
  storage_profile = "Premium-SSD"

  node_cpu          = 2
  node_ram          = 4
  high_availability = true

  timeouts {
    create = "45m"
    update = "45m"
    delete = "30m"
  }
}

output "proxy_end_point" {
  value = fptcloud_database_proxy.db.end_point
}

# Import an existing proxy by its Id (`lb_cluster_id`):
#   terraform import fptcloud_database_proxy.db <lb_cluster_id>
