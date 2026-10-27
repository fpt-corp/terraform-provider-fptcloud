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
  description = "IDs of notification methods for the jobs. Each job needs at least one."
  type        = list(string)
  default     = []
}

# The run, restore and purge variables below feed a resource's triggers: any
# non-empty value turns the action on and a new value repeats it. The time you
# ask for the action, for example "2026-10-26 09:00:00", is recommended so the
# state records when it was last requested. Avoid timestamp(): it changes on
# every plan and repeats the action on every apply. Leave a variable empty to
# skip its action.

variable "run_timeout" {
  description = "How long an apply waits for a manual backup or a purge. A short value ends the wait before the action does, to try the saved outcome: the apply warns, status is tf_timeout, and the next plan reads the real status. Set 60m to wait for the action."
  type        = string
  default     = "5s"
}

# VM backup (v1)

variable "vm_id" {
  description = "ID of a VM that is not in another backup job. Leave empty to skip the VM job, restore and purge."
  type        = string
  default     = ""
}

variable "vm_backup_run" {
  description = "Any value runs a manual backup of the VM job, and a new value runs it again; the request time is recommended. Leave empty to skip."
  type        = string
  default     = ""
}

variable "vm_restore" {
  description = "Any value restores the newest restore point of vm_id, and a new value restores again; the request time is recommended. Leave empty to skip."
  type        = string
  default     = ""
}

variable "vm_restore_mode" {
  description = "keep or overwrite. keep creates a new VM; overwrite replaces the disk of vm_id."
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

variable "vm_restore_points_purge" {
  description = "Any value deletes every restore point of vm_id, under any job, and a new value deletes them again; the request time is recommended. Leave empty to skip."
  type        = string
  default     = ""
}

resource "fptcloud_backup_native_vm_job" "example" {
  count = var.vm_id == "" ? 0 : 1

  vpc_id                  = data.fptcloud_vpc.this.id
  name                    = "tf-example-vm"
  description             = "Backup Native example, VM job 2 update"
  instance_ids            = [var.vm_id]
  notification_method_ids = var.notification_method_ids
  retention               = 1
  schedule_enabled        = true

  # daily takes no full_backup_day. weekly takes 0-6 (0 is Monday),
  # monthly takes 1-31.
  schedule {
    full_backup_frequency = "weekly"
    full_backup_day       = 5
    time                  = "11:17"
    time_zone             = "Asia/Ho_Chi_Minh"
  }
}

resource "fptcloud_backup_native_vm_job_run" "example" {
  count = var.vm_id != "" && var.vm_backup_run != "" ? 1 : 0

  vpc_id   = data.fptcloud_vpc.this.id
  job_id   = fptcloud_backup_native_vm_job.example[0].id
  triggers = { run = var.vm_backup_run }

  timeouts {
    create = var.run_timeout
  }
}

# Deletes every restore point of vm_id, under any job, for good. The VM's next
# backup is a full backup. restore_point_count = 0 means there was nothing to
# delete and no request was sent.
resource "fptcloud_backup_native_vm_restore_points_purge" "example" {
  count = var.vm_id != "" && var.vm_restore_points_purge != "" ? 1 : 0

  vpc_id   = data.fptcloud_vpc.this.id
  vm_id    = var.vm_id
  triggers = { purge = var.vm_restore_points_purge }

  timeouts {
    create = var.run_timeout
  }
}

# Volume backup (v2)

variable "volume_id" {
  description = "ID of an FKE persistent volume (name starts with pv-fke) that is not in another backup job. Leave empty to skip the volume job and purge."
  type        = string
  default     = ""
}

variable "volume_backup_run" {
  description = "Any value runs a manual backup of the volume job, and a new value runs it again; the request time is recommended. Leave empty to skip."
  type        = string
  default     = ""
}

variable "volume_restore_points_purge" {
  description = "Any value deletes every restore point of volume_id, under any job, and a new value deletes them again; the request time is recommended. Leave empty to skip."
  type        = string
  default     = ""
}

resource "fptcloud_backup_native_volume_job" "example" {
  count = var.volume_id == "" ? 0 : 1

  vpc_id                  = data.fptcloud_vpc.this.id
  name                    = "tf-example-volume"
  description             = "Backup Native example, volume job"
  volume_ids              = [var.volume_id]
  notification_method_ids = var.notification_method_ids
  retention               = 7
  schedule_enabled        = true

  # daily takes no full_backup_day. weekly takes 0-6 (0 is Monday),
  # monthly takes 1-31.
  schedule {
    full_backup_frequency = "daily"
    time                  = "01:30"
    time_zone             = "Asia/Ho_Chi_Minh"
  }
}

resource "fptcloud_backup_native_volume_job_run" "example" {
  count = var.volume_id != "" && var.volume_backup_run != "" ? 1 : 0

  vpc_id   = data.fptcloud_vpc.this.id
  job_id   = fptcloud_backup_native_volume_job.example[0].id
  triggers = { run = var.volume_backup_run }

  timeouts {
    create = var.run_timeout
  }
}

# Deletes every restore point of volume_id, under any job, for good. The
# volume's next backup is a full backup. restore_point_count = 0 means there
# was nothing to delete and no request was sent.
resource "fptcloud_backup_native_volume_restore_points_purge" "example" {
  count = var.volume_id != "" && var.volume_restore_points_purge != "" ? 1 : 0

  vpc_id    = data.fptcloud_vpc.this.id
  volume_id = var.volume_id
  triggers  = { purge = var.volume_restore_points_purge }

  timeouts {
    create = var.run_timeout
  }
}
