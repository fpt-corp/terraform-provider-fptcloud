data "fptcloud_alert_notification_methods" "email" {
  vpc_id = var.vpc_id
  type   = "EMAIL"
}

resource "fptcloud_backup_native_vm_job" "web" {
  vpc_id                  = var.vpc_id
  name                    = "web-nightly"
  description             = "Nightly backup of the web tier"
  instance_ids            = [var.web_instance_id]
  notification_method_ids = [for m in data.fptcloud_alert_notification_methods.email.methods : m.id]
  retention               = 7

  schedule {
    full_backup_frequency = "weekly"
    full_backup_day       = 5 # Saturday
    time                  = "02:00"
    time_zone             = "Asia/Ho_Chi_Minh"
  }
}
