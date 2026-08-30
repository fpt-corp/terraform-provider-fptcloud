package fptcloud_mgpu_cluster

import (
	"fmt"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_ssh "terraform-provider-fptcloud/fptcloud/ssh"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type managedGpuCluster struct {
	Id             types.String             `tfsdk:"id"`
	VpcId          types.String             `tfsdk:"vpc_id"`
	ClusterName    types.String             `tfsdk:"cluster_name"`
	NetworkID      types.String             `tfsdk:"network_id"`
	K8SVersion     types.String             `tfsdk:"k8s_version"`
	Purpose        types.String             `tfsdk:"purpose"`
	Pools          []*managedGpuClusterPool `tfsdk:"pools"`
	PodNetwork     types.String             `tfsdk:"pod_network"`
	PodPrefix      types.String             `tfsdk:"pod_prefix"`
	ServiceNetwork types.String             `tfsdk:"service_network"`
	ServicePrefix  types.String             `tfsdk:"service_prefix"`
	K8SMaxPod      types.Int64              `tfsdk:"k8s_max_pod"`
	NetworkType    types.String             `tfsdk:"network_type"`
	EdgeGatewayId  types.String             `tfsdk:"edge_gateway_id"`
	// New block fields
	ClusterAutoscaler     types.Object `tfsdk:"cluster_autoscaler"`
	ClusterEndpointAccess types.Object `tfsdk:"cluster_endpoint_access"`
	IsEnableAutoUpgrade   types.Bool   `tfsdk:"is_enable_auto_upgrade"`
	AutoUpgradeExpression types.List   `tfsdk:"auto_upgrade_expression"`
	AutoUpgradeTimezone   types.String `tfsdk:"auto_upgrade_timezone"`
	InternalSubnetLb      types.String `tfsdk:"internal_subnet_lb"`
	EdgeGatewayName       types.String `tfsdk:"edge_gateway_name"`
	IsRunning             types.Bool   `tfsdk:"is_running"`
	HibernationSchedules  types.List   `tfsdk:"hibernation_schedules"`

	// Bare-metal-only fields.
	NetworkNodePrefix types.Int64  `tfsdk:"network_node_prefix"`
	SshId             types.String `tfsdk:"ssh_key_id"`
	Software          types.Object `tfsdk:"software"`
}

// Software is the operator installed on the cluster: one of the four types,
// at one of the versions that type offers. Mirrors the "GPU information" step
// of the console, where the user picks a single software.
//
// cluster_mig_strategy applies only when software_type is gpu_operator.
type Software struct {
	SoftwareType       types.String `tfsdk:"software_type"`
	SoftwareVersion    types.String `tfsdk:"software_version"`
	ClusterMigStrategy types.String `tfsdk:"cluster_mig_strategy"`
}

type ClusterAutoscaler struct {
	IsEnableAutoScaling           types.Bool    `tfsdk:"is_enable_auto_scaling"`
	ScaleDownDelayAfterAdd        types.Int64   `tfsdk:"scale_down_delay_after_add"`     // seconds
	ScaleDownDelayAfterDelete     types.Int64   `tfsdk:"scale_down_delay_after_delete"`  // seconds
	ScaleDownDelayAfterFailure    types.Int64   `tfsdk:"scale_down_delay_after_failure"` // seconds
	ScaleDownUnneededTime         types.Int64   `tfsdk:"scale_down_unneeded_time"`       // seconds
	ScaleDownUtilizationThreshold types.Float64 `tfsdk:"scale_down_utilization_threshold"`
	ScanInterval                  types.Int64   `tfsdk:"scan_interval"` // seconds
	Expander                      types.String  `tfsdk:"expander"`
}

type ClusterEndpointAccess struct {
	Type      types.String `tfsdk:"type"`
	AllowCidr types.List   `tfsdk:"allow_cidr"`
}

type resourceManagedGpuCluster struct {
	client            *commons.Client
	mgpuClusterClient *MgpuClusterApiClient
	subnetClient      fptcloud_subnet.SubnetService
	sshClient         fptcloud_ssh.SSHKeyService
	tenancyClient     *fptcloud_dfke.TenancyApiClient
}

type managedGpuClusterPool struct {
	WorkerBase       types.Bool   `tfsdk:"worker_base"`
	WorkerPoolID     types.String `tfsdk:"name"`
	HpcFlavorId      types.String `tfsdk:"hpc_flavor_id"`
	HpcFlavorName    types.String `tfsdk:"hpc_flavor_name"`
	HpcNumberServer  types.Int64  `tfsdk:"hpc_number_server"`
	ContainerRuntime types.String `tfsdk:"container_runtime"`
	NetworkID        types.String `tfsdk:"network_id"`
	NetworkName      types.String `tfsdk:"network_name"`
	Tags             types.List   `tfsdk:"tags"`
	Kv               types.Set    `tfsdk:"kv"`
	Taints           types.Set    `tfsdk:"taints"`
	MaxClient        types.Int64  `tfsdk:"max_client"`
	GpuDriver        types.Object `tfsdk:"gpu_driver"`
	GpuType          types.String `tfsdk:"gpu_type"`
	MigProfile       types.String `tfsdk:"mig_profile"`
}

// GpuDriver groups the two fields that pick a GPU driver for a pool:
// installation_type selects the catalog (MANAGED/PRE_INSTALL/USER_INSTALL),
// version is checked against the versions that catalog offers.
type GpuDriver struct {
	InstallationType types.String `tfsdk:"installation_type"`
	Version          types.String `tfsdk:"version"`
}

type KV struct {
	Name  types.String `tfsdk:"name" json:"name"`
	Value types.String `tfsdk:"value" json:"value"`
}

type Taint struct {
	Key    types.String `tfsdk:"key" json:"key"`
	Value  types.String `tfsdk:"value" json:"value"`
	Effect types.String `tfsdk:"effect" json:"effect"`
}

type managedGpuClusterJson struct {
	ClusterName           string                       `json:"cluster_name"`
	NetworkID             string                       `json:"network_id"`
	VmSubnet              string                       `json:"vm_subnet,omitempty"`
	OspNetworkId          string                       `json:"osp_network_id,omitempty"`
	K8SVersion            string                       `json:"k8s_version,omitempty"`
	IsV2                  bool                         `json:"isV2,omitempty"`
	Purpose               string                       `json:"purpose,omitempty"`
	Pools                 []*managedGpuClusterPoolJson `json:"pools"`
	PodNetwork            string                       `json:"pod_network,omitempty"`
	PodPrefix             string                       `json:"pod_prefix,omitempty"`
	ServiceNetwork        string                       `json:"service_network,omitempty"`
	ServicePrefix         string                       `json:"service_prefix,omitempty"`
	K8SMaxPod             int64                        `json:"k8s_max_pod,omitempty"`
	NetworkType           string                       `json:"network_type,omitempty"`
	NetworkOverlay        string                       `json:"network_overlay,omitempty"`
	InternalSubnetLb      interface{}                  `json:"internal_subnet_lb,omitempty"`
	EdgeGatewayId         string                       `json:"edge_gateway_id,omitempty"`
	EdgeGatewayName       string                       `json:"edge_gateway_name,omitempty"`
	ClusterEndpointAccess *ClusterEndpointAccessJson   `json:"clusterEndpointAccess,omitempty"`
	IsEnableAutoUpgrade   bool                         `json:"is_enable_auto_upgrade,omitempty"`
	AutoUpgradeExpression []string                     `json:"auto_upgrade_expression,omitempty"`
	AutoUpgradeTimezone   string                       `json:"auto_upgrade_timezone,omitempty"`
	ClusterAutoscaler     interface{}                  `json:"cluster_autoscaler,omitempty"`
	TypeCreate            string                       `json:"type_create,omitempty"`
	// Hps is always sent as null. Seen in a real create-cluster request with
	// that exact value; meaning unconfirmed with the backend, so no Terraform
	// field backs it yet.
	Hps interface{} `json:"hps"`

	// Bare-metal-only fields, absent from the managed FKE create body.
	DefaultStorageProfile string                 `json:"default_storage_profile,omitempty"`
	CurrentNetworking     string                 `json:"currentNetworking,omitempty"`
	NetworkNodePrefix     int64                  `json:"network_node_prefix,omitempty"`
	LoadBalancerType      string                 `json:"loadBalancerType,omitempty"`
	LbInternalNetwork     *LbInternalNetworkJson `json:"lbInternalNetwork,omitempty"`
	SecretBindingName     string                 `json:"secret_binding_name,omitempty"`
	SshId                 string                 `json:"ssh_id,omitempty"`
	SshName               string                 `json:"ssh_name,omitempty"`
	SshPublicKey          string                 `json:"ssh_public_key,omitempty"`
	Software              *SoftwareJson          `json:"software,omitempty"`
}

// SoftwareJson is the selected operator sent with create-cluster, keyed the
// same way the available-versions catalog is.
type SoftwareJson struct {
	SoftwareType       string `json:"software_type"`
	SoftwareVersion    string `json:"software_version"`
	ClusterMigStrategy string `json:"cluster_mig_strategy,omitempty"`
}

// LbInternalNetworkJson is the internal load balancer network selection sent
// with create-cluster.
type LbInternalNetworkJson struct {
	Cidr          string `json:"cidr"`
	Label         string `json:"label"`
	Label4Sending string `json:"label4sending"`
	Value         string `json:"value"`
	NetworkType   string `json:"networkType"`
}

type ClusterEndpointAccessJson struct {
	Type      string   `json:"type"`
	AllowCidr []string `json:"allowCidr"`
}

// managedGpuClusterPoolJson is one entry of the create-cluster "pools" array.
//
// Bare metal sizes a pool by an HPC flavor plus a server count
// (hpc_flavor_id / hpc_flavor_name / hpc_number_server) rather than by the
// worker_type + scale_min/scale_max triple the managed FKE API uses.
type managedGpuClusterPoolJson struct {
	// int64 fields
	HpcNumberServer int64 `json:"hpc_number_server"`
	MaxClient       int64 `json:"maxClient"`
	DeltaQuotaScale int64 `json:"deltaQuotaScale"`

	// pointer fields
	WorkerPoolID *string `json:"worker_pool_id"`

	// string fields
	StorageProfile         string `json:"storage_profile"`
	HpcFlavorId            string `json:"hpc_flavor_id"`
	HpcFlavorName          string `json:"hpc_flavor_name"`
	NetworkID              string `json:"network_id"`
	NetworkName            string `json:"network_name"`
	VGpuID                 string `json:"vGpuId"`
	DriverInstallationType string `json:"driverInstallationType"`
	GpuDriverVersion       string `json:"gpuDriverVersion"`
	GpuTemplateVersion     string `json:"gpuTemplateVersion"`
	Tags                   string `json:"tags"`
	GpuSharingClient       string `json:"gpuSharingClient"`
	ContainerRuntime       string `json:"container_runtime"`
	Kubernetes             string `json:"kubernetes,omitempty"`
	GpuType                string `json:"gpuType,omitempty"`
	MigProfile             string `json:"migProfile,omitempty"`

	// Resource shape of the pool. Derived from the selected HPC flavor;
	// left empty until a flavor lookup endpoint is available, in which case
	// the backend fills them in.
	Ram       string `json:"ram,omitempty"`
	Cpu       string `json:"cpu,omitempty"`
	GpuAmount string `json:"gpu_amount,omitempty"`

	// slice fields
	Kv     []map[string]string      `json:"kv"`
	Taints []map[string]interface{} `json:"taints"`

	// bool fields
	AutoScale          bool `json:"auto_scale"`
	IsDisplayGPU       bool `json:"isDisplayGPU"`
	IsCreate           bool `json:"isCreate"`
	IsScale            bool `json:"isScale"`
	IsOthers           bool `json:"isOthers"`
	IsEnableAutoRepair bool `json:"isEnableAutoRepair"`
	WorkerBase         bool `json:"worker_base"`
}

// hpcSubnet is one entry of GET .../vmware/vpc/{vpcId}/hpc/subnets. This is
// the OSP-specific subnet catalog create-cluster needs vm_subnet (the CIDR)
// and osp_network_id from — neither is derivable from the regular
// fptcloud_subnet listing, which does not carry an OSP network ID.
type hpcSubnet struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SubnetCidr   string `json:"subnet_cidr"`
	Description  string `json:"description"`
	Gateway      string `json:"gateway"`
	Status       string `json:"status"`
	OspNetworkId string `json:"osp_network_id"`
	NetworkAclId string `json:"network_acl_id"`
}

// hpcSubnetListResponse is the response of GET .../vmware/vpc/{vpcId}/hpc/subnets.
type hpcSubnetListResponse struct {
	Data  []hpcSubnet `json:"data"`
	Total int         `json:"total"`
}

// gpuDriverEntry is one entry of the gpu-drivers catalog: a driver version
// (value) offered for a given driver_installation_type, with its display
// label and backing image. USER_INSTALL reports a single entry with an empty
// value — there is no version to pick, the user installs their own driver.
type gpuDriverEntry struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	ImageID string `json:"imageID"`
}

// gpuDriverListResponse is the response of GET .../fke-gpu/common/vpc/{vpcId}/gpu-drivers.
type gpuDriverListResponse struct {
	DriverType string           `json:"driver_type"`
	DriverList []gpuDriverEntry `json:"driver_list"`
}

// networkSubnet is one entry of the network/subnets listing. That endpoint is the only
// one exposing networkType and the prefix length, both required by config-internal-subnet-lb.
// Beware: it swaps the meaning of id and network_id compared to the /networks endpoint
// backing fptcloud_subnet.SubnetService, so id here is what SubnetService calls NetworkID.
type networkSubnet struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	DefaultGateway     string `json:"defaultGateway"`
	SubnetPrefixLength int    `json:"subnetPrefixLength"`
	NetworkID          string `json:"network_id"`
	NetworkType        string `json:"networkType"`
}

// lbInternalNetworkFromSubnet builds the lbInternalNetwork block create-cluster
// expects from the subnet backing internal_subnet_lb: value is the subnet id,
// label/label4sending come from its description/name, and cidr is its default
// gateway plus prefix length.
func lbInternalNetworkFromSubnet(s *networkSubnet) *LbInternalNetworkJson {
	return &LbInternalNetworkJson{
		Value:         s.ID,
		Label:         s.Description,
		Label4Sending: s.Name,
		Cidr:          fmt.Sprintf("%s/%d", s.DefaultGateway, s.SubnetPrefixLength),
		NetworkType:   s.NetworkType,
	}
}

type networkSubnetListResponse struct {
	Data []networkSubnet `json:"data"`
}

type managedGpuClusterCreateResponse struct {
	Error bool `json:"error"`
	Kpi   struct {
		ClusterId   string `json:"cluster_id"`
		ClusterName string `json:"cluster_name"`
	} `json:"kpi"`
}
type managedGpuClusterReadResponse struct {
	Data  managedGpuClusterData `json:"data"`
	Mess  []string              `json:"mess"`
	Error bool                  `json:"error"`
}

type managedGpuClusterData struct {
	Status   managedGpuClusterDataStatus   `json:"status"`
	Metadata managedGpuClusterDataMetadata `json:"metadata"`
	Spec     managedGpuClusterDataSpec     `json:"spec"`
}

type managedGpuClusterDataStatus struct {
	LastOperation struct {
		Progress int    `json:"progress"`
		State    string `json:"state"`
		Type     string `json:"type"`
	} `json:"lastOperation"`
	Conditions []struct {
		Status string `json:"status"`
	} `json:"conditions"`
	IsRunning bool `json:"is_running"`
}

type managedGpuClusterDataMetadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type managedGpuClusterDataSpec struct {
	Kubernetes struct {
		ClusterAutoscaler managedGpuClusterDataClusterAutoscaler `json:"clusterAutoscaler,omitempty"`
		Kubelet           struct {
			MaxPods int `json:"maxPods"`
		} `json:"kubelet"`
		Version string `json:"version"`
	} `json:"kubernetes"`
	Networking managedGpuClusterDataNetworking `json:"networking"`

	SeedSelector struct {
		MatchLabels struct {
			GardenerCloudPurpose string `json:"gardener_cloud_purpose"`
		} `json:"matchLabels"`
	} `json:"seedSelector"`

	SeedName string `json:"seedName"`

	Provider struct {
		InfrastructureConfig struct {
			Networks struct {
				Id         string `json:"id"`
				Workers    string `json:"workers"`
				Lbv2Subnet string `json:"lbv2Subnet"`
				GatewayRef struct {
					Id   string `json:"id"`
					Name string `json:"name"`
				} `json:"gatewayRef"`
			} `json:"networks"`
			InternalNetworksLB *struct {
				Id      string      `json:"id"`
				Subnets interface{} `json:"subnets"`
			} `json:"internalNetworksLB"`
			// SubnetIDBM is the HPC subnet catalog ID (fptcloud_hpc_subnet's
			// id/network_id, GET .../hpc/subnets) that OSP bare-metal clusters
			// were created with — distinct from Networks.Id, which reports
			// osp_network_id instead.
			SubnetIDBM string `json:"subnetIDBM"`
		} `json:"infrastructureConfig"`
		Workers []*managedGpuClusterDataWorker `json:"workers"`
	} `json:"provider"`

	Extensions  []ExtensionSpec  `json:"extensions,omitempty"`
	Hibernate   *HibernateSpec   `json:"hibernation"`
	AutoUpgrade *AutoUpgradeSpec `json:"autoUpgrade,omitempty"`
	Addons      *AddonsSpec      `json:"addons,omitempty"`
}

// AddonsSpec represents the addons configuration in the API response
type AddonsSpec struct {
	GpuOperator *GpuOperatorSpec `json:"gpuOperator,omitempty"`
}

// GpuOperatorSpec represents the GPU operator configuration
type GpuOperatorSpec struct {
	TimeSliceConfig *TimeSliceConfigSpec `json:"timeSliceConfig,omitempty"`
}

// TimeSliceConfigSpec represents the time slice configuration
type TimeSliceConfigSpec struct {
	MaxClient []string `json:"maxClient"`
}

type managedGpuClusterDataClusterAutoscaler struct {
	Expander                      string  `json:"expander,omitempty"`
	MaxGracefulTerminationSeconds int     `json:"maxGracefulTerminationSeconds,omitempty"`
	MaxNodeProvisionTime          string  `json:"maxNodeProvisionTime,omitempty"`
	ScaleDownDelayAfterAdd        string  `json:"scaleDownDelayAfterAdd,omitempty"`
	ScaleDownDelayAfterDelete     string  `json:"scaleDownDelayAfterDelete,omitempty"`
	ScaleDownDelayAfterFailure    string  `json:"scaleDownDelayAfterFailure,omitempty"`
	ScaleDownUnneededTime         string  `json:"scaleDownUnneededTime,omitempty"`
	ScaleDownUtilizationThreshold float64 `json:"scaleDownUtilizationThreshold,omitempty"`
	ScanInterval                  string  `json:"scanInterval,omitempty"`
}

type managedGpuClusterDataNetworking struct {
	Nodes          string `json:"nodes"`
	Pods           string `json:"pods"`
	Services       string `json:"services"`
	Type           string `json:"type"`
	ProviderConfig struct {
		Overlay struct {
			Enabled bool `json:"enabled"`
		} `json:"overlay"`
		Ipip string `json:"ipip"`
	} `json:"providerConfig"`
}

type HibernateSpec struct {
	Enabled   bool                      `json:"enabled"`
	Schedules []HibernationScheduleJson `json:"schedules,omitempty"`
}

type managedGpuClusterDataWorker struct {
	Annotations map[string]string `json:"annotations"`
	Cri         struct {
		Name string `json:"name"`
	} `json:"cri"`
	Kubernetes struct {
		Kubelet struct {
			ContainerLogMaxFiles int    `json:"containerLogMaxFiles"`
			ContainerLogMaxSize  string `json:"containerLogMaxSize"`
			EvictionHard         struct {
				ImageFSAvailable  string `json:"imageFSAvailable"`
				ImageFSInodesFree string `json:"imageFSInodesFree"`
				MemoryAvailable   string `json:"memoryAvailable"`
				NodeFSAvailable   string `json:"nodeFSAvailable"`
				NodeFSInodesFree  string `json:"nodeFSInodesFree"`
			} `json:"evictionHard"`
			FailSwapOn   bool `json:"failSwapOn"`
			KubeReserved struct {
				CPU              string `json:"cpu"`
				EphemeralStorage string `json:"ephemeralStorage"`
				Memory           string `json:"memory"`
				Pid              string `json:"pid"`
			} `json:"kubeReserved"`
			MaxPods        int `json:"maxPods"`
			SystemReserved struct {
				CPU              string `json:"cpu"`
				EphemeralStorage string `json:"ephemeralStorage"`
				Memory           string `json:"memory"`
				Pid              string `json:"pid"`
			} `json:"systemReserved"`
		} `json:"kubelet"`
		Version string `json:"version"`
	} `json:"kubernetes"`
	Labels  []interface{} `json:"labels"`
	Machine struct {
		Image struct {
			DriverInstallationType string `json:"driverInstallationType"`
			GpuDriverVersion       string `json:"gpuDriverVersion"`
			Name                   string `json:"name"`
			Version                string `json:"version"`
		} `json:"image"`
		Type string `json:"type"`
	} `json:"machine"`
	MaxSurge       int    `json:"maxSurge"`
	MaxUnavailable int    `json:"maxUnavailable"`
	Maximum        int    `json:"maximum"`
	Minimum        int    `json:"minimum"`
	Name           string `json:"name"`
	ProviderConfig struct {
		APIVersion  string      `json:"apiVersion"`
		Kind        string      `json:"kind"`
		NetworkName string      `json:"networkName"`
		ServerGroup interface{} `json:"serverGroup"`
		ServerType  string      `json:"serverType"`
		SshKey      struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"sshKey"`
		UserName string `json:"userName"`
		VGpuID   string `json:"vGpuId"`
	} `json:"providerConfig"`
	SystemComponents struct {
		Allow bool `json:"allow"`
	} `json:"systemComponents"`
	Taints []interface{} `json:"taints"`
	Volume struct {
		Size string `json:"size"`
		Type string `json:"type"`
	} `json:"volume"`
	Zones []string `json:"zones"`
}

type managedGpuClusterEditWorker struct {
	Pools             []*managedGpuClusterPoolJson `json:"pools"`
	K8sVersion        string                       `json:"k8s_version"`
	TypeConfigure     string                       `json:"type_configure"`
	CurrentNetworking string                       `json:"currentNetworking"`
}

// HibernationSchedule represents a single hibernation schedule
type HibernationSchedule struct {
	Start    types.String `tfsdk:"start"`
	End      types.String `tfsdk:"end"`
	Location types.String `tfsdk:"location"`
}

// HibernationScheduleJson represents the JSON structure for hibernation schedules
type HibernationScheduleJson struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Location string `json:"location"`
}

// HibernationSchedulesRequest represents the request body for hibernation schedules
type HibernationSchedulesRequest struct {
	Schedules []HibernationScheduleJson `json:"schedules"`
}

type AutoUpgradeSpec struct {
	TimeUpgrade []string `json:"timeUpgrade"`
	TimeZone    string   `json:"timeZone"`
}

// ExtensionSpec represents the extensions configuration in the API response
type ExtensionSpec struct {
	Type           string                 `json:"type"`
	ProviderConfig map[string]interface{} `json:"providerConfig,omitempty"`
}

// ACLRule represents the ACL rule configuration in the extensions
type ACLRule struct {
	Action string   `json:"action"`
	Cidrs  []string `json:"cidrs"`
	Type   string   `json:"type"`
}
