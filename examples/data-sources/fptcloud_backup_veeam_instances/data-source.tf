data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

# Instances that do not belong to any active backup job yet.
#
# The server also excludes instances that are initialising, unresolved, not
# deployed, or mounted for instant recovery, so this list is shorter than the
# full instance list of the VPC.
data "fptcloud_backup_veeam_instances" "available" {
  vpc_id     = data.fptcloud_vpc.this.id
  not_backup = true
}

# When editing an existing job, pass its id so its own instances stay in the
# result instead of being filtered out as "already in a job".
data "fptcloud_backup_veeam_instances" "for_edit" {
  vpc_id     = data.fptcloud_vpc.this.id
  not_backup = true
  job_id     = "your_backup_job_id"
}

output "available_instance_names" {
  value = [for vm in data.fptcloud_backup_veeam_instances.available.instances : vm.name]
}
