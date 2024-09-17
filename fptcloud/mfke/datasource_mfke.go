package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"
)

var (
	_ datasource.DataSource              = &datasourceManagedKubernetesEngine{}
	_ datasource.DataSourceWithConfigure = &datasourceManagedKubernetesEngine{}
)

type datasourceManagedKubernetesEngine struct {
	client       *commons.Client
	mfkeClient   *MfkeApiClient
	subnetClient *fptcloud_subnet.SubnetClient
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
	d.subnetClient = fptcloud_subnet.NewSubnetClient(d.client)
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

	path := commons.ApiPath.ManagedFKEGet(vpcId, "vmw", id)
	a, err := d.mfkeClient.sendGet(path)
	if err != nil {
		return nil, err
	}

	var response managedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &response)
	if err != nil {
		return nil, err
	}

	if response.Error {
		return nil, errors.New(fmt.Sprintf("Error: %v", response.Mess))
	}

	data := response.Data

	state.Id = types.StringValue(data.Metadata.Name)
	//state.ClusterName = types.StringValue(d.getClusterName(data.Metadata.Name))
	state.VpcId = types.StringValue(vpcId)
	// keep clusterName
	//state.NetworkID
	state.K8SVersion = types.StringValue(data.Spec.Kubernetes.Version)

	cloudPurpose := strings.Split(data.Spec.SeedSelector.MatchLabels.GardenerCloudPurpose, "-")
	state.Purpose = types.StringValue(cloudPurpose[0])

	var poolNames []string

	if len(state.Pools) != 0 {
		existingPool := map[string]*managedKubernetesEnginePool{}
		for _, pool := range state.Pools {
			name := pool.WorkerPoolID.ValueString()
			if _, ok := existingPool[name]; ok {
				return nil, errors.New(fmt.Sprintf("Pool %s already exists", name))
			}

			existingPool[name] = pool
			poolNames = append(poolNames, name)
		}
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

		flavorId, ok := data.Metadata.Labels["fptcloud.com/flavor_pool_test"]
		//if !ok {
		//	return errors.New("missing flavor ID on label fptcloud.com/flavor_pool_test")
		//}
		flavorId = "c89d97cd-c9cb-4d70-a0c1-01f190ea1b02"

		autoRepair := false
		for _, item := range w.Annotations {
			if label, ok := item["worker.fptcloud.com/node-auto-repair"]; ok {
				autoRepair = label == "true"
			}
		}

		networkId, e := getNetworkId(ctx, d.subnetClient, vpcId, w.ProviderConfig.NetworkName)
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
			NetworkName:        types.StringValue(w.ProviderConfig.NetworkName),
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
	state.RangeIPLbStart = types.StringValue(data.Spec.Provider.InfrastructureConfig.Networks.LbIPRangeStart)
	state.RangeIPLbEnd = types.StringValue(data.Spec.Provider.InfrastructureConfig.Networks.LbIPRangeEnd)

	state.LoadBalancerType = types.StringValue(data.Spec.LoadBalancerType)

	return &response, nil
}

func (d *datasourceManagedKubernetesEngine) topFields() map[string]schema.Attribute {
	topLevelAttributes := map[string]schema.Attribute{}
	requiredStrings := []string{
		"vpc_id", "cluster_name", "k8s_version", "purpose",
		"pod_network", "pod_prefix", "service_network", "service_prefix",
		"range_ip_lb_start", "range_ip_lb_end", "load_balancer_type",
	}

	requiredInts := []string{"k8s_max_pod", "network_node_prefix"}

	for _, attribute := range requiredStrings {
		topLevelAttributes[attribute] = schema.StringAttribute{
			Computed: true,
		}
	}

	for _, attribute := range requiredInts {
		topLevelAttributes[attribute] = schema.Int64Attribute{
			Computed: true,
		}
	}

	topLevelAttributes["k8s_version"] = schema.StringAttribute{
		Computed: true,
	}
	topLevelAttributes["network_node_prefix"] = schema.Int64Attribute{
		Computed: true,
	}

	return topLevelAttributes
}
func (d *datasourceManagedKubernetesEngine) poolFields() map[string]schema.Attribute {
	poolLevelAttributes := map[string]schema.Attribute{}
	requiredStrings := []string{
		"name",
		"storage_profile", "worker_type",
		"network_name", "network_id",
		//"driver_installation_type", "gpu_driver_version",
	}
	requiredInts := []string{
		"worker_disk_size", "scale_min", "scale_max",
	}

	requiredBool := []string{
		"auto_scale", "is_enable_auto_repair",
	}

	for _, attribute := range requiredStrings {
		poolLevelAttributes[attribute] = schema.StringAttribute{
			Computed: true,
		}
	}

	for _, attribute := range requiredInts {
		poolLevelAttributes[attribute] = schema.Int64Attribute{
			Computed: true,
		}
	}

	for _, attribute := range requiredBool {
		poolLevelAttributes[attribute] = schema.BoolAttribute{
			Computed: true,
		}
	}

	poolLevelAttributes["scale_min"] = schema.Int64Attribute{
		Computed: true,
	}

	poolLevelAttributes["scale_max"] = schema.Int64Attribute{
		Computed: true,
	}

	return poolLevelAttributes
}
