data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

# Every instance that has restore points in the VPC, one entry per instance and
# backup job - the same table the portal's Restore tab shows. Entries are ordered
# by their most recent restore point, newest first.
data "fptcloud_backup_veeam_restore_groups" "all" {
  vpc_id = data.fptcloud_vpc.this.id
}

# Feed each entry into fptcloud_backup_veeam_restore_points without reading the
# backup jobs first. Legacy entries have no vm_id and cannot be listed, so skip
# them.
data "fptcloud_backup_veeam_restore_points" "each" {
  for_each = {
    for g in data.fptcloud_backup_veeam_restore_groups.all.groups :
    "${g.backup_job_id}/${g.vm_id}" => g
    if g.vm_id != ""
  }
  vpc_id        = data.fptcloud_vpc.this.id
  backup_job_id = each.value.backup_job_id
  vm_id         = each.value.vm_id
}

output "latest_restore_point_per_instance" {
  value = {
    for key, rp in data.fptcloud_backup_veeam_restore_points.each :
    key => try(rp.points[0], null)
  }
}
