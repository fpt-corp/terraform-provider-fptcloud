data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

data "fptcloud_backup_veeam_restore_points" "db" {
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = "your_backup_job_id"
  vm_id         = "your_instance_id"
}

# Brings a restore point back as a NEW instance and leaves the original one
# running - "Restore keep" in the portal.
#
# This resource is an ACTION, not a piece of infrastructure. Applying it
# performs the restore; `terraform destroy` does NOT delete the instance it
# created, it only stops Terraform tracking the restore. Every argument forces a
# new resource, so changing the restore point runs the restore again.
resource "fptcloud_backup_veeam_restore_clone" "db_copy" {
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = "your_backup_job_id"
  vm_id         = "your_instance_id"

  # points[0] is the most recent restore point.
  restore_point_id = data.fptcloud_backup_veeam_restore_points.db.points[0].id

  # The name must not be in use by another instance in the VPC.
  new_instance_name = "db-recovered"

  power_on_after_restore = true
}

output "recovered_instance_id" {
  value = fptcloud_backup_veeam_restore_clone.db_copy.new_instance_id
}
