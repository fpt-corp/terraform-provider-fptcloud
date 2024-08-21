resource "fptcloud_security_group" "security_group_example" {
  vpc_id    = "your_vpc_id"
  name      = "your_security_group_name"
  subnet_id = "your_subnet_id"
  type      = "ACL"
}

resource "fptcloud_security_group_rule" "example_ssh_rule_for_all_traffic" {
  vpc_id            = "your_vpc_id"
  security_group_id = fptcloud_security_group.security_group_example.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "22"
  sources = ["ALL"]
  description = "inbound ssh rule for all traffic"
  # This is required to ensure the security group is created before the rule
  depends_on = [fptcloud_security_group.security_group_example]
}

resource "fptcloud_security_group_rule" "example_all_traffic_rule" {
  vpc_id            = "your_vpc_id"
  security_group_id = fptcloud_security_group.security_group_example.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "ALL"
  port_range        = "ALL"
  sources = ["ALL"]
  description       = "inbound all traffic rule"
  depends_on = [fptcloud_security_group.security_group_example]
}

resource "fptcloud_security_group_rule" "example_all_traffic_rule_for_tcp" {
  vpc_id            = "your_vpc_id"
  security_group_id = fptcloud_security_group.security_group_example.id
  direction         = "INGRESS"
  action            = "ALLOW"
  protocol          = "TCP"
  port_range        = "ALL"
  sources = ["ALL"]
  description       = "inbound all traffic rule for tcp"
  depends_on = [fptcloud_security_group.security_group_example]
}