package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		Description: "Manage managed FKE clusters.",
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
	} else {
		state.Purpose = types.StringValue("private")
	}

	poolNames, err := validatePoolNames(state.Pools)
	if err != nil {
		return nil, err
	}

	workers := map[string]*managedKubernetesEngineDataWorker{}
	for _, worker := range data.Spec.Provider.Workers {
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

		item := managedKubernetesEnginePool{
			WorkerPoolID:       types.StringValue(w.Name),
			StorageProfile:     types.StringValue(w.Volume.Type),
			WorkerType:         types.StringValue(flavorId),
			WorkerDiskSize:     types.Int64Value(int64(parseNumber(w.Volume.Size))),
			AutoScale:          types.BoolValue(w.Maximum != w.Minimum),
			ScaleMin:           types.Int64Value(int64(w.Minimum)),
			ScaleMax:           types.Int64Value(int64(w.Maximum)),
			NetworkID:          types.StringValue(networkId),
			IsEnableAutoRepair: types.BoolValue(autoRepair),
			//DriverInstallationType: types.String{},
			//GpuDriverVersion:       types.StringValue(gpuDriverVersion),
		}

		pool = append(pool, &item)
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

func (d *datasourceManagedKubernetesEngine) topFields() map[string]schema.Attribute {
	topLevelAttributes := map[string]schema.Attribute{}
	// Required string fields
	requiredStrings := []string{
		"vpc_id", "cluster_name", "k8s_version", "purpose",
		"pod_network", "pod_prefix", "service_network", "service_prefix",
		"network_id", "network_overlay", "edge_gateway_id",
	}
	// Optional string fields
	optionalStrings := []string{
		"internal_subnet_lb", "edge_gateway_name", "auto_upgrade_timezone",
	}
	// Required int fields
	requiredInts := []string{"k8s_max_pod", "network_node_prefix"}
	// Optional int fields
	optionalInts := []string{}
	// Optional bool fields
	optionalBools := []string{"is_enable_auto_upgrade"}
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
	topLevelAttributes["network_node_prefix"] = schema.Int64Attribute{
		Required:    true,
		Description: descriptions["network_node_prefix"],
	}

	// Optional nested block: cluster_autoscaler
	topLevelAttributes["cluster_autoscaler"] = schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"isEnableAutoScaling":           schema.BoolAttribute{Optional: true, Description: descriptions["isEnableAutoScaling"]},
			"scaleDownDelayAfterAdd":        schema.Int64Attribute{Optional: true, Description: descriptions["scaleDownDelayAfterAdd"]},
			"scaleDownDelayAfterDelete":     schema.Int64Attribute{Optional: true, Description: descriptions["scaleDownDelayAfterDelete"]},
			"scaleDownDelayAfterFailure":    schema.Int64Attribute{Optional: true, Description: descriptions["scaleDownDelayAfterFailure"]},
			"scaleDownUnneededTime":         schema.Int64Attribute{Optional: true, Description: descriptions["scaleDownUnneededTime"]},
			"scaleDownUtilizationThreshold": schema.Float64Attribute{Optional: true, Description: descriptions["scaleDownUtilizationThreshold"]},
			"scanInterval":                  schema.Int64Attribute{Optional: true, Description: descriptions["scanInterval"]},
			"expander":                      schema.StringAttribute{Optional: true, Description: descriptions["expander"]},
		},
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
	optionalStrings := []string{"tags", "gpuSharingClient", "driverInstallationType"}
	// Required int fields
	requiredInts := []string{"worker_disk_size", "scale_min", "scale_max"}
	// Optional int fields
	optionalInts := []string{"maxClient"}
	// Required bool fields
	requiredBools := []string{"auto_scale", "is_enable_auto_repair"}

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
	// kv: list of map[string]string
	poolLevelAttributes["kv"] = schema.ListAttribute{
		Optional:    true,
		ElementType: types.MapType{ElemType: types.StringType},
		Description: descriptions["kv"],
	}
	// vGpuId: string or null
	poolLevelAttributes["vGpuId"] = schema.StringAttribute{
		Optional:    true,
		Description: descriptions["vGpuId"],
	}
	return poolLevelAttributes
}
