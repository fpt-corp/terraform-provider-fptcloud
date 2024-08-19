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
	"strings"
	"terraform-provider-fptcloud/commons"
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

type resourceManagedKubernetesEngine struct {
	client *commons.Client
}

func NewResourceManagedKubernetesEngine() resource.Resource {
	return &resourceManagedKubernetesEngine{}
}

func (r *resourceManagedKubernetesEngine) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_kubernetes_engine_v1"
}
func (r *resourceManagedKubernetesEngine) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	topLevelAttributes := r.topFields()
	poolAttributes := r.poolFields()

	topLevelAttributes["id"] = schema.StringAttribute{
		Computed: true,
	}
	//topLevelAttributes["pools"] = schema.ListNestedAttribute{Required: true}

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

	var f managedKubernetesEngineJson
	r.remap(&state, &f)
	errDiag := r.fillJson(&f, state.VpcId.ValueString())
	if errDiag != nil {
		response.Diagnostics.Append(errDiag)
		return
	}

	client := r.client
	a, err := client.SendPostRequest(commons.ApiPath.ManagedFKECreate(state.VpcId.ValueString(), "vmw"), f)

	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API", err.Error()))
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

	if err = r.internalRead(ctx, slug, &state); err != nil {
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

	err := r.internalRead(ctx, state.Id.ValueString(), &state)
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

func (r *resourceManagedKubernetesEngine) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	//TODO implement me
	panic("implement me")
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
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API", err.Error()))
		return
	}
}
func (r *resourceManagedKubernetesEngine) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing MFKE cluster ID "+request.ID)

	// lack of ability to import without VPC ID
	response.Diagnostics.Append(diag2.NewErrorDiagnostic("Unimplemented", "Importing DFKE clusters isn't currently supported"))
	return
}
func (r *resourceManagedKubernetesEngine) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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
}
func (r *resourceManagedKubernetesEngine) topFields() map[string]schema.Attribute {
	topLevelAttributes := map[string]schema.Attribute{}
	requiredStrings := []string{
		"vpc_id", "cluster_name", "network_id", "k8s_version", "purpose",
		"pod_network", "pod_prefix", "service_network",
		"range_ip_lb_start", "range_ip_lb_end", "load_balancer_type", "region_id",
	}

	requiredInts := []string{"k8s_max_pod", "network_node_prefix"}

	for _, attribute := range requiredStrings {
		topLevelAttributes[attribute] = schema.StringAttribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersString,
		}
	}

	for _, attribute := range requiredInts {
		topLevelAttributes[attribute] = schema.Int64Attribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersInt,
		}
	}

	return topLevelAttributes
}
func (r *resourceManagedKubernetesEngine) poolFields() map[string]schema.Attribute {
	poolLevelAttributes := map[string]schema.Attribute{}
	requiredStrings := []string{
		"storage_profile", "worker_type", "container_runtime",
		"network_name", "network_id",
		"driver_installation_type", "gpu_driver_version",
	}
	requiredInts := []string{
		"worker_disk_size", "scale_min", "scale_max",
	}

	requiredBool := []string{
		"auto_scale", "is_create", "is_scale", "is_others", "is_enable_auto_repair",
	}

	for _, attribute := range requiredStrings {
		poolLevelAttributes[attribute] = schema.StringAttribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersString,
		}
	}

	for _, attribute := range requiredInts {
		poolLevelAttributes[attribute] = schema.Int64Attribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersInt,
		}
	}

	for _, attribute := range requiredBool {
		poolLevelAttributes[attribute] = schema.BoolAttribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersBool,
		}
	}

	return poolLevelAttributes
}

func (r *resourceManagedKubernetesEngine) fillJson(to *managedKubernetesEngineJson, vpcId string) *diag2.ErrorDiagnostic {
	to.SSHKey = nil
	to.TypeCreate = "create"
	for _, pool := range to.Pools {
		pool.WorkerPoolID = "worker-new"
		pool.ContainerRuntime = "containerd"
		pool.Kv = []struct {
			Name string `json:"name"`
		}([]struct{ Name string }{})
		pool.VGpuID = nil
		pool.IsDisplayGPU = false
	}

	// get k8s versions
	version := to.K8SVersion
	if strings.HasPrefix(version, "v") {
		version = string([]rune(version)[1:])
	}

	osVersion, err := r.getOsVersion(version, vpcId)
	if err != nil {
		return err
	}

	to.OsVersion = osVersion

	return nil
}
func (r *resourceManagedKubernetesEngine) remap(from *managedKubernetesEngine, to *managedKubernetesEngineJson) {
	to.ClusterName = from.ClusterName.ValueString()
	to.NetworkID = from.NetworkID.ValueString()
	to.K8SVersion = from.K8SVersion.ValueString()
	to.Purpose = from.Purpose.ValueString()

	pools := make([]managedKubernetesEnginePoolJson, 0)
	for _, item := range from.Pools {
		newItem := managedKubernetesEnginePoolJson{
			StorageProfile:         item.StorageProfile.ValueString(),
			WorkerType:             item.WorkerType.ValueString(),
			WorkerDiskSize:         item.WorkerDiskSize.ValueInt64(),
			AutoScale:              item.AutoScale.ValueBool(),
			ScaleMin:               item.ScaleMin.ValueInt64(),
			ScaleMax:               item.ScaleMax.ValueInt64(),
			NetworkName:            item.NetworkName.ValueString(),
			NetworkID:              item.NetworkID.ValueString(),
			IsCreate:               item.IsCreate.ValueBool(),
			IsScale:                item.IsScale.ValueBool(),
			IsOthers:               item.IsOthers.ValueBool(),
			IsEnableAutoRepair:     item.IsEnableAutoRepair.ValueBool(),
			DriverInstallationType: item.DriverInstallationType.ValueString(),
			GpuDriverVersion:       item.GpuDriverVersion.ValueString(),
		}

		pools = append(pools, newItem)
	}
	to.Pools = pools

	to.PodNetwork = from.PodNetwork.ValueString()
	to.PodPrefix = from.PodPrefix.ValueString()
	to.ServiceNetwork = from.ServiceNetwork.ValueString()
	to.ServicePrefix = from.ServicePrefix.ValueString()
	to.K8SMaxPod = from.K8SMaxPod.ValueInt64()
	to.NetworkNodePrefix = from.NetworkNodePrefix.ValueInt64()
	to.RangeIPLbStart = from.RangeIPLbStart.ValueString()
	to.RangeIPLbEnd = from.RangeIPLbEnd.ValueString()
	to.LoadBalancerType = from.LoadBalancerType.ValueString()
	to.RegionId = from.RegionId.ValueString()
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

func (r *resourceManagedKubernetesEngine) internalRead(ctx context.Context, id string, state *managedKubernetesEngine) error {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+id+", VPC ID "+vpcId)

	path := commons.ApiPath.ManagedFKEGet(vpcId, "vmw", id)
	a, err := r.client.SendGetRequest(path)
	if err != nil {
		return err
	}

	var d managedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &d)
	if err != nil {
		return err
	}

	if d.Error {
		return errors.New(fmt.Sprintf("Error: %v", d.Mess))
	}

	//state.NetworkID

	return nil
}
func (r *resourceManagedKubernetesEngine) getOsVersion(version string, vpcId string) (interface{}, *diag2.ErrorDiagnostic) {
	var path = commons.ApiPath.GetFKEOSVersion(version, vpcId)
	res, err := r.client.SendGetRequest(path)
	if err != nil {
		diag := diag2.NewErrorDiagnostic("Error calling API", err.Error())
		return nil, &diag
	}

	errorResponse := r.checkForError(res)
	if errorResponse != nil {
		return nil, errorResponse
	}

	var list managedKubernetesEngineOsVersionResponse
	if err = json.Unmarshal(res, &list); err != nil {
		diag := diag2.NewErrorDiagnostic("Error calling API", err.Error())
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

type managedKubernetesEngine struct {
	Id          types.String `tfsdk:"id"`
	VpcId       types.String `tfsdk:"vpc_id"`
	ClusterName types.String `tfsdk:"cluster_name"`
	NetworkID   types.String `tfsdk:"network_id"`
	K8SVersion  types.String `tfsdk:"k8s_version"`
	//OsVersion   struct{} `tfsdk:"os_version"`
	Purpose           types.String                  `tfsdk:"purpose"`
	Pools             []managedKubernetesEnginePool `tfsdk:"pools"`
	PodNetwork        types.String                  `tfsdk:"pod_network"`
	PodPrefix         types.String                  `tfsdk:"pod_prefix"`
	ServiceNetwork    types.String                  `tfsdk:"service_network"`
	ServicePrefix     types.String                  `tfsdk:"service_prefix"`
	K8SMaxPod         types.Int64                   `tfsdk:"k8s_max_pod"`
	NetworkNodePrefix types.Int64                   `tfsdk:"network_node_prefix"`
	RangeIPLbStart    types.String                  `tfsdk:"range_ip_lb_start"`
	RangeIPLbEnd      types.String                  `tfsdk:"range_ip_lb_end"`
	LoadBalancerType  types.String                  `tfsdk:"load_balancer_type"`
	//SSHKey            interface{} `tfsdk:"sshKey"` // just set it nil
	//TypeCreate types.String `tfsdk:"type_create"`
	RegionId types.String `tfsdk:"region_id"`
}
type managedKubernetesEnginePool struct {
	//WorkerPoolID     interface{}  `tfsdk:"worker_pool_id"`
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
	IsCreate               types.Bool   `tfsdk:"is_create"`
	IsScale                types.Bool   `tfsdk:"is_scale"`
	IsOthers               types.Bool   `tfsdk:"is_others"`
	IsEnableAutoRepair     types.Bool   `tfsdk:"is_enable-auto_repair"`
	DriverInstallationType types.String `tfsdk:"driver_installation_type"`
	GpuDriverVersion       types.String `tfsdk:"gpu_driver_version"`
}
type managedKubernetesEngineJson struct {
	ClusterName       string                            `json:"cluster_name"`
	NetworkID         string                            `json:"network_id"`
	K8SVersion        string                            `json:"k8s_version"`
	OsVersion         interface{}                       `json:"os_version"`
	Purpose           string                            `json:"purpose"`
	Pools             []managedKubernetesEnginePoolJson `json:"pools"`
	PodNetwork        string                            `json:"pod_network"`
	PodPrefix         string                            `json:"pod_prefix"`
	ServiceNetwork    string                            `json:"service_network"`
	ServicePrefix     string                            `json:"service_prefix"`
	K8SMaxPod         int64                             `json:"k8s_max_pod"`
	NetworkNodePrefix int64                             `json:"network_node_prefix"`
	RangeIPLbStart    string                            `json:"range_ip_lb_start"`
	RangeIPLbEnd      string                            `json:"range_ip_lb_end"`
	LoadBalancerType  string                            `json:"loadBalancerType"`
	SSHKey            interface{}                       `json:"sshKey"`
	TypeCreate        string                            `json:"type_create"`
	RegionId          string                            `json:"region_id"`
}
type managedKubernetesEnginePoolJson struct {
	WorkerPoolID     interface{} `json:"worker_pool_id"`
	StorageProfile   string      `json:"storage_profile"`
	WorkerType       string      `json:"worker_type"`
	WorkerDiskSize   int64       `json:"worker_disk_size"`
	ContainerRuntime string      `json:"container_runtime"`
	AutoScale        bool        `json:"auto_scale"`
	ScaleMin         int64       `json:"scale_min"`
	ScaleMax         int64       `json:"scale_max"`
	NetworkName      string      `json:"network_name"`
	NetworkID        string      `json:"network_id"`
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
}
