# MFKE Service Tests

File test này sử dụng thư viện `testify` để test các chức năng của MFKE service theo thứ tự sau:

## Cách chạy test

### 1. Chạy tất cả test
```bash
cd fptcloud/mfke
go test -v
```

### 2. Chạy test cụ thể theo nhóm
```bash
# Test tạo cluster cơ bản
go test -v -run TestManagedKubernetesEngine_CreateBasicCluster

# Test các tính năng nâng cao
go test -v -run TestManagedKubernetesEngine_AdvancedFeatures

# Test xóa cluster
go test -v -run TestManagedKubernetesEngine_DeleteCluster

# Test cập nhật cluster
go test -v -run TestManagedKubernetesEngine_UpdateCluster

# Test GPU worker pool
go test -v -run TestManagedKubernetesEngine_GPUWorkerPool
```

### 3. Chạy test với coverage
```bash
go test -v -cover
```

## Cấu trúc test cases

### 1. TestManagedKubernetesEngine_CreateBasicCluster
**Mục đích**: Test tạo cụm cluster bình thường với 1-2 worker pools
- **CreateClusterWithSingleWorkerPool**: Kiểm tra tạo cluster với 1 worker pool và `worker_base = true`
- **CreateClusterWithTwoWorkerPools**: Kiểm tra tạo cluster với 2 worker pools (pool đầu tiên có `worker_base = true`, pool thứ hai có `worker_base = false`)

**Kiểm tra**:
- Tên cluster, phiên bản K8s, mục đích, network type
- Số lượng worker pools
- Thuộc tính `worker_base` của từng pool
- Các thuộc tính cơ bản của worker pool (storage, disk size, scale min/max)

### 2. TestManagedKubernetesEngine_AdvancedFeatures
**Mục đích**: Test các tính năng nâng cao của cluster
- **TestAutoUpgradeEnabled**: Kiểm tra cấu hình auto upgrade với cron expression và timezone
- **TestHibernationSchedules**: Kiểm tra cấu hình hibernation schedules (start/end time, location)
- **TestIsRunning**: Kiểm tra trường computed `is_running`

**Kiểm tra**:
- `is_enable_auto_upgrade`, `auto_upgrade_expression`, `auto_upgrade_timezone`
- `hibernation_schedules` với start, end, location
- Các trường computed

### 3. TestManagedKubernetesEngine_DeleteCluster
**Mục đích**: Test logic xóa cụm cluster
- **TestClusterDeletion**: Giả lập việc xóa cluster và kiểm tra kết quả

### 4. TestManagedKubernetesEngine_UpdateCluster
**Mục đích**: Test cập nhật cluster với nhiều thay đổi khác nhau
- **TestUpdateK8sVersion**: Kiểm tra cập nhật phiên bản Kubernetes
- **TestAddWorkerPool**: Kiểm tra thêm worker pool mới
- **TestRemoveWorkerPool**: Kiểm tra xóa worker pool
- **TestAddKVToWorkerPool**: Kiểm tra thêm KV labels vào worker pool
- **TestAddTaintsToWorkerPool**: Kiểm tra thêm taints vào worker pool

**Kiểm tra**:
- Thay đổi phiên bản K8s
- Thêm/xóa worker pools
- Thêm KV labels (environment, team)
- Thêm taints (dedicated, environment)

### 5. TestManagedKubernetesEngine_GPUWorkerPool
**Mục đích**: Test tạo cluster với worker pool GPU
- **TestGPUWorkerPoolCreation**: Kiểm tra tạo cluster với GPU worker pool
- **TestGPUValidationRules**: Kiểm tra các validation rules cho GPU

**Kiểm tra**:
- Các trường GPU: `vgpu_id`, `max_client`, `gpu_sharing_client`, `driver_installation_type`, `gpu_driver_version`
- KV labels bắt buộc: `nvidia.com/mig.config`, `worker.fptcloud/type`
- Validation rules:
  - `max_client`: chỉ validate khi có `vgpu_id` (GPU pool), phải trong khoảng 2-48
  - `gpu_sharing_client`: chỉ có thể là `""` hoặc `"timeSlicing"`
  - `driver_installation_type`: chỉ có thể là `"pre-install"`
  - `gpu_driver_version`: chỉ có thể là `"default"` hoặc `"latest"`

## Helper Functions

### Config Generation
- `generateBasicClusterConfig(clusterName, poolCount)`: Tạo config cluster cơ bản với số lượng worker pools tùy chỉnh
- `generateClusterWithAutoUpgradeConfig(clusterName, enabled)`: Tạo config cluster với auto upgrade
- `generateClusterWithHibernationConfig(clusterName)`: Tạo config cluster với hibernation schedules
- `generateUpdatedClusterConfig(clusterName, k8sVersion, poolCount)`: Tạo config cluster đã cập nhật
- `generateClusterWithKVConfig(clusterName)`: Tạo config cluster với KV labels
- `generateClusterWithTaintsConfig(clusterName)`: Tạo config cluster với taints
- `generateGPUClusterConfig(clusterName)`: Tạo config cluster với GPU worker pool

### Utility Functions
- `simulateClusterDeletion(clusterName)`: Giả lập việc xóa cluster
- `countWorkerPools(config)`: Đếm số lượng worker pools trong config
- `validateMaxClient(value)`: Validate giá trị max_client (chỉ cho GPU pools)
- `validateGpuSharingClient(value)`: Validate giá trị gpu_sharing_client
- `validateDriverInstallationType(value)`: Validate giá trị driver_installation_type
- `validateGpuDriverVersion(value)`: Validate giá trị gpu_driver_version

## Yêu cầu

- Go 1.21+
- Thư viện `testify` đã được cài đặt
- Các biến môi trường (nếu cần):
  - `FPTCLOUD_TOKEN`
  - `FPTCLOUD_REGION`
  - `FPTCLOUD_TENANT_NAME`

## Lưu ý

Đây là unit tests đơn giản sử dụng `testify`, không phải acceptance tests của Terraform. Các test này kiểm tra:

1. **Logic validation** của provider
2. **Config generation** cho các trường hợp khác nhau
3. **Cấu trúc HCL** được tạo ra
4. **Validation rules** cho các trường GPU
5. **Simulation** của các thao tác CRUD

Các test không kết nối với API thật mà chỉ kiểm tra logic và cấu trúc của provider.
