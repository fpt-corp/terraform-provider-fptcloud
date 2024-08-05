data "fptcloud_floating_ip_rule_ip_address" "example_ip_address" {
  vpc_id = "your_vpc_id"
  filter {
    key = "id"
    values = ["your_ip_address_id"]
  }
}

data "fptcloud_floating_ip_rule_instance" "example_instance" {
  vpc_id = "your_vpc_id"
  filter {
    key = "id"
    values = ["your_instance_id"]
  }
}

resource "fptcloud_floating_ip" "example" {
  vpc_id = "your_vpc_id"
  floating_ip_id = "new"
}

resource "fptcloud_floating_ip" "example" {
  vpc_id = "your_vpc_id"
  floating_ip_id = "new"
  instance_id = fptcloud_floating_ip_rule_instance.example_instance.id
  # This is optional to ensure the floating ip is created before the rule
  depends_on = [fptcloud_floating_ip_rule_instance.example_instance]
}

resource "fptcloud_floating_ip" "example" {
  vpc_id = "your_vpc_id"
  floating_ip_id = fptcloud_floating_ip_rule_ip_address.example_ip_address.id
  instance_id = fptcloud_floating_ip_rule_instance.example_instance.id
  # This is optional to ensure the floating ip is created before the rule
  depends_on = [fptcloud_floating_ip_rule_ip_address.example_ip_address, fptcloud_floating_ip_rule_instance.example_instance]
}

resource "fptcloud_floating_ip" "example" {
  vpc_id = "your_vpc_id"
  floating_ip_id = fptcloud_floating_ip_rule_ip_address.example_ip_address.id
  instance_id = fptcloud_floating_ip_rule_instance.example_instance.id
  floating_ip_port = 1
  instance_port = 1
  # This is optional to ensure the floating ip is created before the rule
  depends_on = [fptcloud_floating_ip_rule_ip_address.example_ip_address, fptcloud_floating_ip_rule_instance.example_instance]
}