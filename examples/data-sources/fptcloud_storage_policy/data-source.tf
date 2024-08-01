data "fptcloud_storage_policy" "example" {
    vpc_id = "your_vpc_id"
    filter {
          key = "name"
          values = ["Premium-SSD"]
     }
}

output "show_value" {
  value = element(data.fptcloud_storage_policy.policy.storage_policies,0)
}