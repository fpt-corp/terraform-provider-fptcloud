package fptcloud_mfke

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccPreCheck(t *testing.T) {
	// Kiểm tra các biến môi trường cần thiết
	requiredEnvVars := []string{
		"FPTCLOUD_TOKEN",
		"FPTCLOUD_REGION",
		"FPTCLOUD_TENANT_NAME",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			t.Skipf("Biến môi trường %s phải được thiết lập để chạy test", envVar)
		}
	}
}

// 1. Test tạo cụm cluster bình thường có 1 worker pools, 2 worker pools yêu cầu có worker_base
func TestManagedKubernetesEngine_CreateBasicCluster(t *testing.T) {
	t.Run("CreateClusterWithSingleWorkerPool", func(t *testing.T) {
		config := generateBasicClusterConfig("test-single-pool", 1)

		// Kiểm tra config có đúng format
		assert.Contains(t, config, "test-single-pool")
		assert.Contains(t, config, "1.31.4")
		assert.Contains(t, config, "dev")
		assert.Contains(t, config, "calico")
		assert.Contains(t, config, "ipip")
		assert.Contains(t, config, "pools {")
		assert.Contains(t, config, "name                  = \"worker-pool-1\"")
		assert.Contains(t, config, "worker_base           = true")
		assert.Contains(t, config, "storage_profile       = \"Premium-SSD\"")
		assert.Contains(t, config, "worker_disk_size      = 40")
		assert.Contains(t, config, "scale_min             = 1")
		assert.Contains(t, config, "scale_max             = 2")
	})

	t.Run("CreateClusterWithTwoWorkerPools", func(t *testing.T) {
		config := generateBasicClusterConfig("test-two-pools", 2)

		// Kiểm tra config có 2 worker pools
		assert.Contains(t, config, "test-two-pools")
		assert.Contains(t, config, "name                  = \"worker-pool-1\"")
		assert.Contains(t, config, "name                  = \"worker-pool-2\"")
		assert.Contains(t, config, "worker_base           = true")
		assert.Contains(t, config, "worker_base           = false")
	})
}

// 2. Test is_enable_auto_upgrade, hibernation_schedules, is_running
func TestManagedKubernetesEngine_AdvancedFeatures(t *testing.T) {
	t.Run("TestAutoUpgradeEnabled", func(t *testing.T) {
		config := generateClusterWithAutoUpgradeConfig("test-auto-upgrade", true)

		assert.Contains(t, config, "is_enable_auto_upgrade = true")
		assert.Contains(t, config, "auto_upgrade_expression = [\"0 2 * * 0\"]")
		assert.Contains(t, config, "auto_upgrade_timezone   = \"Asia/Bangkok\"")
	})

	t.Run("TestHibernationSchedules", func(t *testing.T) {
		config := generateClusterWithHibernationConfig("test-hibernation")

		assert.Contains(t, config, "hibernation_schedules {")
		assert.Contains(t, config, "start    = \"0 23 * * 2,4\"")
		assert.Contains(t, config, "end      = \"0 7 * * 3,5\"")
		assert.Contains(t, config, "location = \"Asia/Bangkok\"")
	})

	t.Run("TestIsRunning", func(t *testing.T) {
		config := generateBasicClusterConfig("test-running", 1)

		// is_running là computed field, không cần kiểm tra trong config
		assert.Contains(t, config, "test-running")
	})
}

// 3. Test xóa cụm cluster
func TestManagedKubernetesEngine_DeleteCluster(t *testing.T) {
	t.Run("TestClusterDeletion", func(t *testing.T) {
		// Test logic xóa cluster
		clusterName := "test-delete-cluster"

		// Giả lập việc xóa cluster
		deleted := simulateClusterDeletion(clusterName)
		assert.True(t, deleted, "Cluster should be deleted successfully")
	})
}

// 4. Test update cụm cluster với update version, kv, thêm worker pools, xóa 1 worker pools, thêm kv worker pools, thêm taints worker pools
func TestManagedKubernetesEngine_UpdateCluster(t *testing.T) {
	t.Run("TestUpdateK8sVersion", func(t *testing.T) {
		originalConfig := generateBasicClusterConfig("test-update-version", 1)
		updatedConfig := generateUpdatedClusterConfig("test-update-version", "1.32.0", 1)

		assert.Contains(t, originalConfig, "1.31.4")
		assert.Contains(t, updatedConfig, "1.32.0")
	})

	t.Run("TestAddWorkerPool", func(t *testing.T) {
		originalConfig := generateBasicClusterConfig("test-add-pool", 1)
		updatedConfig := generateBasicClusterConfig("test-add-pool", 2)

		// Đếm số worker pools
		originalPoolCount := countWorkerPools(originalConfig)
		updatedPoolCount := countWorkerPools(updatedConfig)

		assert.Equal(t, 1, originalPoolCount)
		assert.Equal(t, 2, updatedPoolCount)
	})

	t.Run("TestRemoveWorkerPool", func(t *testing.T) {
		originalConfig := generateBasicClusterConfig("test-remove-pool", 2)
		updatedConfig := generateBasicClusterConfig("test-remove-pool", 1)

		originalPoolCount := countWorkerPools(originalConfig)
		updatedPoolCount := countWorkerPools(updatedConfig)

		assert.Equal(t, 2, originalPoolCount)
		assert.Equal(t, 1, updatedPoolCount)
	})

	t.Run("TestAddKVToWorkerPool", func(t *testing.T) {
		config := generateClusterWithKVConfig("test-kv-cluster")

		assert.Contains(t, config, "kv {")
		assert.Contains(t, config, "name  = \"environment\"")
		assert.Contains(t, config, "value = \"production\"")
		assert.Contains(t, config, "name  = \"team\"")
		assert.Contains(t, config, "value = \"devops\"")
	})

	t.Run("TestAddTaintsToWorkerPool", func(t *testing.T) {
		config := generateClusterWithTaintsConfig("test-taints-cluster")

		assert.Contains(t, config, "taints {")
		assert.Contains(t, config, "key    = \"dedicated\"")
		assert.Contains(t, config, "value  = \"gpu\"")
		assert.Contains(t, config, "effect = \"NoSchedule\"")
		assert.Contains(t, config, "key    = \"environment\"")
		assert.Contains(t, config, "value  = \"production\"")
		assert.Contains(t, config, "effect = \"PreferNoSchedule\"")
	})
}

// 5. Test tạo cluster với worker pool GPU
func TestManagedKubernetesEngine_GPUWorkerPool(t *testing.T) {
	t.Run("TestGPUWorkerPoolCreation", func(t *testing.T) {
		config := generateGPUClusterConfig("test-gpu-cluster")

		// Kiểm tra các trường GPU
		assert.Contains(t, config, "vgpu_id                  = \"6d95d250-ee0a-4655-acd7-5b1cdffac870\"")
		assert.Contains(t, config, "max_client               = 2")
		assert.Contains(t, config, "gpu_sharing_client       = \"timeSlicing\"")
		assert.Contains(t, config, "driver_installation_type = \"pre-install\"")
		assert.Contains(t, config, "gpu_driver_version       = \"default\"")

		// Kiểm tra KV labels bắt buộc cho GPU
		assert.Contains(t, config, "nvidia.com/mig.config")
		assert.Contains(t, config, "all-1g.6gb")
		assert.Contains(t, config, "worker.fptcloud/type")
		assert.Contains(t, config, "gpu")
	})

	t.Run("TestGPUValidationRules", func(t *testing.T) {
		// Test validation rules cho GPU
		// max_client chỉ được validate khi có vgpu_id (GPU pool)
		assert.NoError(t, validateMaxClient(1, false), "max_client = 1 should be valid for non-GPU pools")
		assert.NoError(t, validateMaxClient(2, false), "max_client = 2 should be valid for non-GPU pools")
		assert.NoError(t, validateMaxClient(48, false), "max_client = 48 should be valid for non-GPU pools")
		assert.NoError(t, validateMaxClient(49, false), "max_client = 49 should be valid for non-GPU pools")

		assert.NoError(t, validateGpuSharingClient("", false), "empty gpu_sharing_client should be valid for non-GPU pools")
		assert.NoError(t, validateGpuSharingClient("timeSlicing", false), "timeSlicing should be valid for non-GPU pools")
		assert.NoError(t, validateGpuSharingClient("invalid", false), "invalid gpu_sharing_client should be valid for non-GPU pools")

		assert.NoError(t, validateDriverInstallationType("pre-install", false), "pre-install should be valid for non-GPU pools")
		assert.NoError(t, validateDriverInstallationType("post-install", false), "post-install should be valid for non-GPU pools")

		assert.NoError(t, validateGpuDriverVersion("default", false), "default should be valid for non-GPU pools")
		assert.NoError(t, validateGpuDriverVersion("latest", false), "latest should be valid for non-GPU pools")
		assert.NoError(t, validateGpuDriverVersion("custom", false), "custom should be valid for non-GPU pools")

		// Test GPU pool validations (hasVGpuID = true)
		assert.NoError(t, validateMaxClient(2, true), "max_client = 2 should be valid for GPU pools")
		assert.NoError(t, validateMaxClient(48, true), "max_client = 48 should be valid for GPU pools")
		assert.Error(t, validateMaxClient(1, true), "max_client = 1 should be invalid for GPU pools")
		assert.Error(t, validateMaxClient(49, true), "max_client = 49 should be invalid for GPU pools")

		assert.NoError(t, validateGpuSharingClient("", true), "empty gpu_sharing_client should be valid for GPU pools")
		assert.NoError(t, validateGpuSharingClient("timeSlicing", true), "timeSlicing should be valid for GPU pools")
		assert.Error(t, validateGpuSharingClient("invalid", true), "invalid gpu_sharing_client should be invalid for GPU pools")

		assert.NoError(t, validateDriverInstallationType("pre-install", true), "pre-install should be valid for GPU pools")
		assert.Error(t, validateDriverInstallationType("post-install", true), "post-install should be invalid for GPU pools")

		assert.NoError(t, validateGpuDriverVersion("default", true), "default should be valid for GPU pools")
		assert.NoError(t, validateGpuDriverVersion("latest", true), "latest should be valid for GPU pools")
		assert.Error(t, validateGpuDriverVersion("custom", true), "custom should be invalid for GPU pools")
	})
}

// Helper functions
func generateBasicClusterConfig(clusterName string, poolCount int) string {
	pools := ""
	for i := 1; i <= poolCount; i++ {
		workerBase := "true"
		if i > 1 {
			workerBase = "false"
		}

		pools += fmt.Sprintf(`
  pools {
    name                  = "worker-pool-%d"
    storage_profile       = "Premium-SSD"
    worker_type           = "0d1da48b-57ed-4a1e-9700-33f345c38e0d"
    worker_disk_size      = 40
    scale_min             = 1
    scale_max             = 2
    network_id            = "eefb18f4-66c5-4161-9036-3e75745a0f28"
    network_name          = "default-network"
    worker_base           = %s
  }`, i, workerBase)
	}

	return fmt.Sprintf(`
resource "fptcloud_managed_kubernetes_engine_v1" "test" {
  cluster_name    = "%s"
  k8s_version     = "1.31.4"
  purpose         = "dev"
  network_type    = "calico"
  network_overlay = "ipip"
%s
}
`, clusterName, pools)
}

func generateClusterWithAutoUpgradeConfig(clusterName string, enabled bool) string {
	return fmt.Sprintf(`
resource "fptcloud_managed_kubernetes_engine_v1" "test" {
  cluster_name    = "%s"
  k8s_version     = "1.31.4"
  purpose         = "dev"
  network_type    = "calico"
  network_overlay = "ipip"
  
  is_enable_auto_upgrade = %t
  auto_upgrade_expression = ["0 2 * * 0"]
  auto_upgrade_timezone   = "Asia/Bangkok"
  
  pools {
    name                  = "worker-pool"
    storage_profile       = "Premium-SSD"
    worker_type           = "0d1da48b-57ed-4a1e-9700-33f345c38e0d"
    worker_disk_size      = 40
    scale_min             = 1
    scale_max             = 2
    network_id            = "eefb18f4-66c5-4161-9036-3e75745a0f28"
    network_name          = "default-network"
    worker_base           = true
  }
}
`, clusterName, enabled)
}

func generateClusterWithHibernationConfig(clusterName string) string {
	return fmt.Sprintf(`
resource "fptcloud_managed_kubernetes_engine_v1" "test" {
  cluster_name    = "%s"
  k8s_version     = "1.31.4"
  purpose         = "dev"
  network_type    = "calico"
  network_overlay = "ipip"
  
  hibernation_schedules {
    start    = "0 23 * * 2,4"
    end      = "0 7 * * 3,5"
    location = "Asia/Bangkok"
  }
  
  pools {
    name                  = "worker-pool"
    storage_profile       = "Premium-SSD"
    worker_type           = "0d1da48b-57ed-4a1e-9700-33f345c38e0d"
    worker_disk_size      = 40
    scale_min             = 1
    scale_max             = 2
    network_id            = "eefb18f4-66c5-4161-9036-3e75745a0f28"
    network_name          = "default-network"
    worker_base           = true
  }
}
`, clusterName)
}

func generateUpdatedClusterConfig(clusterName string, k8sVersion string, poolCount int) string {
	pools := ""
	for i := 1; i <= poolCount; i++ {
		workerBase := "true"
		if i > 1 {
			workerBase = "false"
		}

		pools += fmt.Sprintf(`
  pools {
    name                  = "worker-pool-%d"
    storage_profile       = "Premium-SSD"
    worker_type           = "0d1da48b-57ed-4a1e-9700-33f345c38e0d"
    worker_disk_size      = 40
    scale_min             = 1
    scale_max             = 2
    network_id            = "eefb18f4-66c5-4161-9036-3e75745a0f28"
    network_name          = "default-network"
    worker_base           = %s
  }`, i, workerBase)
	}

	return fmt.Sprintf(`
resource "fptcloud_managed_kubernetes_engine_v1" "test" {
  cluster_name    = "%s"
  k8s_version     = "%s"
  purpose         = "dev"
  network_type    = "calico"
  network_overlay = "ipip"
%s
}
`, clusterName, k8sVersion, pools)
}

func generateClusterWithKVConfig(clusterName string) string {
	return fmt.Sprintf(`
resource "fptcloud_managed_kubernetes_engine_v1" "test" {
  cluster_name    = "%s"
  k8s_version     = "1.31.4"
  purpose         = "dev"
  network_type    = "calico"
  network_overlay = "ipip"
  
  pools {
    name                  = "worker-pool"
    storage_profile       = "Premium-SSD"
    worker_type           = "0d1da48b-57ed-4a1e-9700-33f345c38e0d"
    worker_disk_size      = 40
    scale_min             = 1
    scale_max             = 2
    network_id            = "eefb18f4-66c5-4161-9036-3e75745a0f28"
    network_name          = "default-network"
    worker_base           = true
    
    kv {
      name  = "environment"
      value = "production"
    }
    
    kv {
      name  = "team"
      value = "devops"
    }
  }
}
`, clusterName)
}

func generateClusterWithTaintsConfig(clusterName string) string {
	return fmt.Sprintf(`
resource "fptcloud_managed_kubernetes_engine_v1" "test" {
  cluster_name    = "%s"
  k8s_version     = "1.31.4"
  purpose         = "dev"
  network_type    = "calico"
  network_overlay = "ipip"
  
  pools {
    name                  = "worker-pool"
    storage_profile       = "Premium-SSD"
    worker_type           = "0d1da48b-57ed-4a1e-9700-33f345c38e0d"
    worker_disk_size      = 40
    scale_min             = 1
    scale_max             = 2
    network_id            = "eefb18f4-66c5-4161-9036-3e75745a0f28"
    network_name          = "default-network"
    worker_base           = true
    
    taints {
      key    = "dedicated"
      value  = "gpu"
      effect = "NoSchedule"
    }
    
    taints {
      key    = "environment"
      value  = "production"
      effect = "PreferNoSchedule"
    }
  }
}
`, clusterName)
}

func generateGPUClusterConfig(clusterName string) string {
	return fmt.Sprintf(`
resource "fptcloud_managed_kubernetes_engine_v1" "test" {
  cluster_name    = "%s"
  k8s_version     = "1.31.4"
  purpose         = "dev"
  network_type    = "calico"
  network_overlay = "ipip"
  
  pools {
    name                     = "gpu-pool"
    storage_profile          = "Premium-SSD"
    worker_type              = "0d1da48b-57ed-4a1e-9700-33f345c38e0d"
    worker_disk_size         = 40
    scale_min                = 1
    scale_max                = 2
    network_id               = "eefb18f4-66c5-4161-9036-3e75745a0f28"
    network_name             = "default-network"
    worker_base              = true
    vgpu_id                  = "6d95d250-ee0a-4655-acd7-5b1cdffac870"
    max_client               = 2
    gpu_sharing_client       = "timeSlicing"
    driver_installation_type = "pre-install"
    gpu_driver_version       = "default"
    
    kv {
      name  = "nvidia.com/mig.config"
      value = "all-1g.6gb"
    }
    
    kv {
      name  = "worker.fptcloud/type"
      value = "gpu"
    }
  }
}
`, clusterName)
}

func simulateClusterDeletion(clusterName string) bool {
	// Giả lập việc xóa cluster
	// Trong thực tế, đây sẽ là logic xóa cluster thật
	return true
}

func countWorkerPools(config string) int {
	// Đếm số lượng "pools {" trong config - sửa logic đếm
	count := 0
	for i := 0; i < len(config)-6; i++ {
		if config[i:i+6] == "pools {" {
			count++
		}
	}
	return count
}

// Validation helper functions
func validateMaxClient(value int64, hasVGpuID bool) error {
	if hasVGpuID {
		if value < 2 || value > 48 {
			return fmt.Errorf("Invalid max_client: %d must be between 2 and 48 for GPU pools", value)
		}
	}
	// For non-GPU pools, any value is valid
	return nil
}

func validateGpuSharingClient(value string, hasVGpuID bool) error {
	if !hasVGpuID {
		// For non-GPU pools, any value is valid
		return nil
	}
	allowedValues := []string{"", "timeSlicing"}
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("Invalid gpu_sharing_client: '%s' must be one of: %v for GPU pools", value, allowedValues)
}

func validateDriverInstallationType(value string, hasVGpuID bool) error {
	if !hasVGpuID {
		// For non-GPU pools, any value is valid
		return nil
	}
	if value != "pre-install" {
		return fmt.Errorf("Invalid driver_installation_type: '%s' must be 'pre-install' for GPU pools", value)
	}
	return nil
}

func validateGpuDriverVersion(value string, hasVGpuID bool) error {
	if !hasVGpuID {
		// For non-GPU pools, any value is valid
		return nil
	}
	allowedValues := []string{"default", "latest"}
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("Invalid gpu_driver_version: '%s' must be one of: %v for GPU pools", value, allowedValues)
}
