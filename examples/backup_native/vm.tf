# VM backup (v1). Uncomment everything below this header to try it. It uses
# the provider, the VPC and the shared variables (vpc_name,
# notification_method_ids) declared in volume.tf, and the vm_* values in
# terraform.tfvars.
#
# Manual backup: runs the job once, outside its schedule, and waits for the
# run to end. Set vm_backup_run to any value to run one; change the value to
# run it again. Destroying the resource only removes it from state.
#
# Each run creates a restore point, and the restore uses the newest one: turn
# the restore off before running a backup, or the next apply runs the restore
# again.
#
# Restore points exist only after the job has run at least once. Turn the
# restore on after the restore points data source lists one.
#
# Restore: runs once, on create. Changing an argument runs a new restore;
# destroying the resource only removes it from state.

variable "vm_id" {
  description = "ID of a VM that is not in another backup job. Leave empty to skip the VM job and restore."
  type        = string
  default     = ""
}

variable "vm_backup_run" {
  description = "Any value runs the VM job once; a new value runs it again. Leave empty to skip the manual backup."
  type        = string
  default     = ""
}

variable "vm_restore" {
  description = "Restore the newest restore point of vm_id. Creates a new VM (keep) or overwrites vm_id (overwrite)."
  type        = bool
  default     = false
}

variable "vm_restore_mode" {
  description = "keep or overwrite. overwrite replaces the disk of vm_id."
  type        = string
  default     = "keep"
}

variable "vm_restore_flavor_id" {
  description = "keep only. Flavor of the new VM."
  type        = string
  default     = null
}

variable "vm_restore_subnet_id" {
  description = "keep only. Subnet of the new VM (the id of fptcloud_subnet)."
  type        = string
  default     = null
}

variable "vm_restore_security_group_ids" {
  description = "keep only. Security groups of the new VM."
  type        = list(string)
  default     = []
}

resource "fptcloud_backup_native_vm_job" "example" {
  count = var.vm_id == "" ? 0 : 1

  vpc_id                  = data.fptcloud_vpc.this.id
  name                    = "tf-example-vm"
  description             = "Backup Native example, VM job"
  instance_ids            = [var.vm_id]
  notification_method_ids = var.notification_method_ids
  retention               = 7

  # daily takes no full_backup_day. weekly takes 0-6 (0 is Monday),
  # monthly takes 1-31.
  schedule {
    full_backup_frequency = "weekly"
    full_backup_day       = 5
    time                  = "02:00"
    time_zone             = "Asia/Ho_Chi_Minh"
  }
}

resource "fptcloud_backup_native_vm_backup" "example" {
  count = var.vm_id != "" && var.vm_backup_run != "" ? 1 : 0

  vpc_id   = data.fptcloud_vpc.this.id
  job_id   = fptcloud_backup_native_vm_job.example[0].id
  triggers = { run = var.vm_backup_run }
}

data "fptcloud_backup_native_vm_restore_points" "example" {
  count = var.vm_id == "" ? 0 : 1

  vpc_id = data.fptcloud_vpc.this.id
  vm_id  = var.vm_id
}

resource "fptcloud_backup_native_vm_restore" "example" {
  count = var.vm_id != "" && var.vm_restore ? 1 : 0

  vpc_id                   = data.fptcloud_vpc.this.id
  vm_id                    = var.vm_id
  restore_point_created_at = data.fptcloud_backup_native_vm_restore_points.example[0].restore_points[0].created_at
  mode                     = var.vm_restore_mode

  instance_name      = var.vm_restore_mode == "keep" ? "tf-example-restored" : null
  flavor_id          = var.vm_restore_mode == "keep" ? var.vm_restore_flavor_id : null
  subnet_id          = var.vm_restore_mode == "keep" ? var.vm_restore_subnet_id : null
  security_group_ids = var.vm_restore_mode == "keep" ? var.vm_restore_security_group_ids : null
}

data "fptcloud_backup_native_vm_jobs" "all" {
  vpc_id     = data.fptcloud_vpc.this.id
  depends_on = [fptcloud_backup_native_vm_job.example]
}

output "vm_jobs" {
  value = data.fptcloud_backup_native_vm_jobs.all.jobs
}

output "vm_restore_points" {
  value = try(data.fptcloud_backup_native_vm_restore_points.example[0].restore_points, [])
}

output "vm_backup" {
  value = try({
    status      = fptcloud_backup_native_vm_backup.example[0].status
    started_at  = fptcloud_backup_native_vm_backup.example[0].started_at
    finished_at = fptcloud_backup_native_vm_backup.example[0].finished_at
  }, null)
}

output "vm_restore_status" {
  value = try(fptcloud_backup_native_vm_restore.example[0].status, null)
}
