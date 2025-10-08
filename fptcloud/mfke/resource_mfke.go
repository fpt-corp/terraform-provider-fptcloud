package fptcloud_mfke

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
	_ resource.Resource                = &resourceManagedKubernetesEngine{}
	_ resource.ResourceWithConfigure   = &resourceManagedKubernetesEngine{}
	_ resource.ResourceWithImportState = &resourceManagedKubernetesEngine{}

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

func NewResourceManagedKubernetesEngine() resource.Resource {
	return &resourceManagedKubernetesEngine{}
}

func (r *resourceManagedKubernetesEngine) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_kubernetes_engine_v1"
}

func (r *resourceManagedKubernetesEngine) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	topLevelAttributes := TopFields()
	poolAttributes := PoolFields()

	topLevelAttributes["id"] = schema.StringAttribute{
		Computed: true,
	}

	response.Schema = schema.Schema{
		Description: "Manage managed FKE clusters.",
		Attributes:  topLevelAttributes,
		Blocks: map[string]schema.Block{
			"pools": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: poolAttributes,
					Blocks: map[string]schema.Block{
						"kv": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name":  schema.StringAttribute{Required: true, Description: descriptions["name"]},
									"value": schema.StringAttribute{Required: true, Description: descriptions["kv"]},
								},
							},
						},
						"taints": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"key":    schema.StringAttribute{Required: true, Description: "The taint key"},
									"value":  schema.StringAttribute{Required: true, Description: "The taint value"},
									"effect": schema.StringAttribute{Required: true, Description: "The taint effect (NoSchedule, NoExecute, PreferNoSchedule)"},
								},
							},
						},
					},
				},
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

	// Get platform first to set appropriate defaults
	platform, err := r.tenancyClient.GetVpcPlatform(ctx, state.VpcId.ValueString())
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting VPC platform", err.Error()))
		return
	}

	// Set all defaults in one place with platform info
	SetDefaults(&state, platform)

	// Validate all in one place
	if !ValidateCreate(&state, response) {
		return
	}

	var f managedKubernetesEngineJson
	errDiag := MapTerraformToJson(r, ctx, &state, &f, state.VpcId.ValueString())

	if errDiag != nil {
		response.Diagnostics.Append(errDiag)
		return
	}

	if err := validateNetwork(&state, platform); err != nil {
		response.Diagnostics.Append(err)
		return
	}

	path := commons.ApiPath.ManagedFKECreate(state.VpcId.ValueString(), strings.ToLower(platform))
	tflog.Info(ctx, "Calling path "+path)
	a, err := r.mfkeClient.sendPost(ctx, path, platform, f)

	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi(path), err.Error()))
		return
	}

	errorResponse := r.CheckForError(a)
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

func (r *resourceManagedKubernetesEngine) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state managedKubernetesEngine
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

func (r *resourceManagedKubernetesEngine) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state managedKubernetesEngine
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan managedKubernetesEngine
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

func (r *resourceManagedKubernetesEngine) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state managedKubernetesEngine
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

	path := commons.ApiPath.ManagedFKEDelete(vpcId, strings.ToLower(platform), clusterId)

	tflog.Info(ctx, "Attempting to delete cluster "+cluster+", DELETE "+path)

	_, err = r.mfkeClient.sendDelete(path, strings.ToLower(platform))
	if err != nil {
		tflog.Error(ctx, "Error deleting cluster "+cluster+": "+err.Error())
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi(path), err.Error()))
		return
	}

	tflog.Info(ctx, "Successfully deleted cluster "+cluster)
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
