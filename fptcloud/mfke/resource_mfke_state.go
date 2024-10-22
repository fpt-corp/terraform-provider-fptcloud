package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
)

var (
	_ resource.Resource              = &resourceManagedKubernetesEngineState{}
	_ resource.ResourceWithConfigure = &resourceManagedKubernetesEngineState{}
)

type resourceManagedKubernetesEngineState struct {
	client        *commons.Client
	mfkeClient    *MfkeApiClient
	tenancyClient *fptcloud_dfke.TenancyApiClient
}

func NewResourceManagedKubernetesEngineState() resource.Resource {
	return &resourceManagedKubernetesEngineState{}
}

func (r *resourceManagedKubernetesEngineState) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_managed_kubernetes_engine_v1_state"
}

func (r *resourceManagedKubernetesEngineState) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Manage Managed FKE cluster state",
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

func (r *resourceManagedKubernetesEngineState) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var state ManagedKubernetesEngineState
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
	var state ManagedKubernetesEngineState
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
	var state ManagedKubernetesEngineState
	diags := request.Plan.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	vpcId := state.VpcId.ValueString()

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(
			"Error getting platform for VPC "+vpcId,
			err.Error(),
		))
		return
	}

	platform = strings.ToLower(platform)

	var endpoint string
	if state.IsRunning.ValueBool() {
		endpoint = commons.ApiPath.ManagedFKEWakeup(
			state.VpcId.ValueString(),
			platform,
			state.Id.ValueString(),
		)
	} else {
		endpoint = commons.ApiPath.ManagedFKEHibernate(
			state.VpcId.ValueString(),
			platform,
			state.Id.ValueString(),
		)
	}

	a, err2 := r.mfkeClient.sendPatch(endpoint, platform, struct{}{})
	if err2 != nil {
		d := diag2.NewErrorDiagnostic("Error performing hibernation changes", err2.Error())
		response.Diagnostics.Append(d)
		return
	}

	if diagErr2 := checkForError(a); diagErr2 != nil {
		response.Diagnostics.Append(diagErr2)
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

	r.client = client
	r.mfkeClient = newMfkeApiClient(r.client)
	r.tenancyClient = fptcloud_dfke.NewTenancyApiClient(r.client)
}

func (r *resourceManagedKubernetesEngineState) internalRead(ctx context.Context, clusterId string, state *ManagedKubernetesEngineState) error {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+clusterId+", VPC ID "+vpcId)

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return err
	}

	path := commons.ApiPath.ManagedFKEGet(vpcId, platform, clusterId)
	a, err := r.mfkeClient.sendGet(path, platform)
	if err != nil {
		return err
	}

	var d managedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &d)
	if err != nil {
		return err
	}

	hibernate := d.Data.Spec.Hibernation
	hibernated := false
	if hibernate == nil {
		hibernated = false
	} else {
		enabled := hibernate.Enabled
		if enabled == nil {
			hibernated = false
		} else {
			hibernated = *enabled
		}
	}

	state.IsRunning = types.BoolValue(!hibernated)
	return nil
}

type ManagedKubernetesEngineState struct {
	Id        types.String `tfsdk:"id"`
	VpcId     types.String `tfsdk:"vpc_id"`
	IsRunning types.Bool   `tfsdk:"is_running"`
}
