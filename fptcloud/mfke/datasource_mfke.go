package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"

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

type datasourceManagedKubernetesEngine struct {
	client        *commons.Client
	mfkeClient    *MfkeApiClient
	subnetClient  fptcloud_subnet.SubnetService
	tenancyClient *fptcloud_dfke.TenancyApiClient
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
	topLevelAttributes := d.topFields()
	poolAttributes := d.poolFields()

	topLevelAttributes["id"] = schema.StringAttribute{
		Computed: true,
	}
	topLevelAttributes["cluster_name"] = schema.StringAttribute{
		Required: true,
	}
	topLevelAttributes["vpc_id"] = schema.StringAttribute{
		Required: true,
	}

	response.Schema = schema.Schema{
		Description: "Retrieve information about a managed FKE cluster.",
		Attributes:  topLevelAttributes,
	}

	response.Schema.Blocks = map[string]schema.Block{
		"pools": schema.ListNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: poolAttributes,
			},
		},
	}
}

func (d *datasourceManagedKubernetesEngine) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state managedKubernetesEngine
	diags := request.Config.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	_, err := d.internalRead(ctx, state.ClusterName.ValueString(), &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API", err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func NewDataSourceManagedKubernetesEngine() datasource.DataSource {
	return &datasourceManagedKubernetesEngine{}
}

func (d *datasourceManagedKubernetesEngine) internalRead(ctx context.Context, id string, state *managedKubernetesEngine) (*managedKubernetesEngineReadResponse, error) {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+id+", VPC ID "+vpcId)

	platform, err := d.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return nil, err
	}

	platform = strings.ToLower(platform)

	path := commons.ApiPath.ManagedFKEGet(vpcId, platform, id)
	a, err := d.mfkeClient.sendGet(path, platform)
	if err != nil {
		return nil, err
	}

	var response managedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &response)
	if err != nil {
		return nil, err
	}

	if response.Error {
		return nil, fmt.Errorf("error: %v", response.Mess)
	}

	data := response.Data

	state.Id = types.StringValue(data.Metadata.Name)
	state.VpcId = types.StringValue(vpcId)
	// keep clusterName
	//state.NetworkID
	state.K8SVersion = types.StringValue(data.Spec.Kubernetes.Version)

	if strings.Contains(data.Spec.SeedSelector.MatchLabels.GardenerCloudPurpose, "public") {
		state.Purpose = types.StringValue("public")
	} else if strings.Contains(data.Spec.SeedSelector.MatchLabels.GardenerCloudPurpose, "firewall") {
		state.Purpose = types.StringValue("firewall")
	} else {
		state.Purpose = types.StringValue("private")
	}

	poolNames, err := validatePoolNames(state.Pools)
	if err != nil {
		return nil, err
	}

	workers := map[string]*managedKubernetesEngineDataWorker{}

	// Sort workers to ensure consistent order: worker_base first, then by name
	workersList := data.Spec.Provider.Workers
	sort.Slice(workersList, func(i, j int) bool {
		// First sort by worker_base (true first)
		if workersList[i].IsWorkerBase() != workersList[j].IsWorkerBase() {
			return workersList[i].IsWorkerBase()
		}
		// Then sort by name
		return workersList[i].Name < workersList[j].Name
	})

	for _, worker := range workersList {
		workers[worker.Name] = worker

		if len(state.Pools) == 0 {
			poolNames = append(poolNames, worker.Name)
		}
	}

	var pool []*managedKubernetesEnginePool

	for _, name := range poolNames {
		w, ok := workers[name]
		if !ok {
			continue
		}

		flavorPoolKey := "fptcloud.com/flavor_pool_" + name
		flavorId, ok := data.Metadata.Labels[flavorPoolKey]
		if !ok {
			return nil, errors.New("missing flavor ID on label " + flavorPoolKey)
		}

		autoRepair := w.AutoRepair()

		// Only use networkId and error from getNetworkInfoByPlatform
		networkId, _, e := getNetworkInfoByPlatform(ctx, d.subnetClient, vpcId, platform, w, &data)

		if e != nil {
			return nil, e
		}

		item := &managedKubernetesEnginePool{
			WorkerPoolID:           types.StringValue(w.Name),
			StorageProfile:         types.StringValue(w.Volume.Type),
			WorkerType:             types.StringValue(flavorId),
			WorkerDiskSize:         types.Int64Value(int64(parseNumber(w.Volume.Size))),
			ScaleMin:               types.Int64Value(int64(w.Minimum)),
			ScaleMax:               types.Int64Value(int64(w.Maximum)),
			NetworkID:              types.StringValue(networkId),
			IsEnableAutoRepair:     types.BoolValue(autoRepair),
			VGpuID:                 types.StringValue(w.ProviderConfig.VGpuID),
			DriverInstallationType: types.StringValue(w.Machine.Image.DriverInstallationType),
			GpuDriverVersion:       types.StringValue(w.Machine.Image.GpuDriverVersion),
			WorkerBase:             types.BoolValue(w.IsWorkerBase()),
			Tags:                   tagsStringToList(w.Tags()),
		}

		// For GPU pools, read values from addons configuration
		if w.ProviderConfig.VGpuID != "" {
			// Read MaxClient from addons configuration
			maxClientFromAPI := d.MaxClientFromAddons(&data.Spec, w.Name)
			item.MaxClient = types.Int64Value(maxClientFromAPI)

			// Read GpuSharingClient from addons configuration
			gpuSharingClientFromAPI := d.GpuSharingClientFromAddons(&data.Spec, w.Name)
			item.GpuSharingClient = types.StringValue(gpuSharingClientFromAPI)
		} else {
			// Non-GPU pools: set default values
			item.MaxClient = types.Int64Value(0)
			item.GpuSharingClient = types.StringValue("")
		}

		pool = append(pool, item)
	}

	state.Pools = pool

	podNetwork := strings.Split(data.Spec.Networking.Pods, "/")
	state.PodNetwork = types.StringValue(podNetwork[0])
	state.PodPrefix = types.StringValue(podNetwork[1])

	serviceNetwork := strings.Split(data.Spec.Networking.Services, "/")
	state.ServiceNetwork = types.StringValue(serviceNetwork[0])
	state.ServicePrefix = types.StringValue(serviceNetwork[1])

	state.K8SMaxPod = types.Int64Value(int64(data.Spec.Kubernetes.Kubelet.MaxPods))
	// state.NetworkNodePrefix

	return &response, nil
}

// MaxClient reads the maxClient value from the addons configuration
// The maxClient is stored in spec.addons.gpuOperator.timeSliceConfig.maxClient
// Format: ["pool-name:value"] e.g. ["gpu-test:2"]
func (d *datasourceManagedKubernetesEngine) MaxClientFromAddons(spec *managedKubernetesEngineDataSpec, poolName string) int64 {
	if spec.Addons == nil || spec.Addons.GpuOperator == nil || spec.Addons.GpuOperator.TimeSliceConfig == nil {
		return 0
	}

	for _, maxClientStr := range spec.Addons.GpuOperator.TimeSliceConfig.MaxClient {
		// Parse format "pool-name:value" e.g. "gpu-test:2"
		if strings.HasPrefix(maxClientStr, poolName+":") {
			parts := strings.Split(maxClientStr, ":")
			if len(parts) == 2 {
				if value, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					return value
				}
			}
		}
	}
	return 0
}

func (d *datasourceManagedKubernetesEngine) GpuSharingClientFromAddons(spec *managedKubernetesEngineDataSpec, poolName string) string {
	if spec.Addons == nil || spec.Addons.GpuOperator == nil || spec.Addons.GpuOperator.TimeSliceConfig == nil {
		return ""
	}

	// Check if this pool has TimeSliceConfig (maxClient configuration)
	for _, maxClientStr := range spec.Addons.GpuOperator.TimeSliceConfig.MaxClient {
		if strings.HasPrefix(maxClientStr, poolName+":") {
			// If pool has TimeSliceConfig, it means gpu_sharing_client = "timeSlicing"
			return "timeSlicing"
		}
	}

	// If no TimeSliceConfig found for this pool, gpu_sharing_client = "" (empty)
	return ""
}

func (d *datasourceManagedKubernetesEngine) topFields() map[string]schema.Attribute {
	topLevelAttributes := map[string]schema.Attribute{}
	// Required string fields
	requiredStrings := []string{
		"vpc_id", "cluster_name", "k8s_version", "purpose",
		"pod_network", "pod_prefix", "service_network", "service_prefix",
		"network_id", "network_overlay",
	}
	// Optional string fields
	optionalStrings := []string{
		"internal_subnet_lb", "edge_gateway_name", "auto_upgrade_timezone", "network_node_prefix",
	}
	// Required int fields
	requiredInts := []string{}
	// Optional int fields
	optionalInts := []string{"k8s_max_pod"}
	// Optional bool fields
	optionalBools := []string{"is_enable_auto_upgrade", "is_running"}
	// Optional list fields
	optionalLists := []string{"auto_upgrade_expression"}

	for _, attribute := range requiredStrings {
		topLevelAttributes[attribute] = schema.StringAttribute{
			Required:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalStrings {
		topLevelAttributes[attribute] = schema.StringAttribute{
			Optional:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range requiredInts {
		topLevelAttributes[attribute] = schema.Int64Attribute{
			Required:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalInts {
		topLevelAttributes[attribute] = schema.Int64Attribute{
			Optional:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalBools {
		topLevelAttributes[attribute] = schema.BoolAttribute{
			Optional:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalLists {
		topLevelAttributes[attribute] = schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: descriptions[attribute],
		}
	}

	topLevelAttributes["k8s_version"] = schema.StringAttribute{
		Required:    true,
		Description: descriptions["k8s_version"],
	}

	// Flatten cluster_autoscaler into individual attributes
	topLevelAttributes["is_enable_auto_scaling"] = schema.BoolAttribute{
		Optional:    true,
		Description: descriptions["is_enable_auto_scaling"],
	}
	topLevelAttributes["scale_down_delay_after_add"] = schema.Int64Attribute{
		Optional:    true,
		Description: descriptions["scale_down_delay_after_add"],
	}
	topLevelAttributes["scale_down_delay_after_delete"] = schema.Int64Attribute{
		Optional:    true,
		Description: descriptions["scale_down_delay_after_delete"],
	}
	topLevelAttributes["scale_down_delay_after_failure"] = schema.Int64Attribute{
		Optional:    true,
		Description: descriptions["scale_down_delay_after_failure"],
	}
	topLevelAttributes["scale_down_unneeded_time"] = schema.Int64Attribute{
		Optional:    true,
		Description: descriptions["scale_down_unneeded_time"],
	}
	topLevelAttributes["scale_down_utilization_threshold"] = schema.Float64Attribute{
		Optional:    true,
		Description: descriptions["scale_down_utilization_threshold"],
	}
	topLevelAttributes["scan_interval"] = schema.Int64Attribute{
		Optional:    true,
		Description: descriptions["scan_interval"],
	}
	topLevelAttributes["expander"] = schema.StringAttribute{
		Optional:    true,
		Description: descriptions["expander"],
	}

	return topLevelAttributes
}

func (d *datasourceManagedKubernetesEngine) poolFields() map[string]schema.Attribute {
	poolLevelAttributes := map[string]schema.Attribute{}
	// Required string fields
	requiredStrings := []string{
		"name", "storage_profile", "worker_type", "network_id",
	}
	// Optional string fields
	optionalStrings := []string{"gpu_sharing_client", "driver_installation_type", "container_runtime", "gpu_driver_version", "network_name", "vgpu_id"}
	// Required int fields
	requiredInts := []string{"worker_disk_size", "scale_min", "scale_max"}
	// Optional int fields
	optionalInts := []string{"max_client"}
	// Required bool fields
	requiredBools := []string{"auto_scale", "is_enable_auto_repair"}
	// Optional bool fields
	optionalBools := []string{"is_enable_auto_repair"}
	// Optional list fields
	optionalLists := []string{"tags"}

	for _, attribute := range requiredStrings {
		poolLevelAttributes[attribute] = schema.StringAttribute{
			Required:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalStrings {
		poolLevelAttributes[attribute] = schema.StringAttribute{
			Optional:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range requiredInts {
		poolLevelAttributes[attribute] = schema.Int64Attribute{
			Required:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalInts {
		poolLevelAttributes[attribute] = schema.Int64Attribute{
			Optional:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range requiredBools {
		poolLevelAttributes[attribute] = schema.BoolAttribute{
			Required:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalBools {
		poolLevelAttributes[attribute] = schema.BoolAttribute{
			Optional:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalLists {
		poolLevelAttributes[attribute] = schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: descriptions[attribute],
		}
	}
	// kv: list of map[string]string
	poolLevelAttributes["kv"] = schema.ListAttribute{
		Optional:    true,
		ElementType: types.MapType{ElemType: types.StringType},
		Description: descriptions["kv"],
	}
	return poolLevelAttributes
}
