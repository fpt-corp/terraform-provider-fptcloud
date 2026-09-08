data "fptcloud_vpc" "this" {
  name = "your_vpc_name"
}

data "fptcloud_alert_notification_methods" "email" {
  vpc_id = data.fptcloud_vpc.this.id
  type   = "EMAIL"
}

# Match on address, not name: the API groups methods by address, so the name of
# two methods sharing an address is not stable.
output "email_channels" {
  value = [for m in data.fptcloud_alert_notification_methods.email.methods : m.address]
}
