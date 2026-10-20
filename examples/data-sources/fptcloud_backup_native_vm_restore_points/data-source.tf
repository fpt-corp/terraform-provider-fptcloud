data "fptcloud_backup_native_vm_restore_points" "web" {
  vpc_id = var.vpc_id
  vm_id  = var.web_instance_id
}

output "latest_restore_point" {
  value = data.fptcloud_backup_native_vm_restore_points.web.restore_points[0].created_at
}
