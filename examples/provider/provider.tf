terraform {
  required_providers {
    fptcloud = {
      source = "github.com/terraform-provider/fptcloud"
    }
  }
}

provider "fptcloud" {
  token       = "ewogICJ0eXAiOiAiSldUIiwKICAiYWxnIjogIkhTMjU2Igp9.ewogICJpYXQiOiAxNzIxOTIyNzEzLjg3NTE0LAogICJzdWIiOiB7CiAgICAiaWQiOiAiNWY2MGRiMTAtODNjZC00MmNkLWEzNzMtMDc2M2ViOTQ1MDMyIiwKICAgICJlbWFpbCI6ICJ0dWFubm41MkBmcHQuY29tIiwKICAgICJqdGkiOiAiOWU1MzdiYmItMDUzYi00MzIwLTg1NjMtMzdiNTI0YTM4OTJjIgogIH0sCiAgImV4cCI6IDE4MDgzMjYzMTMuODc1MTQKfQ.O4HQd-HG8X1xqrTULYUzwOwQbf9CcPx85oOnuPSJGPo"
  tenant_name = "Revoke Package"
  region      = "VN/HAN"
}



data "fptcloud_instance_group_policy" "data" {
  vpc_id = "120bd194-7031-42ae-86a6-8b53a90ff9ae"
  filter {
    key = "name"
    values = ["Soft Affinity"]
  }
}

output "show_value" {
  value = data.fptcloud_instance_group_policy.data
}


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