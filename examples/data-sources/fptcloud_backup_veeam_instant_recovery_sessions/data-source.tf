data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

# Every Instant Recovery session open in the VPC, including the ones started
# from the portal. An open session blocks the backup job of the instance it was
# mounted from, so this is the list to check when a job stops running.
data "fptcloud_backup_veeam_instant_recovery_sessions" "open" {
  vpc_id = data.fptcloud_vpc.this.id
}

output "open_sessions" {
  value = [
    for session in data.fptcloud_backup_veeam_instant_recovery_sessions.open.sessions :
    {
      name          = session.vm_mount_name
      state         = session.state
      mode          = session.mode
      ready_migrate = session.ready_migrate
    }
  ]
}
