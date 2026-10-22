# Deletes every restore point of the VM, under any backup job, and waits for
# the deletion to end. The VM's next backup is a full backup.
# Change purge (for example to a date) to delete the restore points again.
resource "fptcloud_backup_native_vm_restore_points_purge" "web" {
  vpc_id   = var.vpc_id
  vm_id    = var.web_instance_id
  triggers = { purge = var.restore_points_purge }
}
