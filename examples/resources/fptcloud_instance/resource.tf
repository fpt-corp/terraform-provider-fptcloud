# Create instance with SSH key
resource "fptcloud_instance" "example_01" {
  name              = "example-01"
  vpc_id            = "your_vpc_id"
  ssh_key           = "your_ssh_key"
  image_name        = "UBUNTU-20.04-04072024"
  flavor_name       = "2C2G"
  subnet_id         = "your_subnet_id"
  storage_size_gb   = 60
  storage_policy_id = "your_policy_id"
  status            = "POWERED_ON"
}

# Create instance with password
resource "fptcloud_instance" "example_02" {
  name              = "example-02"
  vpc_id            = "your_vpc_id"
  password          = "your_password"
  image_name        = "UBUNTU-20.04-04072024"
  flavor_name       = "2C2G"
  subnet_id         = "your_subnet_id"
  storage_size_gb   = 60
  storage_policy_id = "your_policy_id"
  status            = "POWERED_ON"
}

# Create instance with security group
resource "fptcloud_instance" "example_02" {
  name              = "example-02"
  vpc_id            = "your_vpc_id"
  password          = "your_password"
  image_name        = "UBUNTU-20.04-04072024"
  flavor_name       = "2C2G"
  subnet_id         = "your_subnet_id"
  storage_size_gb   = 60
  storage_policy_id = "your_policy_id"
  status            = "POWERED_ON"
  security_group_ids = ["your_security_group_id"]
}

# Create instance with instance group
resource "fptcloud_instance" "example_02" {
  name              = "example-02"
  vpc_id            = "your_vpc_id"
  password          = "your_password"
  image_name        = "UBUNTU-20.04-04072024"
  flavor_name       = "2C2G"
  subnet_id         = "your_subnet_id"
  storage_size_gb   = 60
  storage_policy_id = "your_policy_id"
  status            = "POWERED_ON"
  instance_group_id = "your_instance_group_id"
}
