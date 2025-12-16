package commons

import "fmt"

const ObjectStorageApiPrefix = "/v1/vmware/vpc"

var ApiPath = struct {
	SSH                        string
	Storage                    func(vpcId string) string
	StorageUpdateAttached      func(vpcId string, storageId string) string
	UpdateStorageTags          func(vpcId string, storageId string) string
	StoragePolicy              func(vpcId string) string
	Flavor                     func(vpcId string) string
	GetFlavorByName            func(vpcId string) string
	Image                      func(vpcId string) string
	SecurityGroup              func(vpcId string) string
	UpdateSecurityGroupTags    func(vpcId string, securityGroupId string) string
	RenameSecurityGroup        func(vpcId string, securityGroupId string) string
	UpdateApplyToSecurityGroup func(vpcId string, securityGroupId string) string
	SecurityGroupRule          func(vpcId string, securityGroupRuleId string) string
	CreateSecurityGroupRule    func(vpcId string) string
	Instance                   func(vpcId string) string
	RenameInstance             func(vpcId string, instanceId string) string
	ChangeStatusInstance       func(vpcId string, instanceId string) string
	ResizeInstance             func(vpcId string, instanceId string) string
	UpdateInstanceTags         func(vpcId string, instanceId string) string
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
	UpdateFloatingIpTags       func(vpcId string, floatingIpId string) string
	ListIpAddress              func(vpcId string) string
	AssociateFloatingIp        func(vpcId string) string
	DisassociateFloatingIp     func(vpcId string, floatingIpId string) string
	CreateSubnet               func(vpcId string) string
	DeleteSubnet               func(vpcId string, subnetId string) string
	FindSubnetByName           func(vpcId string) string
	FindSubnet                 func(vpcId string, subnetId string) string
	UpdateSubnetTags           func(vpcId string, subnetId string) string
	ListSubnets                func(vpcId string) string

	Subnet          func(vpcId string) string
	EdgeGatewayList func(vpcId string) string

	DatabaseGet    		func(databaseId string) string
	DatabaseCreate 		func() string
	DatabaseDelete 		func(databaseId string) string
	DatabaseStop   		func() string
	DatabaseStart  		func() string
	DatabaseApplyTags	func() string

	// Dedicated FKE
	DedicatedFKEList           func(vpcId string, page, pageSize int) string
	DedicatedFKEGet            func(vpcId string, clusterId string) string
	DedicatedFKEUpgradeVersion func(vpcId string, clusterId string) string
	DedicatedFKEManagement     func(vpcId string, clusterId string) string

	// Managed FKE
	ManagedFKEList                      func(vpcId string, page int, pageSize int, infraType string) string
	ManagedFKEGet                       func(vpcId string, platform string, clusterId string) string
	ManagedFKEDelete                    func(vpcId string, platform string, clusterName string) string
	ManagedFKECreate                    func(vpcId string, platform string) string
	ManagedFKEUpgradeVersion            func(vpcId string, platform string, clusterId string, targetVersion string) string
	ManagedFKEHibernate                 func(vpcId string, platform string, clusterId string, isWakeup bool) string
	ManagedFKEHibernationSchedules      func(vpcId string, platform string, clusterId string) string
	ManagedFKEAutoUpgradeVersion        func(vpcId string, platform string, clusterId string) string
	ManagedFKEConfigWorker              func(vpcId string, platform string, clusterId string) string
	ManagedFKEUpdateEndpointCIDR        func(vpcId string, platform string, clusterId string) string
	ManagedFKEUpdateClusterAutoscaler   func(vpcId string, platform string, clusterId string) string
	ManagedFKECheckEnableServiceAccount func(vpcId string, platform string) string
	ManagedFKECheckQuotaResource        func(vpcId string, platform string) string
	ManagedFKEStoragePolicy             func(vpcId string) string

	// GPU
	GetGPUInfo func(vpcId string) string

	// Object Storage
	// Common
	CheckS3ServiceEnable func(vpcId string) string

	// Bucket
	ListBuckets  func(vpcId, s3ServiceId string, page, pageSize int) string
	CreateBucket func(vpcId, s3ServiceId string) string
	DeleteBucket func(vpcId, s3ServiceId string) string
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
	GetBucketLifecycle    func(vpcId, s3ServiceId, bucketName string, page, pageSize int) string
	PutBucketLifecycle    func(vpcId, s3ServiceId, bucketName string) string
	DeleteBucketLifecycle func(vpcId, s3ServiceId, bucketName string) string
	// Bucket CORS
	GetBucketCORS    func(vpcId, s3ServiceId, bucketName string, page, pageSize int) string
	PutBucketCORS    func(vpcId, s3ServiceId, bucketName string) string
	CreateBucketCors func(vpcId, s3ServiceId, bucketName string) string
	// Bucket ACL
	GetBucketAcl func(vpcId, s3ServiceId, bucketName string) string
	PutBucketAcl func(vpcId, s3ServiceId, bucketName string) string

	// Sub-user
	ListSubUsers           func(vpcId, s3ServiceId string, page, pageSize int) string
	CreateSubUser          func(vpcId, s3ServiceId string) string
	UpdateSubUser          func(vpcId, s3ServiceId, subUserId string) string
	DeleteSubUser          func(vpcId, s3ServiceId, subUserId string) string
	DetailSubUser          func(vpcId, s3ServiceId, subUserId string) string
	CreateSubUserAccessKey func(vpcId, s3ServiceId, subUserId string) string
	DeleteSubUserAccessKey func(vpcId, s3ServiceId, subUserId string) string
	// Access Key
	ListAccessKeys  func(vpcId, s3ServiceId string) string
	CreateAccessKey func(vpcId, s3ServiceId string) string
	DeleteAccessKey func(vpcId, s3ServiceId string) string

	//LBv2
	//Load balancer
	ListLoadBalancers  func(vpcId string, page int, pageSize int) string
	GetLoadBalancer    func(vpcId string, loadBalancerId string) string
	ReadLoadBalancer   func(vpcId string, loadBalancerId string) string
	CreateLoadBalancer func(vpcId string) string
	UpdateLoadBalancer func(vpcId string, loadBalancerId string) string
	ResizeLoadBalancer func(vpcId string, loadBalancerId string) string
	DeleteLoadBalancer func(vpcId string, loadBalancerId string) string
	//Listener
	ListListeners  func(vpcId string, loadBalancerId string, page int, pageSize int) string
	GetListener    func(vpcId string, listenerId string) string
	ReadListener   func(vpcId string, listenerId string) string
	CreateListener func(vpcId string, loadBalancerId string) string
	UpdateListener func(vpcId string, listenerId string) string
	DeleteListener func(vpcId string, listenerId string) string
	//Pool
	ListPools  func(vpcId string, loadBalancerId string, page int, pageSize int) string
	GetPool    func(vpcId string, poolId string) string
	ReadPool   func(vpcId string, poolId string) string
	CreatePool func(vpcId string, loadBalancerId string) string
	UpdatePool func(vpcId string, poolId string) string
	DeletePool func(vpcId string, poolId string) string
	//Certificate
	ListCertificates  func(vpcId string, page int, pageSize int) string
	GetCertificate    func(vpcId string, certificateId string) string
	ReadCertificate   func(vpcId string, certificateId string) string
	CreateCertificate func(vpcId string) string
	DeleteCertificate func(vpcId string, certificateId string) string
	//L7 policy
	ListL7Policies func(vpcId string, listenerId string) string
	GetL7Policy    func(vpcId string, listenerId string, policyId string) string
	ReadL7Policy   func(vpcId string, listenerId string, policyId string) string
	CreateL7Policy func(vpcId string, listenerId string) string
	UpdateL7Policy func(vpcId string, listenerId string, policyId string) string
	DeleteL7Policy func(vpcId string, listenerId string, policyId string) string
	//L7 rule
	ListL7Rules  func(vpcId string, listenerId string, policyId string) string
	GetL7Rule    func(vpcId string, listenerId string, policyId string, ruleId string) string
	ReadL7Rule   func(vpcId string, listenerId string, policyId string, ruleId string) string
	CreateL7Rule func(vpcId string, listenerId string, policyId string) string
	UpdateL7Rule func(vpcId string, listenerId string, policyId string, ruleId string) string
	DeleteL7Rule func(vpcId string, listenerId string, policyId string, ruleId string) string
	//Size
	ListSizes func(vpcId string) string

	// Tagging
	GetTag    func(tenantId, tagId string) string
	ListTags  func(tenantId string) string
	CreateTag func(tenantId string) string
	UpdateTag func(tenantId, tagId string) string
	DeleteTag func(tenantId, tagId string) string
}{
	SSH: "/v1/user/sshs",
	Storage: func(vpcId string) string {
		return fmt.Sprintf("/v2/vpc/%s/storage", vpcId)
	},
	StorageUpdateAttached: func(vpcId string, storageId string) string {
		return fmt.Sprintf("/v2/vpc/%s/storage/%s/update-attached", vpcId, storageId)
	},
	UpdateStorageTags: func(vpcId string, storageId string) string {
		return fmt.Sprintf("/v2/vpc/%s/storage/%s/tags", vpcId, storageId)
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
	UpdateSecurityGroupTags: func(vpcId string, securityGroupId string) string {
		return fmt.Sprintf("/v2/vpc/%s/security-group/%s/tags", vpcId, securityGroupId)
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
	UpdateInstanceTags: func(vpcId string, instanceId string) string {
		return fmt.Sprintf("/v2/vpc/%s/instance/%s/tags", vpcId, instanceId)
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
	UpdateFloatingIpTags: func(vpcId string, floatingIpId string) string {
		return fmt.Sprintf("/v2/vpc/%s/floating-ip/%s/tags", vpcId, floatingIpId)
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
	UpdateSubnetTags: func(vpcId string, subnetId string) string {
		return fmt.Sprintf("/v2/vpc/%s/network/%s/tags", vpcId, subnetId)
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
	DatabaseApplyTags: func() string {
		return "/v1/xplat/database/management/tagging/cluster/apply-tag"
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
	ManagedFKEUpgradeVersion: func(vpcId string, platform string, clusterId string, targetVersion string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/upgrade_version_cluster/shoots/%s/k8s-version/%s",
			vpcId, platform, clusterId, targetVersion,
		)
	},
	ManagedFKEHibernate: func(vpcId string, platform string, clusterId string, isWakeup bool) string {
		action := "hibernate"
		if isWakeup {
			action = "wakeup"
		}
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/hibernation-cluster/shoots/%s/%s",
			vpcId, platform, clusterId, action,
		)
	},
	ManagedFKEHibernationSchedules: func(vpcId string, platform string, clusterId string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/hibernation-cluster/shoots/%s/schedules",
			vpcId, platform, clusterId,
		)
	},

	ManagedFKEAutoUpgradeVersion: func(vpcId string, platform string, clusterId string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/config-auto-upgrade-version/shoots/%s",
			vpcId, platform, clusterId,
		)
	},

	ManagedFKEConfigWorker: func(vpcId string, platform string, clusterId string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/configure-worker-cluster/shoots/%s/0",
			vpcId, platform, clusterId,
		)
	},
	ManagedFKEUpdateEndpointCIDR: func(vpcId string, platform string, clusterId string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/edit-private-cluster-ip/shoots/%s",
			vpcId, platform, clusterId,
		)
	},

	ManagedFKEUpdateClusterAutoscaler: func(vpcId string, platform string, clusterId string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/config-cluster-auto-scaling/shoots/%s",
			vpcId, platform, clusterId,
		)
	},

	ManagedFKECheckEnableServiceAccount: func(vpcId string, platform string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/check-enable-service-account",
			vpcId, platform,
		)
	},

	ManagedFKECheckQuotaResource: func(vpcId string, platform string) string {
		return fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/check-quota-resources",
			vpcId, platform,
		)
	},

	ManagedFKEStoragePolicy: func(vpcId string) string {
		return fmt.Sprintf(
			"/v1/internal/vpc/%s/find_storage_policy",
			vpcId,
		)
	},

	GetGPUInfo: func(vpcId string) string {
		return fmt.Sprintf("/v1/vmware/vgpu/list?vpc_id=%s", vpcId)
	},

	// Object Storage
	// Common
	CheckS3ServiceEnable: func(vpcId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/check-service-enabled?check_unlimited=undefined", vpcId)
	},

	// Bucket
	ListBuckets: func(vpcId, s3ServiceId string, page, pageSize int) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/buckets?page=%d&page_size=%d&s3_service_id=%s", vpcId, page, pageSize, s3ServiceId)
	},
	CreateBucket: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/buckets/create", vpcId, s3ServiceId)
	},

	DeleteBucket: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/buckets/delete", vpcId, s3ServiceId)
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
	GetBucketLifecycle: func(vpcId, s3ServiceId, bucketName string, page, pageSize int) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/lifecycles?page=%d&page_size=%d", vpcId, s3ServiceId, bucketName, page, pageSize)
	},
	PutBucketLifecycle: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/create-bucket-lifecycle-configuration", vpcId, s3ServiceId, bucketName)
	},
	DeleteBucketLifecycle: func(vpcId, s3ServiceId, bucketName string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/delete-bucket-lifecycle-configuration", vpcId, s3ServiceId, bucketName)
	},
	// Bucket CORS
	GetBucketCORS: func(vpcId, s3ServiceId, bucketName string, page, pageSize int) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/bucket/%s/cors?page=%d&page_size=%d", vpcId, s3ServiceId, bucketName, page, pageSize)
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
	ListSubUsers: func(vpcId, serviceId string, page, pageSize int) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/list?page=%d&page_size=%d", vpcId, serviceId, page, pageSize)
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
	DetailSubUser: func(vpcId, s3ServiceId, subUserId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/%s/detail", vpcId, s3ServiceId, subUserId)
	},
	// Sub-user Access Key
	CreateSubUserAccessKey: func(vpcId, s3ServiceId, subUserId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/%s/credentials/create", vpcId, s3ServiceId, subUserId)
	},
	DeleteSubUserAccessKey: func(vpcId, s3ServiceId, subUserId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/sub-users/%s/credentials/delete", vpcId, s3ServiceId, subUserId)
	},

	// Access Key
	ListAccessKeys: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/user/credentials?s3_service_id=%s", vpcId, s3ServiceId)
	},
	CreateAccessKey: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/user/credentials", vpcId, s3ServiceId)
	},
	DeleteAccessKey: func(vpcId, s3ServiceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/s3/%s/user/credentials/delete", vpcId, s3ServiceId)
	},

	//LBv2
	//Load balancer
	ListLoadBalancers: func(vpcId string, page int, pageSize int) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/list?page=%d&page_size=%d", vpcId, page, pageSize)
	},
	GetLoadBalancer: func(vpcId string, loadBalancerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s", vpcId, loadBalancerId)
	},
	ReadLoadBalancer: func(vpcId string, loadBalancerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s/read", vpcId, loadBalancerId)
	},
	CreateLoadBalancer: func(vpcId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/create", vpcId)
	},
	UpdateLoadBalancer: func(vpcId string, loadBalancerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s/update", vpcId, loadBalancerId)
	},
	ResizeLoadBalancer: func(vpcId string, loadBalancerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s/resize", vpcId, loadBalancerId)
	},
	DeleteLoadBalancer: func(vpcId string, loadBalancerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s/delete", vpcId, loadBalancerId)
	},

	//Listener
	ListListeners: func(vpcId string, loadBalancerId string, page int, pageSize int) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s/listeners/list?page=%d&page_size=%d", vpcId, loadBalancerId, page, pageSize)
	},
	GetListener: func(vpcId string, listenerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s", vpcId, listenerId)
	},
	CreateListener: func(vpcId string, loadBalancerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s/listeners/create", vpcId, loadBalancerId)
	},
	UpdateListener: func(vpcId string, listenerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/update", vpcId, listenerId)
	},
	DeleteListener: func(vpcId string, listenerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/delete", vpcId, listenerId)
	},

	//Pool
	ListPools: func(vpcId string, loadBalancerId string, page int, pageSize int) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s/pools/list?page=%d&page_size=%d", vpcId, loadBalancerId, page, pageSize)
	},
	GetPool: func(vpcId string, poolId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/pools/%s", vpcId, poolId)
	},
	CreatePool: func(vpcId string, loadBalancerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/%s/pools/create", vpcId, loadBalancerId)
	},
	UpdatePool: func(vpcId string, poolId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/pools/%s/update", vpcId, poolId)
	},
	DeletePool: func(vpcId string, poolId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/pools/%s/delete", vpcId, poolId)
	},

	//Certificate
	ListCertificates: func(vpcId string, page int, pageSize int) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/certificates?page=%d&page_size=%d", vpcId, page, pageSize)
	},
	GetCertificate: func(vpcId string, certificateId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/certificates/%s", vpcId, certificateId)
	},
	CreateCertificate: func(vpcId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/certificates/create", vpcId)
	},
	DeleteCertificate: func(vpcId string, certificateId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/certificates/%s/delete", vpcId, certificateId)
	},

	//L7 policy
	ListL7Policies: func(vpcId string, listenerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies", vpcId, listenerId)
	},
	GetL7Policy: func(vpcId string, listenerId string, policyId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/%s", vpcId, listenerId, policyId)
	},
	CreateL7Policy: func(vpcId string, listenerId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/create", vpcId, listenerId)
	},
	UpdateL7Policy: func(vpcId string, listenerId string, policyId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/%s/update", vpcId, listenerId, policyId)
	},
	DeleteL7Policy: func(vpcId string, listenerId string, policyId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/%s/delete", vpcId, listenerId, policyId)
	},

	//L7 rule
	ListL7Rules: func(vpcId string, listenerId string, policyId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/%s/rules", vpcId, listenerId, policyId)
	},
	GetL7Rule: func(vpcId string, listenerId string, policyId string, ruleId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/%s/rules/%s", vpcId, listenerId, policyId, ruleId)
	},
	CreateL7Rule: func(vpcId string, listenerId string, policyId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/%s/rules/create", vpcId, listenerId, policyId)
	},
	UpdateL7Rule: func(vpcId string, listenerId string, policyId string, ruleId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/%s/rules/%s/update", vpcId, listenerId, policyId, ruleId)
	},
	DeleteL7Rule: func(vpcId string, listenerId string, policyId string, ruleId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/listeners/%s/l7policies/%s/rules/%s/delete", vpcId, listenerId, policyId, ruleId)
	},

	//Size
	ListSizes: func(vpcId string) string {
		return fmt.Sprintf("/v2/vmware/vpc/%s/load_balancer_v2/sizes", vpcId)
	},

	// Tagging

	ListTags: func(tenantId string) string {
		return fmt.Sprintf("/v2/org/%s/tags", tenantId)
	},
	GetTag: func(tenantId, tagId string) string {
		return fmt.Sprintf("/v2/org/%s/tag/%s", tenantId, tagId)
	},
	CreateTag: func(tenantId string) string {
		return fmt.Sprintf("/v2/org/%s/tag/create", tenantId)
	},
	UpdateTag: func(tenantId, tagId string) string {
		return fmt.Sprintf("/v2/org/%s/tag/%s/update", tenantId, tagId)
	},
	DeleteTag: func(tenantId, tagId string) string {
		return fmt.Sprintf("/v2/org/%s/tag/%s/delete", tenantId, tagId)
	},
}
