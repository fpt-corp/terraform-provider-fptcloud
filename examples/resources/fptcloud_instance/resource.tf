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

resource "fptcloud_instance" "example_03" {
  name              = "example-03"
  vpc_id            = "your_vpc_id"
  password          = "your_password"
  image_name        = "UBUNTU-20.04-04072024"
  flavor_name       = "2C2G"
  subnet_id         = "your_subnet_id"
  storage_size_gb   = 60
  storage_policy_id = "your_policy_id"
  status            = "POWERED_ON"
  tag_ids           = [your_tagging_first_id, your_tagging_id]
}

# Create a GPU instance. flavor_name must be a GPU flavor (see the fptcloud_flavor
# data source's gpu_id/gpu_name fields); gpu_plan sets the GPU billing plan
# ("hold" for reserved, "detach" for payg). Changing flavor_name later to/from a
# GPU flavor attaches/detaches the GPU in place, it does not replace the instance.
resource "fptcloud_instance" "example_gpu" {
  name              = "example-gpu"
  vpc_id            = "your_vpc_id"
  ssh_key           = "your_ssh_key"
  image_name        = "UBUNTU-20.04-04072024"
  flavor_name       = "1-1-Rtx-6000-2"
  gpu_plan          = "hold"
  subnet_id         = "your_subnet_id"
  storage_size_gb   = 60
  storage_policy_id = "your_policy_id"
  status            = "POWERED_ON"
}
