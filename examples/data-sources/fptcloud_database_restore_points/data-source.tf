data "fptcloud_database_restore_points" "db" {
  cluster_id = "owiiqtii"
}

output "restore_points" {
  value = [for p in data.fptcloud_database_restore_points.db.restore_points : p.backup_part if p.backup_state == "successful"]
}
