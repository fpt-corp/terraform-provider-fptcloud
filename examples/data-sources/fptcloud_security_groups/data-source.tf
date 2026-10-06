data "fptcloud_security_groups" "example" {
  vpc_id = "your_vpc_id"
}

# Security group IDs indexed by name
output "security_group_ids" {
  value = { for sg in data.fptcloud_security_groups.example.security_groups : sg.name => sg.id }
}
