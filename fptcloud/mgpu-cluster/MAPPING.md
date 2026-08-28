# mgpu-cluster: Swagger create-cluster body → Terraform

Rà soát từng field của body Swagger `POST .../m-fke/{platform}/hpc/create-cluster`
(bên dưới) đối chiếu với `fptcloud_managed_gpu_cluster`. Đi theo đúng thứ tự
field xuất hiện trong JSON để dễ tick khi rà soát tay.

Ký hiệu:
- **(Đã có map)** — có field Terraform tương ứng, đã cắm dây trong `MapTerraformToJson`/`remapPools` (`utils.go`).
- **(Chưa map)** — JSON có field này nhưng Terraform chưa có field tương ứng / chưa gửi lên.
- **(Provider tự set)** — không phải input Terraform, provider tự tính và gắn cứng hoặc để trống.

Nguồn đối chiếu: `types.go` (`managedGpuCluster`, `managedGpuClusterPool`,
`managedGpuClusterJson`, `managedGpuClusterPoolJson`), `utils.go`
(`MapTerraformToJson`, `remapPools`), `defaults.go`, `validations.go`.

## Body Swagger gốc (để đối chiếu)

```json
{
  "is_enable_auto_upgrade": false,
  "auto_upgrade_expression": ["in fugiat deserunt", "officia proident mollit sunt eu"],
  "auto_upgrade_timezone": "veniam",
  "cluster_autoscaler": {
    "expander": "ut consectetur officia",
    "isEnableAutoScaling": true,
    "maxGracefulTerminationSeconds": -9545742,
    "maxNodeProvisionTime": "veniam aliquip",
    "scaleDownDelayAfterAdd": -84171627,
    "scaleDownDelayAfterDelete": -54985744,
    "scaleDownDelayAfterFailure": 88919150,
    "scaleDownUnneededTime": -66995313,
    "scaleDownUtilizationThreshold": 26947407.575009882,
    "scanInterval": 65426406
  },
  "clusterEndpointAccess": {
    "type": "eu ea",
    "allowCidr": ["sit exercitation esse", "laboris Lorem culpa"]
  },
  "cluster_name": "sit laboris",
  "currentNetworking": "pariatur quis ullamco commodo",
  "default_storage_profile": "amet nisi reprehenderit irure anim",
  "edge_gateway_id": "sed aliqua nulla",
  "edge_gateway_name": "occaecat id aliqua",
  "internal_subnet_lb": "reprehenderit la",
  "k8s_max_pod": -60982206,
  "k8s_version": "cupidatat Duis",
  "lbInternalNetwork": {
    "cidr": "ut culpa dolore incididunt",
    "label": "in esse",
    "label4sending": "in culpa",
    "value": "qui tempor officia"
  },
  "loadBalancerType": "culpa dolor",
  "network_id": "elit ut",
  "network_node_prefix": 77817645,
  "network_overlay": "ut sunt",
  "network_type": "elit adipisicing ut nostrud",
  "pod_network": "id Lorem proident dolore",
  "pod_prefix": "aliqua sed ipsum",
  "pools": [
    {
      "ram": "string", "cpu": "string", "gpu_amount": "string",
      "vGpuId": "string", "isCreate": "string", "isScale": "string", "isOthers": "string",
      "deltaQuotaScale": "string", "isEnableAutoRepair": "string",
      "driverInstallationType": "string", "auto_scale": "string",
      "container_runtime": "string", "gpuDriverVersion": "string",
      "gpuSharingClient": "string", "gpuTemplateVersion": "string",
      "hpc_flavor_id": "string", "hpc_flavor_name": "string", "hpc_number_server": "string",
      "isDisplayGPU": "string", "kubernetes": "string", "kv": "string",
      "maxClient": "string", "network_id": "string", "network_name": "string",
      "scale_max": "string", "scale_min": "string", "storage_profile": "string",
      "tags": "string", "taints": "string", "worker_base": "string",
      "worker_disk_size": "string", "worker_pool_id": "string", "worker_type": "string"
    }
  ],
  "purpose": "elit in",
  "secret_binding_name": "in",
  "service_network": "et amet quis aute",
  "service_prefix": "eiusmod magna",
  "ssh_id": "eiusmod fugiat commodo",
  "ssh_name": "voluptate ven",
  "ssh_public_key": "incididunt ut"
}
```

> Lưu ý: đây là schema mẫu do Swagger auto-generate (toàn bộ field trong `pools`
> bị ép về kiểu `"string"`), không phải một request thật — dùng để liệt kê đủ
> tên field, không dùng để suy ra kiểu dữ liệu thật.

## Top level — theo đúng thứ tự trong body

| # | JSON field | Trạng thái | Terraform field |
|---|---|---|---|
| 1 | `is_enable_auto_upgrade` | **Đã có map** | `is_enable_auto_upgrade` |
| 2 | `auto_upgrade_expression` | **Đã có map** | `auto_upgrade_expression` |
| 3 | `auto_upgrade_timezone` | **Đã có map** | `auto_upgrade_timezone` |
| 4 | `cluster_autoscaler` | **Đã có map** (một phần) | `cluster_autoscaler` — xem [Cluster autoscaler](#cluster-autoscaler-field-lệch) |
| 5 | `clusterEndpointAccess` | **Đã có map** | `cluster_endpoint_access` |
| 6 | `cluster_name` | **Đã có map** | `cluster_name` (có suffix ngẫu nhiên nếu chưa đúng định dạng) |
| 7 | `currentNetworking` | **Chưa map** | — provider gửi rỗng, không có field Terraform tương ứng |
| 8 | `default_storage_profile` | **Đã có map** | `default_storage_profile` |
| 9 | `edge_gateway_id` | **Đã có map** | `edge_gateway_id` (bị xoá rỗng nếu platform = OSP) |
| 10 | `edge_gateway_name` | **Đã có map** | `edge_gateway_name` — nhưng do **provider tự tính** (lookup theo `edge_gateway_id`), không lấy trực tiếp từ input user |
| 11 | `internal_subnet_lb` | **Đã có map** | `internal_subnet_lb` |
| 12 | `k8s_max_pod` | **Đã có map** | `k8s_max_pod` |
| 13 | `k8s_version` | **Đã có map** | `k8s_version` |
| 14 | `lbInternalNetwork` | **Chưa map** | — không có field Terraform, provider không gửi (field reserved) |
| 15 | `loadBalancerType` | **Đã có map** | `load_balancer_type` |
| 16 | `network_id` | **Đã có map** | `network_id` |
| 17 | `network_node_prefix` | **Đã có map** | `network_node_prefix` |
| 18 | `network_overlay` | **Đã có map** | `network_overlay` |
| 19 | `network_type` | **Đã có map** | `network_type` |
| 20 | `pod_network` | **Đã có map** | `pod_network` |
| 21 | `pod_prefix` | **Đã có map** | `pod_prefix` |
| 22 | `pools` | **Đã có map** | `pools` (block) — xem [Pool](#pool) |
| 23 | `purpose` | **Đã có map** | `purpose` |
| 24 | `secret_binding_name` | **Đã có map** | `secret_binding_name` |
| 25 | `service_network` | **Đã có map** | `service_network` |
| 26 | `service_prefix` | **Đã có map** | `service_prefix` |
| 27 | `ssh_id` | **Đã có map** | `ssh_id` |
| 28 | `ssh_name` | **Đã có map** | `ssh_name` |
| 29 | `ssh_public_key` | **Đã có map** | `ssh_public_key` |

### Có trong Terraform nhưng KHÔNG có trong body Swagger mẫu này

Những field này Terraform vẫn gửi lên (theo `managedGpuClusterJson`), nhưng
Swagger mẫu bạn gửi không liệt kê — khả năng cao doc Swagger đang **thiếu**,
không phải provider gửi thừa. Cần đối chiếu lại với backend:

| JSON field (provider gửi) | Terraform field | Ghi chú |
|---|---|---|
| `vGpuId` không nằm ở top level, `software` mới đúng | `software` | **Cả object `software` không xuất hiện trong body Swagger mẫu** — cần xác nhận lại tên key thật trên backend (`software` hay khác) |
| `isV2` | — (provider tự set) | chỉ gửi khi gọi endpoint v2, `k8s_version >= 1.33` |
| `type_create` | — (provider tự set) | luôn `"create"` |
| `os_version` | — (provider tự set, không set) | reserved, không populate |

### Provider tự set (không phải input Terraform)

| JSON field | Giá trị |
|---|---|
| `type_create` | luôn `"create"` |
| `isV2` *(omit nếu false)* | `true` khi gọi endpoint v2 |
| `currentNetworking` *(omit)* | reserved, không populate khi create |
| `lbInternalNetwork` *(omit)* | reserved; không populate khi create |
| `os_version` *(omit)* | reserved, không populate |

## Pool

Body Swagger để mọi field trong `pools[]` là kiểu `"string"` (do auto-gen), đối
chiếu theo **tên field**, không theo kiểu.

| # | JSON field trong `pools[]` | Trạng thái | Terraform field |
|---|---|---|---|
| 1 | `ram` | **Chưa map** | — không có field Terraform; do backend tự tính từ `hpc_flavor_id`, provider không gửi |
| 2 | `cpu` | **Chưa map** | — như trên |
| 3 | `gpu_amount` | **Chưa map** | — như trên |
| 4 | `vGpuId` | **Đã có map** | `vgpu_id` |
| 5 | `isCreate` | **Provider tự set** | — `true` khi `worker_pool_id = null` (pool mới) |
| 6 | `isScale` | **Provider tự set** | — luôn `false` khi create |
| 7 | `isOthers` | **Provider tự set** | — luôn `false` khi create |
| 8 | `deltaQuotaScale` | **Provider tự set** | — luôn gửi `0`, không có field Terraform |
| 9 | `isEnableAutoRepair` | **Đã có map** | `is_enable_auto_repair` |
| 10 | `driverInstallationType` | **Đã có map** | `driver_installation_type` |
| 11 | `auto_scale` | **Provider tự set** | — luôn `false` (pool bare-metal là số server cố định, không autoscale theo pool) |
| 12 | `container_runtime` | **Đã có map** | `container_runtime` |
| 13 | `gpuDriverVersion` | **Đã có map** | `gpu_driver_version` |
| 14 | `gpuSharingClient` | **Đã có map** | `gpu_sharing_client` |
| 15 | `gpuTemplateVersion` | **Đã có map** | `gpu_template_version` |
| 16 | `hpc_flavor_id` | **Đã có map** | `hpc_flavor_id` |
| 17 | `hpc_flavor_name` | **Đã có map** | `hpc_flavor_name` |
| 18 | `hpc_number_server` | **Đã có map** | `hpc_number_server` |
| 19 | `isDisplayGPU` | **Provider tự set** | — `true` khi `vgpu_id != ""` |
| 20 | `kubernetes` | **Chưa map** | — không có field Terraform, provider không gửi (reserved) |
| 21 | `kv` | **Đã có map** | `kv` (Set `{name, value}`) — format khác hẳn `"string"` trong doc, xem [KV](#kv) |
| 22 | `maxClient` | **Đã có map** | `max_client` |
| 23 | `network_id` | **Đã có map** | `network_id` |
| 24 | `network_name` | **Đã có map** | `network_name` |
| 25 | `scale_max` | **Chưa map (mgpu không có)** | mgpu dùng `hpc_number_server` cố định thay vì `scale_min`/`scale_max`; field này chỉ tồn tại bên `mfke` — **nghi ngờ doc Swagger đang copy nhầm từ mfke** |
| 26 | `scale_min` | **Chưa map (mgpu không có)** | như trên |
| 27 | `storage_profile` | **Đã có map** | `storage_profile` |
| 28 | `tags` | **Đã có map** | `tags` (list Terraform → string nối bằng `\n` khi gửi) |
| 29 | `taints` | **Đã có map** | `taints` (Set `{key, value, effect}`) — format khác `"string"`, xem [Taints](#taints) |
| 30 | `worker_base` | **Đã có map** | `worker_base` |
| 31 | `worker_disk_size` | **Đã có map** | `worker_disk_size` |
| 32 | `worker_pool_id` | **Đã có map** | `name` — `null` khi tạo pool mới |
| 33 | `worker_type` | **Chưa map (mgpu không có)** | mgpu dùng `hpc_flavor_id` thay cho `worker_type` của mfke — **nghi ngờ doc Swagger đang copy nhầm từ mfke** |

### Nghi vấn cần xác nhận lại với backend/API doc

1. **`scale_min` / `scale_max` / `worker_type`** xuất hiện trong doc Swagger
   của mgpu nhưng thực tế đây là field của **mfke** (worker pool autoscale
   theo range). Bare-metal GPU pool dùng `hpc_number_server` (số lượng máy cố
   định) + `hpc_flavor_id`/`hpc_flavor_name` thay thế. → có khả năng Swagger
   doc dùng chung schema với mfke, chưa tách riêng cho mgpu. Cần hỏi backend
   xem 3 field này có thật sự được API mgpu chấp nhận không, hay bị bỏ qua.
2. **`software` block** hoàn toàn không xuất hiện trong body Swagger mẫu, dù
   Terraform coi đây là field `Required` khi tạo cluster GPU (chọn GPU
   Operator / Network Operator / Slurm Operator / vGPU Scheduler). Cần xác
   nhận lại tên key JSON thật (`software`? nằm lồng trong `pools`? tên khác?).
3. **`ram` / `cpu` / `gpu_amount` / `kubernetes`** trong `pools[]` — hiện tại
   provider không gửi các field này (giả định backend tự suy ra từ
   `hpc_flavor_id`). Nếu Swagger liệt kê chúng là field request hợp lệ, cần
   xác nhận provider có cần gửi lên không, hoặc đây là field chỉ xuất hiện ở
   response.

## Chi tiết các field lệch tên/format

### Cluster autoscaler (field lệch)

Terraform → JSON, đổi tên key sang camelCase. Riêng 2 field
`maxGracefulTerminationSeconds` và `maxNodeProvisionTime` xuất hiện trong
Swagger nhưng **không có field Terraform tương ứng** — không có input để
điền, provider không gửi 2 field này.

| Terraform | JSON gửi lên | Trạng thái |
|---|---|---|
| `is_enable_auto_scaling` | `isEnableAutoScaling` | Đã có map |
| `scale_down_delay_after_add` | `scaleDownDelayAfterAdd` | Đã có map |
| `scale_down_delay_after_delete` | `scaleDownDelayAfterDelete` | Đã có map |
| `scale_down_delay_after_failure` | `scaleDownDelayAfterFailure` | Đã có map |
| `scale_down_unneeded_time` | `scaleDownUnneededTime` | Đã có map |
| `scale_down_utilization_threshold` | `scaleDownUtilizationThreshold` | Đã có map |
| `scan_interval` | `scanInterval` | Đã có map |
| `expander` | `expander` (lower-case hoá) | Đã có map |
| — | `maxGracefulTerminationSeconds` | **Chưa map** — không có field Terraform |
| — | `maxNodeProvisionTime` | **Chưa map** — không có field Terraform |

### Cluster endpoint access

```hcl
cluster_endpoint_access = { type = "private", allow_cidr = ["0.0.0.0/0"] }
```
```json
"clusterEndpointAccess": { "type": "private", "allowCidr": ["0.0.0.0/0"] }
```

`allow_cidr` → `allowCidr`; tên key ngoài cùng cũng đổi (`cluster_endpoint_access` → `clusterEndpointAccess`).

### KV

`types.Set` `{name, value}` → mảng các map 1-key, sort theo key. Bỏ qua entry
key/value rỗng và key hệ thống (`isSystemGeneratedKey`, hiện tại là
`nvidia.com/device-plugin.config`).

```hcl
kv = [{ name = "team", value = "ml" }]
```
```json
"kv": [{ "team": "ml" }]
```

Với pool GPU (`vgpu_id != ""`), provider tự động thêm 2 key hệ thống nếu
người dùng chưa khai:
- `nvidia.com/mig.config` (mặc định `all-1g.6gb`)
- `worker.fptcloud/type: gpu`

### Taints

```hcl
taints = [{ key = "dedicated", value = "gpu", effect = "NoSchedule" }]
```
```json
"taints": [{ "dedicated": { "value": "gpu", "effect": "NoSchedule" } }]
```

`effect` ∈ `NoSchedule`, `PreferNoSchedule`, `NoExecute`. Pool có
`worker_base = true` chỉ được có taint khi cluster có ≥ 2 pool, và chỉ được
đúng 1 taint `CriticalAddonsOnly=true:NoSchedule`.

### Pool identity

`worker_pool_id` là `*string`: `null` khi tạo pool mới, tên pool khi update
pool đã tồn tại. Tên `worker-new` bị cấm dùng (`validatePool`).

### Pool networking

`MapTerraformToJson` resolve `network_id` của cluster trước, dùng làm fallback:

| Pool config | `network_id` gửi | `network_name` gửi |
|---|---|---|
| không set gì | của cluster | của cluster |
| chỉ set `network_name` | của cluster | của pool |
| chỉ set `network_id` | của pool | của cluster |
| set cả 2 | của pool | của pool |

### Edge gateway

- **OSP** — `edge_gateway_id` và `edge_gateway_name` đều bị xoá rỗng.
- **Khác OSP** — lookup theo `edge_gateway_id`, `edge_gateway_name` lấy từ kết
  quả lookup (không lấy trực tiếp input); lookup fail → lỗi cứng.

### Version routing

`k8s_version >= 1.33` → gọi `/hpc/v2/...`, kèm `isV2: true`. Version thấp hơn
→ `/hpc/...`, không gửi `isV2`.

Danh sách version hợp lệ (`allowedK8sVersions`): `1.36.2`, `1.35.6`, `1.34.6`,
`1.33.12`, `1.32.5`, `1.31.4`, `1.30.8`, `1.29.8`.

### Cluster name

`checkClusterName` kiểm tra tên đã kết thúc bằng dấu `-` + đúng 8 ký tự
alphanumeric thường chưa. Nếu chưa, `GenerateRandomSuffix` tự thêm suffix
trước khi gửi request.

## Field chỉ dùng ở state / endpoint khác, không nằm trong body create

| Terraform field | Vì sao không có trong body create |
|---|---|
| `id` | tính từ response sau khi create (`{cluster_name}-{cluster_id}`) |
| `is_running` | gửi ở endpoint hibernate/wakeup riêng, không phải create |
| `hibernation_schedules` | gửi ở endpoint schedules riêng |
| `vpc_id` | chỉ nằm trên URL path, không nằm trong body |

## Việc cần làm tiếp

1. Xác nhận với backend: `scale_min`, `scale_max`, `worker_type` trong doc
   Swagger của mgpu có thật hay bị lẫn từ schema mfke.
2. Xác nhận tên key JSON thật của block `software` (Swagger mẫu không có).
3. Xác nhận `ram`/`cpu`/`gpu_amount`/`kubernetes` trong `pools[]` — request
   field hay chỉ xuất hiện ở response.
4. Xác nhận `maxGracefulTerminationSeconds`/`maxNodeProvisionTime` trong
   `cluster_autoscaler` có cần expose lên Terraform không.
5. Xác nhận `lbInternalNetwork`, `currentNetworking` có thật sự cần gửi khi
   create hay chỉ dùng cho update/edit-worker.
