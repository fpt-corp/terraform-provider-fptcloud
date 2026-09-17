data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

data "fptcloud_backup_veeam_restore_points" "db" {
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = "your_backup_job_id"
  vm_id         = "your_instance_id"
}

# Asks the platform to mount a backup as a NEW instance that runs straight from
# it, leaving the original instance alone - useful for checking a backup, or for
# getting a service back up while a full restore runs. Apply ends once the
# request has been accepted.
#
# Terraform does not manage the session afterwards: keeping the mounted
# instance (migrate) or stopping the session (unmount) is done in the portal's
# Instant Recovery tab, and `terraform destroy` here leaves the session running.
#
# While a session is open, the backup job of the instance it was mounted from
# cannot run.
resource "fptcloud_backup_veeam_instant_recovery" "db_sandbox" {
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = "your_backup_job_id"
  vm_id         = "your_instance_id"

  restore_point_id = data.fptcloud_backup_veeam_restore_points.db.points[0].id

  # The name must not be in use by another instance in the VPC.
  new_instance_name = "db-sandbox"

  power_up = true

  # Off by default: a mounted instance with its network connected can collide
  # with the original one, which is still running.
  nics_enabled = false
}

# Empty right after the first apply, filled in by a later refresh once the
# platform has finished mounting.
output "session_state" {
  value = fptcloud_backup_veeam_instant_recovery.db_sandbox.state
}

# The sessions data source lists every open session in the VPC, including ones
# started in the portal.
data "fptcloud_backup_veeam_instant_recovery_sessions" "open" {
  vpc_id = data.fptcloud_vpc.this.id
}
