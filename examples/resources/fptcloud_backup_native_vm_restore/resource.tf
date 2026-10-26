data "fptcloud_backup_native_vm_restore_points" "web" {
  vpc_id = var.vpc_id
  vm_id  = var.web_instance_id
}

# Restore the latest restore point into a new VM and keep the original.
# Set restore to the time you ask for the restore; a new time runs it again,
# and the state records when each restore was requested.
resource "fptcloud_backup_native_vm_restore" "web_copy" {
  vpc_id                   = var.vpc_id
  vm_id                    = var.web_instance_id
  restore_point_created_at = data.fptcloud_backup_native_vm_restore_points.web.restore_points[0].created_at
  mode                     = "keep"

  instance_name      = "web-restored"
  flavor_id          = var.flavor_id
  subnet_id          = var.subnet_id
  security_group_ids = [var.security_group_id]

  triggers = { restore = "2026-10-26 09:00:00" }
}

# Find the new VM by name.
data "fptcloud_instances" "all" {
  vpc_id     = var.vpc_id
  depends_on = [fptcloud_backup_native_vm_restore.web_copy]
}
