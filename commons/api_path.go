package commons

import "fmt"

const ObjectStorageApiPrefix = "/v1/vmware/vpc"

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

	Subnet          func(vpcId string) string
	EdgeGatewayList func(vpcId string) string

	DatabaseGet    func(databaseId string) string
	DatabaseCreate func() string
	DatabaseDelete func(databaseId string) string
	DatabaseStop   func() string
	DatabaseStart  func() string

	DedicatedFKEList           func(vpcId string, page, pageSize int) string
	DedicatedFKEGet            func(vpcId string, clusterId string) string
	DedicatedFKEUpgradeVersion func(vpcId string, clusterId string) string
	DedicatedFKEManagement     func(vpcId string, clusterId string) string

	ManagedFKEList   func(vpcId string, page int, pageSize int, infraType string) string
	ManagedFKEGet    func(vpcId string, platform string, clusterId string) string
	ManagedFKEDelete func(vpcId string, platform string, clusterName string) string
	ManagedFKECreate func(vpcId string, platform string) string
	GetFKEOSVersion  func(vpcId string, platform string) string

	// Object Storage
	// Common
	CheckS3ServiceEnable func(vpcId string) string

	// Bucket
	ListBuckets  func(vpcId, s3ServiceId string, page, pageSize int) string
	CreateBucket func(vpcId, s3ServiceId string) string
	DeleteBucket func(vpcId, s3ServiceId, bucketName string) string
	// Bucket Policy
	GetBucketPolicy func(vpcId, s3ServiceId, bucketName string) string
	PutBucketPolicy func(vpcId, s3ServiceId, bucketName string) string
	// Bucket Static Website
	GetBucketWebsite          func(vpcId, s3ServiceId, bucketName string) string
	PutBucketWebsite          func(vpcId, s3ServiceId, bucketName string) string
	DeleteBucketStaticWebsite func(vpcId, s3ServiceId, bucketName string) string
	// Bucket Versioning
	GetBucketVersioning func(vpcId, s3ServiceId, bucketName string) string
	PutBucketVersioning func(vpcId, s3ServiceId, bucketName string) string
	// Bucket Lifecycle
	GetBucketLifecycle    func(vpcId, s3ServiceId, bucketName, page, pageSize string) string
	PutBucketLifecycle    func(vpcId, s3ServiceId, bucketName string) string
	DeleteBucketLifecycle func(vpcId, s3ServiceId, bucketName string) string
	// Bucket CORS
	GetBucketCORS    func(vpcId, s3ServiceId, bucketName string) string
	PutBucketCORS    func(vpcId, s3ServiceId, bucketName string) string
	CreateBucketCors func(vpcId, s3ServiceId, bucketName string) string
	// Bucket ACL
	GetBucketAcl func(vpcId, s3ServiceId, bucketName string) string
	PutBucketAcl func(vpcId, s3ServiceId, bucketName string) string

	// Sub-user
	ListSubUsers           func(vpcId, s3ServiceId string) string
	CreateSubUser          func(vpcId, s3ServiceId string) string
	UpdateSubUser          func(vpcId, s3ServiceId, subUserId string) string
	DeleteSubUser          func(vpcId, s3ServiceId, subUserId string) string
	DetailSubUser          func(vpcId, s3ServiceId, subUserId string) string
	CreateSubUserAccessKey func(vpcId, s3ServiceId, subUserId string) string
	DeleteSubUserAccessKey func(vpcId, s3ServiceId, subUserId, accessKeyId string) string
	// Access Key
	ListAccessKeys  func(vpcId, s3ServiceId string) string
	CreateAccessKey func(vpcId, s3ServiceId string) string
	DeleteAccessKey func(vpcId, s3ServiceId string) string
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

	EdgeGatewayList: func(vpcId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/edge_gateway/list", vpcId)
	},

	DatabaseGet: func(databaseId string) string {
		return fmt.Sprintf("/v1/xplat/database/management/cluster/detail/%s", databaseId)
	},
	DatabaseCreate: func() string {
		return "/v1/xplat/database/provision/create"
	},
	DatabaseDelete: func(databaseId string) string {
		return fmt.Sprintf("/v1/xplat/database/provision/delete/%s", databaseId)
	},
	DatabaseStop: func() string {
		return "/v1/xplat/database/management/cluster/stop"
	},
	DatabaseStart: func() string {
		return "/v1/xplat/database/management/cluster/start"
	},

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

	// Object Storage
	// Common
	CheckS3ServiceEnable: func(vpcId string) string {
		fmt.Println("vpcId: ", vpcId)
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/check-service-enabled?check_unlimited=undefined", vpcId)
	},

	// Bucket
	ListBuckets: func(vpcId, s3ServiceId string, page, pageSize int) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/buckets?page=%d&page_size=%d&s3_service_id=%s", vpcId, page, pageSize, s3ServiceId)
	},
	CreateBucket: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/buckets/create", vpcId, s3ServiceId)
	},

	DeleteBucket: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/buckets/%s/delete", vpcId, s3ServiceId, bucketName)
	},

	// Bucket Versioning
	GetBucketVersioning: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/get-versioning", vpcId, s3ServiceId, bucketName)
	},
	PutBucketVersioning: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/put-versioning", vpcId, s3ServiceId, bucketName)
	},
	// Bucket Policy
	GetBucketPolicy: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/get-policy", vpcId, s3ServiceId, bucketName)
	},
	PutBucketPolicy: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/put-policy", vpcId, s3ServiceId, bucketName)
	},
	// Bucket Static Website
	GetBucketWebsite: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/get-config", vpcId, s3ServiceId, bucketName)
	},
	PutBucketWebsite: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/put-config", vpcId, s3ServiceId, bucketName)
	},
	DeleteBucketStaticWebsite: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/delete-config", vpcId, s3ServiceId, bucketName)
	},
	// Bucket Lifecycle
	GetBucketLifecycle: func(vpcId, s3ServiceId, bucketName, page, pageSize string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/lifecycles?page=%s&page_size=%s", vpcId, s3ServiceId, bucketName, page, pageSize)
	},
	PutBucketLifecycle: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/create-bucket-lifecycle-configuration`", vpcId, s3ServiceId, bucketName)
	},
	DeleteBucketLifecycle: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/delete-bucket-lifecycle-configuration", vpcId, s3ServiceId, bucketName)
	},
	// Bucket CORS
	GetBucketCORS: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/cors", vpcId, s3ServiceId, bucketName)
	},
	PutBucketCORS: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/put-bucket-cors", vpcId, s3ServiceId, bucketName)
	},
	CreateBucketCors: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/create-bucket-cors", vpcId, s3ServiceId, bucketName)
	},
	// Bucket ACL
	GetBucketAcl: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/acl", vpcId, s3ServiceId, bucketName)
	},
	PutBucketAcl: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/acl", vpcId, s3ServiceId, bucketName)
	},
	// Sub-user
	ListSubUsers: func(vpcId, serviceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/list", vpcId, serviceId)
	},
	CreateSubUser: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/create", vpcId, s3ServiceId)
	},
	UpdateSubUser: func(vpcId, s3ServiceId, subUserId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/sub-users/%s/update", vpcId, subUserId)
	},
	DeleteSubUser: func(vpcId, s3ServiceId, subUserId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/%s/delete", vpcId, s3ServiceId, subUserId)
	},
	CreateSubUserAccessKey: func(vpcId, s3ServiceId, subUserId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/%s/credentials/create", vpcId, s3ServiceId, subUserId)
	},
	DeleteSubUserAccessKey: func(vpcId, s3ServiceId, subUserId, accessKeyId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/%s/credentials/%s/delete", vpcId, s3ServiceId, subUserId, accessKeyId)
	},
	DetailSubUser: func(vpcId, s3ServiceId, subUserId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/%s/detail", vpcId, s3ServiceId, subUserId)
	},

	// Access Key
	ListAccessKeys: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/user/credentials?s3_service_id=%s", vpcId, s3ServiceId)
	},
	CreateAccessKey: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/user/credentials", vpcId, s3ServiceId)
	},
	// https://console-api.fptcloud.com/api/v1/vmware/vpc/1dce0aa0-a78d-4e19-89a3-d688bcff7f1b/s3/d8c82109-3d17-4ac2-8b21-5fedb2d81c54/user/credentials/delete
	DeleteAccessKey: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/user/credentials/delete", vpcId, s3ServiceId)
	},
}
