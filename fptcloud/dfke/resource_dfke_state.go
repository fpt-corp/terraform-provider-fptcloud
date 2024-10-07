package fptcloud_dfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-fptcloud/commons"
)

var (
	_ resource.Resource                = &resourceDedicatedKubernetesEngineState{}
	_ resource.ResourceWithConfigure   = &resourceDedicatedKubernetesEngineState{}
	_ resource.ResourceWithImportState = &resourceDedicatedKubernetesEngineState{}
)

type resourceDedicatedKubernetesEngineState struct {
	client *commons.Client
}

func NewResourceDedicatedKubernetesEngineState() resource.Resource {
	return &resourceDedicatedKubernetesEngineState{}
}

func (r *resourceDedicatedKubernetesEngineState) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing state for DFKE cluster ID "+request.ID)

	var state dedicatedKubernetesEngineState
	state.Id = types.StringValue(request.ID)

	state.VpcId = types.StringValue("188af427-269b-418a-90bb-0cb27afc6c1e")

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

	// lack of ability to import without VPC ID
	response.Diagnostics.Append(diag2.NewErrorDiagnostic("Unimplemented", "Importing DFKE clusters isn't currently supported"))
}

func (r *resourceDedicatedKubernetesEngineState) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dedicated_kubernetes_engine_v1_state"
}

func (r *resourceDedicatedKubernetesEngineState) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Manage dedicated FKE cluster state",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"vpc_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
			},
			"is_running": schema.BoolAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *resourceDedicatedKubernetesEngineState) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var state dedicatedKubernetesEngineState
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

func (r *resourceDedicatedKubernetesEngineState) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state dedicatedKubernetesEngineState
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

func (r *resourceDedicatedKubernetesEngineState) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state dedicatedKubernetesEngineState
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	management := dedicatedKubernetesEngineManagement{
		ClusterId:  state.Id.ValueString(),
		MgmtAction: "stop",
		DiskExtend: "0",
		ExtendType: "",
		Flavor:     "",
		NodeType:   "",
	}

	path := commons.ApiPath.DedicatedFKEManagement(state.VpcId.ValueString(), state.Id.ValueString())

	a, err2 := r.client.SendPostRequest(path, management)
	if err2 != nil {
		d := diag2.NewErrorDiagnostic("Error calling management API", err2.Error())
		response.Diagnostics.Append(d)
		return
	}

	if diagErr2 := checkForError(a); diagErr2 != nil {
		response.Diagnostics.Append(diagErr2)
		return
	}
}

func (r *resourceDedicatedKubernetesEngineState) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	response.Diagnostics.AddError("Unsupported operation", "Deleting state of a cluster isn't supported")
}

func (r *resourceDedicatedKubernetesEngineState) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *resourceDedicatedKubernetesEngineState) internalRead(ctx context.Context, clusterId string, state *dedicatedKubernetesEngineState) error {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+clusterId+", VPC ID "+vpcId)

	a, err := r.client.SendGetRequest(commons.ApiPath.DedicatedFKEGet(vpcId, clusterId))

	if err != nil {
		return err
	}

	var d dedicatedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &d)
	if err != nil {
		return err
	}

	data := d.Cluster
	if data.Status != "STOPPED" && !data.IsRunning {
		return errors.New("cluster is not running, but status is " + data.Status + " instead of STOPPED")
	}

	if data.Status != "SUCCEEDED" && data.IsRunning {
		return errors.New("cluster is running, but status is " + data.Status + " instead of SUCCEEDED")
	}

	state.IsRunning = types.BoolValue(data.IsRunning)
	return nil
}

type dedicatedKubernetesEngineState struct {
	Id        types.String `tfsdk:"id"`
	VpcId     types.String `tfsdk:"vpc_id"`
	IsRunning types.Bool   `tfsdk:"is_running"`
}
