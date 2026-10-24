# Run the full job once. Change `triggers` to run it again.
resource "fptcloud_database_backup_run_now" "full" {
  cluster_id = fptcloud_database_backup.db.cluster_id
  job_id     = fptcloud_database_backup.db.full_job[0].job_id

  triggers = {
    run = "2026-10-23"
  }
}
