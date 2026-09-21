data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

data "fptcloud_backup_veeam_jobs" "all" {
  vpc_id = data.fptcloud_vpc.this.id
}

output "job_names" {
  value = [for job in data.fptcloud_backup_veeam_jobs.all.jobs : job.name]
}

# status reflects the last synchronisation with Veeam, not a real-time value.
output "job_status" {
  value = {
    for job in data.fptcloud_backup_veeam_jobs.all.jobs : job.name => job.status
  }
}
