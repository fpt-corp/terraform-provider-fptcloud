package commons

import "fmt"

var ApiPath = struct {
	SSH                        string
	Storage                    func(vpcId string) string
	StorageUpdateAttached      func(vpcId string, storageId string) string
	StoragePolicy              func(vpcId string) string
	Flavor                     func(vpcId string) string
	GetFlavorByName            func(vpcId string) string
	Image                      func(vpcId string) string
	SecurityGroup              func(vpcId string) string
	RenameSecurityGroup        func(vpcId string, securityGroupId string) string
	UpdateApplyToSecurityGroup func(vpcId string, securityGroupId string) string
	SecurityGroupRule          func(vpcId string, securityGroupRuleId string) string
	CreateSecurityGroupRule    func(vpcId string) string
	Instance                   func(vpcId string) string
	RenameInstance             func(vpcId string, instanceId string) string
	ChangeStatusInstance       func(vpcId string, instanceId string) string
	ResizeInstance             func(vpcId string, instanceId string) string
	Tenant                     func(tenantName string) string
	Vpc                        func(tenantId string) string
	VMGroupPolicies            func(vpcId string) string
	CreateInstanceGroup        func(vpcId string) string
	FindInstanceGroup          func(vpcId string) string
	DeleteInstanceGroup        func(vpcId string, instanceGroupId string) string
	CreateFloatingIp           func(vpcId string) string
	FindFloatingIp             func(vpcId string, floatingIpId string) string
	FindFloatingIpByAddress    func(vpcId string) string
	ListFloatingIp             func(vpcId string) string
	DeleteFloatingIp           func(vpcId string, floatingIpId string) string
	ListIpAddress              func(vpcId string) string
	AssociateFloatingIp        func(vpcId string) string
	DisassociateFloatingIp     func(vpcId string, floatingIpId string) string
	CreateSubnet               func(vpcId string) string
	DeleteSubnet               func(vpcId string, subnetId string) string
	FindSubnetByName           func(vpcId string) string
	FindSubnet                 func(vpcId string, subnetId string) string
	ListSubnets                func(vpcId string) string

	Subnet                     func(vpcId string) string
	DedicatedFKEList           func(vpcId string, page, pageSize int) string
	DedicatedFKEGet            func(vpcId string, clusterId string) string
	DedicatedFKEUpgradeVersion func(vpcId string, clusterId string) string
	DedicatedFKEManagement     func(vpcId string, clusterId string) string

	ManagedFKEList   func(vpcId string, page int, pageSize int, infraType string) string
	ManagedFKEGet    func(vpcId string, platform string, clusterId string) string
	ManagedFKEDelete func(vpcId string, platform string, clusterName string) string
	ManagedFKECreate func(vpcId string, platform string) string
	GetFKEOSVersion  func(vpcId string, platform string) string
}{
	SSH: "/v1/user/sshs",
	Storage: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/storage", vpcId)
	},
	StorageUpdateAttached: func(vpcId string, storageId string) string {
		return fmt.Sprintf("/v2/vpc/%s/storage/%s/update-attached", vpcId, storageId)
	},
	StoragePolicy: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/storage-policies", vpcId)
	},
	Flavor: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/flavors", vpcId)
	},
	Image: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/images", vpcId)
	},
	SecurityGroup: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/security-group", vpcId)
	},
	RenameSecurityGroup: func(vpcId string, securityGroupId string) string {
		return fmt.Sprintf("/v2/vpc/%s/security-group/%s/rename", vpcId, securityGroupId)
	},
	UpdateApplyToSecurityGroup: func(vpcId string, securityGroupId string) string {
		return fmt.Sprintf("/v2/vpc/%s/security-group/%s/apply-to", vpcId, securityGroupId)
	},
	SecurityGroupRule: func(vpcId string, securityGroupRuleId string) string {
		return fmt.Sprintf("/v2/vpc/%s/security-group-rule/%s", vpcId, securityGroupRuleId)
	},
	CreateSecurityGroupRule: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/security-group-rule", vpcId)
	},
	Instance: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/instance", vpcId)
	},
	RenameInstance: func(vpcId string, instanceId string) string {
		return fmt.Sprintf("/v2/vpc/%s/instance/%s/rename", vpcId, instanceId)
	},
	ChangeStatusInstance: func(vpcId string, instanceId string) string {
		return fmt.Sprintf("/v2/vpc/%s/instance/%s/change-status", vpcId, instanceId)
	},
	ResizeInstance: func(vpcId string, instanceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/compute/instance/%s/reconfigure-vm", vpcId, instanceId)
	},
	GetFlavorByName: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/flavor/find-by-name", vpcId)
	},
	Tenant: func(tenantName string) string {
		return fmt.Sprintf("/v2/tenant/%s", tenantName)
	},
	Vpc: func(tenantId string) string {
		return fmt.Sprintf("/v2/org/%s/vpc", tenantId)
	},
	VMGroupPolicies: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/vm-group-policies", vpcId)
	},
	CreateInstanceGroup: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/vm-group", vpcId)
	},
	FindInstanceGroup: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/vm-groups", vpcId)
	},
	DeleteInstanceGroup: func(vpcId string, instanceGroupId string) string {
		return fmt.Sprintf("/v2/vpc/%s/vm-group/%s", vpcId, instanceGroupId)
	},
	CreateFloatingIp: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/floating-ip", vpcId)
	},
	FindFloatingIp: func(vpcId string, floatingIpId string) string {
		return fmt.Sprintf("/v2/vpc/%s/floating-ip/%s", vpcId, floatingIpId)
	},
	FindFloatingIpByAddress: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/floating-ip-address", vpcId)
	},
	ListFloatingIp: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/floating-ips", vpcId)
	},
	DeleteFloatingIp: func(vpcId string, floatingIpId string) string {
		return fmt.Sprintf("/v2/vpc/%s/floating-ip/%s/release", vpcId, floatingIpId)
	},
	AssociateFloatingIp: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/floating-ip/associate", vpcId)
	},
	DisassociateFloatingIp: func(vpcId string, floatingIpId string) string {
		return fmt.Sprintf("/v2/vpc/%s/floating-ip/%s/disassociate", vpcId, floatingIpId)
	},
	CreateSubnet: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/networks", vpcId)
	},
	DeleteSubnet: func(vpcId string, subnetId string) string {
		return fmt.Sprintf("/v2/vpc/%s/network/%s", vpcId, subnetId)
	},
	FindSubnetByName: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/network-by-name", vpcId)
	},
	FindSubnet: func(vpcId string, subnetId string) string {
		return fmt.Sprintf("/v2/vpc/%s/network/%s", vpcId, subnetId)
	},
	ListSubnets: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/networks", vpcId)
	},

	Subnet: func(vpcId string) string { return fmt.Sprintf("/v1/vmware/vpc/%s/network/subnets", vpcId) },

	DedicatedFKEList: func(vpcId string, page, pageSize int) string {
		return fmt.Sprintf("/v1/xplat/fke/vpc/%s/kubernetes?page=%d&page_size=%d", vpcId, page, pageSize)
	},
	DedicatedFKEGet: func(vpcId string, clusterId string) string {
		return fmt.Sprintf("/v1/xplat/fke/vpc/%s/cluster/%s?page=1&page_size=25", vpcId, clusterId)
	},
	DedicatedFKEUpgradeVersion: func(vpcId string, clusterId string) string {
		return fmt.Sprintf("/v1/xplat/fke/vpc/%s/cluster/%s/upgrade-version", vpcId, clusterId)
	},
	DedicatedFKEManagement: func(vpcId string, clusterId string) string {
		return fmt.Sprintf("/v1/xplat/fke/vpc/%s/kubernetes/%s/management", vpcId, clusterId)
	},

	ManagedFKEList: func(vpcId string, page int, pageSize int, infraType string) string {
		return fmt.Sprintf("/v1/xplat/fke/vpc/%s/m-fke/%s/get-shoot-cluster/shoots?page=%d&page_size=%d", vpcId, infraType, page, pageSize)
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
		return fmt.Sprintf("/v1/xplat/fke/vpc/%s/m-fke/%s/get_k8s_versions", vpcId, platform)
	},
}
