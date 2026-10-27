package fptcloud_mfke

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &datasourceManagedKubernetesEngines{}
	_ datasource.DataSourceWithConfigure = &datasourceManagedKubernetesEngines{}
)

// datasourceManagedKubernetesEngines lists every managed FKE cluster in a VPC.
// Attribute names follow the fptcloud_managed_kubernetes_engine_v1 resource.
type datasourceManagedKubernetesEngines struct {
	client        *commons.Client
	mfkeClient    *MfkeApiClient
	tenancyClient *fptcloud_dfke.TenancyApiClient
}

type managedKubernetesEngines struct {
	VpcId    types.String                     `tfsdk:"vpc_id"`
	Clusters []managedKubernetesEngineSummary `tfsdk:"clusters"`
}

type managedKubernetesEngineSummary struct {
	Id                    types.String                         `tfsdk:"id"`
	ClusterName           types.String                         `tfsdk:"cluster_name"`
	K8SVersion            types.String                         `tfsdk:"k8s_version"`
	Purpose               types.String                         `tfsdk:"purpose"`
	IsRunning             types.Bool                           `tfsdk:"is_running"`
	ClusterEndpointAccess types.Object                         `tfsdk:"cluster_endpoint_access"`
	NetworkType           types.String                         `tfsdk:"network_type"`
	NodeNetwork           types.String                         `tfsdk:"node_network"`
	NodePrefix            types.String                         `tfsdk:"node_prefix"`
	PodNetwork            types.String                         `tfsdk:"pod_network"`
	PodPrefix             types.String                         `tfsdk:"pod_prefix"`
	ServiceNetwork        types.String                         `tfsdk:"service_network"`
	ServicePrefix         types.String                         `tfsdk:"service_prefix"`
	K8SMaxPod             types.Int64                          `tfsdk:"k8s_max_pod"`
	CreatedAt             types.String                         `tfsdk:"created_at"`
	Tags                  types.Set                            `tfsdk:"tags"`
	Pools                 []managedKubernetesEngineSummaryPool `tfsdk:"pools"`
}

type managedKubernetesEngineSummaryPool struct {
	Name               types.String `tfsdk:"name"`
	WorkerType         types.String `tfsdk:"worker_type"`
	MachineType        types.String `tfsdk:"machine_type"`
	VGpuID             types.String `tfsdk:"vgpu_id"`
	StorageProfile     types.String `tfsdk:"storage_profile"`
	WorkerDiskSize     types.Int64  `tfsdk:"worker_disk_size"`
	ScaleMin           types.Int64  `tfsdk:"scale_min"`
	ScaleMax           types.Int64  `tfsdk:"scale_max"`
	WorkerBase         types.Bool   `tfsdk:"worker_base"`
	IsEnableAutoRepair types.Bool   `tfsdk:"is_enable_auto_repair"`
}

func NewDataSourceManagedKubernetesEngines() datasource.DataSource {
	return &datasourceManagedKubernetesEngines{}
}

func (d *datasourceManagedKubernetesEngines) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*commons.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *commons.Client, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	}

	d.client = client
	d.mfkeClient = newMfkeApiClient(d.client)
	d.tenancyClient = fptcloud_dfke.NewTenancyApiClient(d.client)
}

func (d *datasourceManagedKubernetesEngines) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_kubernetes_engines_v1"
}

// managedKubernetesEnginePoolAttrTypes and managedKubernetesEngineSummaryAttrTypes
// describe one pool and one cluster of the list. They are plain object types
// rather than nested attributes because the provider is served over protocol
// version 5, which has no nested attributes.
var managedKubernetesEnginePoolAttrTypes = map[string]attr.Type{
	"name":                  types.StringType,
	"worker_type":           types.StringType,
	"machine_type":          types.StringType,
	"vgpu_id":               types.StringType,
	"storage_profile":       types.StringType,
	"worker_disk_size":      types.Int64Type,
	"scale_min":             types.Int64Type,
	"scale_max":             types.Int64Type,
	"worker_base":           types.BoolType,
	"is_enable_auto_repair": types.BoolType,
}

// clusterEndpointAccessAttrTypes matches the object clusterEndpointAccessFrom
// builds, which is also the resource's cluster_endpoint_access.
var clusterEndpointAccessAttrTypes = map[string]attr.Type{
	"type":       types.StringType,
	"allow_cidr": types.ListType{ElemType: types.StringType},
}

var managedKubernetesEngineSummaryAttrTypes = map[string]attr.Type{
	"id":                      types.StringType,
	"cluster_name":            types.StringType,
	"k8s_version":             types.StringType,
	"purpose":                 types.StringType,
	"is_running":              types.BoolType,
	"cluster_endpoint_access": types.ObjectType{AttrTypes: clusterEndpointAccessAttrTypes},
	"network_type":            types.StringType,
	"node_network":            types.StringType,
	"node_prefix":             types.StringType,
	"pod_network":             types.StringType,
	"pod_prefix":              types.StringType,
	"service_network":         types.StringType,
	"service_prefix":          types.StringType,
	"k8s_max_pod":             types.Int64Type,
	"created_at":              types.StringType,
	"tags":                    types.SetType{ElemType: types.StringType},
	"pools":                   types.ListType{ElemType: types.ObjectType{AttrTypes: managedKubernetesEnginePoolAttrTypes}},
}

// managedKubernetesEnginesClustersDescription documents the fields of each
// cluster, since object types carry no per-field description.
const managedKubernetesEnginesClustersDescription = "The managed FKE clusters in the VPC. Attribute names follow the " +
	"`fptcloud_managed_kubernetes_engine_v1` resource. Each cluster has:\n" +
	"  - `id`: the full name of the cluster, e.g. `my-cluster-ab12cd34`, the same value as the resource's `id`.\n" +
	"  - `cluster_name`: the full name of the cluster, the same as `id`, as in the `fptcloud_managed_kubernetes_engine_v1` data source, which takes it as `cluster_name`.\n" +
	"  - `k8s_version`: Kubernetes version.\n" +
	"  - `purpose`: `public`, `private` or `firewall`.\n" +
	"  - `is_running`: `false` when the cluster is hibernated.\n" +
	"  - `cluster_endpoint_access`: access to the Kubernetes API server: `type` (`public`, `private` or `mixed`) and the allowed CIDR blocks in `allow_cidr`.\n" +
	"  - `network_type`: network plugin, e.g. `calico` or `cilium`.\n" +
	"  - `node_network`, `node_prefix`: node network address and prefix.\n" +
	"  - `pod_network`, `pod_prefix`: pod network address and prefix.\n" +
	"  - `service_network`, `service_prefix`: service network address and prefix.\n" +
	"  - `k8s_max_pod`: max pods per node.\n" +
	"  - `created_at`: creation time, in RFC 3339 format.\n" +
	"  - `tags`: the tag IDs of the cluster.\n" +
	"  - `pools`: the worker pools, each with `name`, `worker_type` (flavor ID), `machine_type` (flavor name, e.g. `2C4G`), " +
	"`vgpu_id` (empty for a pool without GPU), `storage_profile`, " +
	"`worker_disk_size` (GB), `scale_min`, `scale_max`, `worker_base` and `is_enable_auto_repair`."

func (d *datasourceManagedKubernetesEngines) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "List every managed FKE cluster in a VPC, v1 and v2 alike.",
		Attributes: map[string]schema.Attribute{
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the VPC whose clusters are listed.",
			},
			"clusters": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: managedKubernetesEngineSummaryAttrTypes},
				Description: managedKubernetesEnginesClustersDescription,
			},
		},
	}
}

func (d *datasourceManagedKubernetesEngines) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state managedKubernetesEngines
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Listing managed FKE clusters of VPC "+vpcId)

	platform, err := d.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting VPC platform", err.Error()))
		return
	}

	items, err := d.mfkeClient.listClusters(vpcId, strings.ToLower(platform))
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error listing managed FKE clusters", err.Error()))
		return
	}

	state.Clusters = make([]managedKubernetesEngineSummary, 0, len(items))
	for _, item := range items {
		summary, diags := toManagedKubernetesEngineSummary(item)
		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}
		state.Clusters = append(state.Clusters, summary)
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func toManagedKubernetesEngineSummary(item managedKubernetesEngineShoot) (managedKubernetesEngineSummary, diag2.Diagnostics) {
	nodeNetwork, nodePrefix := splitCidr(item.Spec.Networking.Nodes)
	podNetwork, podPrefix := splitCidr(item.Spec.Networking.Pods)
	serviceNetwork, servicePrefix := splitCidr(item.Spec.Networking.Services)

	endpointAccess, diags := clusterEndpointAccessFrom(item.Metadata.Labels, item.Spec.Extensions)

	pools := make([]managedKubernetesEngineSummaryPool, 0, len(item.Spec.Provider.Workers))
	for _, w := range item.Spec.Provider.Workers {
		pools = append(pools, managedKubernetesEngineSummaryPool{
			Name:               types.StringValue(w.Name),
			WorkerType:         types.StringValue(item.Metadata.Labels["fptcloud.com/flavor_pool_"+w.Name]),
			MachineType:        types.StringValue(w.Machine.Type),
			VGpuID:             types.StringValue(w.ProviderConfig.VGpuID),
			StorageProfile:     types.StringValue(w.Volume.Type),
			WorkerDiskSize:     types.Int64Value(int64(parseNumber(w.Volume.Size))),
			ScaleMin:           types.Int64Value(int64(w.Minimum)),
			ScaleMax:           types.Int64Value(int64(w.Maximum)),
			WorkerBase:         types.BoolValue(w.SystemComponents.Allow),
			IsEnableAutoRepair: types.BoolValue(w.Annotations["worker.fptcloud.com/node-auto-repair"] == "true"),
		})
	}

	return managedKubernetesEngineSummary{
		Id:          types.StringValue(item.Metadata.Name),
		ClusterName: types.StringValue(item.Metadata.Name),
		K8SVersion:  types.StringValue(item.Spec.Kubernetes.Version),
		Purpose:     types.StringValue(purposeFromSeedPurpose(item.Spec.SeedSelector.MatchLabels.GardenerCloudPurpose)),
		// Same rule as the resource: a missing or disabled hibernation spec
		// means the cluster is running.
		IsRunning:             types.BoolValue(item.Spec.Hibernation == nil || !item.Spec.Hibernation.Enabled),
		ClusterEndpointAccess: endpointAccess,
		NetworkType:           types.StringValue(item.Spec.Networking.Type),
		NodeNetwork:           types.StringValue(nodeNetwork),
		NodePrefix:            types.StringValue(nodePrefix),
		PodNetwork:            types.StringValue(podNetwork),
		PodPrefix:             types.StringValue(podPrefix),
		ServiceNetwork:        types.StringValue(serviceNetwork),
		ServicePrefix:         types.StringValue(servicePrefix),
		K8SMaxPod:             types.Int64Value(int64(item.Spec.Kubernetes.Kubelet.MaxPods)),
		CreatedAt:             types.StringValue(item.Metadata.CreationTimestamp),
		Tags:                  tagSpecsToList(item.Tags),
		Pools:                 pools,
	}, diags
}

// purposeFromSeedPurpose maps the gardener_cloud_purpose seed label to the
// purpose value the resource uses.
func purposeFromSeedPurpose(seedPurpose string) string {
	switch {
	case strings.Contains(seedPurpose, "public"):
		return "public"
	case strings.Contains(seedPurpose, "firewall"):
		return "firewall"
	default:
		return "private"
	}
}

// splitCidr splits "100.96.0.0/11" into its address and prefix. A value
// without a prefix is returned as the address with an empty prefix.
func splitCidr(cidr string) (string, string) {
	address, prefix, _ := strings.Cut(cidr, "/")
	return address, prefix
}
