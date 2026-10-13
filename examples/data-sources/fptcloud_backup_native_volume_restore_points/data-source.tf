data "fptcloud_backup_native_volume_restore_points" "data" {
  vpc_id    = var.vpc_id
  volume_id = var.pv_volume_id
}
