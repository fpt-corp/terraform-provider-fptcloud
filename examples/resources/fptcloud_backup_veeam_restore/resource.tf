data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

data "fptcloud_backup_veeam_restore_points" "db" {
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = "your_backup_job_id"
  vm_id         = "your_instance_id"
}

# Restores the instance from a restore point, overwriting whatever is running.
#
# This resource is an ACTION, not a piece of infrastructure. Applying it
# performs the restore; `terraform destroy` does NOT undo it, it only stops
# Terraform tracking that the restore happened. Every argument forces a new
# resource, so changing the restore point runs the restore again.
resource "fptcloud_backup_veeam_restore" "rollback" {
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = "your_backup_job_id"
  vm_id         = "your_instance_id"

  # points[0] is the most recent restore point.
  restore_point_id = data.fptcloud_backup_veeam_restore_points.db.points[0].id

  # Restore only the blocks that changed since the point was taken. Faster, and
  # what the portal calls "Quick rollback".
  quick_rollback = false

  power_on_after_restore = true
}

# To repeat a restore from the SAME restore point, change something in triggers.
# Without it nothing in the configuration differs, so Terraform would see no
# reason to act.
resource "fptcloud_backup_veeam_restore" "repeat" {
  vpc_id           = data.fptcloud_vpc.this.id
  backup_job_id    = "your_backup_job_id"
  vm_id            = "your_instance_id"
  restore_point_id = data.fptcloud_backup_veeam_restore_points.db.points[0].id

  triggers = {
    reason = "incident-4821"
  }
}

output "restored_from" {
  value = fptcloud_backup_veeam_restore.rollback.restored_at
}
