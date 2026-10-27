package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &datasourceManagedKubernetesEngine{}
	_ datasource.DataSourceWithConfigure = &datasourceManagedKubernetesEngine{}
)

// datasourceManagedKubernetesEngine reads one managed FKE cluster. It has its
// own model rather than the resource's, so a change to the resource cannot
// break it, and it reports the same attributes as one element of
// fptcloud_managed_kubernetes_engines_v1, plus what only the detail endpoint
// returns.
type datasourceManagedKubernetesEngine struct {
	client        *commons.Client
	mfkeClient    *MfkeApiClient
	subnetClient  fptcloud_subnet.SubnetService
	tenancyClient *fptcloud_dfke.TenancyApiClient
}

type managedKubernetesEngineDetail struct {
	VpcId                 types.String                        `tfsdk:"vpc_id"`
	ClusterName           types.String                        `tfsdk:"cluster_name"`
	Id                    types.String                        `tfsdk:"id"`
	K8SVersion            types.String                        `tfsdk:"k8s_version"`
	Purpose               types.String                        `tfsdk:"purpose"`
	IsRunning             types.Bool                          `tfsdk:"is_running"`
	ClusterEndpointAccess types.Object                        `tfsdk:"cluster_endpoint_access"`
	NetworkType           types.String                        `tfsdk:"network_type"`
	NodeNetwork           types.String                        `tfsdk:"node_network"`
	NodePrefix            types.String                        `tfsdk:"node_prefix"`
	PodNetwork            types.String                        `tfsdk:"pod_network"`
	PodPrefix             types.String                        `tfsdk:"pod_prefix"`
	ServiceNetwork        types.String                        `tfsdk:"service_network"`
	ServicePrefix         types.String                        `tfsdk:"service_prefix"`
	K8SMaxPod             types.Int64                         `tfsdk:"k8s_max_pod"`
	CreatedAt             types.String                        `tfsdk:"created_at"`
	Tags                  types.Set                           `tfsdk:"tags"`
	Pools                 []managedKubernetesEngineDetailPool `tfsdk:"pools"`
}

type managedKubernetesEngineDetailPool struct {
	Name                   types.String `tfsdk:"name"`
	WorkerType             types.String `tfsdk:"worker_type"`
	MachineType            types.String `tfsdk:"machine_type"`
	VGpuID                 types.String `tfsdk:"vgpu_id"`
	StorageProfile         types.String `tfsdk:"storage_profile"`
	WorkerDiskSize         types.Int64  `tfsdk:"worker_disk_size"`
	ScaleMin               types.Int64  `tfsdk:"scale_min"`
	ScaleMax               types.Int64  `tfsdk:"scale_max"`
	WorkerBase             types.Bool   `tfsdk:"worker_base"`
	IsEnableAutoRepair     types.Bool   `tfsdk:"is_enable_auto_repair"`
	NetworkID              types.String `tfsdk:"network_id"`
	PoolTags               types.Set    `tfsdk:"pool_tags"`
	DriverInstallationType types.String `tfsdk:"driver_installation_type"`
	GpuDriverVersion       types.String `tfsdk:"gpu_driver_version"`
	MaxClient              types.Int64  `tfsdk:"max_client"`
	GpuSharingClient       types.String `tfsdk:"gpu_sharing_client"`
}

// managedKubernetesEngineDetailPoolAttrTypes is a pool of the list data source
// plus the fields only the detail endpoint returns.
var managedKubernetesEngineDetailPoolAttrTypes = func() map[string]attr.Type {
	attrTypes := map[string]attr.Type{
		"network_id":               types.StringType,
		"pool_tags":                types.SetType{ElemType: types.StringType},
		"driver_installation_type": types.StringType,
		"gpu_driver_version":       types.StringType,
		"max_client":               types.Int64Type,
		"gpu_sharing_client":       types.StringType,
	}
	for name, attrType := range managedKubernetesEnginePoolAttrTypes {
		attrTypes[name] = attrType
	}
	return attrTypes
}()

func NewDataSourceManagedKubernetesEngine() datasource.DataSource {
	return &datasourceManagedKubernetesEngine{}
}

func (d *datasourceManagedKubernetesEngine) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*commons.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *commons.Client, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	}

	d.client = client
	d.mfkeClient = newMfkeApiClient(d.client)
	d.subnetClient = fptcloud_subnet.NewSubnetService(d.client)
	d.tenancyClient = fptcloud_dfke.NewTenancyApiClient(d.client)
}

func (d *datasourceManagedKubernetesEngine) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_kubernetes_engine_v1"
}

func (d *datasourceManagedKubernetesEngine) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, Description: description}
	}
	computedInt := func(description string) schema.Int64Attribute {
		return schema.Int64Attribute{Computed: true, Description: description}
	}
	computedBool := func(description string) schema.BoolAttribute {
		return schema.BoolAttribute{Computed: true, Description: description}
	}

	response.Schema = schema.Schema{
		Description: "Retrieve information about a managed FKE cluster. Attribute names follow the " +
			"`fptcloud_managed_kubernetes_engine_v1` resource.",
		Attributes: map[string]schema.Attribute{
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the VPC the cluster belongs to.",
			},
			"cluster_name": schema.StringAttribute{
				Required: true,
				Description: "The full name of the cluster, e.g. `my-cluster-ab12cd34`, as `id` and `cluster_name` " +
					"of `fptcloud_managed_kubernetes_engines_v1` report it.",
			},
			"id":              computedString("The full name of the cluster, the same as `cluster_name`."),
			"k8s_version":     computedString("Kubernetes version"),
			"purpose":         computedString("Cluster purpose: `public`, `private` or `firewall`."),
			"is_running":      computedBool("`false` when the cluster is hibernated."),
			"network_type":    computedString("Network plugin, e.g. `calico` or `cilium`."),
			"node_network":    computedString("Node network (address)"),
			"node_prefix":     computedString("Node network (prefix)"),
			"pod_network":     computedString("Pod network (address)"),
			"pod_prefix":      computedString("Pod network (prefix)"),
			"service_network": computedString("Service network (address)"),
			"service_prefix":  computedString("Service network (prefix)"),
			"k8s_max_pod":     computedInt("Max pods per node"),
			"created_at":      computedString("Creation time, in RFC 3339 format."),
			"tags": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The tag IDs of the cluster.",
			},
			"cluster_endpoint_access": schema.ObjectAttribute{
				Computed:       true,
				AttributeTypes: clusterEndpointAccessAttrTypes,
				Description: "Access to the Kubernetes API server: `type` (`public`, `private` or `mixed`) and the " +
					"allowed CIDR blocks in `allow_cidr`.",
			},
			"pools": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: managedKubernetesEngineDetailPoolAttrTypes},
				Description: "The worker pools of the cluster, each with `name`, `worker_type` (flavor ID), " +
					"`machine_type` (flavor name, e.g. `2C4G`), `vgpu_id` (empty for a pool without GPU), " +
					"`storage_profile`, `worker_disk_size` (GB), `scale_min`, `scale_max`, `worker_base`, " +
					"`is_enable_auto_repair`, `network_id`, `pool_tags` (the tag IDs of the pool, inherited " +
					"cluster tags included), `driver_installation_type`, `gpu_driver_version`, `max_client` and " +
					"`gpu_sharing_client`.",
			},
		},
	}
}

func (d *datasourceManagedKubernetesEngine) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state managedKubernetesEngineDetail
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	vpcId := state.VpcId.ValueString()
	clusterName := state.ClusterName.ValueString()
	tflog.Info(ctx, "Reading managed FKE cluster "+clusterName+" of VPC "+vpcId)

	platform, err := d.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting VPC platform", err.Error()))
		return
	}
	platform = strings.ToLower(platform)

	shoot, err := d.mfkeClient.getCluster(vpcId, platform, clusterName)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error reading managed FKE cluster "+clusterName, err.Error()))
		return
	}

	detail, diags := toManagedKubernetesEngineDetail(shoot, d.poolNetworkIds(ctx, vpcId, platform, shoot))
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	detail.VpcId = state.VpcId
	response.Diagnostics.Append(response.State.Set(ctx, &detail)...)
}

// poolNetworkIds returns the network ID of each worker pool, keyed by pool
// name. On OSP every pool sits on the cluster network. On VMW a pool names its
// network, which is looked up once per name; a name that is not found leaves
// the pool's network_id empty.
func (d *datasourceManagedKubernetesEngine) poolNetworkIds(ctx context.Context, vpcId string, platform string, shoot managedKubernetesEngineShoot) map[string]string {
	clusterNetworkId := shoot.Spec.Provider.InfrastructureConfig.Networks.Id
	ids := make(map[string]string, len(shoot.Spec.Provider.Workers))
	byName := map[string]string{}

	for _, w := range shoot.Spec.Provider.Workers {
		networkName := w.ProviderConfig.NetworkName
		if platform != "vmw" || networkName == "" {
			ids[w.Name] = clusterNetworkId
			continue
		}

		id, ok := byName[networkName]
		if !ok {
			subnet, err := d.subnetClient.FindSubnetByName(fptcloud_subnet.FindSubnetDTO{NetworkName: networkName, VpcId: vpcId})
			if err != nil {
				tflog.Warn(ctx, "Network "+networkName+" of pool "+w.Name+" not found: "+err.Error())
			} else if subnet != nil {
				id = subnet.NetworkID
			}
			byName[networkName] = id
		}
		ids[w.Name] = id
	}

	return ids
}

// toManagedKubernetesEngineDetail maps a shoot read from the detail endpoint.
// The attributes it shares with fptcloud_managed_kubernetes_engines_v1 come from
// the same mapping, so the two data sources always agree.
func toManagedKubernetesEngineDetail(shoot managedKubernetesEngineShoot, networkIds map[string]string) (managedKubernetesEngineDetail, diag2.Diagnostics) {
	summary, diags := toManagedKubernetesEngineSummary(shoot)

	pools := make([]managedKubernetesEngineDetailPool, 0, len(summary.Pools))
	for i, p := range summary.Pools {
		w := shoot.Spec.Provider.Workers[i]
		pools = append(pools, managedKubernetesEngineDetailPool{
			Name:                   p.Name,
			WorkerType:             p.WorkerType,
			MachineType:            p.MachineType,
			VGpuID:                 p.VGpuID,
			StorageProfile:         p.StorageProfile,
			WorkerDiskSize:         p.WorkerDiskSize,
			ScaleMin:               p.ScaleMin,
			ScaleMax:               p.ScaleMax,
			WorkerBase:             p.WorkerBase,
			IsEnableAutoRepair:     p.IsEnableAutoRepair,
			NetworkID:              types.StringValue(networkIds[w.Name]),
			PoolTags:               tagSpecsToList(w.Tags),
			DriverInstallationType: types.StringValue(w.Machine.Image.DriverInstallationType),
			GpuDriverVersion:       types.StringValue(w.Machine.Image.GpuDriverVersion),
			MaxClient:              types.Int64Value(maxClientFromAddons(shoot.Spec.Addons, w.Name)),
			GpuSharingClient:       types.StringValue(gpuSharingClientFromAddons(shoot.Spec.Addons, w.Name)),
		})
	}

	return managedKubernetesEngineDetail{
		ClusterName:           summary.ClusterName,
		Id:                    summary.Id,
		K8SVersion:            summary.K8SVersion,
		Purpose:               summary.Purpose,
		IsRunning:             summary.IsRunning,
		ClusterEndpointAccess: summary.ClusterEndpointAccess,
		NetworkType:           summary.NetworkType,
		NodeNetwork:           summary.NodeNetwork,
		NodePrefix:            summary.NodePrefix,
		PodNetwork:            summary.PodNetwork,
		PodPrefix:             summary.PodPrefix,
		ServiceNetwork:        summary.ServiceNetwork,
		ServicePrefix:         summary.ServicePrefix,
		K8SMaxPod:             summary.K8SMaxPod,
		CreatedAt:             summary.CreatedAt,
		Tags:                  summary.Tags,
		Pools:                 pools,
	}, diags
}

// maxClientFromAddons reads a GPU pool's time-slicing client count from
// spec.addons.gpuOperator.timeSliceConfig.maxClient, whose entries look like
// "pool-name:2". A pool without an entry reports 0.
func maxClientFromAddons(addons *AddonsSpec, poolName string) int64 {
	if addons == nil || addons.GpuOperator == nil || addons.GpuOperator.TimeSliceConfig == nil {
		return 0
	}

	for _, entry := range addons.GpuOperator.TimeSliceConfig.MaxClient {
		name, value, found := strings.Cut(entry, ":")
		if !found || name != poolName {
			continue
		}
		if n, err := strconv.ParseInt(value, 10, 64); err == nil {
			return n
		}
	}
	return 0
}

// gpuSharingClientFromAddons reports "timeSlicing" for a pool that has a
// time-slicing entry, and "" otherwise.
func gpuSharingClientFromAddons(addons *AddonsSpec, poolName string) string {
	if addons == nil || addons.GpuOperator == nil || addons.GpuOperator.TimeSliceConfig == nil {
		return ""
	}

	for _, entry := range addons.GpuOperator.TimeSliceConfig.MaxClient {
		if name, _, found := strings.Cut(entry, ":"); found && name == poolName {
			return "timeSlicing"
		}
	}
	return ""
}

// getCluster reads one cluster from the detail endpoint. A cluster is served
// by the v1 or the v2 API family depending on how it was created, which the
// data source cannot know up front, so a 404 on v1 retries v2.
func (m *MfkeApiClient) getCluster(vpcId string, platform string, clusterName string) (managedKubernetesEngineShoot, error) {
	a, err := m.sendGetV2Aware(
		commons.ApiPath.ManagedFKEGet(vpcId, platform, clusterName),
		commons.ApiPath.ManagedFKEGetV2(vpcId, platform, clusterName),
		platform,
	)
	if err != nil {
		return managedKubernetesEngineShoot{}, err
	}

	var response managedKubernetesEngineShootResponse
	if err = json.Unmarshal(a, &response); err != nil {
		return managedKubernetesEngineShoot{}, err
	}
	if response.Error {
		return managedKubernetesEngineShoot{}, fmt.Errorf("error: %v", response.Mess)
	}

	return response.Data, nil
}
