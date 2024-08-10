terraform {
    required_providers {
        fptcloud = {
            source = "github.com/terraform-provider/fptcloud"
        }
    }
}

provider "fptcloud" {
    token       = "ewogICJ0eXAiOiAiSldUIiwKICAiYWxnIjogIkhTMjU2Igp9.ewogICJpYXQiOiAxNzIzMTczMDQ1LjAzODg1MzIsCiAgInN1YiI6IHsKICAgICJpZCI6ICI1ZjYwZGIxMC04M2NkLTQyY2QtYTM3My0wNzYzZWI5NDUwMzIiLAogICAgImVtYWlsIjogInR1YW5ubjUyQGZwdC5jb20iLAogICAgImp0aSI6ICJjM2YzMGU0ZC1mZDc5LTQ2ODEtYmNkMy1jYjUyNTFiMWNlNzUiCiAgfSwKICAiZXhwIjogMTgwOTU3NjY0NS4wMzg4NTMyCn0.FUudXW4zmO3kU20-8gr_YqL2cZYtsmjKoU8ZJ3EPYO8"
    tenant_name = "Revoke Package"
    region      = "VN/HAN"
}
#
# resource "fptcloud_subnet" "example" {
#     vpc_id         = "45a41029-106f-43a5-846f-8e7fbb805055"
#     name           = "subnet-test-terraform"
#     type           = "NAT_ROUTED"
#     cidr           = "172.19.3.0/24"
#     gateway_ip     = "172.19.3.1"
#     static_ip_pool = "172.19.3.5-172.19.3.10"
# }



data "fptcloud_subnet" "example" {
  vpc_id = "45a41029-106f-43a5-846f-8e7fbb805055"
#   filter {
#     key = "id"
#     values = ["679fe554-0a53-442c-931f-1874db4f731a"]
#   }

}

output "show_value" {
  value = data.fptcloud_subnet.example
}

# ==============================================================
# ==============================================================
# ==============================================================

# data "fptcloud_instance_group" "example" {
# #   vpc_id = "6daffc98-fc17-4e5d-aa5d-a221517785f6"
#   vpc_id = "ff6ab93f-4b05-4f04-94f8-ebfdb5bba6f3"
# #   filter {
# #     key = "id"
# #     values = ["679fe554-0a53-442c-931f-1874db4f731a"]
# #   }
#   filter {
#     key = "name"
#     values = ["John Doe"]
#   }
# }
#
# output "show_value" {
#   value = data.fptcloud_instance_group.example
# }

# ==============================================================
# ==============================================================
# ==============================================================

# resource "fptcloud_instance_group" "example" {
#   vpc_id = "6daffc98-fc17-4e5d-aa5d-a221517785f6"
#   name = "instance-test"
#   policy_id = "6e1c5151-39cd-4735-ae27-dfd77233630a"
#   vm_ids = "vm_ids"
# }

# ==============================================================
# ==============================================================
# ==============================================================

# data "fptcloud_floating_ip_rule_ip_address" "example" {
#   vpc_id = "120bd194-7031-42ae-86a6-8b53a90ff9ae"
#   filter {
#     key = "id"
#     values = ["b18a55c5-039d-44c5-9871-bbfb2f0fb09f"]
#   }
#   filter {
#     key = "name"
#     values = ["103.160.80.101"]
#   }
# }
#
# output "show_value" {
#   value = data.fptcloud_floating_ip_rule_ip_address.example
# }

# ==============================================================
# ==============================================================
# ==============================================================

# data "fptcloud_floating_ip_rule_instance" "example" {
#   vpc_id = "120bd194-7031-42ae-86a6-8b53a90ff9ae"
#   filter {
#     key = "name"
#     values = ["vm-24071722550-k43e47jt"]
#   }
# }
#
# output "show_value" {
#   value = data.fptcloud_floating_ip_rule_instance.example
# }

# ==============================================================
# ==============================================================
# ==============================================================

#
# resource "fptcloud_floating_ip" "example" {
#   vpc_id = "120bd194-7031-42ae-86a6-8b53a90ff9ae"
#   floating_ip_id = "new"
#   instance_id= "b90ddce4-7c85-4194-9814-2ca0218410cc"
#   floating_ip_port = 1
#   instance_port = 1
# }

# resource "fptcloud_floating_ip" "example" {
#   vpc_id = "934a79d8-8de9-40a2-a5e6-cca500132f15"
#   floating_ip_id = "new"
# }

# data "fptcloud_floating_ip" "example" {
#   vpc_id = "120bd194-7031-42ae-86a6-8b53a90ff9ae"
#   filter {
#     key = "ip_address"
#     values = ["103.160.80.184"]
#   }
# }
#
# output "show_value" {
#   value = data.fptcloud_floating_ip.example
# }

# ==============================================================
# ==============================================================
# ==============================================================

# data "fptcloud_instance_group_policy" "data" {
#   vpc_id = "6daffc98-fc17-4e5d-aa5d-a221517785f6"
#   filter {
#     key = "name"
#     values = ["Soft Affinity"]
#   }
# }
#
# output "show_value" {
#   value = data.fptcloud_instance_group_policy.data
# }

# ==============================================================
# ==============================================================
# ==============================================================

# resource "fptcloud_ssh_key" "example" {
#   name = "your_ssh_name"
#   public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDLAWx447unnmJLgdT0U3mu6luLioI6DI7/1zXBYT+9VAgqiBcff4kfnLLNt1k2dIO6DlzpWMgVDpXbAwr+UWGnavLrw2+2du4EQE3HQPajChuJY3bV3U6CNOzsnFdTZjwPEqifhIOQTm407wIkutZcQ8Jc/RqiB6+tA5scXdbvOOoG+wjapoz3eqw07OYgJGRgZ987LGpn1jcyHxspoE4XbYiFjRDBcQlF5bMSMsMTdUfcmG2VToSXeMgN3aeCAC+r9PcHbtGfphOsMIKMe7lda/hOepsS3Py69QzWkVOn+w/k0ZIU2chAdQo8T49Ce3PnVRpYSrxbq+X8rEKNC+aB"
# }

# ==============================================================
# ==============================================================
# ==============================================================

# data "fptcloud_storage" "example" {
#   vpc_id = "120bd194-7031-42ae-86a6-8b53a90ff9ae"
#   name = "storage-disk-2272"
# }
#
# output "output-example" {
#   value = data.fptcloud_storage.example
# }

# ==============================================================
# ==============================================================
# ==============================================================

# data "fptcloud_storage_policy" "example" {
#   vpc_id = "120bd194-7031-42ae-86a6-8b53a90ff9ae"
#   filter {
#     key = "name"
#     values = ["Premium-SSD"]
#   }
# }
#
# output "show_value" {
#   value = element(data.fptcloud_storage_policy.example,0)
#
# }