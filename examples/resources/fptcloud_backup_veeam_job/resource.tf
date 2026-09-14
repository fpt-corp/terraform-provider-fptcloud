data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

# Notification methods must already exist in the portal under Alert.
# Always source the IDs from this data source: the API silently ignores IDs it
# does not recognise, so a hand-typed ID leaves the job with no notifications
# and no error.
data "fptcloud_alert_notification_methods" "email" {
  vpc_id = data.fptcloud_vpc.this.id
  type   = "EMAIL"
}

resource "fptcloud_backup_veeam_job" "db_daily" {
  vpc_id      = data.fptcloud_vpc.this.id
  name        = "job-db-daily"
  description = "Daily backup for the database instances"

  # One job protects several instances. Adding or removing an instance updates
  # the job in place and does not delete existing restore points.
  vm_ids = [
    "your_instance_id_1",
    "your_instance_id_2",
  ]

  schedule_enabled = true

  retention {
    cycles     = 7
    limit_type = "Days" # or "Cycles", shown as "Restore points" in the portal
  }

  schedule {
    type = "daily"

    daily {
      type   = "weekDays" # or "Everyday"
      run_at = "22:00:00"
    }
  }

  notification_method_ids = [
    for m in data.fptcloud_alert_notification_methods.email.methods :
    m.id if m.address == "ops@example.com"
  ]

  timeouts {
    create = "45m"
  }
}

# Runs every four hours, but only between 20:00 and 23:59.
#
# start_hour must be less than or equal to end_hour: the window cannot wrap
# past midnight. A wrapping window would produce a schedule that never runs,
# and the API would accept it without complaint, so Terraform rejects it during
# plan instead.
resource "fptcloud_backup_veeam_job" "app_hourly" {
  vpc_id = data.fptcloud_vpc.this.id
  name   = "job-app-hourly"
  vm_ids = ["your_instance_id_3"]

  retention {
    cycles     = 14
    limit_type = "Cycles"
  }

  schedule {
    type = "period"

    period {
      full_period = 4
      start_hour  = 20
      end_hour    = 23
    }
  }
}

# Runs on the last day of every month.
resource "fptcloud_backup_veeam_job" "archive_monthly" {
  vpc_id = data.fptcloud_vpc.this.id
  name   = "job-archive-monthly"
  vm_ids = ["your_instance_id_4"]

  retention {
    cycles     = 12
    limit_type = "Cycles"
  }

  schedule {
    type = "monthly"

    monthly {
      run_at              = "23:00:00"
      day_number_in_month = "onDay"
      day_of_month        = 32 # 32 means the last day of the month
    }
  }
}
