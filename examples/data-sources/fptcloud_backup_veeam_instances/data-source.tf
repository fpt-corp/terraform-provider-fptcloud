data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

# Instances you can put into a backup job.
#
# unprotected_only defaults to true, so this returns only instances that do not
# belong to a job yet. An instance can belong to only one job, so building a job
# from an already protected instance fails with duplicateVm.
#
# The server also excludes instances mounted for instant recovery. It filters by
# power state only when `status` is given, so without it the result can include
# instances that are initialising, unresolved, or not yet deployed.
data "fptcloud_backup_veeam_instances" "available" {
  vpc_id = data.fptcloud_vpc.this.id
}

# When editing an existing job, pass its id so its own instances stay in the
# result instead of being filtered out as already protected.
data "fptcloud_backup_veeam_instances" "for_edit" {
  vpc_id = data.fptcloud_vpc.this.id
  job_id = "your_backup_job_id"
}

# Every instance in the VPC, protected or not.
data "fptcloud_backup_veeam_instances" "all" {
  vpc_id           = data.fptcloud_vpc.this.id
  unprotected_only = false
}

output "available_instances" {
  value = data.fptcloud_backup_veeam_instances.available.instances
}
