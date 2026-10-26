terraform {
  required_providers {
    fptcloud = {
      source = "fpt-corp/fptcloud"
    }
  }
}

# Credentials come from FPTCLOUD_TOKEN, FPTCLOUD_TENANT_NAME and
# FPTCLOUD_REGION (and FPTCLOUD_API_URL for a non-default endpoint).
provider "fptcloud" {
}

data "fptcloud_vpc" "this" {
  name = var.vpc_name
}

variable "vpc_name" {
  description = "Name of the VPC as shown in the portal."
  type        = string

  validation {
    condition     = var.vpc_name != ""
    error_message = "Set vpc_name in terraform.tfvars."
  }
}

variable "notification_method_ids" {
  description = "IDs of notification methods for the jobs. The VM job needs at least one."
  type        = list(string)
  default     = []
}

# VM backup (v1)

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

variable "vm_restore_points_purge" {
  description = "Time you ask for the purge, for example 2026-10-26 09:00:00. Deletes every restore point of vm_id, under any job; a new time deletes them again. Leave empty to skip."
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
    full_backup_day       = 4
    time                  = "02:00"
    time_zone             = "Asia/Ho_Chi_Minh"
  }
}

# A manual backup runs the job once, outside its schedule, and waits for the
# run to end. Set run to the time you ask for the backup; a new time runs it
# again, and the state records when each run was requested. Write the time as
# a literal: timestamp() changes on every plan and runs a backup on every
# apply. Destroying the resource only removes it from state.
#
# Each run creates a restore point, and the restore below uses the newest one,
# so the apply after a backup runs the restore again.
resource "fptcloud_backup_native_vm_job_run" "example" {
  vpc_id   = data.fptcloud_vpc.this.id
  job_id   = fptcloud_backup_native_vm_job.example.id
  triggers = { run = "2026-10-26 09:00:00" }
}

data fptcloud_backup_native_vm_restore_points "example" {
  vpc_id = data.fptcloud_vpc.this.id
  vm_id  = var.vm_id
}

output "restore_points" {
  value = data.fptcloud_backup_native_vm_restore_points.example.restore_points
}


output "vpc_id" {
  value = data.fptcloud_vpc.this.id
}

resource "fptcloud_backup_native_vm_restore" "example2" {
  vpc_id                   = data.fptcloud_vpc.this.id
  vm_id                    = var.vm_id
  restore_point_created_at = data.fptcloud_backup_native_vm_restore_points.example.restore_points[0].created_at
  mode                     = "keep"
  power_on                 = true

  instance_name      = "helllo-restore-vm"
  flavor_id          = var.vm_restore_flavor_id
  subnet_id          = var.vm_restore_subnet_id
  security_group_ids = var.vm_restore_security_group_ids
  triggers           = { restore = "2026-10-26 09:00:01" }
}

# resource "fptcloud_backup_native_vm_restore" "restore_overwrite" {
#   vpc_id                   = data.fptcloud_vpc.this.id
#   vm_id                    = var.vm_id
#   restore_point_created_at = data.fptcloud_backup_native_vm_restore_points.example.restore_points[0].created_at
#   mode                     = "overwrite"
#   power_on                 = true
#   triggers                 = { restore = "2026-10-26 09:00:01" }
# }