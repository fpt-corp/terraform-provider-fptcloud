package commons

import "fmt"

var ApiPath = struct {
	SSH                   string
	Storage               func(vpcId string) string
	StorageUpdateAttached func(vpcId string, storageId string) string
	StoragePolicy         func(vpcId string) string
	Flavor                func(vpcId string) string
	Image                 func(vpcId string) string
	Subnet                func(vpcId string) string
	ManagedFKEList        func(vpcId string, page int, pageSize int) string
	ManagedFKEGet         func(vpcId string, platform string, clusterId string) string
	ManagedFKEDelete      func(vpcId string, platform string, clusterName string) string
	ManagedFKECreate      func(vpcId string, platform string) string
	GetFKEOSVersion       func(vpcId string, platform string) string
}{
	SSH: "/v1/user/sshs",
	Storage: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/storage", vpcId)
	},
	StorageUpdateAttached: func(vpcId string, storageId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/storage/%s/update-attached", vpcId, storageId)
	},
	StoragePolicy: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/storage-policies", vpcId)
	},
	Flavor: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/flavors", vpcId)
	},
	Image: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/images", vpcId)
	},
	Subnet: func(vpcId string) string { return fmt.Sprintf("/v1/vmware/vpc/%s/network/subnets", vpcId) },
	ManagedFKEList: func(vpcId string, page int, pageSize int) string {
		return fmt.Sprintf("/v1/xplat/fke/vpc/%s/m-fke/vmw/get-shoot-cluster/shoots?page=%d&page_size=%d", vpcId, page, pageSize)
	},
	ManagedFKEDelete: func(vpcId string, platform string, clusterName string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/delete-shoot-cluster/shoots/%s",
			vpcId, platform, clusterName,
		)
	},
	ManagedFKECreate: func(vpcId string, platform string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/create-cluster",
			vpcId, platform,
		)
	},
	ManagedFKEGet: func(vpcId string, platform string, clusterId string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/get-shoot-specific/shoots/%s",
			vpcId, platform, clusterId,
		)
	},
	GetFKEOSVersion: func(vpcId string, platform string) string {
		return fmt.Sprintf("v1/xplat/fke/vpc/%s/m-fke/%s/get_k8s_versions", vpcId, platform)
	},
}
