package fptcloud_dfke

import (
	"context"
	"encoding/json"
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-fptcloud/commons"
)

var (
	_ resource.Resource                = &resourceDedicatedKubernetesEngine{}
	_ resource.ResourceWithConfigure   = &resourceDedicatedKubernetesEngine{}
	_ resource.ResourceWithImportState = &resourceDedicatedKubernetesEngine{}

	forceNewPlanModifiersString = []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	}

	forceNewPlanModifiersInt = []planmodifier.Int64{
		int64planmodifier.RequiresReplace(),
	}
)

type resourceDedicatedKubernetesEngine struct {
	client     *commons.Client
	dfkeClient *dfkeApiClient
}

func (r *resourceDedicatedKubernetesEngine) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var state dedicatedKubernetesEngine
	diags := request.Plan.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var f dedicatedKubernetesEngineJson
	r.remap(&state, &f)

	f.CustomScript = ""
	f.EnableCustomScript = false
	f.PublicKey = ""
	f.UpstreamDNS = ""
	f.RegionId = "saigon-vn"

	client := r.client
	a, err := client.SendPostRequest(fmt.Sprintf("xplat/fke/vpc/%s/kubernetes", state.vpcId()), f)

	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API", err.Error()))
		return
	}

	errorResponse := r.checkForError(a)
	if errorResponse != nil {
		response.Diagnostics.Append(errorResponse)
		return
	}

	var createResponse dedicatedKubernetesEngineCreateResponse
	if err = json.Unmarshal(a, &createResponse); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error()))
		return
	}

	tflog.Info(ctx, "Created cluster with id "+createResponse.Cluster.ID)

	if err = r.internalRead(ctx, createResponse.Cluster.ID, &state); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error reading cluster state", err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceDedicatedKubernetesEngine) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state dedicatedKubernetesEngine
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

func (r *resourceDedicatedKubernetesEngine) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	//TODO implement me
	panic("implement me")
}

func (r *resourceDedicatedKubernetesEngine) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state dedicatedKubernetesEngine
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.client.SendDeleteRequest(fmt.Sprintf("xplat/fke/vpc/%s/cluster/%s/delete", state.vpcId(), state.Id))
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API", err.Error()))
		return
	}
}

func (r *resourceDedicatedKubernetesEngine) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing cluster ID "+request.ID)
	var state dedicatedKubernetesEngine

	state.Id = types.StringValue(request.ID)
	err := r.internalRead(ctx, request.ID, &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API", err.Error()))
		return
	}

	diags := response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func NewResourceDedicatedKubernetesEngine() resource.Resource {
	return &resourceDedicatedKubernetesEngine{}
}

func (r *resourceDedicatedKubernetesEngine) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dedicated_kubernetes_engine_v1"
}

func (r *resourceDedicatedKubernetesEngine) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Manage dedicated FKE clusters.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"cluster_id": schema.StringAttribute{
				Computed: true,
			},
			"cluster_name": schema.StringAttribute{
				Required: true,
			},
			"k8s_version": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"master_type": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"master_count": schema.Int64Attribute{
				Required: true,
			},
			"master_disk_size": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
			},
			"worker_type": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"worker_disk_size": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
			},
			"network_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"lb_size": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"pod_network": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"service_network": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"network_node_prefix": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
			},
			"max_pod_per_node": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
			},
			"nfs_status": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"nfs_disk_size": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
			},
			//"public_key": schema.StringAttribute{
			//	Required:true,
			//	PlanModifiers: forceNewPlanModifiersString,
			//},
			"storage_policy": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"edge_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			//"upstream_dns": schema.Int64Attribute{
			//	Required:true,
			//	PlanModifiers: forceNewPlanModifiersInt,
			//},
			"scale_min": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
			},
			"scale_max": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
			},
			"node_dns": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"ip_public_firewall": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"ip_private_firewall": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
		},
	}
}

func (r *resourceDedicatedKubernetesEngine) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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
	a, err := newDfkeApiClient(client)
	if err != nil {
		response.Diagnostics.AddError(
			"Error configuring API client",
			fmt.Sprintf("%s", err.Error()),
		)

		return
	}

	r.dfkeClient = a
}

func (r *resourceDedicatedKubernetesEngine) internalRead(ctx context.Context, clusterId string, state *dedicatedKubernetesEngine) error {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+clusterId+", VPC ID "+vpcId)

	a, err := r.client.SendGetRequest(fmt.Sprintf("xplat/fke/vpc/%s/cluster/%s?page=1&page_size=25", vpcId, clusterId))

	if err != nil {
		return err
	}

	var d dedicatedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &d)
	data := d.Cluster
	if err != nil {
		return err
	}

	var awx dedicatedKubernetesEngineParams
	err = json.Unmarshal([]byte(d.Cluster.AwxParams), &awx)

	if err != nil {
		return err
	}

	// resolve edge ID
	edge, err := r.dfkeClient.FindEdgeByEdgeGatewayId(vpcId, data.EdgeID)
	if err != nil {
		return err
	}

	state.ClusterId = types.StringValue(data.ClusterID)
	state.ClusterName = types.StringValue(data.Name)
	state.Version = types.StringValue(awx.K8SVersion)
	state.MasterType = types.StringValue(awx.MasterType)
	state.MasterCount = types.Int64Value(int64(awx.MasterCount))
	state.MasterDiskSize = types.Int64Value(int64(awx.MasterDiskSize))
	state.WorkerType = types.StringValue(awx.WorkerType)
	state.WorkerDiskSize = types.Int64Value(int64(awx.WorkerDiskSize))
	state.NetworkID = types.StringValue(data.NetworkID)
	state.LbSize = types.StringValue(awx.LbSize)
	state.PodNetwork = types.StringValue(awx.PodNetwork + "/" + awx.PodPrefix)
	state.ServiceNetwork = types.StringValue(awx.ServiceNetwork + "/" + awx.ServicePrefix)
	state.NetworkNodePrefix = types.Int64Value(int64(awx.NetworkNodePrefix))
	state.MaxPodPerNode = types.Int64Value(int64(awx.K8SMaxPod))
	state.NfsStatus = types.StringValue(awx.NfsStatus)
	state.NfsDiskSize = types.Int64Value(int64(awx.NfsDiskSize))
	state.StoragePolicy = types.StringValue(awx.StorageProfile)
	state.EdgeID = types.StringValue(edge.Id)
	state.ScaleMin = types.Int64Value(int64(awx.ScaleMinSize))
	state.ScaleMax = types.Int64Value(int64(awx.ScaleMaxSize))
	state.NodeDNS = types.StringValue(awx.NodeDNS)
	state.IPPublicFirewall = types.StringValue(awx.IPPublicFirewall)
	state.IPPrivateFirewall = types.StringValue(awx.IPPrivateFirewall)
	state.VpcId = types.StringValue(data.VpcID)
	//state.CustomScript = awx.CustomScript
	//state.EnableCustomScript = awx.EnableCustomScript

	return nil
}

func (r *resourceDedicatedKubernetesEngine) checkForError(a []byte) *diag2.ErrorDiagnostic {
	var re map[string]interface{}
	err := json.Unmarshal(a, &re)
	if err != nil {
		res := diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error())
		return &res
	}

	if _, ok := re["error"]; ok {
		res := diag2.NewErrorDiagnostic("Response contained an error field", "Response body was "+string(a))
		return &res
	}

	return nil
}

func (r *resourceDedicatedKubernetesEngine) remap(from *dedicatedKubernetesEngine, to *dedicatedKubernetesEngineJson) {
	to.ClusterName = from.ClusterName.ValueString()
	to.ClusterId = from.ClusterId.ValueString()
	to.Id = from.Id.ValueString()
	to.Version = from.Version.ValueString()
	to.MasterType = from.MasterType.ValueString()
	to.MasterCount = from.MasterCount.ValueInt64()
	to.MasterDiskSize = from.MasterDiskSize.ValueInt64()
	to.WorkerType = from.WorkerType.ValueString()
	to.WorkerDiskSize = from.WorkerDiskSize.ValueInt64()
	to.NetworkID = from.NetworkID.ValueString()
	to.LbSize = from.LbSize.ValueString()
	to.PodNetwork = from.PodNetwork.ValueString()
	to.ServiceNetwork = from.ServiceNetwork.ValueString()
	to.NetworkNodePrefix = from.NetworkNodePrefix.ValueInt64()
	to.MaxPodPerNode = from.MaxPodPerNode.ValueInt64()
	to.NfsStatus = from.NfsStatus.ValueString()
	to.NfsDiskSize = from.NfsDiskSize.ValueInt64()
	to.StoragePolicy = from.StoragePolicy.ValueString()
	to.EdgeID = from.EdgeID.ValueString()
	to.ScaleMin = from.ScaleMin.ValueInt64()
	to.ScaleMax = from.ScaleMax.ValueInt64()
	to.NodeDNS = from.NodeDNS.ValueString()
	to.IPPublicFirewall = from.IPPublicFirewall.ValueString()
	to.IPPrivateFirewall = from.IPPrivateFirewall.ValueString()
}

func (e *dedicatedKubernetesEngine) vpcId() string {
	return e.VpcId.ValueString()
}

type dedicatedKubernetesEngine struct {
	ClusterName    types.String `tfsdk:"cluster_name" json:"cluster_name"`
	ClusterId      types.String `tfsdk:"cluster_id" json:"cluster_id,omitempty"`
	Id             types.String `tfsdk:"id" json:"id"`
	Version        types.String `tfsdk:"k8s_version" json:"k8s_version"`
	MasterType     types.String `tfsdk:"master_type"` // tfsdk:"master_type"
	MasterCount    types.Int64  `tfsdk:"master_count" json:"master_count"`
	MasterDiskSize types.Int64  `tfsdk:"master_disk_size" json:"master_disk_size"`
	WorkerType     types.String `tfsdk:"worker_type" json:"worker_type"`
	WorkerDiskSize types.Int64  `tfsdk:"worker_disk_size" json:"worker_disk_size"`
	NetworkID      types.String `tfsdk:"network_id" json:"network_id"`
	LbSize         types.String `tfsdk:"lb_size" json:"lb_size"`

	PodNetwork     types.String `tfsdk:"pod_network" json:"pod_network"`
	ServiceNetwork types.String `tfsdk:"service_network" json:"service_network"`

	NetworkNodePrefix types.Int64 `tfsdk:"network_node_prefix" json:"network_node_prefix"`

	MaxPodPerNode types.Int64  `tfsdk:"max_pod_per_node" json:"max_pod_per_node"`
	NfsStatus     types.String `tfsdk:"nfs_status" json:"nfs_status"`
	NfsDiskSize   types.Int64  `tfsdk:"nfs_disk_size" json:"nfs_disk_size"`

	StoragePolicy types.String `tfsdk:"storage_policy" json:"storage_policy"`
	EdgeID        types.String `tfsdk:"edge_id"`

	ScaleMin types.Int64 `tfsdk:"scale_min" json:"scale_min"`
	ScaleMax types.Int64 `tfsdk:"scale_max" json:"scale_max"`

	NodeDNS           types.String `tfsdk:"node_dns" json:"node_dns"`
	IPPublicFirewall  types.String `tfsdk:"ip_public_firewall" json:"ip_public_firewall"`
	IPPrivateFirewall types.String `tfsdk:"ip_private_firewall" json:"ip_private_firewall"`
	VpcId             types.String `tfsdk:"vpc_id" json:"vpc_id"`
}

type dedicatedKubernetesEngineJson struct {
	ClusterName    string `json:"cluster_name"`
	ClusterId      string `json:"cluster_id,omitempty"`
	Id             string `json:"id,omitempty"`
	Version        string `json:"k8s_version"`
	IpPublic       string `json:"ip_public"`
	MasterType     string `json:"master_type"`
	MasterCount    int64  `json:"master_count"`
	MasterDiskSize int64  `json:"master_disk_size"`
	WorkerType     string `json:"worker_type"`
	WorkerDiskSize int64  `json:"worker_disk_size"`
	NetworkID      string `json:"network_id"`
	LbSize         string `json:"lb_size"`

	PodNetwork     string `json:"pod_network"`
	ServiceNetwork string `json:"service_network"`

	NetworkNodePrefix int64 `json:"network_node_prefix"`

	MaxPodPerNode int64  `json:"max_pod_per_node"`
	NfsStatus     string `json:"nfs_status"`
	NfsDiskSize   int64  `json:"nfs_disk_size"`

	StoragePolicy string `json:"storage_policy"`
	EdgeID        string `json:"edge_id"`

	ScaleMin int64 `json:"scale_min"`
	ScaleMax int64 `json:"scale_max"`

	NodeDNS           string `json:"node_dns"`
	IPPublicFirewall  string `json:"ip_public_firewall"`
	IPPrivateFirewall string `json:"ip_private_firewall"`

	CustomScript       string `json:"custom_script"`
	EnableCustomScript bool   `json:"enable_custom_script"`
	PublicKey          string `json:"public_key"`
	UpstreamDNS        string `json:"upstream_dns"`

	RegionId string `json:"region_id"`
}

type dedicatedKubernetesEngineData struct {
	ID           string      `json:"id"`
	ClusterID    string      `json:"cluster_id"`
	VpcID        string      `json:"vpc_id"`
	EdgeID       string      `json:"edge_id"`
	Name         string      `json:"name"`
	AwxParams    string      `json:"awx_params"`
	Status       string      `json:"status"`
	NetworkID    string      `json:"network_id"`
	NfsDiskSize  int         `json:"nfs_disk_size"`
	NfsStatus    string      `json:"nfs_status"`
	ErrorMessage interface{} `json:"error_message"`
	IsRunning    bool        `json:"is_running"`
	AutoScale    string      `json:"auto_scale"`
	ScaleMin     int         `json:"scale_min"`
	ScaleMax     int         `json:"scale_max"`
	Templates    string      `json:"templates"`
	NetworkName  string      `json:"network_name"`
}

type dedicatedKubernetesEngineParams struct {
	VcdURL               string      `json:"vcd_url"`
	PublicDomain         string      `json:"public_domain"`
	ClusterID            string      `json:"cluster_id"`
	ClusterName          string      `json:"cluster_name"`
	OrgName              string      `json:"org_name"`
	VdcName              string      `json:"vdc_name"`
	MasterType           string      `json:"master_type"`
	MasterOs             string      `json:"master_os"`
	MasterCPU            int         `json:"master_cpu"`
	MasterRAM            int         `json:"master_ram"`
	MasterCount          int         `json:"master_count"`
	MasterDiskSize       int         `json:"master_disk_size"`
	WorkerOs             string      `json:"worker_os"`
	WorkerCPU            int         `json:"worker_cpu"`
	WorkerRAM            int         `json:"worker_ram"`
	WorkerCount          int         `json:"worker_count"`
	WorkerType           string      `json:"worker_type"`
	WorkerDiskSize       int         `json:"worker_disk_size"`
	VMPass               string      `json:"vm_pass"`
	StorageProfile       string      `json:"storage_profile"`
	VMNetwork            string      `json:"vm_network"`
	EdgeGatewayID        string      `json:"edge_gateway_id"`
	K8SVersion           string      `json:"k8s_version"`
	PodNetwork           string      `json:"pod_network"`
	PodPrefix            string      `json:"pod_prefix"`
	ServiceNetwork       string      `json:"service_network"`
	ServicePrefix        string      `json:"service_prefix"`
	NetworkNodePrefix    int         `json:"network_node_prefix"`
	K8SMaxPod            int         `json:"k8s_max_pod"`
	IPPublic             string      `json:"ip_public"`
	IDServiceEngineGroup string      `json:"id_service_engine_group"`
	VirtualIPAddress     interface{} `json:"virtual_ip_address"`
	NfsStatus            string      `json:"nfs_status"`
	NfsDiskSize          int         `json:"nfs_disk_size"`
	LbSize               string      `json:"lb_size"`
	DashboardLink        string      `json:"dashboard_link"`
	APILink              string      `json:"api_link"`
	UserName             string      `json:"user_name"`
	AwxJobType           string      `json:"awx_job_type"`
	AutoScaleStatus      string      `json:"auto_scale_status"`
	ScaleMinSize         int         `json:"scale_min_size"`
	ScaleMaxSize         int         `json:"scale_max_size"`
	VpcID                string      `json:"vpc_id"`
	NodeDNS              string      `json:"node_dns"`
	CallbackURL          string      `json:"callback_url"`
	CallbackAction       string      `json:"callback_action"`
	AccessToken          string      `json:"access_token"`
	IPPublicFirewall     string      `json:"ip_public_firewall"`
	IPPrivateFirewall    string      `json:"ip_private_firewall"`
	CustomScript         string      `json:"custom_script"`
	EnableCustomScript   bool        `json:"enable_custom_script"`
	VcdProvider          string      `json:"vcd_provider"`
	VcdPod               string      `json:"vcd_pod"`
	RequestUserID        string      `json:"request_user_id"`
}

type dedicatedKubernetesEngineCreateResponse struct {
	Cluster dedicatedKubernetesEngineData `json:"cluster"`
}

type dedicatedKubernetesEngineReadResponse struct {
	Cluster dedicatedKubernetesEngineData `json:"cluster"`
}
