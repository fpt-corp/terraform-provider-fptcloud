package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
)

var (
	_ resource.Resource                = &resourceManagedKubernetesEngineState{}
	_ resource.ResourceWithConfigure   = &resourceManagedKubernetesEngineState{}
	_ resource.ResourceWithImportState = &resourceManagedKubernetesEngineState{}
)

type resourceManagedKubernetesEngineState struct {
	//client        *commons.Client
	mfkeClient    *MfkeApiClient
	tenancyClient *fptcloud_dfke.TenancyApiClient
}

func NewResourceDedicatedKubernetesEngineState() resource.Resource {
	return &resourceManagedKubernetesEngineState{}
}

func (r *resourceManagedKubernetesEngineState) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	pieces := strings.Split(request.ID, "/")
	if len(pieces) != 2 {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Wrong import format", "Expected import ID in format vpc_id/cluster_name"))
	}

	vpcId := pieces[0]
	clusterName := pieces[1]

	tflog.Info(ctx, "Importing state for VPC "+vpcId+", cluster "+clusterName)

	var state managedKubernetesEngineState
	state.Id = types.StringValue(clusterName)
	state.VpcId = types.StringValue(vpcId)

	err := r.internalRead(ctx, clusterName, &state)
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

func (r *resourceManagedKubernetesEngineState) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_kubernetes_engine_v1_state"
}

func (r *resourceManagedKubernetesEngineState) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Manage managed FKE cluster hibernation state",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:      true,
				Description:   "Cluster ID, as seen on portal.",
				PlanModifiers: forceNewPlanModifiersString,
			},
			"vpc_id": schema.StringAttribute{
				Required:      true,
				Description:   "VPC ID (an UUID string)",
				PlanModifiers: forceNewPlanModifiersString,
			},
			"is_running": schema.BoolAttribute{
				Required:    true,
				Description: "Whether the cluster runs",
			},
		},
	}
}

func (r *resourceManagedKubernetesEngineState) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var state managedKubernetesEngineState
	diags := request.Plan.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := r.internalRead(ctx, state.Id.ValueString(), &state); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error reading cluster state", err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceManagedKubernetesEngineState) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state managedKubernetesEngineState
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := r.internalRead(ctx, state.Id.ValueString(), &state); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error reading cluster state", err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceManagedKubernetesEngineState) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state managedKubernetesEngineState
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var desired managedKubernetesEngineState
	diags = request.Plan.Get(ctx, &desired)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, state.VpcId.ValueString())
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting platform", err.Error()))
		return
	}

	isWakeup := desired.IsRunning.ValueBool()

	path := commons.ApiPath.ManagedFKEHibernate(state.VpcId.ValueString(), strings.ToLower(platform), state.Id.ValueString(), isWakeup)
	a, err := r.mfkeClient.sendPatch(path, platform, nil)
	if err != nil {
		d := diag2.NewErrorDiagnostic("Error calling hibernate API", err.Error())
		response.Diagnostics.Append(d)
		return
	}

	if diagErr2 := fptcloud_dfke.CheckForError(a); diagErr2 != nil {
		response.Diagnostics.Append(diagErr2)
		return
	}

	err = r.internalRead(ctx, state.Id.ValueString(), &state)
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

func (r *resourceManagedKubernetesEngineState) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	response.Diagnostics.AddError("Unsupported operation", "Deleting state of a cluster isn't supported")
}

func (r *resourceManagedKubernetesEngineState) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

	//r.client = client
	r.mfkeClient = newMfkeApiClient(client)
	r.tenancyClient = fptcloud_dfke.NewTenancyApiClient(client)
}

func (r *resourceManagedKubernetesEngineState) internalRead(ctx context.Context, clusterId string, state *managedKubernetesEngineState) error {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+clusterId+", VPC ID "+vpcId)

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, state.VpcId.ValueString())
	if err != nil {
		return fmt.Errorf("error getting VPC platform: %v", err)
	}

	a, err := r.mfkeClient.sendGet(commons.ApiPath.ManagedFKEGet(vpcId, strings.ToLower(platform), clusterId), platform)

	if err != nil {
		return err
	}

	var d managedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &d)
	if err != nil {
		return err
	}

	status := strings.ToUpper(d.Data.Status.LastOperation.State)
	isRunning := false
	if len(d.Data.Status.Conditions) > 0 {
		isRunning = d.Data.Status.Conditions[0].Status == "True"
	}
	if d.Data.Spec.Hibernate != nil {
		isRunning = !d.Data.Spec.Hibernate.Enabled
	}

	tflog.Info(ctx, fmt.Sprintf("spec.hibernate: %v", d.Data.Spec.Hibernate))

	if status != "SUCCEEDED" && status != "PROCESSING" {
		return errors.New("cluster is running, but status is " + status + " instead of SUCCEEDED")
	}

	state.IsRunning = types.BoolValue(isRunning)
	return nil
}

type managedKubernetesEngineState struct {
	Id        types.String `tfsdk:"id"`
	VpcId     types.String `tfsdk:"vpc_id"`
	IsRunning types.Bool   `tfsdk:"is_running"`
}
