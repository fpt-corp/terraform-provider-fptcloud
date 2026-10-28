# Deletes every restore point of the VM, under any backup job, and waits for
# the deletion to end. The VM's next backup is a full backup.
# Any value works and a new value deletes the restore points again. A request
# time, as below, is recommended so the state records when each purge was
# requested. Avoid timestamp(): it changes on every plan and deletes the
# restore points on every apply.
resource "fptcloud_backup_native_vm_restore_points_purge" "web" {
  vpc_id   = var.vpc_id
  vm_id    = var.web_instance_id
  triggers = { purge = "2026-10-26 09:00:00" }
}
