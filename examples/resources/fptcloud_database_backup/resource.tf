# Backup service of a database cluster, with a full and a diff job.
# The cluster needs an alert with alert_service "backup" and at least one recipient (Alert tab of the portal).
resource "fptcloud_database_backup" "db" {
  cluster_id  = fptcloud_database.main.id
  vpc_id      = var.vpc_id
  capacity_gb = 200 # at least 10, can only be increased

  full_job {
    name         = "daily-full" # 6-25 characters, cannot be changed after the job is created
    retention    = 7
    hours        = [1]
    minute       = 30
    days_of_week = [1, 2, 3, 4, 5, 6, 7] # 1 = Monday ... 7 = Sunday
  }

  # Optional, needs full_job. Its retention is the one of the full job.
  # Not available for every engine (for example not for MongoDB, Redis, Kafka, ClickHouse, OpenSearch, RabbitMQ).
  diff_job {
    name         = "hourly-diff"
    hours        = [5, 9, 13] # not an hour used by full_job
    minute       = 0
    days_of_week = [1, 2, 3, 4, 5]
  }

  timeouts {
    create = "45m"
  }
}
