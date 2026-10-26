data "fptcloud_backup_native_volume_restore_points" "data" {
  vpc_id    = var.vpc_id
  volume_id = var.pv_volume_id
}

# Set restore to the time you ask for the restore; a new time runs it again,
# and the state records when each restore was requested.
resource "fptcloud_backup_native_volume_restore" "data_copy" {
  vpc_id           = var.vpc_id
  volume_id        = var.pv_volume_id
  restore_point_id = data.fptcloud_backup_native_volume_restore_points.data.restore_points[0].id
  name             = "pv-data-restored"
  size_gb          = 20
  triggers         = { restore = "2026-10-26 09:00:00" }
}
