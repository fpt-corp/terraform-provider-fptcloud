package fptcloud_mfke

import (
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type managedKubernetesEngine struct {
	Id             types.String                   `tfsdk:"id"`
	VpcId          types.String                   `tfsdk:"vpc_id"`
	ClusterName    types.String                   `tfsdk:"cluster_name"`
	NetworkID      types.String                   `tfsdk:"network_id"`
	K8SVersion     types.String                   `tfsdk:"k8s_version"`
	Purpose        types.String                   `tfsdk:"purpose"`
	Pools          []*managedKubernetesEnginePool `tfsdk:"pools"`
	PodNetwork     types.String                   `tfsdk:"pod_network"`
	PodPrefix      types.String                   `tfsdk:"pod_prefix"`
	ServiceNetwork types.String                   `tfsdk:"service_network"`
	ServicePrefix  types.String                   `tfsdk:"service_prefix"`
	K8SMaxPod      types.Int64                    `tfsdk:"k8s_max_pod"`
	NetworkType    types.String                   `tfsdk:"network_type"`
	NetworkOverlay types.String                   `tfsdk:"network_overlay"`
	EdgeGatewayId  types.String                   `tfsdk:"edge_gateway_id"`
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

type resourceManagedKubernetesEngine struct {
	client        *commons.Client
	mfkeClient    *MfkeApiClient
	subnetClient  fptcloud_subnet.SubnetService
	tenancyClient *fptcloud_dfke.TenancyApiClient
}

type managedKubernetesEnginePool struct {
	WorkerBase             types.Bool   `tfsdk:"worker_base"`
	WorkerPoolID           types.String `tfsdk:"name"`
	StorageProfile         types.String `tfsdk:"storage_profile"`
	WorkerType             types.String `tfsdk:"worker_type"`
	WorkerDiskSize         types.Int64  `tfsdk:"worker_disk_size"`
	ContainerRuntime       types.String `tfsdk:"container_runtime"`
	ScaleMin               types.Int64  `tfsdk:"scale_min"`
	ScaleMax               types.Int64  `tfsdk:"scale_max"`
	NetworkID              types.String `tfsdk:"network_id"`
	NetworkName            types.String `tfsdk:"network_name"`
	Tags                   types.String `tfsdk:"tags"`
	Kv                     []KV         `tfsdk:"kv"`
	Taints                 []Taint      `tfsdk:"taints"`
	VGpuID                 types.String `tfsdk:"vgpu_id"`
	MaxClient              types.Int64  `tfsdk:"max_client"`
	GpuSharingClient       types.String `tfsdk:"gpu_sharing_client"`
	IsEnableAutoRepair     types.Bool   `tfsdk:"is_enable_auto_repair"`
	DriverInstallationType types.String `tfsdk:"driver_installation_type"`
	GpuDriverVersion       types.String `tfsdk:"gpu_driver_version"`
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

type managedKubernetesEngineJson struct {
	ClusterName           string                             `json:"cluster_name"`
	NetworkID             string                             `json:"network_id"`
	K8SVersion            string                             `json:"k8s_version,omitempty"`
	OsVersion             interface{}                        `json:"os_version,omitempty"`
	Purpose               string                             `json:"purpose,omitempty"`
	Pools                 []*managedKubernetesEnginePoolJson `json:"pools"`
	PodNetwork            string                             `json:"pod_network,omitempty"`
	PodPrefix             string                             `json:"pod_prefix,omitempty"`
	ServiceNetwork        string                             `json:"service_network,omitempty"`
	ServicePrefix         string                             `json:"service_prefix,omitempty"`
	K8SMaxPod             int64                              `json:"k8s_max_pod,omitempty"`
	NetworkOverlay        string                             `json:"network_overlay,omitempty"`
	InternalSubnetLb      interface{}                        `json:"internal_subnet_lb,omitempty"`
	EdgeGatewayId         string                             `json:"edge_gateway_id,omitempty"`
	EdgeGatewayName       string                             `json:"edge_gateway_name,omitempty"`
	ClusterEndpointAccess *ClusterEndpointAccessJson         `json:"clusterEndpointAccess,omitempty"`
	IsEnableAutoUpgrade   bool                               `json:"is_enable_auto_upgrade,omitempty"`
	AutoUpgradeExpression []string                           `json:"auto_upgrade_expression,omitempty"`
	AutoUpgradeTimezone   string                             `json:"auto_upgrade_timezone,omitempty"`
	ClusterAutoscaler     interface{}                        `json:"cluster_autoscaler,omitempty"`
	TypeCreate            string                             `json:"type_create,omitempty"`
}

type ClusterEndpointAccessJson struct {
	Type      string   `json:"type"`
	AllowCidr []string `json:"allowCidr"`
}

type managedKubernetesEnginePoolJson struct {
	// int64 fields
	WorkerDiskSize int64 `json:"worker_disk_size"`
	ScaleMin       int64 `json:"scale_min"`
	ScaleMax       int64 `json:"scale_max"`
	MaxClient      int64 `json:"maxClient"`

	// pointer fields
	WorkerPoolID *string `json:"worker_pool_id"`

	// string fields
	StorageProfile         string `json:"storage_profile"`
	WorkerType             string `json:"worker_type"`
	NetworkID              string `json:"network_id"`
	NetworkName            string `json:"network_name"`
	VGpuID                 string `json:"vGpuId"`
	DriverInstallationType string `json:"driverInstallationType"`
	GpuDriverVersion       string `json:"gpuDriverVersion"`
	Tags                   string `json:"tags"`
	GpuSharingClient       string `json:"gpuSharingClient"`
	ContainerRuntime       string `json:"container_runtime"`

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

type managedKubernetesEngineCreateResponse struct {
	Error bool `json:"error"`
	Kpi   struct {
		ClusterId   string `json:"cluster_id"`
		ClusterName string `json:"cluster_name"`
	} `json:"kpi"`
}
type managedKubernetesEngineReadResponse struct {
	Data  managedKubernetesEngineData `json:"data"`
	Mess  []string                    `json:"mess"`
	Error bool                        `json:"error"`
}

type managedKubernetesEngineData struct {
	Status   managedKubernetesEngineDataStatus   `json:"status"`
	Metadata managedKubernetesEngineDataMetadata `json:"metadata"`
	Spec     managedKubernetesEngineDataSpec     `json:"spec"`
}

type managedKubernetesEngineDataStatus struct {
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

type managedKubernetesEngineDataMetadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type managedKubernetesEngineDataSpec struct {
	Kubernetes struct {
		ClusterAutoscaler managedKubernetesEngineDataClusterAutoscaler `json:"clusterAutoscaler,omitempty"`
		Kubelet           struct {
			MaxPods int `json:"maxPods"`
		} `json:"kubelet"`
		Version string `json:"version"`
	} `json:"kubernetes"`
	Networking managedKubernetesEngineDataNetworking `json:"networking"`

	SeedSelector struct {
		MatchLabels struct {
			GardenerCloudPurpose string `json:"gardener_cloud_purpose"`
		} `json:"matchLabels"`
	} `json:"seedSelector"`

	Provider struct {
		InfrastructureConfig struct {
			Networks struct {
				Id      string `json:"id"`
				Workers string `json:"workers"`
			} `json:"networks"`
		} `json:"infrastructureConfig"`
		Workers []*managedKubernetesEngineDataWorker `json:"workers"`
	} `json:"provider"`

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

type managedKubernetesEngineDataClusterAutoscaler struct {
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

type managedKubernetesEngineDataNetworking struct {
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

type managedKubernetesEngineDataWorker struct {
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
		UserName    string      `json:"userName"`
		VGpuID      string      `json:"vGpuId"`
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

type managedKubernetesEngineEditWorker struct {
	Pools             []*managedKubernetesEnginePoolJson `json:"pools"`
	K8sVersion        string                             `json:"k8s_version"`
	TypeConfigure     string                             `json:"type_configure"`
	CurrentNetworking string                             `json:"currentNetworking"`
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
