data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

# Read a job by id.
#
# Use this data source rather than fptcloud_backup_veeam_jobs when you need the
# job's configuration: the plural data source lists jobs with their status but
# returns no schedule, retention, instances or notification methods.
data "fptcloud_backup_veeam_job" "by_id" {
  vpc_id = data.fptcloud_vpc.this.id
  job_id = "your_backup_job_id"
}

# Or read it by name, matched exactly. Job names are unique within a VPC.
data "fptcloud_backup_veeam_job" "by_name" {
  vpc_id = data.fptcloud_vpc.this.id
  name   = "job-db-daily"
}

output "job" {
  value = data.fptcloud_backup_veeam_job.by_name
}

# Reuse an existing job's instance list, for example to build a second job that
# protects everything the first one does not.
output "protected_instance_ids" {
  value = data.fptcloud_backup_veeam_job.by_name.vm_ids
}
