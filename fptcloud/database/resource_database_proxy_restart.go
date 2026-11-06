package fptcloud_database

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &resourceDatabaseProxyRestart{}
	_ resource.ResourceWithConfigure = &resourceDatabaseProxyRestart{}
)

// resourceDatabaseProxyRestart is an action resource: creating it, or changing `triggers`, restarts a proxy.
type resourceDatabaseProxyRestart struct {
	api *databaseApiClient
}

func NewResourceDatabaseProxyRestart() resource.Resource {
	return &resourceDatabaseProxyRestart{}
}

type databaseProxyRestartModel struct {
	Id          types.String `tfsdk:"id"`
	ProxyId     types.String `tfsdk:"proxy_id"`
	VpcId       types.String `tfsdk:"vpc_id"`
	Triggers    types.Map    `tfsdk:"triggers"`
	RestartedAt types.String `tfsdk:"restarted_at"`
	Timeouts    types.Object `tfsdk:"timeouts"`
}

func (r *resourceDatabaseProxyRestart) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_proxy_restart"
}

func (r *resourceDatabaseProxyRestart) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (r *resourceDatabaseProxyRestart) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Restarts a DB proxy and waits until it is running again. Creating the resource, or changing `triggers`, restarts the proxy; connections through it are dropped. " +
			"Changing another argument only updates the state; to restart again without changing `triggers`, use `terraform apply -replace=...`. Destroying it calls no API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "The Id of the proxy.",
			},
			"proxy_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the proxy, for example `fptcloud_database_proxy.db.id`. It must be running (or in warning) to restart.",
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the VPC of the proxy.",
			},
			"triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arbitrary values; when one changes, the proxy restarts again.",
			},
			"restarted_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the proxy was last restarted by Terraform (UTC, RFC 3339).",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeoutsBlock(true),
		},
	}
}

// restart checks that the proxy can be restarted, restarts it and waits for it to run again.
func (r *resourceDatabaseProxyRestart) restart(ctx context.Context, m *databaseProxyRestartModel, waitTimeout time.Duration) error {
	id, vpcId := m.ProxyId.ValueString(), m.VpcId.ValueString()
	detail, err := r.api.getProxy(id)
	if err != nil {
		return fmt.Errorf("proxy %s: %v", id, err)
	}
	if !proxyCanChange(string(detail.Status)) {
		return fmt.Errorf("The proxy is %s; it must be running before it can be restarted. Start it on the portal first.", detail.Status)
	}
	// The portal does not check the status itself, and a restart sent while the VMs are not synced is lost.
	syncVpcInfrastructure(ctx, r.api.Client, vpcId)
	if err := r.api.restartProxy(id); err != nil {
		return err
	}
	tflog.Info(ctx, "Restarted proxy "+id)
	if _, err := r.api.waitProxyRestarted(ctx, vpcId, id, waitTimeout); err != nil {
		return err
	}
	m.RestartedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	return nil
}

func (r *resourceDatabaseProxyRestart) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan databaseProxyRestartModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	validateTimeouts(plan.Timeouts, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	if err := r.restart(ctx, &plan, timeoutFrom(plan.Timeouts, "create", proxyUpdateTimeout)); err != nil {
		response.Diagnostics.AddError("Error restarting the proxy", err.Error())
		return
	}
	plan.Id = plan.ProxyId
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *resourceDatabaseProxyRestart) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state databaseProxyRestartModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	// The restart already happened: there is nothing on the portal to compare with. Whether the proxy still
	// exists is the business of fptcloud_database_proxy.
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *resourceDatabaseProxyRestart) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state databaseProxyRestartModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	validateTimeouts(plan.Timeouts, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	plan.Id = plan.ProxyId
	plan.RestartedAt = state.RestartedAt
	if !plan.Triggers.Equal(state.Triggers) {
		if err := r.restart(ctx, &plan, timeoutFrom(plan.Timeouts, "update", proxyUpdateTimeout)); err != nil {
			response.Diagnostics.AddError("Error restarting the proxy", err.Error())
			return
		}
	} else {
		response.Diagnostics.AddWarning("The proxy was not restarted again",
			"`proxy_id` or `vpc_id` changed, so only the state was updated. To restart, change `triggers` or use `terraform apply -replace=...`.")
	}
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *resourceDatabaseProxyRestart) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	// Nothing to undo: a restart that happened stays.
}
