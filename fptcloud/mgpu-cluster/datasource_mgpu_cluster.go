package fptcloud_mgpu_cluster

import (
	"context"
	"encoding/json"
	"errors"
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
	_ datasource.DataSource              = &datasourceManagedGpuCluster{}
	_ datasource.DataSourceWithConfigure = &datasourceManagedGpuCluster{}
)

type datasourceManagedGpuCluster struct {
	client            *commons.Client
	mgpuClusterClient *MgpuClusterApiClient
	subnetClient      fptcloud_subnet.SubnetService
	tenancyClient     *fptcloud_dfke.TenancyApiClient
}

func (d *datasourceManagedGpuCluster) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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
	d.mgpuClusterClient = newMgpuClusterApiClient(d.client)
	d.subnetClient = fptcloud_subnet.NewSubnetService(d.client)
	d.tenancyClient = fptcloud_dfke.NewTenancyApiClient(d.client)
}

func (d *datasourceManagedGpuCluster) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_gpu_cluster"
}

func (d *datasourceManagedGpuCluster) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
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
		Description: "Retrieve information about a Bare Metal (GPU) Kubernetes cluster.",
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

func (d *datasourceManagedGpuCluster) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state managedGpuCluster
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

func NewDataSourceManagedGpuCluster() datasource.DataSource {
	return &datasourceManagedGpuCluster{}
}

func (d *datasourceManagedGpuCluster) internalRead(ctx context.Context, id string, state *managedGpuCluster) (*managedGpuClusterReadResponse, error) {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+id+", VPC ID "+vpcId)

	platform, err := d.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return nil, err
	}

	platform = strings.ToLower(platform)

	v1Path := commons.ApiPath.ManagedGpuClusterGet(vpcId, platform, id)
	v2Path := commons.ApiPath.ManagedGpuClusterGetV2(vpcId, platform, id)
	primaryPath, fallbackPath := v1Path, v2Path
	if requiresV2API(state.K8SVersion.ValueString()) {
		primaryPath, fallbackPath = v2Path, v1Path
	}
	a, err := d.mgpuClusterClient.sendGetV2Aware(primaryPath, fallbackPath, platform)
	if err != nil {
		return nil, err
	}

	var response managedGpuClusterReadResponse
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

	workers := map[string]*managedGpuClusterDataWorker{}

	// Sort workers to ensure consistent order: worker_base first, then by name
	for _, worker := range data.Spec.Provider.Workers {
		workers[worker.Name] = worker

		if len(state.Pools) == 0 {
			poolNames = append(poolNames, worker.Name)
		}
	}

	var pool []*managedGpuClusterPool

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

		item := &managedGpuClusterPool{
			WorkerPoolID:           types.StringValue(w.Name),
			StorageProfile:         types.StringValue(w.Volume.Type),
			HpcFlavorId:            types.StringValue(flavorId),
			HpcFlavorName:          types.StringValue(w.Machine.Type),
			HpcNumberServer:        types.Int64Value(int64(w.Maximum)),
			WorkerDiskSize:         types.Int64Value(int64(parseNumber(w.Volume.Size))),
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
func (d *datasourceManagedGpuCluster) MaxClientFromAddons(spec *managedGpuClusterDataSpec, poolName string) int64 {
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

func (d *datasourceManagedGpuCluster) GpuSharingClientFromAddons(spec *managedGpuClusterDataSpec, poolName string) string {
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

func (d *datasourceManagedGpuCluster) topFields() map[string]schema.Attribute {
	topLevelAttributes := map[string]schema.Attribute{}
	// Required string fields
	requiredStrings := []string{
		"vpc_id", "cluster_name", "k8s_version", "purpose",
		"pod_network", "pod_prefix", "service_network", "service_prefix",
		"network_id", "network_overlay",
	}
	// Optional string fields
	optionalStrings := []string{
		"internal_subnet_lb", "edge_gateway_name", "auto_upgrade_timezone",
		"default_storage_profile", "load_balancer_type", "secret_binding_name", "ssh_id", "ssh_name", "ssh_public_key",
	}
	// Required int fields
	requiredInts := []string{}
	// Optional int fields
	optionalInts := []string{"k8s_max_pod", "network_node_prefix"}
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

	// Object-typed attributes. These must mirror the fields of
	// managedGpuCluster exactly: the struct is shared with the resource, and a
	// schema that disagrees with it fails at state-conversion time.
	topLevelAttributes["cluster_autoscaler"] = schema.ObjectAttribute{
		Computed:    true,
		Description: descriptions["cluster_autoscaler"],
		AttributeTypes: map[string]attr.Type{
			"is_enable_auto_scaling":           types.BoolType,
			"scale_down_delay_after_add":       types.Int64Type,
			"scale_down_delay_after_delete":    types.Int64Type,
			"scale_down_delay_after_failure":   types.Int64Type,
			"scale_down_unneeded_time":         types.Int64Type,
			"scale_down_utilization_threshold": types.Float64Type,
			"scan_interval":                    types.Int64Type,
			"expander":                         types.StringType,
		},
	}

	topLevelAttributes["cluster_endpoint_access"] = schema.ObjectAttribute{
		Computed:    true,
		Description: descriptions["cluster_endpoint_access"],
		AttributeTypes: map[string]attr.Type{
			"type":       types.StringType,
			"allow_cidr": types.ListType{ElemType: types.StringType},
		},
	}

	topLevelAttributes["software"] = schema.ObjectAttribute{
		Computed:    true,
		Description: descriptions["software"],
		AttributeTypes: map[string]attr.Type{
			"software_type":        types.StringType,
			"software_version":     types.StringType,
			"cluster_mig_strategy": types.StringType,
		},
	}

	topLevelAttributes["hibernation_schedules"] = schema.ListAttribute{
		Computed:    true,
		Description: descriptions["hibernation_schedules"],
		ElementType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"start":    types.StringType,
				"end":      types.StringType,
				"location": types.StringType,
			},
		},
	}

	topLevelAttributes["network_type"] = schema.StringAttribute{
		Computed:    true,
		Description: descriptions["network_type"],
	}
	topLevelAttributes["edge_gateway_id"] = schema.StringAttribute{
		Computed:    true,
		Description: descriptions["edge_gateway_id"],
	}

	return topLevelAttributes
}

func (d *datasourceManagedGpuCluster) poolFields() map[string]schema.Attribute {
	poolLevelAttributes := map[string]schema.Attribute{}
	// Required string fields
	requiredStrings := []string{
		"name", "storage_profile", "hpc_flavor_id", "network_id",
	}
	// Optional string fields
	optionalStrings := []string{"gpu_sharing_client", "driver_installation_type", "container_runtime", "gpu_driver_version", "network_name", "vgpu_id", "hpc_flavor_name", "gpu_template_version"}
	// Required int fields
	requiredInts := []string{"worker_disk_size", "hpc_number_server"}
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
