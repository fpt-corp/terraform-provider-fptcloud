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

# Volume backup (v2)

variable "volume_id" {
  description = "ID of an FKE persistent volume (name starts with pv-fke) that is not in another backup job. Leave empty to skip the volume job and restore."
  type        = string
  default     = ""
}

variable "volume_backup_run" {
  description = "Any value runs the volume job once; a new value runs it again. Leave empty to skip the manual backup."
  type        = string
  default     = ""
}

variable "volume_restore" {
  description = "Restore the newest restore point of volume_id into a new volume."
  type        = bool
  default     = false
}

variable "volume_restore_size_gb" {
  description = "Size of the restored volume in GB, at least the size of volume_id."
  type        = number
  default     = 20
}

resource "fptcloud_backup_native_volume_job" "example" {
  count = var.volume_id == "" ? 0 : 1

  vpc_id                  = data.fptcloud_vpc.this.id
  name                    = "tf-example-volume"
  description             = "Backup Native example, volume job"
  volume_ids              = [var.volume_id]
  notification_method_ids = var.notification_method_ids
  retention               = 7

  # daily takes no full_backup_day. weekly takes 0-6 (0 is Monday),
  # monthly takes 1-31.
  schedule {
    full_backup_frequency = "daily"
    time                  = "01:30"
    time_zone             = "Asia/Ho_Chi_Minh"
  }
}

# A manual backup runs the job once, outside its schedule, and waits for the
# run to end. Set volume_backup_run to any value to run one; change the value
# to run it again. Destroying the resource only removes it from state.
#
# Each run creates a restore point, and the restore below uses the newest one:
# turn the restore off before running a backup, or the next apply runs the
# restore again.
resource "fptcloud_backup_native_volume_job_run" "example" {
  count = var.volume_id != "" && var.volume_backup_run != "" ? 1 : 0

  vpc_id   = data.fptcloud_vpc.this.id
  job_id   = fptcloud_backup_native_volume_job.example[0].id
  triggers = { run = var.volume_backup_run }
}

# Restore points exist only after the job has run at least once. Turn the
# restore on after this data source lists a restore point.
data "fptcloud_backup_native_volume_restore_points" "example" {
  count = var.volume_id == "" ? 0 : 1

  vpc_id    = data.fptcloud_vpc.this.id
  volume_id = var.volume_id
}

# A restore runs once, on create. Changing an argument runs a new restore;
# destroying the resource only removes it from state.
resource "fptcloud_backup_native_volume_restore" "example" {
  count = var.volume_id != "" && var.volume_restore ? 1 : 0

  vpc_id           = data.fptcloud_vpc.this.id
  volume_id        = var.volume_id
  restore_point_id = data.fptcloud_backup_native_volume_restore_points.example[0].restore_points[0].id
  name             = "tf-example-restored"
  size_gb          = var.volume_restore_size_gb
}

data "fptcloud_backup_native_volume_jobs" "all" {
  vpc_id     = data.fptcloud_vpc.this.id
  depends_on = [fptcloud_backup_native_volume_job.example]
}

output "volume_jobs" {
  value = data.fptcloud_backup_native_volume_jobs.all.jobs
}

output "volume_restore_points" {
  value = try(data.fptcloud_backup_native_volume_restore_points.example[0].restore_points, [])
}

output "volume_job_run" {
  value = try({
    status      = fptcloud_backup_native_volume_job_run.example[0].status
    started_at  = fptcloud_backup_native_volume_job_run.example[0].started_at
    finished_at = fptcloud_backup_native_volume_job_run.example[0].finished_at
  }, null)
}

output "volume_restore_status" {
  value = try(fptcloud_backup_native_volume_restore.example[0].status, null)
}
