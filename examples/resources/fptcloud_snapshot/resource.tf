# Each resource is one snapshot. To keep several, declare several: editing an
# argument does not add a snapshot, it replaces the one this resource owns.
# To retake a snapshot without changing anything:
#   terraform apply -replace='fptcloud_snapshot.before_upgrade'
resource "fptcloud_snapshot" "before_upgrade" {
  vpc_id      = "your_vpc_id"
  instance_id = fptcloud_instance.server.id
  name        = "before-upgrade"

  # Optional.
  tag_ids     = ["your_tag_id"]
  include_ram = false
}

output "snapshot_id" {
  value = fptcloud_snapshot.before_upgrade.id
}
