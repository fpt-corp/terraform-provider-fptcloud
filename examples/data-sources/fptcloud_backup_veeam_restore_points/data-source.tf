data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

# The restore points of one protected instance - the same table the portal shows
# when you pick "Restore" on an instance. Points are ordered newest first, so
# points[0] is the most recent backup.
data "fptcloud_backup_veeam_restore_points" "db" {
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = "your_backup_job_id"
  vm_id         = "your_instance_id"
}

# Only full restore points, if an incremental one will not do.
data "fptcloud_backup_veeam_restore_points" "db_full" {
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = "your_backup_job_id"
  vm_id         = "your_instance_id"
  point_type    = "full"
}

output "latest_restore_point" {
  value = data.fptcloud_backup_veeam_restore_points.db.points[0]
}
