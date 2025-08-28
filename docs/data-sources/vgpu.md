# fptcloud_vgpu Data Source

Provides information about vGPUs available in FPT Cloud.

## Example Usage

```hcl
# Get all vGPUs for a VPC
data "fptcloud_vgpu" "all_vgpus" {
  vpc_id = var.vpc_id
}

# Get specific vGPU by name using filter
data "fptcloud_vgpu" "nvidia_a30" {
  vpc_id = var.vpc_id
  filter {
    key    = "name"
    values = ["nvidia_a30"]
  }
}

# Get multiple vGPU types using filter
data "fptcloud_vgpu" "nvidia_gpus" {
  vpc_id = var.vpc_id
  filter {
    key    = "name"
    values = ["nvidia_a30", "nvidia_a100", "nvidia_rtx4090"]
  }
}

# Use vGPU ID in managed Kubernetes cluster
resource "fptcloud_managed_kubernetes_engine_v1" "cluster" {
  vpc_id       = var.vpc_id
  cluster_name = "my-cluster"
  network_id   = var.network_id

  pools {
    name             = "gpu-pool"
    storage_profile  = "Premium-SSD"
    worker_type      = var.gpu_flavor_id
    worker_disk_size = 40
    scale_min        = 1
    scale_max        = 2

    vgpu_id                  = element(data.fptcloud_vgpu.nvidia_a30.vgpus, 0).id
    max_client               = 3
    gpu_sharing_client       = "timeSlicing"
    gpu_driver_version       = "default"
    driver_installation_type = "pre-install"

    worker_base = false
  }
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The VPC ID to list vGPUs for.
* `filter` - (Optional) Filter the results. The `filter` block supports:
  * `key` - (Required) The field to filter by. Valid values are `name`, `display_name`, `status`, `platform`.
  * `values` - (Required) A list of values to filter by.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `vgpus` - A list of vGPUs. Each vGPU has the following attributes:
  * `id` - The ID of the vGPU.
  * `name` - The name of the vGPU (e.g., "nvidia_a30", "nvidia_a100").
  * `display_name` - The display name of the vGPU (e.g., "NVIDIA Tesla A30").
  * `created_at` - The creation date of the vGPU.
  * `memory` - The memory size (GB) of the vGPU.
  * `status` - The status of the vGPU.
  * `is_dedicated` - Whether the vGPU is dedicated.
  * `service_type_id` - The service type ID of the vGPU.
  * `platform` - The platform of the vGPU.
  * `parent_id` - The parent ID of the vGPU.
  * `enable_nvme` - Whether NVMe is enabled for the vGPU.

## Common vGPU Names

The following are common vGPU names that can be used for filtering:

* `nvidia_a30` - NVIDIA Tesla A30
* `nvidia_a100` - NVIDIA Tesla A100
* `nvidia_a10` - NVIDIA Tesla A10
* `nvidia_rtx4090` - NVIDIA RTX4090
* `hgx_h100` - NVIDIA H100 SXM5
