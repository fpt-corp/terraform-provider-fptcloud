# Runs fptcloud_backup_native_volume_job.data once and waits for the run to end.
# Any value works and a new value runs it again. A request time, as below, is
# recommended so the state records when each run was requested. Avoid
# timestamp(): it changes on every plan and runs a backup on every apply.
resource "fptcloud_backup_native_volume_job_run" "data" {
  vpc_id   = var.vpc_id
  job_id   = fptcloud_backup_native_volume_job.data.id
  triggers = { run = "2026-10-26 09:00:00" }
}
