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

## Đối chiếu với request thật (OSP, v2, tạo cluster thành công)

Khác với body Swagger mẫu bên dưới (dữ liệu giả kiểu Lorem ipsum), đoạn này
dựa trên 1 request `POST .../hpc/v2/create-cluster` thật, response `200` với
`"Cluster creation is started successfully."`. Đáng tin hơn Swagger doc.

```json
{
  "cluster_name": "mycluster-qfr72jbb",
  "network_id": "c009571e-72c2-4eb2-8dc1-c125d1e76901",
  "vm_subnet": "10.102.17.0/24",
  "osp_network_id": "3f248d56-28ae-40df-98c6-3f9d1b8ea5e1",
  "k8s_version": "1.33.12",
  "isV2": true,
  "os_version": { "...": "catalog rất lớn, mọi zone/driver — không phải input" },
  "purpose": "public",
  "lbInternalNetwork": {
    "label4sending": "phuonght71-9i9y3r2h",
    "label": "phuonght71",
    "value": "f1574c39-1baa-45e7-92cc-34cca8663dd5",
    "cidr": "172.30.250.1/24",
    "networkType": "NAT_ROUTED"
  },
  "pools": [
    {
      "worker_pool_id": "worker-316qy5vg",
      "container_runtime": "containerd",
      "hpc_flavor_id": "3eb8a0c2-f810-11ef-838f-005056b46212",
      "hpc_flavor_name": "Metal Cloud GPU H200",
      "hpc_number_server": 1,
      "auto_scale": false,
      "kv": [],
      "isCreate": true,
      "isScale": false,
      "isOthers": false,
      "gpuType": "H200",
      "migProfile": "all-disabled",
      "maxClient": 0,
      "driverInstallationType": "PRE_INSTALL",
      "gpuDriverVersion": "550.90.07",
      "gpuTemplateVersion": "22.04.20250325-ncp-550-prod"
    }
  ],
  "pod_network": "100.96.0.0",
  "pod_prefix": "11",
  "service_network": "100.64.0.0",
  "service_prefix": "13",
  "k8s_max_pod": 110,
  "network_node_prefix": 23,
  "network_type": "calico",
  "ssh_public_key": "ssh-rsa ...",
  "ssh_name": "terraform-test-ssh-key",
  "ssh_id": "314fcabc-574d-4776-b3c0-7b24653cd0a0",
  "auto_upgrade_expression": [],
  "auto_upgrade_timezone": "",
  "type_create": "create",
  "hps": null,
  "clusterEndpointAccess": { "type": "public", "allowCidr": [] }
}
```

Điểm khác biệt so với code (đã xử lý / còn treo):

| Field | Trạng thái | Xử lý |
|---|---|---|
| `os_version` | **Đã bỏ** | Field từng tồn tại trong `managedGpuClusterJson` (`OsVersion interface{}`) nhưng chưa từng được gán giá trị ở đâu — request thật cho thấy nó là 1 catalog cực lớn (mọi zone × mọi driver), không phải input hợp lý cho Terraform. Đã xoá field khỏi struct theo quyết định của bạn. |
| `vm_subnet` + `osp_network_id` | **Đã có map** | Không có field Terraform riêng — `network_id` vẫn là field duy nhất người dùng khai (có thể lấy từ data source `fptcloud_hpc_subnet` mới, xem [HPC subnet](#hpc-subnet-vm_subnet--osp_network_id)). Trên OSP, `MapTerraformToJson` tự gọi lại `GET /v2/vmware/vpc/{vpcId}/hpc/subnets`, tìm entry có `id == network_id`, lấy `subnet_cidr` → `vm_subnet` và `osp_network_id` để gửi kèm. Không phải input trực tiếp, không phải omit — là 1 lookup nội bộ theo `network_id`. |
| `lbInternalNetwork` | **Đã có map** | Không có field Terraform riêng — dựng từ subnet backing `internal_subnet_lb` (trên OSP, giá trị đó là subnet ID). `MapTerraformToJson` gọi `findNetworkSubnetById` (đã có sẵn cho `config-internal-subnet-lb`), lấy `GET /v1/vmware/vpc/{vpcId}/network/subnets`, rồi map: `value`←`id`, `label`←`description`, `label4sending`←`name`, `cidr`←`defaultGateway`+`/`+`subnetPrefixLength`, `networkType`←`networkType` (`lbInternalNetworkFromSubnet` trong `types.go`). Bỏ qua khi `internal_subnet_lb` rỗng, hoặc trên platform khác OSP (ở đó `internal_subnet_lb` là CIDR chứ không phải subnet ID, không tra được theo cách này). |
| `pools[].gpuType` | **Đã có map (2 chiều)** | Field Terraform `pools[].gpu_type` (Optional string, validate theo `A100`/`A30`/`H100`/`H200`). Gửi lên qua `managedGpuClusterPoolJson.GpuType` (json tag `gpuType`); đọc lại từ API GPU software — xem [GPU software](#gpu-software-api-thu-2-khi-create). |
| `pools[].migProfile` | **Đã có map (2 chiều)** | Nay nằm trong block `pools[].mig { strategy, profile }` (không còn là field rời `mig_profile`). Gửi lên qua `MigProfile` (json tag `migProfile`) + `WorkerMigStrategy` (json tag `workerMigStrategy`); đọc lại từ API GPU software. |
| `pools[].workerMigStrategy` | **Đã có map** | `pools[].mig.strategy` — `NONE`/`SINGLE`/`MIXED`. |
| `pools[].sharingClient` | **Đã có map** | `pools[].gpu_sharing.client_type` — `NONE`/`MPS`/`TIMESLICING`. |
| `pools[].maxClient` | **Đã có map** | Chuyển từ field rời `max_client` vào `pools[].gpu_sharing.max_client` (chỉ có nghĩa cùng `client_type`). |
| `hps` | **Đã có map (tạm)** | Không có field Terraform — `managedGpuClusterJson.Hps interface{}` (không `omitempty`) luôn serialize thành `"hps": null`, khớp giá trị null gửi trong request thật. Ý nghĩa của field vẫn chưa xác nhận với backend; giữ `null` cố định cho tới khi có xác nhận. |

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
| 8 | `default_storage_profile` | **Đã bỏ** | Không còn field Terraform — field Terraform `default_storage_profile` đã bị xoá khỏi schema theo yêu cầu. Provider không gửi field này lên nữa (bỏ hẳn khỏi `managedGpuClusterJson`), không phải omitempty do giá trị rỗng. |
| 9 | `edge_gateway_id` | **Đã có map** | `edge_gateway_id` (bị xoá rỗng nếu platform = OSP) |
| 10 | `edge_gateway_name` | **Đã có map** | `edge_gateway_name` — nhưng do **provider tự tính** (lookup theo `edge_gateway_id`), không lấy trực tiếp từ input user |
| 11 | `internal_subnet_lb` | **Đã có map** | `internal_subnet_lb` (Required, không ForceNew — cập nhật qua API riêng, không tạo lại cluster) |
| 12 | `k8s_max_pod` | **Đã có map** | `k8s_max_pod` |
| 13 | `k8s_version` | **Đã có map** | `k8s_version` |
| 14 | `lbInternalNetwork` | **Chưa map** | — không có field Terraform, provider không gửi (field reserved) |
| 15 | `loadBalancerType` | **Đã bỏ** | Không còn field Terraform — field Terraform `load_balancer_type` đã bị xoá khỏi schema theo yêu cầu. `managedGpuClusterJson.LoadBalancerType` vẫn tồn tại (do `omitempty`) nhưng luôn nhận giá trị rỗng, nên không được gửi lên. |
| 16 | `network_id` | **Đã có map** | `network_id` |
| 17 | `network_node_prefix` | **Đã có map** | `network_node_prefix` |
| 18 | `network_overlay` | **Đã bỏ** | Không còn field Terraform — field Terraform `network_overlay` (cả validate theo `network_type`: `cilium` → rỗng, `calico` → `Always`/`CrossSubnet`) đã bị xoá theo yêu cầu. `managedGpuClusterJson.NetworkOverlay` vẫn tồn tại (do `omitempty`) nhưng luôn nhận giá trị rỗng, nên không được gửi lên. |
| 19 | `network_type` | **Đã có map** | `network_type` |
| 20 | `pod_network` | **Đã có map** | `pod_network` |
| 21 | `pod_prefix` | **Đã có map** | `pod_prefix` |
| 22 | `pools` | **Đã có map** | `pools` (block) — xem [Pool](#pool) |
| 23 | `purpose` | **Đã có map** | `purpose` |
| 24 | `secret_binding_name` | **Đã bỏ** | Không còn field Terraform — field Terraform `secret_binding_name` đã bị xoá khỏi schema theo yêu cầu. `managedGpuClusterJson.SecretBindingName` vẫn tồn tại (do `omitempty`) nhưng luôn nhận giá trị rỗng, nên không được gửi lên. |
| 25 | `service_network` | **Đã có map** | `service_network` |
| 26 | `service_prefix` | **Đã có map** | `service_prefix` |
| 27 | `ssh_id` | **Đã có map** | `ssh_key_id` (Required) — `ssh_name`/`ssh_public_key` tự tra, xem [SSH key](#ssh-key) |
| 28 | `ssh_name` | **Đã có map** | không có field Terraform riêng — tự tra từ `ssh_key_id` |
| 29 | `ssh_public_key` | **Đã có map** | không có field Terraform riêng — tự tra từ `ssh_key_id` |

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
| 4 | `vGpuId` | **Đã bỏ** | Không còn field Terraform — field Terraform `vgpu_id` đã bị xoá khỏi schema theo yêu cầu. `managedGpuClusterPoolJson.VGpuID` vẫn tồn tại trong wire format nhưng luôn nhận giá trị rỗng (không `omitempty`, vẫn gửi `"vGpuId": ""`); kéo theo `isDisplayGPU` cũng luôn `false` (xem dòng 19 bên dưới). |
| 5 | `isCreate` | **Provider tự set** | — `true` khi `worker_pool_id = null` (pool mới) |
| 6 | `isScale` | **Provider tự set** | — luôn `false` khi create |
| 7 | `isOthers` | **Provider tự set** | — luôn `false` khi create |
| 8 | `deltaQuotaScale` | **Provider tự set** | — luôn gửi `0`, không có field Terraform |
| 9 | `isEnableAutoRepair` | **Đã bỏ** | Không còn field Terraform — field Terraform `is_enable_auto_repair` đã bị xoá khỏi schema theo yêu cầu. `managedGpuClusterPoolJson.IsEnableAutoRepair` vẫn tồn tại trong wire format nhưng luôn gửi `false` (zero value, field bool không có `omitempty`). |
| 10 | `driverInstallationType` | **Đã có map** | `gpu_driver.installation_type` (block, không còn là field rời) — validate theo API `gpu-drivers`, xem [GPU driver](#gpu-driver) |
| 11 | `auto_scale` | **Provider tự set** | — luôn `false` (pool bare-metal là số server cố định, không autoscale theo pool) |
| 12 | `container_runtime` | **Đã có map** | `container_runtime` |
| 13 | `gpuDriverVersion` | **Đã có map** | `gpu_driver.version` (block, không còn là field rời) — validate theo API `gpu-drivers`, xem [GPU driver](#gpu-driver) |
| 14 | `gpuSharingClient` | **Đã bỏ** | Không còn field Terraform — field Terraform `gpu_sharing_client` (và validate đi kèm: chỉ nhận `""`/`timeSlicing`, cùng ràng buộc `max_client` 2-48 khi `= timeSlicing`) đã bị xoá theo yêu cầu. `managedGpuClusterPoolJson.GpuSharingClient` vẫn tồn tại trong wire format nhưng luôn gửi `""`. `max_client` vẫn còn field Terraform (không nằm trong danh sách xoá) nhưng không còn validate range vì ràng buộc đó phụ thuộc `gpu_sharing_client`. |
| 15 | `gpuTemplateVersion` | **Đã bỏ** | Không còn field Terraform — field Terraform `gpu_template_version` đã bị xoá khỏi schema theo yêu cầu. `managedGpuClusterPoolJson.GpuTemplateVersion` vẫn tồn tại trong wire format nhưng luôn gửi `""`. |
| 16 | `hpc_flavor_id` | **Đã có map** | `hpc_flavor_id` |
| 17 | `hpc_flavor_name` | **Đã có map** | `hpc_flavor_name` |
| 18 | `hpc_number_server` | **Đã có map** | `hpc_number_server` |
| 19 | `isDisplayGPU` | **Provider tự set** | — trước đây `true` khi `vgpu_id != ""`; từ khi `vgpu_id` bị xoá khỏi Terraform, `vGpuId` luôn rỗng nên `isDisplayGPU` luôn `false` |
| 20 | `kubernetes` | **Chưa map** | — không có field Terraform, provider không gửi (reserved) |
| 21 | `kv` | **Đã có map** | `kv` (Set `{name, value}`) — format khác hẳn `"string"` trong doc, xem [KV](#kv) |
| 22 | `maxClient` | **Đã có map** | `max_client` |
| 23 | `network_id` | **Đã có map** | `network_id` |
| 24 | `network_name` | **Đã có map** | `network_name` |
| 25 | `scale_max` | **Không áp dụng cho mgpu** | Chốt: đây là field của `mfke` (worker pool autoscale theo range), doc Swagger dùng chung schema với mfke. mgpu dùng `hpc_number_server` cố định. Không có field Terraform, không gửi, không cần xử lý thêm. |
| 26 | `scale_min` | **Không áp dụng cho mgpu** | như trên |
| 27 | `storage_profile` | **Đã bỏ** | Không còn field Terraform — field Terraform `storage_profile` (trước đây Required trong pool) đã bị xoá khỏi schema theo yêu cầu. `managedGpuClusterPoolJson.StorageProfile` vẫn tồn tại trong wire format nhưng luôn gửi `""` (chưa có default nào khác thay thế). |
| 28 | `tags` | **Đã có map** | `tags` (list Terraform → string nối bằng `\n` khi gửi) |
| 29 | `taints` | **Đã có map** | `taints` (Set `{key, value, effect}`) — format khác `"string"`, xem [Taints](#taints) |
| 30 | `worker_base` | **Đã có map** | `worker_base` |
| 31 | `worker_disk_size` | **Đã có map** | `worker_disk_size` |
| 32 | `worker_pool_id` | **Đã có map** | `name` — `null` khi tạo pool mới |
| 33 | `worker_type` | **Không áp dụng cho mgpu** | Chốt: mgpu dùng `hpc_flavor_id` thay cho `worker_type` của mfke. Không có field Terraform, không gửi, không cần xử lý thêm. |

### Nghi vấn cần xác nhận lại với backend/API doc

1. **`software` block** hoàn toàn không xuất hiện trong body Swagger mẫu, dù
   Terraform coi đây là field khi tạo cluster GPU (chọn GPU Operator /
   Network Operator / Slurm Operator / vGPU Scheduler, hiện Optional). Cần
   xác nhận lại tên key JSON thật (`software`? nằm lồng trong `pools`? tên
   khác?).
2. **`ram` / `cpu` / `gpu_amount` / `kubernetes`** trong `pools[]` — hiện tại
   provider không gửi các field này (giả định backend tự suy ra từ
   `hpc_flavor_id`). Nếu Swagger liệt kê chúng là field request hợp lệ, cần
   xác nhận provider có cần gửi lên không, hoặc đây là field chỉ xuất hiện ở
   response.

`scale_min`/`scale_max`/`worker_type` đã **chốt xong**: không áp dụng cho mgpu
(field của mfke, doc Swagger dùng chung schema) — không còn là nghi vấn, xem
bảng Pool ở trên.

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

Đã bỏ: trước đây provider tự động thêm 2 key hệ thống
(`nvidia.com/mig.config`, `worker.fptcloud/type: gpu`) cho pool GPU
(`vgpu_id != ""`) khi người dùng chưa khai. Logic này đã bị xoá khỏi
`MapTerraformToJson` — `kv` giờ chỉ gửi đúng những gì người dùng khai trong
`.tf`, không tự thêm gì nữa.

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

### GPU driver

`driver_installation_type` and `gpu_driver_version` are grouped into one
`gpu_driver` block per pool:

```hcl
gpu_driver = {
  installation_type = "MANAGED"       # MANAGED | PRE_INSTALL | USER_INSTALL
  version            = "535.247.01"
}
```

```json
{ "driverInstallationType": "MANAGED", "gpuDriverVersion": "535.247.01" }
```

The block itself is optional — a pool that never sets `gpu_driver` sends
nothing for either wire field (no defaulting, `gpu_driver` stays `null`
through plan/apply). When set, both fields are validated live against
`GET /v2/xplat/fke-gpu/common/vpc/{vpcId}/gpu-drivers?driver_type=...&zone=...&kubernetes_version=...`
(`validateGpuDriver` in `validations.go`, `fetchGpuDrivers` in `utils.go`).

```json
{
  "driver_type": "MANAGED",
  "driver_list": [
    { "label": "535.247.01 - CUDA version: 12.2", "value": "535.247.01", "imageID": "22.04.20250715-ncp-standard-prod" }
  ]
}
```

| `gpu_driver.installation_type` | `gpu_driver.version` |
|---|---|
| `MANAGED` | must be one of the `driver_list[].value` returned for `driver_type=MANAGED` |
| `PRE_INSTALL` | must be one of the `driver_list[].value` returned for `driver_type=PRE_INSTALL` |
| `USER_INSTALL` | must be left empty — the catalog returns a single entry with `value: ""` (no version to pick; the user installs their own driver) |

The `zone` query param is derived from the client's region
(`gpuDriverZoneForRegion` in `utils.go`), not a Terraform field:

| Client region | `zone` |
|---|---|
| `JP/JCSI2` | `jcncp01` |
| everything else | **not yet defined** — `validateGpuDriver` fails with an explicit error until the mapping is confirmed |

Reading a cluster back (`InternalRead`, the data source) rebuilds `gpu_driver`
from the worker's `machine.image.{driverInstallationType,gpuDriverVersion}`;
when the API reports both empty, the block is read back as `null` rather than
`{ installation_type = "", version = "" }`, so a config that never set
`gpu_driver` does not see a permanent diff (`gpuDriverObjectValue` in
`utils.go`).

### GPU software (API thứ 2 khi create)

Theo SRS mục 1 và 2.3: **một MGPU cluster là 2 tài nguyên backend riêng biệt**,
phải thao tác tuần tự. Tạo xong `create-cluster` mới chỉ là một nửa — GPU
operator/driver/sharing/MIG nằm ở backend thứ 2 và phải cài bằng 1 request
riêng, nếu không cluster tạo ra không có GPU software gì cả.

| # | Hệ thống | Endpoint |
|---|---|---|
| 1 | m-fke cluster | `POST .../fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/create-cluster` |
| 2 | GPU software | `POST .../xplat/fke-gpu/common/vpc/{vpcId}/gpu-clusters/{clusterName}` |

Lưu ý URL API 2 **không có** segment `/m-fke`, và prefix `/v1` hay `/v2` bám
theo đúng family mà cluster được tạo (`gpuSoftwareApiVersion` trong
`commons/api_path.go`).

**`tenant_id`** — API 2 cần `tenant_id` trong body (và trong query khi
GET/DELETE). Provider lấy qua `vpc.GetTenant()` (`GET /v2/tenant/{tenant_name}`,
service đã có sẵn trong `fptcloud/vpc`), dùng `tenant_name` từ provider config.
Đã verify bằng call thật: tenant `NCP-PRODUCT-DEV-JP` trả về
`afb02abc-1e90-4bf6-a8a1-283c69e37032`, trùng đúng `tenant_id` trong body
request thật của console (và trùng header `OrgId`). Không cần duyệt
tenant/region/VPC, không cần field Terraform mới.

**`region`** — body API 2 nhận đúng giá trị mà header `fpt-region` dùng
(`JP/JCSI2` → `tokyo-jp`). `gpuSoftwareRegion` trong `gpu_software.go` giữ
chung 1 mapping với `sendRequestWithHeader` để 2 chỗ không lệch nhau.

Body gửi lên (`gpuSoftwareRequest`, `types.go`):

| JSON | Nguồn |
|---|---|
| `isV2` | đồng bộ với API 1 (`requiresV2API(k8s_version)`) |
| `name` | tên cluster thật (đã có hậu tố random) |
| `infra_type` | platform, upper-case (`OSP`) |
| `region` | `gpuSoftwareRegion(client.Region)` |
| `tenant_id` | `vpc.GetTenant().Id` |
| `kubernetes_version` | `k8s_version` |
| `operator_version` | `{ <software.software_type>: <software.software_version> }` |
| `cluster_type` | hardcode `"BM"` |
| `mig_strategy` | `software.cluster_mig_strategy`, upper-case (`single` → `SINGLE`) |
| `worker_groups[]` | 1-1 với `pools[]`, khớp theo tên |
| `status` | hardcode `"NOTREADY"` |

`worker_groups[]` (`gpuSoftwareWorkerJson`):

| JSON | Nguồn Terraform |
|---|---|
| `name` | `pools[].name` |
| `mig_mode` | `pools[].mig.strategy` (`NONE` nếu không khai) |
| `mig_profile` | `pools[].mig.profile` (`NONE` nếu không khai) |
| `sharing_client_type` | `pools[].gpu_sharing.client_type` (`NONE` nếu không khai) |
| `max_client` | `pools[].gpu_sharing.max_client` |
| `gpu_scheduler` | luôn `NONE` — không có field Terraform, console cũng chỉ gửi `NONE` |
| `driver_type` | `pools[].gpu_driver.installation_type` |
| `driver_version` | `pools[].gpu_driver.version` |
| `enable_operand` | hardcode `true` |
| `gpu_type` | `pools[].gpu_type` |

Riêng pool có `driver_type = USER_INSTALL`: người dùng tự cài driver nên
platform không quản MIG/sharing — mọi field đó bị ép về giá trị tắt
(`mig_mode`/`mig_profile`/`gpu_scheduler` = `NONE`, `sharing_client_type` =
`all-disabled`, `max_client` = 0, `driver_version` = `""`), và validate cấm
khai `gpu_sharing`/`mig` cho pool đó.

**Đọc lại (Read/refresh)** — `GET .../gpu-clusters/{clusterName}?tenant_id=...&region=...`
là nguồn duy nhất cho `gpu_type`, `gpu_sharing`, `mig` (get-shoot-specific
không trả field nào trong số đó). `fetchGpuSoftwareWorkers` gọi nó trong
`InternalRead` và trong data source, key theo tên pool. Lỗi ở bước này
**không** làm hỏng cả lần read: GPU software là backend độc lập và một cluster
cài lỗi thì đơn giản là không có bản ghi ở đó, nên 3 block chỉ để `null`.

Response GET dùng `deploy_operand` cho đúng thứ request gọi là
`enable_operand`, và `max_client` trả về dạng số (dù request gửi chuỗi) —
`gpuSoftwareWorkerRead` phản ánh đúng shape đó.

Giá trị `NONE`/`all-disabled` khi đọc lại được `normalizeGpuNone` map ngược về
rỗng, để pool chưa từng khai sharing/MIG đọc ra block `null` chứ không phải
object toàn `NONE` — nếu không sẽ diff vĩnh viễn.

**Xoá (Delete)** — theo SRS 5.2, `DELETE .../gpu-clusters/{clusterName}?tenant_id=...`
chỉ chạy **sau khi** xoá cluster thành công.

**Tính không nguyên tử** — 2 lời gọi tới 2 hệ thống, backend không có rollback
(SRS 11.1). `Create` xử lý theo đúng khuyến nghị đó: nếu API 2 lỗi thì vẫn ghi
cluster id vào state **trước** rồi mới trả lỗi, để Terraform không mất dấu
cluster đã tạo; thông báo lỗi nói rõ chạy `terraform apply` lại để cài lại GPU
software.

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

### HPC subnet (vm_subnet / osp_network_id)

Data source riêng: `fptcloud_hpc_subnet` (`datasource_hpc_subnet.go`), đọc
`GET /v2/vmware/vpc/{vpcId}/hpc/subnets`. Cùng shape với `fptcloud_subnet`:
không có `filter` → trả về toàn bộ subnet trong VPC; có `filter` (key `id`
hoặc `name`) → lọc list đó lại. Luôn trả về `subnets` (list), kể cả khi filter
chỉ còn 1 phần tử:

```hcl
data "fptcloud_hpc_subnet" "hpc_subnet" {
  vpc_id = data.fptcloud_vpc.vpc.id
  filter {
    key    = "name"
    values = ["check-k8s"]
  }
}
```

Mỗi phần tử của `subnets[]`: `id` (khớp `network_id` của cluster), `name`,
`subnet_cidr` (chính là `vm_subnet` request cần), `osp_network_id`, `gateway`,
`status`, `description`, `network_acl_id`.

Cách dùng: index thẳng vào `subnets[0]` — không cần `local`, không có field
`vm_subnet`/`osp_network_id` riêng trên resource:

```hcl
resource "fptcloud_managed_gpu_cluster" "example" {
  network_id          = data.fptcloud_hpc_subnet.hpc_subnet.subnets[0].id
  internal_subnet_lb  = data.fptcloud_hpc_subnet.hpc_subnet.subnets[0].id
  ...
}
```

Bên trong `MapTerraformToJson`, khi platform là OSP, provider tự gọi lại
`GET /v2/vmware/vpc/{vpcId}/hpc/subnets`, tìm entry có `id == network_id`, và
lấy `subnet_cidr`/`osp_network_id` từ đó để gửi kèm `vm_subnet`/
`osp_network_id` trong body create (`fetchHpcSubnetById` trong `utils.go`).
Không tìm thấy → lỗi rõ ràng, trỏ người dùng về data source này.
Trên platform khác OSP, không lookup, không gửi 2 field này.

**Bug đã sửa:** `network_id` cũng phải resolve từ chính catalog HPC subnet
này trên OSP, **không** phải từ catalog subnet thường (`fptcloud_subnet`,
`GET /v2/vpc/{vpcId}/networks`) — 2 catalog không dùng chung ID. Trước đây
`MapTerraformToJson` luôn gọi `getNetworkByIdOrName` (tra `ListSubnet`, catalog
subnet thường) bất kể platform, nên `network_id` lấy từ
`fptcloud_hpc_subnet` không bao giờ khớp và luôn lỗi `no such network found`
khi apply thật (dù `terraform plan`/`validate` không phát hiện ra, vì lookup
chỉ chạy lúc apply). Giờ trên OSP, `network_id`/`network_name` cũng lấy từ
`fetchHpcSubnetById` (cùng 1 lần gọi lấy cả 4 giá trị: `network_id`,
`network_name`, `vm_subnet`, `osp_network_id`); trên platform khác OSP vẫn
dùng `getNetworkByIdOrName` như cũ.

### SSH key

`ssh_key_id` (Required, thay cho `ssh_id` cũ) là field Terraform duy nhất —
không còn `ssh_name`/`ssh_public_key` trên resource. Dùng data source có sẵn
`fptcloud_ssh_key` để lấy id:

```hcl
data "fptcloud_ssh_key" "ssh_key" {
  name = "terraform-test-ssh-key"
}

resource "fptcloud_managed_gpu_cluster" "example" {
  ssh_key_id = data.fptcloud_ssh_key.ssh_key.id
  ...
}
```

`MapTerraformToJson` gọi `sshClient.FindSSHKey(ssh_key_id)` (service có sẵn
trong `fptcloud/ssh`, exact-match theo ID) để lấy `name`/`public_key`, rồi gửi
cả 3 field `ssh_id`/`ssh_name`/`ssh_public_key` trong body — giống hệt cách
`vm_subnet`/`osp_network_id` được tự động tra từ `network_id`. Không tìm thấy
→ lỗi rõ ràng.

### lbInternalNetwork

Không có field Terraform riêng — dựng ngầm từ subnet mà `internal_subnet_lb`
trỏ tới, chỉ trên OSP (nơi `internal_subnet_lb` là subnet ID; trên platform
khác nó là CIDR nên không tra được theo cách này). `internal_subnet_lb` là
**Required** (không phải Optional), nhưng **không ForceNew** — thay đổi giá
trị này chỉ gọi `updateInternalSubnetLb` (API riêng), không tạo lại cluster.

```hcl
resource "fptcloud_managed_gpu_cluster" "example" {
  internal_subnet_lb = "f1574c39-1baa-45e7-92cc-34cca8663dd5" # subnet ID (OSP)
  ...
}
```

`MapTerraformToJson` gọi `findNetworkSubnetById` (hàm đã có sẵn, dùng chung
với `config-internal-subnet-lb`) trên `GET /v1/vmware/vpc/{vpcId}/network/subnets`,
rồi map (`lbInternalNetworkFromSubnet` trong `types.go`):

| `lbInternalNetwork` field | Nguồn (từ response `network/subnets`) |
|---|---|
| `value` | `id` |
| `label` | `description` |
| `label4sending` | `name` |
| `cidr` | `defaultGateway` + `"/"` + `subnetPrefixLength` |
| `networkType` | `networkType` |

Ví dụ đối chiếu với request thật đã xác nhận:

```json
// GET network/subnets trả về (rút gọn 1 entry):
{ "id": "f1574c39-...", "name": "phuonght71-9i9y3r2h", "description": "phuonght71",
  "defaultGateway": "172.30.250.1", "subnetPrefixLength": 24, "networkType": "NAT_ROUTED" }

// → lbInternalNetwork gửi trong create-cluster:
{ "value": "f1574c39-...", "label": "phuonght71", "label4sending": "phuonght71-9i9y3r2h",
  "cidr": "172.30.250.1/24", "networkType": "NAT_ROUTED" }
```

Bỏ qua (không gửi `lbInternalNetwork`) trên platform khác OSP.

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

So với body request thật `mycluster-qfr72jbb` (xem [Đối chiếu với request
thật](#đối-chiếu-với-request-thật-osp-v2-tạo-cluster-thành-công) ở đầu file),
`pools[].gpuType`/`pools[].migProfile` (từng là khoảng trống duy nhất khi so
trực tiếp với body đó) **đã map được chiều gửi lên** — xem [GPU type / MIG
profile](#gpu-type--mig-profile-pools-gpu_type--pools-mig_profile). Chiều đọc
lại (Read/refresh) vẫn còn thiếu — xem mục cuối danh sách bên dưới. Các mục
còn lại đến từ nguồn khác (doc Swagger, suy luận logic), không phải phát hiện
mới từ body request thật này.

1. Xác nhận tên key JSON thật của block `software` (Swagger mẫu không có).
2. Xác nhận `ram`/`cpu`/`gpu_amount`/`kubernetes` trong `pools[]` — request
   field hay chỉ xuất hiện ở response.
3. Xác nhận `maxGracefulTerminationSeconds`/`maxNodeProvisionTime` trong
   `cluster_autoscaler` có cần expose lên Terraform không.
4. Xác nhận `currentNetworking` có thật sự cần gửi khi create hay chỉ dùng
   cho update/edit-worker.
5. **`hps`** xác nhận ý nghĩa thật với backend — hiện tạm gửi `null` cố định,
   đúng như request thật quan sát được, nhưng chưa biết nó dùng để làm gì.
6. **Đồng bộ GPU software khi Update pool** (SRS 4.1, phần "Đồng bộ GPU
   software"). Hiện `Create` và `Delete` đã gọi đủ 2 backend, nhưng `Update`
   thì **chưa**: sau `configure-worker-cluster` cần `PUT .../gpu-clusters/
   {clusterName}` theo kiểu read-modify-write (GET trước, thay
   `worker_groups`, PUT lại toàn bộ — không được tự dựng body từ đầu). Chưa
   làm vì còn vướng vấn đề `gpu_driver` immutable ở mục 4 phần "Điều cần hỏi
   anh Kiên" (cơ chế xoá/tạo lại pool) — làm xong cái đó rồi mới nối phần này
   cho khớp.

## Điều cần hỏi anh Kiên

1. **HPC bare-metal flavor catalog (`hpc_flavor_id`).** Chưa xác định được
   đúng API để tra danh sách flavor bare-metal (dùng cho `pools[].hpc_flavor_id`/
   `hpc_flavor_name`). Đã thử `GET /v2/vpc/{vpcId}/flavors` (API mà data
   source `fptcloud_flavor` dùng) — gọi thật bằng token trong `main.tf`, xác
   nhận nó **không phải** đúng catalog: có 8 flavor `type: "GPU_SIZE"` trong
   đó (vd `id: 5a25eeab-db24-4675-9e75-0c2e83772f44`, `name:
   B_24C192G_H200_141G`), nhưng ID/tên khác hoàn toàn format `hpc_flavor_id`
   thật từng thấy trong 1 request create-cluster thành công (`id:
   3eb8a0c2-f810-11ef-838f-005056b46212`, `name: "Metal Cloud GPU H200"`).
   Đây rõ ràng là 2 catalog khác nhau (giống việc `hpc/subnets` khác
   `/v2/vpc/{vpcId}/networks`). Cần anh Kiên cho biết đúng API list HPC
   bare-metal flavor để viết data source `fptcloud_hpc_flavor` (tương tự
   `fptcloud_hpc_subnet`) — hiện `main.tf` phải để `hpc_flavor_id`/
   `hpc_flavor_name` trống, người dùng tự điền tay UUID.

2. **`os_version` trong body create-cluster.** SRS mục 2.2 đánh dấu field này
   là **bắt buộc**, và request thật của console có gửi (một catalog rất lớn:
   mọi zone × mọi driver). Provider hiện **không gửi** field này, nhưng
   create-cluster vẫn trả `200` và cluster vẫn tạo thành công qua nhiều lần
   test thật — nên nhiều khả năng backend tự điền khi vắng mặt. Cần xác nhận:
   có thật sự bắt buộc không, hay chỉ là UI gửi kèm cho đủ? Nếu bắt buộc thì
   provider cần lấy catalog đó từ đâu (API `get_k8s_versions` trả kèm?).

3. **Zone cho API `gpu-drivers` (`gpuDriverZoneForRegion` trong `utils.go`).**
   API `GET /v2/xplat/fke-gpu/common/vpc/{vpcId}/gpu-drivers` (dùng để
   validate `gpu_driver.installation_type`/`gpu_driver.version`) cần query
   param `zone` — hiện code chỉ hardcode được đúng 1 mapping xác nhận qua
   test thật: region `JP/JCSI2 → zone jcncp01`. Các region còn lại
   (`VN/HAN`, `VN/SGN`, `VN/HAN2`, `VN/SGN2`) **chưa có zone tương ứng** và
   sẽ lỗi cứng ("gpu-drivers zone mapping is not yet defined for region")
   nếu người dùng ở các region đó khai `gpu_driver` cho pool. Cần anh Kiên
   cho biết zone đúng của từng region còn lại (hoặc 1 API tra `region → zone`
   thay vì hardcode) để bổ sung vào `gpuDriverZoneForRegion`.

4. **`gpu_driver` là immutable — cần cơ chế xoá/tạo lại pool thay vì update.**
   Xác nhận qua test thật trên cluster `mgpu-cluster-01-z0tdfcyl` (2 lần gọi
   `configure-worker-cluster`/`ConfigWorker` để đổi `driverInstallationType`/
   `gpuDriverVersion`/`gpuTemplateVersion` trên 1 pool đã tồn tại): API nhận
   request `200` nhưng **không áp dụng** driver info (đọc lại sau apply vẫn là
   giá trị cũ), và có side effect ngoài ý muốn — **tự sinh SSH key mới** cho
   pool đó, vì payload endpoint này không có field `sshKey` để giữ nguyên key
   cũ. Anh Kiên xác nhận (kèm ảnh UI console FPT Cloud: form "Worker Group",
   mục GPU Driver chỉ xuất hiện ở form **tạo mới**, không có ở form **sửa**)
   đây là hành vi thiết kế đúng: `gpu_driver` chỉ set được lúc tạo pool, không
   sửa được sau đó.

   Nguyện vọng: khi `gpu_driver` của 1 pool đổi trong `.tf`, muốn Terraform
   **xoá pool đó và tạo pool khác thay thế trong cùng cluster** (không phá huỷ
   toàn bộ cluster) — vì "1 cluster có thể có nhiều worker pool". Đã tìm trong
   code hiện có (`utils.go`): cơ chế "tạo pool mới" đã tồn tại (dòng ~748,
   `remapPools` — tên pool rỗng/`worker-new`/`WorkerPoolID` null hoặc unknown
   thì `worker_pool_id` gửi `null`, backend tạo pool mới), nhưng **chưa rõ**
   `configure-worker-cluster` có hỗ trợ xoá 1 pool khỏi cluster hay không (vd.
   gửi `pools[]` không còn liệt kê pool đó nữa — có bị backend hiểu là "xoá
   pool" không, hay chỉ đơn giản bị bỏ qua/giữ nguyên?), và có API xoá 1 worker
   pool riêng biệt không. Cần anh Kiên xác nhận cơ chế đúng trước khi code (để
   tránh lặp lại sai lầm như các API khác đã bị đoán nhầm catalog trước đây)
   rồi mới cài đặt: `DiffPool` phát hiện `gpu_driver` đổi → coi đó là
   xoá-pool-cũ + tạo-pool-mới thay vì gọi update in-place.

## Kết quả rà soát field thừa/thiếu (đối chiếu trực tiếp code, không dựa tài liệu cũ)

Đã rà từng field trong `managedGpuCluster`/`managedGpuClusterPool` (input
Terraform) và `managedGpuClusterJson`/`managedGpuClusterPoolJson` (JSON gửi
API) theo cả 2 chiều:

- **Field Terraform "thừa"** (khai được trong `.tf` nhưng không bao giờ đi vào
  body request) — **không có**. `id`, `vpc_id`, `is_running`,
  `hibernation_schedules` trông giống thừa nhưng đều hợp lệ: `vpc_id` truyền
  qua tham số riêng (không đọc từ struct), còn 3 field kia xử lý ở endpoint
  khác (xem [Field chỉ dùng ở state / endpoint khác](#field-chỉ-dùng-ở-state--endpoint-khác-không-nằm-trong-body-create)).
- **Field JSON "chưa map"** (tồn tại trong struct Go nhưng không có input
  Terraform, luôn ở giá trị zero) — đúng 6 field, tất cả đã liệt kê ở trên:
  `currentNetworking`, `deltaQuotaScale` (luôn `0`, provider tự set), `ram`,
  `cpu`, `gpu_amount`, `kubernetes` (pool). Không phát sinh field mới nào
  ngoài danh sách đã có trong tài liệu này.
- `isV2` **không** phải field chưa map — được gán trong `resource_mgpu_cluster.go`
  (`Create`, dựa trên `requiresV2API`), không phải trong `MapTerraformToJson`.

Đã chốt xong (không còn là việc cần làm): `scale_min`/`scale_max`/`worker_type`
(không áp dụng cho mgpu, field của mfke).
