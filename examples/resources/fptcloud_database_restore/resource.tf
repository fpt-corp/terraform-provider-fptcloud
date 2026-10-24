# Restore a restore point into a new cluster.
data "fptcloud_database_restore_points" "src" {
  cluster_id = fptcloud_database.main.id
}

resource "fptcloud_database_restore" "copy" {
  cluster_id   = fptcloud_database.main.id
  restore_name = "main-restored"
  backup_part  = data.fptcloud_database_restore_points.src.restore_points[0].backup_part
  backup_type  = data.fptcloud_database_restore_points.src.restore_points[0].backup_type

  # Optional: without them the new cluster has the flavor and storage profile of the source.
  # flavor_id       = "..."        # or node_cpu and node_ram
  # storage_profile = "Premium-SSD-3000"

  # Keep the default (false) to leave the restored cluster in place when this resource is destroyed.
  delete_cluster_on_destroy = false
}
