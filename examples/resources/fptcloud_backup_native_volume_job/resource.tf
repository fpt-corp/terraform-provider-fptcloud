resource "fptcloud_backup_native_volume_job" "data" {
  vpc_id                  = var.vpc_id
  name                    = "pv-data-daily"
  description             = "Daily backup of the FKE data volume"
  volume_ids              = [var.pv_volume_id] # an FKE persistent volume, name pv-fke-*
  notification_method_ids = [var.notification_method_id]
  retention               = 14

  schedule {
    full_backup_frequency = "daily"
    time                  = "01:30"
    time_zone             = "Asia/Ho_Chi_Minh"
  }
}
