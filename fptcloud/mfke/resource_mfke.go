package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strconv"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_edge_gateway "terraform-provider-fptcloud/fptcloud/edge_gateway"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"
	"unicode"
)

var (
	_ resource.Resource                = &resourceManagedKubernetesEngine{}
	_ resource.ResourceWithConfigure   = &resourceManagedKubernetesEngine{}
	_ resource.ResourceWithImportState = &resourceManagedKubernetesEngine{}

	forceNewPlanModifiersString = []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	}

	forceNewPlanModifiersInt = []planmodifier.Int64{
		int64planmodifier.RequiresReplace(),
	}

	forceNewPlanModifiersBool = []planmodifier.Bool{
		boolplanmodifier.RequiresReplace(),
	}
)

const (
	errorCallingApi        = "Error calling API"
	platformVpcErrorPrefix = "Error getting platform for VPC "
)

type resourceManagedKubernetesEngine struct {
	client        *commons.Client
	mfkeClient    *MfkeApiClient
	subnetClient  fptcloud_subnet.SubnetService
	tenancyClient *fptcloud_dfke.TenancyApiClient
}

func NewResourceManagedKubernetesEngine() resource.Resource {
	return &resourceManagedKubernetesEngine{}
}

func (r *resourceManagedKubernetesEngine) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_kubernetes_engine_v1"
}
func (r *resourceManagedKubernetesEngine) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	topLevelAttributes := r.topFields()
	poolAttributes := r.poolFields()

	topLevelAttributes["id"] = schema.StringAttribute{
		Computed: true,
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
func (r *resourceManagedKubernetesEngine) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var state managedKubernetesEngine
	diags := request.Plan.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validatePool(state.Pools); err != nil {
		response.Diagnostics.Append(err)
		return
	}

	var f managedKubernetesEngineJson
	r.remap(&state, &f)
	errDiag := r.fillJson(ctx, &f, state.VpcId.ValueString())

	if errDiag != nil {
		response.Diagnostics.Append(errDiag)
		return
	}

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, state.VpcId.ValueString())
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting VPC platform", err.Error()))
		return
	}

	if err := validateNetwork(&state, platform); err != nil {
		response.Diagnostics.Append(err)
		return
	}

	path := commons.ApiPath.ManagedFKECreate(state.VpcId.ValueString(), strings.ToLower(platform))
	tflog.Info(ctx, "Calling path "+path)
	a, err := r.mfkeClient.sendPost(path, platform, f)

	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi, err.Error()))
		return
	}

	errorResponse := r.checkForError(a)
	if errorResponse != nil {
		response.Diagnostics.Append(errorResponse)
		return
	}

	var createResponse managedKubernetesEngineCreateResponse
	if err = json.Unmarshal(a, &createResponse); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error()))
		return
	}

	slug := fmt.Sprintf("%s-%s", createResponse.Kpi.ClusterName, createResponse.Kpi.ClusterId)

	tflog.Info(ctx, "Created cluster with id "+slug)

	if _, err = r.internalRead(ctx, slug, &state); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error reading cluster state", err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceManagedKubernetesEngine) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state managedKubernetesEngine
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.internalRead(ctx, state.Id.ValueString(), &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi, err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceManagedKubernetesEngine) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state managedKubernetesEngine
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan managedKubernetesEngine
	request.Plan.Get(ctx, &plan)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	errDiag := r.diff(ctx, &state, &plan)
	if errDiag != nil {
		response.Diagnostics.Append(errDiag)
		return
	}

	_, err := r.internalRead(ctx, state.Id.ValueString(), &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error refreshing state", err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceManagedKubernetesEngine) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state managedKubernetesEngine
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.client.SendDeleteRequest(
		commons.ApiPath.ManagedFKEDelete(state.VpcId.ValueString(), "vmw", state.ClusterName.ValueString()),
	)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi, err.Error()))
		return
	}
}
func (r *resourceManagedKubernetesEngine) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing MFKE cluster ID "+request.ID)

	var state managedKubernetesEngine

	id := request.ID
	pieces := strings.Split(id, "/")
	if len(pieces) != 2 {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Invalid format", "must be in format vpcId/clusterId"))
		return
	}

	vpcId := pieces[0]
	clusterId := pieces[1]

	state.VpcId = types.StringValue(vpcId)

	state.Id = types.StringValue(clusterId)

	_, err := r.internalRead(ctx, clusterId, &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi, err.Error()))
		return
	}

	diags := response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}
func (r *resourceManagedKubernetesEngine) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

	r.client = client
	r.mfkeClient = newMfkeApiClient(r.client)
	r.subnetClient = fptcloud_subnet.NewSubnetService(r.client)
	r.tenancyClient = fptcloud_dfke.NewTenancyApiClient(r.client)
}
func (r *resourceManagedKubernetesEngine) topFields() map[string]schema.Attribute {
	topLevelAttributes := map[string]schema.Attribute{}
	requiredStrings := []string{
		"vpc_id", "cluster_name", "k8s_version", "purpose",
		"pod_network", "pod_prefix", "service_network", "service_prefix",
		"range_ip_lb_start", "range_ip_lb_end", "load_balancer_type", "network_id", "network_overlay",
		"edge_gateway_id",
	}

	requiredInts := []string{"k8s_max_pod", "network_node_prefix"}

	for _, attribute := range requiredStrings {
		topLevelAttributes[attribute] = schema.StringAttribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersString,
			Description:   descriptions[attribute],
		}
	}

	for _, attribute := range requiredInts {
		topLevelAttributes[attribute] = schema.Int64Attribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersInt,
			Description:   descriptions[attribute],
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

	return topLevelAttributes
}
func (r *resourceManagedKubernetesEngine) poolFields() map[string]schema.Attribute {
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
			Required:      true,
			PlanModifiers: forceNewPlanModifiersString,
			Description:   descriptions[attribute],
		}
	}

	for _, attribute := range requiredInts {
		poolLevelAttributes[attribute] = schema.Int64Attribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersInt,
			Description:   descriptions[attribute],
		}
	}

	for _, attribute := range requiredBool {
		poolLevelAttributes[attribute] = schema.BoolAttribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersBool,
			Description:   descriptions[attribute],
		}
	}

	poolLevelAttributes["scale_min"] = schema.Int64Attribute{
		Required:    true,
		Description: descriptions["scale_min"],
	}

	poolLevelAttributes["scale_max"] = schema.Int64Attribute{
		Required:    true,
		Description: descriptions["scale_max"],
	}

	return poolLevelAttributes
}

func (r *resourceManagedKubernetesEngine) fillJson(ctx context.Context, to *managedKubernetesEngineJson, vpcId string) *diag2.ErrorDiagnostic {
	to.SSHKey = nil
	to.TypeCreate = "create"
	to.NetworkType = "calico"
	for _, pool := range to.Pools {
		pool.ContainerRuntime = "containerd"
		pool.DriverInstallationType = "pre-install"
		pool.GpuDriverVersion = "default"
		pool.Kv = []struct {
			Name string `json:"name"`
		}([]struct{ Name string }{})
		pool.VGpuID = nil
		pool.IsDisplayGPU = false
		pool.IsCreate = true
		pool.IsScale = false
		pool.IsOthers = false

		pool.GpuSharingClient = ""
		pool.Tags = ""
	}

	// get k8s versions
	version := to.K8SVersion
	if strings.HasPrefix(version, "v") {
		version = string([]rune(version)[1:])
	}

	osVersion, err := r.getOsVersion(ctx, version, vpcId)
	if err != nil {
		return err
	}

	to.OsVersion = osVersion
	to.InternalSubnetLb = nil

	platform, e := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if e != nil {
		d := diag2.NewErrorDiagnostic("Error getting platform for VPC "+vpcId, e.Error())
		return &d
	}

	if strings.ToLower(platform) == "osp" {
		to.EdgeGatewayId = ""
		to.EdgeGatewayName = ""
	} else {
		// get edge gateway name
		edgeGatewayId := to.EdgeGatewayId
		edge, err := r.getEdgeGateway(ctx, edgeGatewayId, vpcId)
		if err != nil {
			return err
		}
		to.EdgeGatewayName = edge.Name
	}

	to.ClusterEndpointAccess = struct {
		Type      string   `json:"type"`
		AllowCidr []string `json:"allowCidr"`
	}{Type: "public", AllowCidr: []string{}}

	return nil
}
func (r *resourceManagedKubernetesEngine) remap(from *managedKubernetesEngine, to *managedKubernetesEngineJson) {
	to.ClusterName = from.ClusterName.ValueString()
	to.K8SVersion = from.K8SVersion.ValueString()
	to.Purpose = from.Purpose.ValueString()

	pools := make([]*managedKubernetesEnginePoolJson, 0)
	for _, item := range from.Pools {
		name := item.WorkerPoolID.ValueString()
		newItem := managedKubernetesEnginePoolJson{
			StorageProfile:     item.StorageProfile.ValueString(),
			WorkerType:         item.WorkerType.ValueString(),
			WorkerDiskSize:     item.WorkerDiskSize.ValueInt64(),
			AutoScale:          item.AutoScale.ValueBool(),
			ScaleMin:           item.ScaleMin.ValueInt64(),
			ScaleMax:           item.ScaleMax.ValueInt64(),
			NetworkName:        item.NetworkName.ValueString(),
			NetworkID:          item.NetworkID.ValueString(),
			IsEnableAutoRepair: item.IsEnableAutoRepair.ValueBool(),
			WorkerPoolID:       &name,
		}

		pools = append(pools, &newItem)
	}
	to.Pools = pools

	to.NetworkID = to.Pools[0].NetworkID

	to.PodNetwork = from.PodNetwork.ValueString()
	to.PodPrefix = from.PodPrefix.ValueString()
	to.ServiceNetwork = from.ServiceNetwork.ValueString()
	to.ServicePrefix = from.ServicePrefix.ValueString()
	to.K8SMaxPod = from.K8SMaxPod.ValueInt64()
	to.NetworkNodePrefix = from.NetworkNodePrefix.ValueInt64()
	to.RangeIPLbStart = from.RangeIPLbStart.ValueString()
	to.RangeIPLbEnd = from.RangeIPLbEnd.ValueString()
	to.LoadBalancerType = from.LoadBalancerType.ValueString()
	to.NetworkOverlay = from.NetworkOverlay.ValueString()
	to.EdgeGatewayId = from.EdgeGatewayId.ValueString()
}

func (r *resourceManagedKubernetesEngine) remapPools(item *managedKubernetesEnginePool, name string) *managedKubernetesEnginePoolJson {
	newItem := managedKubernetesEnginePoolJson{
		StorageProfile:     item.StorageProfile.ValueString(),
		WorkerType:         item.WorkerType.ValueString(),
		WorkerDiskSize:     item.WorkerDiskSize.ValueInt64(),
		AutoScale:          item.AutoScale.ValueBool(),
		ScaleMin:           item.ScaleMin.ValueInt64(),
		ScaleMax:           item.ScaleMax.ValueInt64(),
		NetworkName:        item.NetworkName.ValueString(),
		NetworkID:          item.NetworkID.ValueString(),
		IsEnableAutoRepair: item.IsEnableAutoRepair.ValueBool(),
		WorkerPoolID:       &name,
	}

	return &newItem
}

func (r *resourceManagedKubernetesEngine) checkForError(a []byte) *diag2.ErrorDiagnostic {
	var re map[string]interface{}
	err := json.Unmarshal(a, &re)
	if err != nil {
		res := diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error())
		return &res
	}

	if e, ok := re["error"]; ok {
		if e == true {
			res := diag2.NewErrorDiagnostic("Response contained an error field", "Response body was "+string(a))
			return &res
		}
	}

	return nil
}

func (r *resourceManagedKubernetesEngine) diff(ctx context.Context, from *managedKubernetesEngine, to *managedKubernetesEngine) *diag2.ErrorDiagnostic {
	if from.K8SVersion != to.K8SVersion {
		if err := r.upgradeVersion(ctx, from, to); err != nil {
			return err
		}
	}
	if from.NetworkNodePrefix != to.NetworkNodePrefix {
		from.NetworkNodePrefix = to.NetworkNodePrefix
	}

	editGroup := r.diffPool(ctx, from, to)

	if editGroup {
		d, err := r.internalRead(ctx, from.Id.ValueString(), from)
		if err != nil {
			di := diag2.NewErrorDiagnostic("Error reading cluster state", err.Error())
			return &di
		}

		pools := []*managedKubernetesEnginePoolJson{}

		for _, pool := range to.Pools {
			item := r.remapPools(pool, pool.WorkerPoolID.ValueString())
			pools = append(pools, item)
		}

		body := managedKubernetesEngineEditWorker{
			K8sVersion:        to.K8SVersion.ValueString(),
			CurrentNetworking: d.Data.Spec.Networking.Nodes,
			Pools:             pools,
			TypeConfigure:     "configure",
		}

		vpcId := from.VpcId.ValueString()
		platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
		if err != nil {
			d := diag2.NewErrorDiagnostic(platformVpcErrorPrefix+vpcId, err.Error())
			return &d
		}

		platform = strings.ToLower(platform)

		path := fmt.Sprintf(
			"/v1/xplat/fke/vpc/%s/m-fke/%s/configure-worker-cluster/shoots/%s/0",
			from.VpcId.ValueString(),
			platform,
			from.Id.ValueString(),
		)

		res, err := r.mfkeClient.sendPatch(path, platform, body)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error configuring worker", err.Error())
			return &d
		}

		if e2 := r.checkForError(res); e2 != nil {
			return e2
		}
	}

	return nil
}

func (r *resourceManagedKubernetesEngine) upgradeVersion(ctx context.Context, from, to *managedKubernetesEngine) *diag2.ErrorDiagnostic {
	// upgrade version
	vpcId := from.VpcId.ValueString()
	cluster := from.Id.ValueString()
	targetVersion := to.K8SVersion.ValueString()

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		d := diag2.NewErrorDiagnostic(platformVpcErrorPrefix+vpcId, err.Error())
		return &d
	}

	platform = strings.ToLower(platform)

	path := fmt.Sprintf(
		"/v1/xplat/fke/vpc/%s/m-fke/%s/upgrade_version_cluster/shoots/%s/k8s-version/%s",
		vpcId,
		platform,
		cluster,
		targetVersion,
	)

	body, err := r.mfkeClient.sendPatch(path, platform, struct{}{})
	if err != nil {
		d := diag2.NewErrorDiagnostic(
			fmt.Sprintf("Error upgrading version to %s", to.K8SVersion.ValueString()),
			err.Error(),
		)

		return &d
	}

	if diagErr2 := r.checkForError(body); diagErr2 != nil {
		return diagErr2
	}

	return nil
}

func (r *resourceManagedKubernetesEngine) diffPool(_ context.Context, from *managedKubernetesEngine, to *managedKubernetesEngine) bool {
	fromPool := map[string]*managedKubernetesEnginePool{}
	toPool := map[string]*managedKubernetesEnginePool{}

	for _, pool := range from.Pools {
		fromPool[pool.WorkerPoolID.ValueString()] = pool
	}

	for _, pool := range to.Pools {
		toPool[pool.WorkerPoolID.ValueString()] = pool
	}

	if len(fromPool) != len(toPool) {
		return true
	}

	for _, pool := range from.Pools {
		f := fromPool[pool.WorkerPoolID.ValueString()]
		t := toPool[pool.WorkerPoolID.ValueString()]
		if f.ScaleMin != t.ScaleMin || f.ScaleMax != t.ScaleMax {
			return true
		}
	}

	return false
}

func (r *resourceManagedKubernetesEngine) internalRead(ctx context.Context, id string, state *managedKubernetesEngine) (*managedKubernetesEngineReadResponse, error) {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+id+", VPC ID "+vpcId)

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return nil, err
	}

	platform = strings.ToLower(platform)

	path := commons.ApiPath.ManagedFKEGet(vpcId, platform, id)
	a, err := r.mfkeClient.sendGet(path, platform)
	if err != nil {
		return nil, err
	}

	var d managedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &d)
	if err != nil {
		return nil, err
	}

	if d.Error {
		return nil, fmt.Errorf("error: %v", d.Mess)
	}

	data := d.Data

	state.Id = types.StringValue(data.Metadata.Name)
	state.ClusterName = types.StringValue(getClusterName(data.Metadata.Name))
	state.VpcId = types.StringValue(vpcId)
	// keep clusterName
	//state.NetworkID
	state.K8SVersion = types.StringValue(data.Spec.Kubernetes.Version)

	cloudPurpose := strings.Split(data.Spec.SeedSelector.MatchLabels.GardenerCloudPurpose, "-")
	state.Purpose = types.StringValue(cloudPurpose[0])

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

		networkId, networkName, e := getNetworkInfoByPlatform(ctx, r.subnetClient, vpcId, platform, w, &data)
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

		if strings.ToLower(platform) == "osp" {
			item.NetworkName = types.StringValue(networkName)
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

	return &d, nil
}
func (r *resourceManagedKubernetesEngine) getOsVersion(ctx context.Context, version string, vpcId string) (interface{}, *diag2.ErrorDiagnostic) {
	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		d := diag2.NewErrorDiagnostic(platformVpcErrorPrefix+vpcId, err.Error())
		return nil, &d
	}

	platform = strings.ToLower(platform)

	var path = commons.ApiPath.GetFKEOSVersion(vpcId, platform)
	tflog.Info(ctx, "Getting OS version for version "+version+", VPC ID "+vpcId)
	tflog.Info(ctx, "Calling "+path)

	res, err := r.mfkeClient.sendGet(path, platform)
	if err != nil {
		diag := diag2.NewErrorDiagnostic(errorCallingApi, err.Error())
		return nil, &diag
	}

	errorResponse := r.checkForError(res)
	if errorResponse != nil {
		return nil, errorResponse
	}

	var list managedKubernetesEngineOsVersionResponse
	if err = json.Unmarshal(res, &list); err != nil {
		diag := diag2.NewErrorDiagnostic(errorCallingApi, err.Error())
		return nil, &diag
	}

	for _, item := range list.Data {
		if item.Value == version {
			return item.OsVersion, nil
		}
	}

	diag := diag2.NewErrorDiagnostic("Error finding OS version", "K8s version "+version+" not found")
	return nil, &diag
}
func (r *resourceManagedKubernetesEngine) getEdgeGateway(_ context.Context, edgeId string, vpcId string) (*fptcloud_edge_gateway.EdgeGatewayData, *diag2.ErrorDiagnostic) {
	res, err := r.client.SendGetRequest(commons.ApiPath.EdgeGatewayList(vpcId))

	if err != nil {
		d := diag2.NewErrorDiagnostic(errorCallingApi, err.Error())
		return nil, &d
	}

	var resp fptcloud_edge_gateway.EdgeGatewayResponse
	if err = json.Unmarshal(res, &resp); err != nil {
		diag := diag2.NewErrorDiagnostic(errorCallingApi, err.Error())
		return nil, &diag
	}

	for _, item := range resp.Data {
		if item.EdgeGatewayId == edgeId {
			return &item, nil
		}
	}

	diag := diag2.NewErrorDiagnostic("No such Edge Gateway in this VPC", fmt.Sprintf("No edge gateway with ID %s was found in VPC %s", edgeId, vpcId))
	return nil, &diag
}

func getClusterName(name string) string {
	var indices []int
	for i, c := range name {
		if c == '-' {
			indices = append(indices, i)
		}
	}

	if len(indices) == 0 {
		return name
	}

	last := indices[len(indices)-1]
	clusterName := string([]rune(name)[:last])

	return clusterName
}

func parseNumber(s string) int {
	out := ""
	for _, c := range s {
		if unicode.IsDigit(c) {
			out += string(c)
		}
	}

	if out == "" {
		out = "0"
	}

	f, _ := strconv.Atoi(out)
	return f
}

type managedKubernetesEngine struct {
	Id          types.String `tfsdk:"id"`
	VpcId       types.String `tfsdk:"vpc_id"`
	ClusterName types.String `tfsdk:"cluster_name"`
	NetworkID   types.String `tfsdk:"network_id"`
	K8SVersion  types.String `tfsdk:"k8s_version"`
	//OsVersion   struct{} `tfsdk:"os_version"`
	Purpose           types.String                   `tfsdk:"purpose"`
	Pools             []*managedKubernetesEnginePool `tfsdk:"pools"`
	PodNetwork        types.String                   `tfsdk:"pod_network"`
	PodPrefix         types.String                   `tfsdk:"pod_prefix"`
	ServiceNetwork    types.String                   `tfsdk:"service_network"`
	ServicePrefix     types.String                   `tfsdk:"service_prefix"`
	K8SMaxPod         types.Int64                    `tfsdk:"k8s_max_pod"`
	NetworkNodePrefix types.Int64                    `tfsdk:"network_node_prefix"`
	RangeIPLbStart    types.String                   `tfsdk:"range_ip_lb_start"`
	RangeIPLbEnd      types.String                   `tfsdk:"range_ip_lb_end"`
	LoadBalancerType  types.String                   `tfsdk:"load_balancer_type"`
	NetworkOverlay    types.String                   `tfsdk:"network_overlay"`
	//SSHKey            interface{} `tfsdk:"sshKey"` // just set it nil
	//TypeCreate types.String `tfsdk:"type_create"`
	//RegionId types.String `tfsdk:"region_id"`

	EdgeGatewayId types.String `tfsdk:"edge_gateway_id"`
}
type managedKubernetesEnginePool struct {
	WorkerPoolID   types.String `tfsdk:"name"`
	StorageProfile types.String `tfsdk:"storage_profile"`
	WorkerType     types.String `tfsdk:"worker_type"`
	WorkerDiskSize types.Int64  `tfsdk:"worker_disk_size"`
	//ContainerRuntime types.String `tfsdk:"container_runtime"`
	AutoScale   types.Bool   `tfsdk:"auto_scale"`
	ScaleMin    types.Int64  `tfsdk:"scale_min"`
	ScaleMax    types.Int64  `tfsdk:"scale_max"`
	NetworkName types.String `tfsdk:"network_name"`
	NetworkID   types.String `tfsdk:"network_id"`
	//Kv               []struct {
	//	Name types.String `tfsdk:"name"`
	//} `tfsdk:"kv"`
	//VGpuID                 interface{}  `tfsdk:"vGpuId"`
	//IsDisplayGPU           bool         `tfsdk:"isDisplayGPU"`
	//IsCreate               types.Bool   `tfsdk:"is_create"`
	//IsScale                types.Bool   `tfsdk:"is_scale"`
	//IsOthers               types.Bool   `tfsdk:"is_others"`
	IsEnableAutoRepair types.Bool `tfsdk:"is_enable_auto_repair"`
	//DriverInstallationType types.String `tfsdk:"driver_installation_type"`
	//GpuDriverVersion       types.String `tfsdk:"gpu_driver_version"`
}
type managedKubernetesEngineJson struct {
	ClusterName       string                             `json:"cluster_name"`
	NetworkID         string                             `json:"network_id"`
	K8SVersion        string                             `json:"k8s_version"`
	OsVersion         interface{}                        `json:"os_version"`
	Purpose           string                             `json:"purpose"`
	Pools             []*managedKubernetesEnginePoolJson `json:"pools"`
	PodNetwork        string                             `json:"pod_network"`
	PodPrefix         string                             `json:"pod_prefix"`
	ServiceNetwork    string                             `json:"service_network"`
	ServicePrefix     string                             `json:"service_prefix"`
	K8SMaxPod         int64                              `json:"k8s_max_pod"`
	NetworkNodePrefix int64                              `json:"network_node_prefix"`
	RangeIPLbStart    string                             `json:"range_ip_lb_start"`
	RangeIPLbEnd      string                             `json:"range_ip_lb_end"`
	LoadBalancerType  string                             `json:"loadBalancerType"`
	NetworkType       string                             `json:"network_type"`
	SSHKey            interface{}                        `json:"sshKey"`
	TypeCreate        string                             `json:"type_create"`
	NetworkOverlay    string                             `json:"network_overlay"`
	InternalSubnetLb  interface{}                        `json:"internal_subnet_lb"`
	//RegionId          string                             `json:"region_id"`
	EdgeGatewayId         string      `json:"edge_gateway_id,omitempty"`
	EdgeGatewayName       string      `json:"edge_gateway_name,omitempty"`
	ClusterEndpointAccess interface{} `json:"clusterEndpointAccess"`
}
type managedKubernetesEnginePoolJson struct {
	WorkerPoolID     *string `json:"worker_pool_id"`
	StorageProfile   string  `json:"storage_profile"`
	WorkerType       string  `json:"worker_type"`
	WorkerDiskSize   int64   `json:"worker_disk_size"`
	ContainerRuntime string  `json:"container_runtime"`
	AutoScale        bool    `json:"auto_scale"`
	ScaleMin         int64   `json:"scale_min"`
	ScaleMax         int64   `json:"scale_max"`
	NetworkName      string  `json:"network_name"`
	NetworkID        string  `json:"network_id"`
	Kv               []struct {
		Name string `json:"name"`
	} `json:"kv"`
	VGpuID                 interface{} `json:"vGpuId"`
	IsDisplayGPU           bool        `json:"isDisplayGPU"`
	IsCreate               bool        `json:"isCreate"`
	IsScale                bool        `json:"isScale"`
	IsOthers               bool        `json:"isOthers"`
	IsEnableAutoRepair     bool        `json:"isEnableAutoRepair"`
	DriverInstallationType string      `json:"driverInstallationType"`
	GpuDriverVersion       string      `json:"gpuDriverVersion"`
	Tags                   string      `json:"tags"`
	GpuSharingClient       string      `json:"gpuSharingClient"`
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
type managedKubernetesEngineOsVersionResponse struct {
	Error bool `json:"error"`
	Data  []struct {
		Label     string      `json:"label"`
		OsVersion interface{} `json:"os_version"`
		Value     string      `json:"value"`
	} `json:"data"`
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
}
type managedKubernetesEngineDataMetadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}
type managedKubernetesEngineDataSpec struct {
	Kubernetes struct {
		Kubelet struct {
			MaxPods int `json:"maxPods"`
		} `json:"kubelet"`
		Version string `json:"version"`
	} `json:"kubernetes"`
	LoadBalancerType string `json:"loadBalancerType"`
	Networking       struct {
		Nodes    string `json:"nodes"`
		Pods     string `json:"pods"`
		Services string `json:"services"`
		Type     string `json:"type"`
	} `json:"networking"`

	SeedSelector struct {
		MatchLabels struct {
			GardenerCloudPurpose string `json:"gardener_cloud_purpose"`
		} `json:"matchLabels"`
	} `json:"seedSelector"`

	Provider struct {
		InfrastructureConfig struct {
			Networks struct {
				Id             string `json:"id"`
				LbIPRangeEnd   string `json:"lbIpRangeEnd"`
				LbIPRangeStart string `json:"lbIpRangeStart"`
				Workers        string `json:"workers"`
			} `json:"networks"`
		} `json:"infrastructureConfig"`
		Workers []*managedKubernetesEngineDataWorker `json:"workers"`
	} `json:"provider"`

	Hibernate *struct {
		Enabled bool `json:"enabled"`
	} `json:"hibernation"`
}

type managedKubernetesEngineDataWorker struct {
	Annotations map[string]string `json:"annotations"`
	Kubernetes  struct {
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
			DriverInstallationType *string `json:"driverInstallationType"`
			GpuDriverVersion       *string `json:"gpuDriverVersion"`
			Name                   string  `json:"name"`
			Version                string  `json:"version"`
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
		VGpuID      interface{} `json:"vGpuId"`
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

func (w *managedKubernetesEngineDataWorker) AutoRepair() bool {
	autoRepair := false
	if label, ok := w.Annotations["worker.fptcloud.com/node-auto-repair"]; ok {
		autoRepair = label == "true"
	}

	return autoRepair
}

type managedKubernetesEngineEditWorker struct {
	Pools             []*managedKubernetesEnginePoolJson `json:"pools"`
	K8sVersion        string                             `json:"k8s_version"`
	TypeConfigure     string                             `json:"type_configure"`
	CurrentNetworking string                             `json:"currentNetworking"`
}
