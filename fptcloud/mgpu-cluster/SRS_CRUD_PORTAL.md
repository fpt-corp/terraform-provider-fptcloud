# SRS — MGPU / Managed GPU Cluster (CRUD)

Tài liệu đặc tả các API và trường dữ liệu của dịch vụ **Managed GPU Cluster (MGPU)** để làm cơ sở phát triển Terraform provider.

Nguồn: reverse-engineer từ portal UI (`apps/web/src/pages/kubernetes/hpc/`, `apps/web/src/api/path.js`). Đây là mô tả **hành vi hiện tại của UI**, không phải spec chính thức từ backend — xem mục [Điểm cần xác nhận với backend](#12-điểm-cần-xác-nhận-với-backend).

---

## 1. Tổng quan

MGPU là dịch vụ Kubernetes cluster chạy trên **bare-metal GPU server**. Về mặt code nó dùng chung backend "managed FKE" (m-fke) với FKE thường, phân biệt bằng segment `/hpc` trong URL.

**Đặc điểm quan trọng nhất cho Terraform:** một MGPU cluster là **hai tài nguyên backend riêng biệt** phải được thao tác tuần tự:

| # | Hệ thống | Vai trò |
|---|---|---|
| 1 | **m-fke cluster** (`/fke/.../m-fke/<platform>/hpc/...`) | Cluster K8s, worker pool, network, SSH, HPS |
| 2 | **GPU software** (`/fke-gpu/common/vpc/{vpcId}/gpu-clusters/...`) | GPU operator, driver, MIG, GPU sharing |

Cả Create, Update (sửa pool) và Delete đều phải gọi **cả hai**. UI hiện tại gọi tuần tự và **không có rollback** — xem [mục 11](#11-lưu-ý-triển-khai-terraform).

### 1.1. Base path & biến môi trường

```
apiPath        = /api/v1
apiPathV2      = /api/v2
apiXplatPath   = /api/v1/xplat
apiXplatPathV2 = /api/v2/xplat
```

Mẫu URL m-fke:
```
{apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/<action>
                                      ^^^^^^^^^^ ^^^^  ^^^^
                                      lowercase  bắt buộc cho MGPU
```

- `platform`: lấy từ `localStorage('platform')`, viết thường. Với MGPU thực tế **chỉ hỗ trợ `osp`** (UI gate `platform === 'OSP'`).
- `/hpc`: segment cố định phân biệt MGPU với FKE thường (trong code là cờ `isBareMetal: true`).
- `/v2`: tuỳ chọn, bật khi `isV2 = true`. UI lấy từ query param `?v2=`. Provider nên expose thành 1 biến cấu hình.

### 1.2. Header bắt buộc

Mọi request m-fke và GPU-software đều gửi kèm:

| Header | Giá trị | Bắt buộc |
|---|---|---|
| `infra-type` | giá trị `platform` (vd `OSP`), fallback `0` | Có |
| `fpt-region` | `regionId` (vd `HAN1`, `HCM3`) | Có |

Cộng thêm header xác thực chuẩn của portal (bearer token).

### 1.3. Route UI tương ứng (tham chiếu)

| Chức năng | Đường dẫn UI |
|---|---|
| List | `/:tenantName/:regionId/vpc/:vpcId/hpc/kubernetes/` |
| Create | `/:tenantName/:regionId/vpc/:vpcId/hpc/kubernetes/create` |
| Detail | `/:tenantName/:regionId/vpc/:vpcId/hpc/kubernetes/detail/:clusterId` |

---

## 2. CREATE — Tạo cluster

Luồng UI: 4 bước (Basics → Nodes Pool → Advanced → Review) rồi gọi 2 API tuần tự.

### 2.1. Bước tiền đề: kiểm tra quota

```http
POST {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/check-quota-resources
```

UI gọi trước khi cho phép submit. Nếu response `status_code === 1134` thì UI cho phép bỏ qua (cờ `ignoreCheckQuota`). Provider nên gọi ở giai đoạn `plan`/`validate` để fail sớm.

### 2.2. API 1 — Tạo cluster

```http
POST {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/create-cluster
```

**Lưu ý về `cluster_name`:** UI tự sinh ID theo `"{tên người dùng nhập}".toLowerCase() + "-" + random(...)`, rồi gửi ID đó làm `cluster_name`. Tức là tên thật của cluster **không phải** tên người dùng nhập. Terraform nên:
- cho người dùng khai báo `name` (phần prefix), và
- lưu `cluster_name` thực tế trả về vào state làm `id`.

#### Body

| Trường | Kiểu | Bắt buộc | Mô tả / Ràng buộc |
|---|---|---|---|
| `cluster_name` | string | **Có** | ID cluster. Prefix do user nhập: 3–20 ký tự, regex `^[a-zA-Z0-9-]+$`, sẽ được lowercase + hậu tố random |
| `network_id` | string | **Có** | ID subnet (từ API list subnet) |
| `vm_subnet` | string | **Có** | `subnet_cidr` của network đã chọn |
| `osp_network_id` | string | **Có** | `osp_network_id` của network đã chọn |
| `k8s_version` | string | **Có** | Từ API `get_k8s_versions` |
| `os_version` | string | **Có** | Đi kèm bản k8s đã chọn (field `os_version`) |
| `isV2` | bool | **Có** | Mặc định `false` |
| `purpose` | enum | **Có** | `"public"` nếu `clusterEndpointAccess.type == "public"`, ngược lại `"private"` |
| `clusterEndpointAccess` | object | **Có** | Xem [2.2.1](#221-clusterendpointaccess) |
| `lbInternalNetwork` | object \| null | **Có** | Subnet cho internal LB. Bắt buộc theo validation UI. Không được overlap node-network/`172.17.0.0/16` |
| `pools` | array | **Có** | ≥ 1 phần tử. Xem [2.2.2](#222-pools-worker-group) |
| `pod_network` | string | **Có** | Phần địa chỉ của CIDR (vd `100.96.0.0`). Mặc định UI `100.96.0.0/11` |
| `pod_prefix` | string | **Có** | Phần prefix (vd `11`) |
| `service_network` | string | **Có** | Mặc định UI `100.64.0.0/13` → `100.64.0.0` |
| `service_prefix` | string | **Có** | vd `13` |
| `k8s_max_pod` | int | **Có** | 1–110. Mặc định `110` |
| `network_node_prefix` | int | **Có** | Mặc định `23` |
| `network_type` | string | **Có** | Hardcode `"calico"` |
| `ssh_public_key` | string \| null | **Có** | Nội dung public key |
| `ssh_name` | string \| null | **Có** | Tên SSH key |
| `ssh_id` | string \| null | **Có** | ID SSH key |
| `auto_upgrade_expression` | array | Không | Cron expression. Mặc định `[]` |
| `auto_upgrade_timezone` | string | Không | Mặc định `""` |
| `type_create` | string | **Có** | Hardcode `"create"` |
| `hps` | object \| null | Không | `null` nếu không bật HPS. Xem [2.2.3](#223-hps-high-performance-storage) |

> **Về `pod_network` / `service_network`:** UI tách CIDR thành 2 trường riêng. Terraform nên nhận `pod_network = "100.96.0.0/11"` rồi tự split khi build request.

##### 2.2.1. `clusterEndpointAccess`

| Trường | Kiểu | Bắt buộc | Ràng buộc |
|---|---|---|---|
| `type` | enum | **Có** | `public` \| `private` \| `mixed` \| `null` |
| `allowCidr` | array\<string\> | Không | Mặc định `[]` |

##### 2.2.2. `pools[]` (worker group)

| Trường | Kiểu | Bắt buộc | Ràng buộc |
|---|---|---|---|
| `worker_pool_id` | string | **Có** | Regex `^[0-9a-z][0-9a-z-]{0,14}$`, không kết thúc bằng `-`/`_`. Tối đa 15 ký tự, chỉ chữ thường/số/gạch ngang |
| `container_runtime` | string | **Có** | Hiện chỉ `"containerd"` |
| `hpc_flavor_id` | string | **Có** | Từ API `listServerPackage`. Flavor phải còn `limit` > 0 |
| `hpc_flavor_name` | string | **Có** | Tên flavor tương ứng |
| `hpc_number_server` | int | **Có** | 1 – 100 |
| `auto_scale` | bool | **Có** | Hardcode `false` (MGPU chưa hỗ trợ autoscale) |
| `kv` | array\<object\> | Không | Labels dạng `{"key":"value"}`. UI lọc bỏ `{"":""}` |
| `taints` | array\<object\> | Không | **Chỉ gửi cho pool index > 0** — pool đầu (base) không được có taint |
| `isCreate` | bool | **Có** | `true` khi tạo mới |
| `isScale` | bool | **Có** | `false` |
| `isOthers` | bool | **Có** | `false` |
| `gpuType` | enum | Có (nếu dùng GPU) | `A100` \| `A30` \| `H100` \| `H200` |
| `driverInstallationType` | enum | **Có** | `MANAGED` \| `PRE_INSTALL` \| `USER_INSTALL` |
| `gpuDriverVersion` | string | Có khi driver ≠ `USER_INSTALL` | Từ API `gpu-drivers` |
| `gpuTemplateVersion` | string | Có khi driver ≠ `USER_INSTALL` | `imageID` của driver version |
| `workerMigStrategy` | enum | Có có điều kiện¹ | `NONE` \| `SINGLE` \| `MIXED` |
| `migProfile` | string | Có có điều kiện¹ | Từ API `mig-profiles` |
| `sharingClient` | enum | Có có điều kiện² | `NONE` \| `MPS` \| `TIMESLICING` |
| `maxClient` | int | Có có điều kiện³ | 0 nếu sharing = `NONE`; 2–48 nếu `MPS`/`TIMESLICING` |
| `gpuScheduler` | string | Có khi bật GPU scheduler | |

**Ràng buộc điều kiện:**
1. `workerMigStrategy` / `migProfile` bắt buộc khi cluster-level `hasGpuOperator ∈ {SINGLE, MIXED}` **và** `driverInstallationType == PRE_INSTALL`.
2. `sharingClient` bắt buộc khi `hasGpuOperator == SINGLE` **và** `driverInstallationType == PRE_INSTALL`.
3. `maxClient`: nếu `sharingClient ∈ {null, "", NONE}` và driver = `PRE_INSTALL` → bắt buộc `= 0`. Ngược lại bắt buộc trong `[2, 48]`.
4. Nếu `driverInstallationType == USER_INSTALL`: MIG/sharing bị vô hiệu, xem [2.3](#23-api-2--cài-gpu-software).

##### 2.2.3. `hps` (High Performance Storage)

`null` nếu không bật. Nếu bật, **cả 3 trường policy + mount point đều bắt buộc**:

| Trường | Kiểu | Bắt buộc | Nguồn |
|---|---|---|---|
| `hps_tenant_id` | string | **Có** | `mountPointHPS.value.hpsTenantId` |
| `qos_policy` | string | **Có** | API `fsaas/qos_policies` |
| `view_policy` | string | **Có** | API `fsaas/view_policies` |
| `vip_pool` | string | **Có** | `mountPointHPS.value.vipPool` |
| `mount_point` | string | **Có** | `mountPointHPS.value.mountPoint` |

### 2.3. API 2 — Cài GPU software

Gọi **ngay sau khi** API 1 trả về thành công, dùng chính `cluster_name` vừa tạo.

```http
POST {apiXplatPath|apiXplatPathV2}/fke-gpu/common/vpc/{vpcId}/gpu-clusters/{clusterName}
```

> Lưu ý: với MGPU (`isBareMetal = true`) URL **không** có segment `/m-fke`. Base là `apiXplatPathV2` khi `isV2 = true`.

#### Body

| Trường | Kiểu | Bắt buộc | Mô tả |
|---|---|---|---|
| `isV2` | bool | **Có** | Đồng bộ với API 1 |
| `name` | string | **Có** | = `cluster_name` |
| `infra_type` | string | **Có** | vd `OSP` |
| `region` | string | **Có** | vd `HAN1` |
| `tenant_id` | string | **Có** | |
| `kubernetes_version` | string | **Có** | Trùng `k8s_version` của API 1 |
| `operator_version` | object | **Có** | Map `{<software_type>: <version>}` |
| `cluster_type` | string | **Có** | Hardcode `"BM"` (bare metal) |
| `mig_strategy` | string \| null | Không | Lấy từ `softwareOption` của `gpu_operator` |
| `worker_groups` | array | **Có** | Xem dưới |
| `status` | string | **Có** | Hardcode `"NOTREADY"` |

`operator_version` — key hợp lệ:
```
gpu_operator | network_operator | slurm_operator | vgpu_scheduler
```
Danh sách version lấy từ API `operator-versions`.

##### `worker_groups[]`

Phải khớp 1-1 với `pools[]` của API 1 (theo `name` == `worker_pool_id`).

| Trường | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `name` | string | **Có** | = `worker_pool_id` |
| `mig_mode` | string | **Có** | `NONE` nếu driver = `USER_INSTALL`, ngược lại = `workerMigStrategy` |
| `mig_profile` | string | **Có** | `NONE` nếu driver = `USER_INSTALL` |
| `sharing_client_type` | string | **Có** | `all-disabled` nếu driver = `USER_INSTALL`, ngược lại = `sharingClient` |
| `max_client` | int | **Có** | `0` nếu driver = `USER_INSTALL` |
| `gpu_scheduler` | string | **Có** | `NONE` nếu không bật GPU scheduler |
| `driver_type` | string | **Có** | `MANAGED` \| `PRE_INSTALL` \| `USER_INSTALL` |
| `driver_version` | string | **Có** | `""` nếu driver = `USER_INSTALL` |
| `enable_operand` | bool | **Có** | Hardcode `true` |
| `gpu_type` | string | **Có** | `A100`/`A30`/`H100`/`H200` |

---

## 3. READ — Đọc thông tin cluster

### 3.1. Chi tiết cluster

```http
GET {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/get-shoot-specific/shoots/{clusterId}
```

Response là object kiểu Gardener shoot. Các đường dẫn UI đang đọc (dùng để map về state Terraform):

| Thuộc tính | Đường dẫn trong response |
|---|---|
| Tên cluster | `metadata.name` |
| K8s version | `spec.kubernetes.version` |
| Node network | `spec.networking.nodes` |
| Worker pools | `workers[]` |
| SSH key | `workers[0].providerConfig.sshKey.{name,id}` |

### 3.2. Danh sách cluster

```http
GET {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/get-shoot-cluster/shoots?page={page}&page_size={pageSize}
```

### 3.3. Thông tin GPU software

```http
GET {apiXplatPathV?}/fke-gpu/common/vpc/{vpcId}/gpu-clusters/{clusterName}?tenant_id={tenantId}&region={regionId}
```

**Bắt buộc gọi trước khi Update/Delete pool** — xem [mục 4](#4-update--sửa-cluster).

Trạng thái cài đặt GPU software:
```http
GET .../fke-gpu/common/vpc/{vpcId}/gpu-clusters/{clusterName}/status?tenant_id={tenantId}&region={regionId}
```

### 3.4. Kubeconfig

```http
GET {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/get-kubeconfig/{clusterName}?direct=1
```

### 3.5. Danh sách VM/node của cluster

```http
GET .../hpc[/v2]/get_vm_by_cluster_id/{clusterId}?page={page}&page_size={pageSize}
```

---

## 4. UPDATE — Sửa cluster

Không có API "update cluster" tổng quát. Mỗi thuộc tính có endpoint riêng, và **phần lớn thuộc tính là immutable** (bắt buộc `ForceNew` trong Terraform).

### 4.1. Sửa / scale worker pool (`configure_worker`)

Đây là endpoint quan trọng nhất. Nó xử lý **thêm pool, sửa pool, scale số node, và xoá pool** — tất cả bằng cách gửi lại **toàn bộ** danh sách pool (semantics thay thế, không phải patch từng phần).

```http
PATCH {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/configure-worker-cluster/shoots/{clusterName}/0
```

Segment cuối (`0`) là `disableTraceLog`.

#### Body

| Trường | Kiểu | Bắt buộc | Mô tả |
|---|---|---|---|
| `ssh_name` | string \| null | **Có** | Lấy từ `workers[0].providerConfig.sshKey.name` của cluster hiện tại |
| `ssh_id` | string \| null | **Có** | Tương tự |
| `pools` | array | **Có** | **Toàn bộ** pool sau khi thay đổi |
| `k8s_version` | string \| null | **Có** | `cluster.spec.kubernetes.version` hiện tại |
| `currentNetworking` | string | **Có** | `cluster.spec.networking.nodes` hiện tại |
| `type_configure` | string | **Có** | Hardcode `"configure"` |

#### `pools[]` khi update

Giống cấu trúc [2.2.2](#222-pools-worker-group), **thêm** các trường:

| Trường | Kiểu | Bắt buộc | Mô tả |
|---|---|---|---|
| `worker_base` | bool | **Có** | Đánh dấu pool gốc. **Đúng 1 pool** có `true` |
| `deltaQuotaScale` | int | **Có** | Chênh lệch số node so với hiện tại. `0` nếu không scale |
| `isCreate` | bool | **Có** | `true` với pool mới thêm |
| `isScale` | bool | **Có** | `true` khi thay đổi `hpc_number_server` |
| `isOthers` | bool | **Có** | `true` khi sửa thuộc tính khác |

**Ràng buộc bắt buộc:**
- Mảng `pools` phải được **sort sao cho pool có `worker_base = true` đứng đầu**:
  ```js
  pools.sort((a, b) => Number(b.worker_base) - Number(a.worker_base))
  ```
- `taints` chỉ gửi cho pool **không phải** base.
- Regex tên pool khi edit dùng `REG_WORKER_GROUP_NAME_EDIT` (khác lúc create — cho phép hậu tố `-(base)` mà UI sẽ strip trước khi gửi).
- `flavor` bắt buộc với pool mới (`isNewWG = true`), tuỳ chọn với pool đã tồn tại → **không đổi được flavor của pool hiện có**.

#### Đồng bộ GPU software

Sau khi `configure_worker` thành công, phải gọi:

```http
PUT {apiXplatPathV?}/fke-gpu/common/vpc/{vpcId}/gpu-clusters/{clusterName}
```

Body = **object GPU-software info đọc từ [3.3](#33-thông-tin-gpu-software)**, với `worker_groups` được thay bằng danh sách mới (cùng mapping như [2.3](#23-api-2--cài-gpu-software)).

> Đây là read-modify-write: bắt buộc GET trước, sửa field `worker_groups`, rồi PUT lại toàn bộ. Không được tự dựng body từ đầu.

### 4.2. Xoá một worker group

Không có endpoint DELETE riêng. Thực hiện bằng `configure_worker` với pool đó **đã bị lọc khỏi mảng `pools`**, body y hệt [4.1](#41-sửa--scale-worker-pool-configure_worker).

Sau đó `PUT` GPU software với `worker_groups` đã lọc bỏ group tương ứng (so khớp theo `name`).

### 4.3. Nâng cấp phiên bản Kubernetes

```http
PATCH {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}[/v2]/upgrade_version_cluster/shoots/{clusterName}/k8s-version/{versionUpgrade}
```

> ⚠️ Endpoint này **không có segment `/hpc`** (khác với tất cả endpoint lifecycle khác). Cần xác nhận với backend — xem [mục 12](#12-điểm-cần-xác-nhận-với-backend).

### 4.4. Đổi Internal LB Network

```http
PATCH {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}[/v2]/... (configLbInternalNetwork)
```

Chỉ khả dụng khi `isBareMetal && !isV2`.

### 4.5. HPS integration

| Thao tác | Method | Endpoint |
|---|---|---|
| Bật/tích hợp | `POST` | `{apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc/shoots/{clusterName}/hps` |
| Tắt/gỡ | `DELETE` | (cùng URL) |

### 4.6. Hibernation (ngủ đông / đánh thức)

```http
PATCH {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}[/v2]/hibernation-cluster/shoots/{clusterName}/{hibernation}
```
`hibernation`: `true` (ngủ) / `false` (đánh thức).

### 4.7. Các thao tác khác

| Thao tác | Endpoint |
|---|---|
| Retry cluster lỗi | `PATCH .../hpc[/v2]/retry-cluster/shoots/{clusterName}` |
| Rotate kubeconfig | `PATCH .../hpc[/v2]/rotate-kubeconfig/shoots/{clusterName}` |
| Sửa tag | `PUT .../m-fke/{platform}[/v2]/shoots/{clusterName}/tags` |

---

## 5. DELETE — Xoá cluster

Hai bước tuần tự.

### 5.1. Xoá cluster

```http
DELETE {apiXplatPath}/fke/vpc/{vpcId}/m-fke/{platform}/hpc[/v2]/delete-shoot-cluster/shoots/{clusterName}
```
Không có body.

### 5.2. Xoá GPU software

Chỉ gọi **sau khi** bước 5.1 thành công (UI đặt trong `onSuccess`):

```http
DELETE {apiXplatPathV?}/fke-gpu/common/vpc/{vpcId}/gpu-clusters/{clusterName}?tenant_id={tenantId}
```

> UI yêu cầu người dùng gõ đúng tên cluster để xác nhận — đây là ràng buộc UI, không phải API.

---

## 6. API tra cứu (lookup / data source)

Terraform cần các API này để resolve tên → ID.

| Mục đích | Endpoint |
|---|---|
| Danh sách K8s version | `GET .../m-fke/{platform}/hpc/get_k8s_versions` |
| Danh sách SSH key | (API SSH key chung của portal) |
| Danh sách subnet MGPU | `GET {apiPathV2}/vmware/vpc/{vpcId}/hpc/subnets?page=&pageSize=` |
| Subnet khả dụng | `GET {apiPathV2}/vmware/vpc/{vpcId}/hpc/subnets/available` |
| **Danh sách flavor GPU** | `GET {apiPathV2}/vmware/vpc/{vpcId}/hpc/server_package` |
| Kiểm tra dịch vụ bật chưa | `GET {apiPathV2}/vmware/vpc/{vpcId}/hpc/server/check-service-enable` |
| Kiểm tra quota | `POST .../hpc[/v2]/check-quota-resources` |
| Purposes | `GET .../hpc[/v2]/purposes` |
| GPU operator versions | `GET .../fke-gpu/common/vpc/{vpcId}/operator-versions` |
| GPU driver versions | `GET .../fke-gpu/common/vpc/{vpcId}/gpu-drivers?driver_type=&zone=&kubernetes_version=` |
| MIG profiles | `GET .../fke-gpu/common/vpc/{vpcId}/mig-profiles?gpu_type=&mig_mode=` |

### HPS (chỉ khi dùng HPS)

| Mục đích | Endpoint |
|---|---|
| HPS tenant/store | `GET {apiPathV2}/fsaas/stores?tenant_id={tenantId}` |
| HPS network | `GET {apiPathV2}/fsaas/subnets?tenant_id={tenantId}` |
| Chi tiết HPS network | `GET {apiPathV2}/fsaas/subnets/{networkId}` |
| Mount point | `POST {apiPathV2}/fsaas/networkmappings/used_by` |
| QoS policy | `GET {apiPathV2}/fsaas/qos_policies/{qosPolicyId}` |
| View policy | `GET {apiPathV2}/fsaas/view_policies/{viewPolicyId}` |

---

## 7. Bảng enum

```
gpu_type              : A100 | A30 | H100 | H200
driver_type           : MANAGED | PRE_INSTALL | USER_INSTALL
mig_mode / strategy   : NONE | SINGLE | MIXED
sharing_client_type   : NONE | MPS | TIMESLICING   (+ "all-disabled" khi USER_INSTALL)
software_type         : gpu_operator | network_operator | slurm_operator | vgpu_scheduler
container_runtime     : containerd
network_type          : calico
cluster_type          : BM
endpoint_access_type  : public | private | mixed
```

## 8. Giới hạn số

| Tham số | Giới hạn |
|---|---|
| Độ dài `cluster_name` (phần user nhập) | 3 – 20 |
| Độ dài tên worker group | 1 – 15 |
| Số node mỗi pool (`hpc_number_server`) | 1 – 100 |
| Max pod / node (`k8s_max_pod`) | 1 – 110 |
| `max_client` khi bật GPU sharing | 2 – 48 |
| `max_client` khi không sharing | 0 |

## 9. Giá trị mặc định

| Trường | Mặc định |
|---|---|
| `pod_network` | `100.96.0.0/11` |
| `service_network` | `100.64.0.0/13` |
| `network_node_prefix` | `23` |
| `k8s_max_pod` | `110` |
| `network_type` | `calico` |
| `container_runtime` | `containerd` |
| `hpc_number_server` | `1` |
| `max_client` | `0` |
| `auto_scale` | `false` |
| `clusterEndpointAccess.type` | `public` |
| `isV2` | `false` |

## 10. Ràng buộc validation cần implement ở provider

1. **CIDR không được overlap** — `pod_network`, `service_network`, `lbInternalNetwork.cidr` phải không đè lên nhau, không đè lên node network dành riêng theo region/platform, và không đè `172.17.0.0/16` (Docker subnet).
2. **Flavor phải còn chỗ** — UI kiểm tra `flavor.limit`; nếu hết sẽ báo "no more usable space".
3. **Regex tên** — cluster `^[a-zA-Z0-9-]+$`; worker group `^[0-9a-z][0-9a-z-]{0,14}$` và không kết thúc bằng `-`/`_`.
4. **Ràng buộc MIG/sharing** theo `driverInstallationType` — xem [2.2.2](#222-pools-worker-group).
5. **Đúng một pool có `worker_base = true`** khi update, và pool đó phải đứng đầu mảng.
6. **Pool base không được có `taints`.**
7. **HPS: bật thì cả 5 trường con đều bắt buộc.**

---

## 11. Lưu ý triển khai Terraform

### 11.1. Tính không nguyên tử (quan trọng nhất)

Create / Update pool / Delete đều gồm 2 lời gọi tới 2 hệ thống khác nhau, **không có rollback**. UI hiện tại nếu bước 2 lỗi thì chỉ hiện thông báo, để lại trạng thái lệch.

Khuyến nghị cho provider:
- **Create**: nếu API GPU-software lỗi → vẫn phải `d.SetId(clusterName)` **trước** khi trả lỗi, nếu không Terraform sẽ mất dấu cluster đã tạo (orphan resource). Trả về error để user biết cần `terraform apply` lại.
- **Update**: đọc lại GPU-software info (GET) trước mỗi lần PUT (read-modify-write).
- **Delete**: nếu bước 5.1 thành công nhưng 5.2 lỗi → không được xoá ID khỏi state ngay; cần retry bước 5.2.

### 11.2. Semantics thay thế toàn bộ của `configure_worker`

`pools` là **danh sách đầy đủ**, không phải patch. Provider phải:
- đọc trạng thái hiện tại (`get-shoot-specific`),
- merge với config mới,
- gửi lại toàn bộ.

Đồng thời tự tính `deltaQuotaScale`, `isCreate`, `isScale`, `isOthers` — backend dựa vào các cờ này để phân biệt loại thao tác.

### 11.3. Trường nên đặt `ForceNew`

Không có API sửa, nên đổi các trường sau phải tạo lại cluster:
`network_id`, `vm_subnet`, `osp_network_id`, `pod_network`, `pod_prefix`, `service_network`, `service_prefix`, `network_node_prefix`, `k8s_max_pod`, `network_type`, `ssh_*`, `purpose` / `clusterEndpointAccess.type`.

Đổi flavor của một pool đã tồn tại cũng không được hỗ trợ → `ForceNew` ở cấp pool.

### 11.5. Import

`id` của resource nên là `cluster_name` (ID thật, đã có hậu tố random) — không phải tên user nhập. Khi import cần đọc cả 2 nguồn: cluster detail + GPU software info.

---

## 12. Điểm cần xác nhận với backend

Các điểm sau phát hiện từ code UI nhưng **chưa được xác minh**, nên hỏi lại team backend trước khi implement:

1. **`upgrade_version` thiếu segment `/hpc`** ([`api/path.js:939-945`](apps/web/src/api/path.js#L939-L945)) trong khi mọi endpoint lifecycle khác của MGPU đều có. Cần xác nhận đây là chủ ý hay bug.
=> tẹo nữa thử call API để xác minh
2. **Ngữ nghĩa chính xác của `deltaQuotaScale`, `isCreate`, `isScale`, `isOthers`** — backend dùng để làm gì, có bắt buộc chính xác không, hay chỉ để trace log.
=> Chỉ trace logs
3. **`worker_base`** — có được đổi pool base sau khi tạo không? UI cho chọn nhưng chưa rõ backend xử lý thế nào.
=> Có
4. **Có API đọc/validate cluster theo tên trước khi tạo không** (để tránh trùng), vì UI tự sinh hậu tố random thay vì kiểm tra.
=> Không
5. **Behavior khi GPU-software và cluster lệch nhau** — có API reconcile không (ngoài `activate`: `POST .../gpu-clusters/{clusterName}/activate`).
=> Không, chỉ có API activate, activate đến khi được, nếu không được thì trả ra lỗi cho khách hàng, nma cluster vẫn được tạo
6. **`isV2`** — điều kiện nào thì phải bật? Ảnh hưởng gì tới payload ngoài đường dẫn URL?
Version K8s >= 1.33
7. **MGPU có hỗ trợ platform nào khác `OSP` không** — UI hardcode gate `platform === 'OSP'`.
Chỉ hỗ trợ OSP

---

## Phụ lục — Tham chiếu file nguồn

| Chức năng | File |
|---|---|
| Payload create + 2 mutation | [`create/CreateHPCK8SPage.js`](apps/web/src/pages/kubernetes/hpc/create/CreateHPCK8SPage.js) |
| Validation schema create | [`create/config.js`](apps/web/src/pages/kubernetes/hpc/create/config.js) |
| Schema GPU software | [`schema-joi/GpuSoftware.js`](apps/web/src/pages/kubernetes/schema-joi/GpuSoftware.js) |
| Payload update pool | [`detail/components/EditPools.js`](apps/web/src/pages/kubernetes/hpc/detail/components/EditPools.js) |
| Xoá worker group | [`button/BtnRemoveWorkerGroup.js`](apps/web/src/pages/kubernetes/hpc/button/BtnRemoveWorkerGroup.js) |
| Xoá cluster | [`openstack-v2/button/BtnDelete.js`](apps/web/src/pages/kubernetes/openstack-v2/button/BtnDelete.js) |
| Định nghĩa toàn bộ endpoint | [`api/path.js`](apps/web/src/api/path.js) |
| Enum & hằng số | [`hpc/constants/constants.js`](apps/web/src/pages/kubernetes/hpc/constants/constants.js) |
| Giới hạn số | [`openstack-v2/FmkeConstants.js`](apps/web/src/pages/kubernetes/openstack-v2/FmkeConstants.js) |
 