data "fptcloud_instances" "example" {
  vpc_id = "your_vpc_id"
}

# Names of all instances that are currently running
output "running_instance_names" {
  value = [for instance in data.fptcloud_instances.example.instances : instance.name if instance.status == "POWERED_ON"]
}
