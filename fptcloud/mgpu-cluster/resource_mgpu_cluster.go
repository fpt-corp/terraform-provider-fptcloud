package fptcloud_mgpu_cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"

	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &resourceManagedGpuCluster{}
	_ resource.ResourceWithConfigure   = &resourceManagedGpuCluster{}
	_ resource.ResourceWithImportState = &resourceManagedGpuCluster{}

	forceNewPlanModifiersString = []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	}

	forceNewPlanModifiersInt = []planmodifier.Int64{
		int64planmodifier.RequiresReplace(),
	}
)

const (
	platformVpcErrorPrefix = "Error getting platform for VPC "
)

func NewResourceManagedGpuCluster() resource.Resource {
	return &resourceManagedGpuCluster{}
}

func (r *resourceManagedGpuCluster) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_gpu_cluster"
}

func (r *resourceManagedGpuCluster) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	topLevelAttributes := TopFields()
	poolAttributes := PoolFields()

	topLevelAttributes["id"] = schema.StringAttribute{
		Computed: true,
	}

	response.Schema = schema.Schema{
		Description: "Manage Bare Metal (GPU) Kubernetes clusters.",
		Attributes:  topLevelAttributes,
		Blocks: map[string]schema.Block{
			"pools": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: poolAttributes,
				},
			},
		},
	}
}

func (r *resourceManagedGpuCluster) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var state managedGpuCluster
	diags := request.Plan.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, state.VpcId.ValueString())
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting VPC platform", err.Error()))
		return
	}

	// Set all defaults in one place
	SetDefaults(&state)

	// Validate all in one place
	if !ValidateCreate(&state, response) {
		return
	}

	var f managedGpuClusterJson
	errDiag := MapTerraformToJson(r, ctx, &state, &f, state.VpcId.ValueString())

	if errDiag != nil {
		response.Diagnostics.Append(errDiag)
		return
	}

	if checkClusterName(f.ClusterName) {
		originalName := f.ClusterName
		randomSuffix := GenerateRandomSuffix()
		f.ClusterName = fmt.Sprintf("%s-%s", f.ClusterName, randomSuffix)
		tflog.Info(ctx, fmt.Sprintf("Auto-generated random suffix for cluster_name: %s -> %s", originalName, f.ClusterName))
	}

	if err := validateNetwork(&state, platform); err != nil {
		response.Diagnostics.Append(err)
		return
	}

	// Check service account before creating
	serviceAccountEnabled, err := r.mgpuClusterClient.checkServiceAccount(ctx, state.VpcId.ValueString(), platform)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error checking service account, please try again", err.Error()))
		return
	}
	if !serviceAccountEnabled {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("VPC does not have service account", "VPC does not have service account"))
		return
	}

	// Check quota resource before creating
	// quotaCheckPassed, err := r.mgpuClusterClient.checkQuotaResource(ctx, state.VpcId.ValueString(), platform)
	// if err != nil {
	// 	response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error checking quota resource", err.Error()))
	// 	return
	// }
	// if !quotaCheckPassed {
	// 	response.Diagnostics.Append(diag2.NewErrorDiagnostic("Quota resource check failed", "Quota resource check failed"))
	// 	return
	// }

	path := commons.ApiPath.ManagedGpuClusterCreate(state.VpcId.ValueString(), strings.ToLower(platform))
	if requiresV2API(f.K8SVersion) {
		path = commons.ApiPath.ManagedGpuClusterCreateV2(state.VpcId.ValueString(), strings.ToLower(platform))
		f.IsV2 = true
	}
	tflog.Info(ctx, "Calling path "+path)
	a, err := r.mgpuClusterClient.sendPost(ctx, path, platform, f)

	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi(path), err.Error()))
		return
	}

	errorResponse := r.CheckForError(a)
	if errorResponse != nil {
		response.Diagnostics.Append(errorResponse)
		return
	}

	var createResponse managedGpuClusterCreateResponse
	if err = json.Unmarshal(a, &createResponse); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error()))
		return
	}

	// Build slug: if cluster_name already ends with cluster_id, use cluster_name directly
	// Otherwise, append cluster_id to cluster_name
	var slug string
	suffix := "-" + createResponse.Kpi.ClusterId
	if strings.HasSuffix(createResponse.Kpi.ClusterName, suffix) {
		slug = createResponse.Kpi.ClusterName
	} else {
		slug = fmt.Sprintf("%s-%s", createResponse.Kpi.ClusterName, createResponse.Kpi.ClusterId)
	}

	tflog.Info(ctx, "Created cluster with id "+slug)

	if _, err = r.InternalRead(ctx, slug, &state); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error reading cluster state", err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceManagedGpuCluster) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state managedGpuCluster
	diags := request.State.Get(ctx, &state)
	tflog.Info(ctx, "State after request.State.Get: "+fmt.Sprintf("%#v", state))

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.InternalRead(ctx, state.Id.ValueString(), &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi("internalRead"), err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceManagedGpuCluster) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state managedGpuCluster
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan managedGpuCluster
	request.Plan.Get(ctx, &plan)

	// Default optional fields in plan to state if not specified
	SetDefaultsUpdate(&plan, &state)

	// Validate all in one place for update
	if !ValidateUpdate(&state, &plan, response) {
		return
	}

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "[DEBUG] State in Update: "+fmt.Sprintf("%#v", state))
	tflog.Info(ctx, "[DEBUG] Plan in Update: "+fmt.Sprintf("%#v", plan))

	errDiag := r.Diff(ctx, &state, &plan)
	if errDiag != nil {
		response.Diagnostics.Append(errDiag)
		return
	}

	_, err := r.InternalRead(ctx, state.Id.ValueString(), &state)
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

func (r *resourceManagedGpuCluster) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state managedGpuCluster
	diags := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	vpcId := state.VpcId.ValueString()
	cluster := state.ClusterName.ValueString()
	clusterId := state.Id.ValueString()

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(platformVpcErrorPrefix+vpcId, err.Error()))
		return
	}

	platformLower := strings.ToLower(platform)
	v1Path := commons.ApiPath.ManagedGpuClusterDelete(vpcId, platformLower, clusterId)
	v2Path := commons.ApiPath.ManagedGpuClusterDeleteV2(vpcId, platformLower, clusterId)
	path, fallbackPath := v1Path, v2Path
	if requiresV2API(state.K8SVersion.ValueString()) {
		path, fallbackPath = v2Path, v1Path
	}

	tflog.Info(ctx, "Attempting to delete cluster "+cluster+", DELETE "+path)

	_, err = r.mgpuClusterClient.sendDeleteV2Aware(path, fallbackPath, platformLower)
	if err != nil {
		tflog.Error(ctx, "Error deleting cluster "+cluster+": "+err.Error())
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi(path), err.Error()))
		return
	}

	tflog.Info(ctx, "Successfully deleted cluster "+cluster)
}

func (r *resourceManagedGpuCluster) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing MFKE cluster ID "+request.ID)

	var state managedGpuCluster

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

	_, err := r.InternalRead(ctx, clusterId, &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi("internalRead"), err.Error()))
		return
	}

	diags := response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceManagedGpuCluster) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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
	r.mgpuClusterClient = newMgpuClusterApiClient(r.client)
	r.subnetClient = fptcloud_subnet.NewSubnetService(r.client)
	r.tenancyClient = fptcloud_dfke.NewTenancyApiClient(r.client)
}
