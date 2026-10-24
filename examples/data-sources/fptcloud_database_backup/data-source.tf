data "fptcloud_database_backup" "db" {
  cluster_id = "owiiqtii"
}

output "backup_status" {
  value = data.fptcloud_database_backup.db.status
}
