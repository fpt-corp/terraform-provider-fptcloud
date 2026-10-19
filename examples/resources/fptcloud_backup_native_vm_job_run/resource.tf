# Runs fptcloud_backup_native_vm_job.web once and waits for the run to end.
# Change backup_run (for example to a date) to run the backup again.
resource "fptcloud_backup_native_vm_job_run" "web" {
  vpc_id   = var.vpc_id
  job_id   = fptcloud_backup_native_vm_job.web.id
  triggers = { run = var.backup_run }
}
