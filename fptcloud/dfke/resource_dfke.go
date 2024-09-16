package fptcloud_dfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-fptcloud/commons"
	"time"
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
	client           *commons.Client
	dfkeClient       *dfkeApiClient
	tenancyApiClient *tenancyApiClient
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

	client := r.client
	a, err := client.SendPostRequest(fmt.Sprintf("/v1/xplat/fke/vpc/%s/kubernetes", state.vpcId()), f)

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

	state.Id = types.StringValue(createResponse.Cluster.ID)

	if err = r.waitForSucceeded(ctx, &state, 30*time.Minute, true); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error waiting cluster up", err.Error()))
		return
	}
	if _, err = r.internalRead(ctx, createResponse.Cluster.ID, &state); err != nil {
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

	_, err := r.internalRead(ctx, state.Id.ValueString(), &state)
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
	var state dedicatedKubernetesEngine
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan dedicatedKubernetesEngine
	request.Plan.Get(ctx, &plan)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	//tflog.Info(ctx, "Reading existing state of cluster ID "+state.Id.ValueString()+", VPC "+state.vpcId())
	//err := r.internalRead(ctx, state.Id.ValueString(), &existing)
	//if err != nil {
	//	response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting existing state", err.Error()))
	//	return
	//}

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

func (r *resourceDedicatedKubernetesEngine) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state dedicatedKubernetesEngine
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.client.SendDeleteRequest(fmt.Sprintf("/v1/xplat/fke/vpc/%s/cluster/%s/delete", state.vpcId(), state.Id))
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API", err.Error()))
		return
	}
}

func (r *resourceDedicatedKubernetesEngine) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing DFKE cluster ID "+request.ID)

	var state dedicatedKubernetesEngine

	// format: vpcId/clusterId
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
				Required: true,
			},
			"master_count": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
			},
			"master_disk_size": schema.Int64Attribute{
				Required: true,
			},
			"worker_type": schema.StringAttribute{
				Required: true,
			},
			"worker_disk_size": schema.Int64Attribute{
				Required: true,
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
				Required: true,
			},
			"scale_max": schema.Int64Attribute{
				Required: true,
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
			"vpc_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"region_id": schema.StringAttribute{
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

	t := newTenancyApiClient(client)
	r.tenancyApiClient = t
}

func (r *resourceDedicatedKubernetesEngine) internalRead(ctx context.Context, clusterId string, state *dedicatedKubernetesEngine) (*dedicatedKubernetesEngineReadResponse, error) {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+clusterId+", VPC ID "+vpcId)

	a, err := r.client.SendGetRequest(commons.ApiPath.DedicatedFKEGet(vpcId, clusterId))

	if err != nil {
		return nil, err
	}

	var d dedicatedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &d)
	if err != nil {
		return nil, err
	}
	data := d.Cluster

	var awx dedicatedKubernetesEngineParams
	err = json.Unmarshal([]byte(d.Cluster.AwxParams), &awx)

	if err != nil {
		return nil, err
	}

	// resolve edge ID
	edge, err := r.dfkeClient.FindEdgeByEdgeGatewayId(ctx, vpcId, data.EdgeID)
	if err != nil {
		return nil, err
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
	region, err := r.getRegionFromVpcId(ctx, vpcId)
	if err != nil {
		return nil, err
	}
	state.RegionId = types.StringValue(region)

	return &d, nil
}

func (r *resourceDedicatedKubernetesEngine) checkForError(a []byte) *diag2.ErrorDiagnostic {
	var re map[string]interface{}
	err := json.Unmarshal(a, &re)
	if err != nil {
		res := diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error())
		return &res
	}

	if errorField, ok := re["error"]; ok {
		e2, isBool := errorField.(bool)
		if isBool && e2 != false {
			res := diag2.NewErrorDiagnostic(
				fmt.Sprintf("Response contained an error field and value was %t", e2),
				"Response body was "+string(a),
			)
			return &res
		}

		if isBool {
			return nil
		}

		if errorField != nil {
			res := diag2.NewErrorDiagnostic("Response contained an error field", "Response body was "+string(a))
			return &res
		}
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
	to.RegionId = from.RegionId.ValueString()
}

func (r *resourceDedicatedKubernetesEngine) diff(ctx context.Context, from *dedicatedKubernetesEngine, to *dedicatedKubernetesEngine) *diag2.ErrorDiagnostic {
	master := from.MasterDiskSize.ValueInt64()
	master2 := to.MasterDiskSize.ValueInt64()
	// status: EXTENDING
	if master != master2 {
		if master2 < master {
			d := diag2.NewErrorDiagnostic("Wrong master disk size", "Disk cannot be shrinked")
			return &d
		}

		tflog.Info(ctx, fmt.Sprintf("Resizing master from %d to %d", master, master2))

		time.Sleep(5 * time.Second)
		management := dedicatedKubernetesEngineManagement{
			ClusterId:  to.clusterUUID(),
			MgmtAction: "",
			DiskExtend: fmt.Sprintf("%d", master2-master),
			ExtendType: "master",
			Flavor:     "",
			NodeType:   "",
		}

		if err := r.manage(from, management); err != nil {
			return err
		}
		tflog.Info(ctx, fmt.Sprintf("Resized master from %d to %d", master, master2))

		err := r.waitForSucceeded(ctx, from, 5*time.Minute, false)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error waiting for cluster after resizing master disk to return to SUCCEEDED state", err.Error())
			return &d
		}
	}

	worker := from.WorkerDiskSize.ValueInt64()
	worker2 := to.WorkerDiskSize.ValueInt64()
	// status: EXTENDING
	if worker != worker2 {
		if worker2 < worker {
			d := diag2.NewErrorDiagnostic("Wrong worker disk size", "Disk cannot be shrinked")
			return &d
		}

		tflog.Info(ctx, fmt.Sprintf("Resizing worker from %d to %d", worker, worker2))
		management := dedicatedKubernetesEngineManagement{
			ClusterId:  to.clusterUUID(),
			MgmtAction: "",
			DiskExtend: fmt.Sprintf("%d", worker2-worker),
			ExtendType: "worker",
			Flavor:     "",
			NodeType:   "",
		}

		if err := r.manage(from, management); err != nil {
			return err
		}
		tflog.Info(ctx, fmt.Sprintf("Resized worker from %d to %d", worker, worker2))

		err := r.waitForSucceeded(ctx, from, 5*time.Minute, false)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error waiting for cluster after resizing worker disk to return to SUCCEEDED state", err.Error())
			return &d
		}
	}

	masterType := from.MasterType.ValueString()
	master2Type := to.MasterType.ValueString()
	if masterType != master2Type {
		tflog.Info(ctx, fmt.Sprintf("Changing master from %s to %s", masterType, master2Type))

		management := dedicatedKubernetesEngineManagement{
			ClusterId:  from.clusterUUID(),
			MgmtAction: "",
			DiskExtend: "0",
			ExtendType: "",
			Flavor:     master2Type,
			NodeType:   "master",
		}

		if err := r.manage(from, management); err != nil {
			return err
		}
		tflog.Info(ctx, fmt.Sprintf("Changed master from %s to %s", masterType, master2Type))

		err := r.waitForSucceeded(ctx, from, 20*time.Minute, false)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error waiting for cluster after changing master type to return to SUCCEEDED state", err.Error())
			return &d
		}
	}

	workerType := from.WorkerType.ValueString()
	worker2Type := to.WorkerType.ValueString()
	if from.WorkerType != to.WorkerType {
		tflog.Info(ctx, fmt.Sprintf("Changing worker from %s to %s", workerType, worker2Type))

		management := dedicatedKubernetesEngineManagement{
			ClusterId:  from.clusterUUID(),
			MgmtAction: "",
			DiskExtend: "0",
			ExtendType: "",
			Flavor:     worker2Type,
			NodeType:   "worker",
		}

		if err := r.manage(from, management); err != nil {
			return err
		}

		tflog.Info(ctx, fmt.Sprintf("Changed worker from %s to %s", workerType, worker2Type))

		err := r.waitForSucceeded(ctx, from, 20*time.Minute, false)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error waiting for cluster after changing worker type to return to SUCCEEDED state", err.Error())
			return &d
		}
	}

	if (from.ScaleMin.ValueInt64() != to.ScaleMin.ValueInt64()) || (from.ScaleMax.ValueInt64() != to.ScaleMax.ValueInt64()) {
		tflog.Info(ctx, fmt.Sprintf(
			"Changing autoscale from (%d-%d) to (%d-%d)",
			from.ScaleMin.ValueInt64(), from.ScaleMax.ValueInt64(),
			to.ScaleMin.ValueInt64(), to.ScaleMax.ValueInt64(),
		))
		autoScale := dedicatedKubernetesEngineAutoscale{
			ClusterId:   to.clusterUUID(),
			ScaleMin:    to.ScaleMin.ValueInt64(),
			ScaleMax:    to.ScaleMax.ValueInt64(),
			ActionScale: "update",
		}

		if err := r.manage(from, autoScale); err != nil {
			return err
		}

		tflog.Info(ctx, fmt.Sprintf(
			"Changed autoscale to to (%d-%d)",
			to.ScaleMin.ValueInt64(), to.ScaleMax.ValueInt64(),
		))

		err := r.waitForSucceeded(ctx, from, 5*time.Minute, false)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error waiting for cluster after updating autoscale to return to SUCCEEDED state", err.Error())
			return &d
		}
	}

	if from.Version.ValueString() != to.Version.ValueString() {
		//	version changed, call bump version
		path := commons.ApiPath.DedicatedFKEUpgradeVersion(from.vpcId(), from.Id.ValueString())

		version := to.Version.ValueString()
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		change := dedicatedKubernetesEngineUpgradeVersion{
			VersionUpgrade: version,
			ClusterId:      from.clusterUUID(),
		}

		tflog.Info(ctx, fmt.Sprintf("Bumping version to %s", to.Version))

		a, err2 := r.client.SendPostRequest(path, change)
		if err2 != nil {
			d := diag2.NewErrorDiagnostic("Error calling upgrade version API", err2.Error())
			return &d
		}

		if diagErr2 := r.checkForError(a); diagErr2 != nil {
			return diagErr2
		}

		err := r.waitForSucceeded(ctx, from, 1*time.Hour, false)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error waiting for cluster after upgrading to return to SUCCEEDED state", err.Error())
			return &d
		}
	}

	return nil
}

func (r *resourceDedicatedKubernetesEngine) manage(state *dedicatedKubernetesEngine, params interface{}) *diag2.ErrorDiagnostic {
	//path := commons.ApiPath.DedicatedFKEManagement(state.vpcId(), state.clusterUUID())
	path := fmt.Sprintf("/v1/xplat/fke/vpc/%s/cluster/%s/auto-scale", state.vpcId(), state.clusterUUID())

	a, err2 := r.client.SendPostRequest(path, params)
	if err2 != nil {
		d := diag2.NewErrorDiagnostic("Error calling autoscale API", err2.Error())
		return &d
	}

	if diagErr2 := r.checkForError(a); diagErr2 != nil {
		return diagErr2
	}

	return nil
}

func (r *resourceDedicatedKubernetesEngine) waitForSucceeded(ctx context.Context, state *dedicatedKubernetesEngine, timeout time.Duration, ignoreError bool) error {
	clusterId := state.clusterUUID()
	durationText := fmt.Sprintf("%v", timeout)
	tflog.Info(ctx, "Waiting for cluster "+clusterId+" to succeed, duration "+durationText)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	to := time.NewTimer(timeout)
	defer to.Stop()

	for {
		select {
		case <-to.C:
			return errors.New("Timed out waiting for cluster " + clusterId + " to return to success state")
		case <-ticker.C:
			{
				var e error
				tflog.Info(ctx, "Checking status of cluster "+clusterId)

				localTimeout := 200 * time.Millisecond

				for i := 0; i < 5; i++ {
					status, err := r.internalRead(ctx, clusterId, &dedicatedKubernetesEngine{
						ClusterId: state.ClusterId,
						VpcId:     state.VpcId,
					})
					e = err

					if err != nil {
						time.Sleep(localTimeout)
						localTimeout *= 2
						localTimeout = min(localTimeout, 30*time.Second)

						continue
					}

					state := status.Cluster.Status
					tflog.Info(ctx, "Status of cluster "+clusterId+" is currently "+state)
					if state == "SUCCEEDED" {
						return nil
					}

					if state == "ERROR" {
						return errors.New("cluster in error state")
					}

					if state == "STOPPED" {
						return errors.New("cluster is stopped")
					}
				}
				if e != nil && !ignoreError {
					return e
				}
			}
		}
	}
}

func (e *dedicatedKubernetesEngine) vpcId() string {
	return e.VpcId.ValueString()
}
func (e *dedicatedKubernetesEngine) clusterUUID() string {
	return e.Id.ValueString()
}
func (r *resourceDedicatedKubernetesEngine) getRegionFromVpcId(ctx context.Context, vpcId string) (string, error) {
	client := r.tenancyApiClient

	t, err := client.GetTenancy(ctx)
	if err != nil {
		return "", err
	}

	user := t.UserId

	for _, tenant := range t.Tenants {
		regions, e := client.GetRegions(ctx, tenant.Id)
		if e != nil {
			return "", e
		}

		for _, region := range regions {
			vpcs, e2 := client.ListVpcs(ctx, tenant.Id, user, region.Id)
			if e2 != nil {
				return "", e2
			}

			for _, vpc := range vpcs {
				if vpc.Id == vpcId {
					return region.Abbr, nil
				}
			}
		}
	}

	return "", errors.New("no VPC found under this account with vpcId " + vpcId)
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
	RegionId          types.String `tfsdk:"region_id"`
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
	RegionId           string `json:"region_id"`
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

type dedicatedKubernetesEngineUpgradeVersion struct {
	ClusterId      string `json:"cluster_id"`
	VersionUpgrade string `json:"version_upgrade"`
}

type dedicatedKubernetesEngineManagement struct {
	ClusterId  string `json:"cluster_id"`
	MgmtAction string `json:"mgmt_action"`
	DiskExtend string `json:"disk_extend"`
	ExtendType string `json:"extend_type"`
	Flavor     string `json:"flavor"`
	NodeType   string `json:"node_type"`
}

type dedicatedKubernetesEngineAutoscale struct {
	ClusterId   string `json:"cluster_id"`
	ScaleMin    int64  `json:"scale_min"`
	ScaleMax    int64  `json:"scale_max"`
	ActionScale string `json:"action_scale"`
}
