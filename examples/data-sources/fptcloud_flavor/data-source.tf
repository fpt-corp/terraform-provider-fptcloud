data "fptcloud_flavor" "example" {
  vpc_id = "your_vpc_id"
  filter {
    key = "name"
    values = ["Extra-8"]
  }
}

data "fptcloud_flavor" "example_filter" {
  vpc_id = "your_vpc_id"
  filter {
    key = "cpu"
    values = [2]
  }
  filter {
    key = "memory_mb"
    values = [4 * 1024]
  }
  filter {
    key = "type"
    values = ["VM_SIZE"]
  }
}

output "show_value" {
  value = element(data.fptcloud_flavor.example.flavors, 0)
}

# GPU flavors only. Each flavor carries gpu_id/gpu_name (set when it has a GPU
# attached) and is_nvme (true when its storage is a physical NVMe disk instead
# of a regular storage_policy_id-backed disk).
data "fptcloud_flavor" "gpu_only" {
  vpc_id = "your_vpc_id"
  filter {
    key    = "type"
    values = ["GPU_SIZE"]
  }
}

output "gpu_flavors" {
  value = data.fptcloud_flavor.gpu_only.flavors
}