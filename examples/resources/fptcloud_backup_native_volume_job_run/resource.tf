# Runs fptcloud_backup_native_volume_job.data once and waits for the run to end.
# Change backup_run (for example to a date) to run the backup again.
resource "fptcloud_backup_native_volume_job_run" "data" {
  vpc_id   = var.vpc_id
  job_id   = fptcloud_backup_native_volume_job.data.id
  triggers = { run = var.backup_run }
}
