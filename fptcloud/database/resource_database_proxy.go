package fptcloud_database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &resourceDatabaseProxy{}
	_ resource.ResourceWithConfigure      = &resourceDatabaseProxy{}
	_ resource.ResourceWithImportState    = &resourceDatabaseProxy{}
	_ resource.ResourceWithModifyPlan     = &resourceDatabaseProxy{}
	_ resource.ResourceWithValidateConfig = &resourceDatabaseProxy{}
)

var proxyNameRe = regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)

// resourceDatabaseProxy is the DB proxy of a database cluster (MaxScale for MariaDB, MySQL Router for MySQL).
// Nothing forces a new proxy: the only change the portal takes in place is a resize, which rebuilds every node of the proxy.
type resourceDatabaseProxy struct {
	api *databaseApiClient
}

func NewResourceDatabaseProxy() resource.Resource {
	return &resourceDatabaseProxy{}
}

type databaseProxyModel struct {
	Id               types.String `tfsdk:"id"`
	DbClusterId      types.String `tfsdk:"db_cluster_id"`
	VpcId            types.String `tfsdk:"vpc_id"`
	Name             types.String `tfsdk:"name"`
	LbType           types.String `tfsdk:"lb_type"`
	StorageProfile   types.String `tfsdk:"storage_profile"`
	NodeCpu          types.Int64  `tfsdk:"node_cpu"`
	NodeRam          types.Int64  `tfsdk:"node_ram"`
	HighAvailability types.Bool   `tfsdk:"high_availability"`
	Flavor           types.String `tfsdk:"flavor"`
	FlavorId         types.String `tfsdk:"flavor_id"`
	Status           types.String `tfsdk:"status"`
	EndPoint         types.String `tfsdk:"end_point"`
	NodeCount        types.Int64  `tfsdk:"node_count"`
	CreatedAt        types.String `tfsdk:"created_at"`
	Timeouts         types.Object `tfsdk:"timeouts"`
}

func (r *resourceDatabaseProxy) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_proxy"
}

func (r *resourceDatabaseProxy) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (r *resourceDatabaseProxy) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	fixed := " Cannot be changed after creation: a plan that changes it fails and shows the current value."
	response.Schema = schema.Schema{
		Description: "Manages the DB proxy of a FPT Cloud database cluster. A change of `node_cpu`, `node_ram`, `flavor` or `flavor_id` resizes the proxy in place; " +
			"the resize rebuilds every node of the proxy, so connections through it are dropped. Start and stop are not managed here.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The Id of the proxy (`lb_cluster_id`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"db_cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster the proxy belongs to. The cluster must be running when the proxy is created." + fixed,
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the VPC of the cluster." + fixed,
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the proxy, 8 to 25 characters: letters, digits, spaces, `_` and `-`." + fixed,
			},
			"lb_type": schema.StringAttribute{
				Required:    true,
				Description: "Type of the proxy: `maxscale` for MariaDB, `mysqlrouter` for MySQL. See the data source `fptcloud_database_proxy_types`." + fixed,
			},
			"storage_profile": schema.StringAttribute{
				Required:    true,
				Description: "Name of the storage policy of the proxy nodes." + fixed,
			},
			"node_cpu": schema.Int64Attribute{
				Required:    true,
				Description: "Number of vCPU of each node. Must match a flavor of the VPC. Changing it resizes the proxy.",
			},
			"node_ram": schema.Int64Attribute{
				Required:    true,
				Description: "RAM of each node in GB. Must match a flavor of the VPC. Changing it resizes the proxy.",
			},
			"high_availability": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the proxy has 2 nodes. Defaults to false." + fixed,
			},
			"flavor": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Name of the flavor, for example `4C8G`. Taken from `node_cpu` and `node_ram` when not set. Changing it resizes the proxy.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"flavor_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Id of the flavor. Taken from the flavor when not set. Changing it resizes the proxy.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:      true,
				Description:   "Status of the proxy on the portal. Read only: it is not part of the configuration, so a proxy stopped outside of Terraform is not a change.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"end_point": schema.StringAttribute{
				Computed:      true,
				Description:   "Connection end points of the proxy, one per line.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"node_count": schema.Int64Attribute{
				Computed:      true,
				Description:   "Number of nodes of the proxy.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "Creation time of the proxy.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeoutsBlock(true),
		},
	}
}

func (r *resourceDatabaseProxy) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var config databaseProxyModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}
	if !config.Name.IsNull() && !config.Name.IsUnknown() {
		name := config.Name.ValueString()
		if n := len([]rune(name)); n < 8 || n > 25 || !proxyNameRe.MatchString(name) {
			response.Diagnostics.AddAttributeError(path.Root("name"), "Invalid name",
				fmt.Sprintf("%q is not a valid proxy name: use 8 to 25 characters among letters, digits, spaces, `_` and `-`.", name))
		}
	}
	for _, a := range []struct {
		name string
		v    types.Int64
	}{{"node_cpu", config.NodeCpu}, {"node_ram", config.NodeRam}} {
		if !a.v.IsNull() && !a.v.IsUnknown() && a.v.ValueInt64() <= 0 {
			response.Diagnostics.AddAttributeError(path.Root(a.name), "Invalid "+a.name, a.name+" must be greater than 0.")
		}
	}
	validateTimeouts(config.Timeouts, &response.Diagnostics)
}

// proxySnapshot is what Terraform applied and checks again on the next Read: the size of the proxy.
func proxySnapshot(m *databaseProxyModel) driftSnapshot {
	return driftSnapshot{
		"node_cpu": strconv.FormatInt(m.NodeCpu.ValueInt64(), 10),
		"node_ram": strconv.FormatInt(m.NodeRam.ValueInt64(), 10),
		"flavor":   m.Flavor.ValueString(),
	}
}

// setFromDetail writes what the portal reports into the model. vpc_id and flavor_id are not in the detail; they stay.
func (m *databaseProxyModel) setFromDetail(d *proxyDetail) {
	m.Id = types.StringValue(d.LbClusterId)
	m.DbClusterId = types.StringValue(string(d.DbClusterId))
	m.Name = types.StringValue(string(d.LbClusterName))
	m.LbType = types.StringValue(string(d.LbType))
	m.StorageProfile = types.StringValue(string(d.StorageProfile))
	m.NodeCpu = types.Int64Value(int64(d.NodeCpu))
	m.NodeRam = types.Int64Value(int64(d.NodeRam))
	m.HighAvailability = types.BoolValue(proxyIsHA(d.IsLbCluster))
	m.Status = types.StringValue(string(d.Status))
	m.NodeCount = types.Int64Value(d.NodeCount)
	m.CreatedAt = stringValueOrNull(string(d.CreatedAt))
	m.EndPoint = stringValueOrNull(strings.TrimSpace(string(d.EndPoint)))
	m.Flavor = stringValueOrNull(string(d.Flavor))
}

// fillFlavorId looks for the id of the flavor the proxy has, when the state does not have it or it is another flavor.
// A lookup that fails leaves the id empty rather than failing a read.
func (r *resourceDatabaseProxy) fillFlavorId(ctx context.Context, m *databaseProxyModel, previous types.String, previousFlavor types.String) {
	if !previous.IsNull() && !previous.IsUnknown() && previous.ValueString() != "" && previousFlavor.ValueString() == m.Flavor.ValueString() {
		m.FlavorId = previous
		return
	}
	m.FlavorId = types.StringNull()
	if m.Flavor.IsNull() || m.VpcId.ValueString() == "" {
		return
	}
	flavors, err := r.api.listProxyFlavors(m.VpcId.ValueString())
	if err != nil {
		tflog.Warn(ctx, "Could not look for the flavor id: "+err.Error())
		return
	}
	for _, f := range flavors {
		if f.FlavorName == m.Flavor.ValueString() && int64(f.Vcpu) == m.NodeCpu.ValueInt64() && f.ramGb() == m.NodeRam.ValueInt64() {
			m.FlavorId = types.StringValue(f.FlavorId)
			return
		}
	}
}

func (r *resourceDatabaseProxy) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan databaseProxyModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	clusterId := plan.DbClusterId.ValueString()
	vpcId := plan.VpcId.ValueString()

	cluster, err := r.api.getClusterData(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading the database cluster", err.Error())
		return
	}
	if !strings.EqualFold(cluster.Status, proxyStatusRunning) {
		response.Diagnostics.AddAttributeError(path.Root("db_cluster_id"), "The database cluster is not running",
			fmt.Sprintf("The cluster %s is %s; it must be running before a proxy can be created.", clusterId, cluster.Status))
		return
	}
	if d := r.checkType(clusterId, plan.LbType.ValueString()); d != "" {
		response.Diagnostics.AddAttributeError(path.Root("lb_type"), "Invalid lb_type", d)
		return
	}
	existing, err := r.api.listProxies(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading the proxies of the cluster", err.Error())
		return
	}
	for _, p := range existing {
		if string(p.LbClusterName) == plan.Name.ValueString() {
			response.Diagnostics.AddAttributeError(path.Root("name"), "Name already used",
				fmt.Sprintf("The cluster %s already has a proxy named %q (%s, id %s). Choose another name, or import it with `terraform import`.",
					clusterId, p.LbClusterName, p.Status, p.LbClusterId))
			return
		}
	}
	flavor, err := r.flavorFor(vpcId, &plan)
	if err != nil {
		response.Diagnostics.AddAttributeError(path.Root("node_cpu"), "Invalid flavor", err.Error())
		return
	}

	haText := "no"
	if plan.HighAvailability.ValueBool() {
		haText = "yes"
	}
	id, err := r.api.createProxy(ctx, proxyCreateRequest{
		DbClusterId:    clusterId,
		LbClusterName:  plan.Name.ValueString(),
		IsLbCluster:    haText,
		LbType:         plan.LbType.ValueString(),
		NodeCpu:        plan.NodeCpu.ValueInt64(),
		NodeRam:        plan.NodeRam.ValueInt64(),
		Flavor:         flavor.FlavorName,
		StorageProfile: plan.StorageProfile.ValueString(),
	})
	if err != nil {
		response.Diagnostics.AddError("Error creating the proxy", err.Error())
		return
	}

	// The proxy exists on the portal: record it now, so a failure or a timeout below does not lose it.
	plan.Id = types.StringValue(id)
	plan.Flavor = types.StringValue(flavor.FlavorName)
	plan.FlavorId = types.StringValue(flavor.FlavorId)
	plan.Status = types.StringValue("processing")
	plan.EndPoint = types.StringNull()
	plan.NodeCount = types.Int64Value(0)
	plan.CreatedAt = types.StringNull()
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
	response.Diagnostics.Append(saveDriftSnapshot(ctx, response.Private, proxySnapshot(&plan))...)
	if response.Diagnostics.HasError() {
		return
	}

	detail, err := r.api.waitProxyReady(ctx, vpcId, id, timeoutFrom(plan.Timeouts, "create", proxyCreateTimeout))
	if err != nil {
		response.Diagnostics.AddError("Error waiting for the proxy", err.Error())
		return
	}
	plan.setFromDetail(detail)
	plan.FlavorId = types.StringValue(flavor.FlavorId)
	// What was applied is what was asked for, not what the portal returns.
	applied := databaseProxyModel{NodeCpu: types.Int64Value(plan.NodeCpu.ValueInt64()), NodeRam: plan.NodeRam, Flavor: types.StringValue(flavor.FlavorName)}
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
	response.Diagnostics.Append(saveDriftSnapshot(ctx, response.Private, proxySnapshot(&applied))...)
}

// checkType returns a message when lb_type is not one the cluster accepts.
func (r *resourceDatabaseProxy) checkType(clusterId, lbType string) string {
	allowed, err := r.api.listProxyTypes(clusterId)
	if err != nil {
		return "could not read the proxy types of the cluster: " + err.Error()
	}
	for _, t := range allowed {
		if t == lbType {
			return ""
		}
	}
	return fmt.Sprintf("%q is not a proxy type of cluster %s; use one of: %s.", lbType, clusterId, strings.Join(allowed, ", "))
}

// flavorFor finds the flavor of the planned size, and checks that it is one the VPC has.
func (r *resourceDatabaseProxy) flavorFor(vpcId string, m *databaseProxyModel) (*proxyFlavor, error) {
	flavors, err := r.api.listProxyFlavors(vpcId)
	if err != nil {
		return nil, err
	}
	name, id := "", ""
	if !m.Flavor.IsNull() && !m.Flavor.IsUnknown() {
		name = m.Flavor.ValueString()
	}
	if !m.FlavorId.IsNull() && !m.FlavorId.IsUnknown() {
		id = m.FlavorId.ValueString()
	}
	return resolveProxyFlavor(flavors, m.NodeCpu.ValueInt64(), m.NodeRam.ValueInt64(), name, id)
}

func (r *resourceDatabaseProxy) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state databaseProxyModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	applied, snapshotDiags := loadDriftSnapshot(ctx, request.Private)
	response.Diagnostics.Append(snapshotDiags...)

	detail, err := r.api.getProxy(state.Id.ValueString())
	if errors.Is(err, errProxyNotFound) {
		tflog.Info(ctx, "The proxy "+state.Id.ValueString()+" is gone from the portal, removing it from the state")
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Error reading the proxy", err.Error())
		return
	}

	previousFlavorId, previousFlavor := state.FlavorId, state.Flavor
	state.setFromDetail(detail)
	r.fillFlavorId(ctx, &state, previousFlavorId, previousFlavor)

	addDriftWarnings(&response.Diagnostics, applied, proxySnapshot(&state))
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *resourceDatabaseProxy) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state databaseProxyModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	id := state.Id.ValueString()
	vpcId := state.VpcId.ValueString()

	// Plan values of computed attributes that are unknown are the ones the resize can change.
	target := plan
	flavor, err := r.flavorFor(vpcId, &target)
	if err != nil {
		response.Diagnostics.AddAttributeError(path.Root("node_cpu"), "Invalid flavor", err.Error())
		return
	}

	current, err := r.api.getProxy(id)
	if err != nil {
		response.Diagnostics.AddError("Error reading the proxy", err.Error())
		return
	}
	sameSize := int64(current.NodeCpu) == plan.NodeCpu.ValueInt64() && int64(current.NodeRam) == plan.NodeRam.ValueInt64() &&
		string(current.Flavor) == flavor.FlavorName
	if sameSize {
		// The portal refuses a resize to the size the proxy has. Nothing to do but record the state.
		tflog.Info(ctx, "The proxy already has the planned size, nothing to resize")
	} else {
		if err := r.checkOperable(ctx, state.DbClusterId.ValueString(), vpcId, id, "resized"); err != nil {
			response.Diagnostics.AddError("The proxy cannot be resized", err.Error())
			return
		}
		if err := r.api.resizeProxy(proxyResizeRequest{
			LbClusterId: id,
			NodeCpu:     plan.NodeCpu.ValueInt64(),
			NodeRam:     plan.NodeRam.ValueInt64(),
			Flavor:      flavor.FlavorName,
			FlavorId:    flavor.FlavorId,
		}); err != nil {
			response.Diagnostics.AddError("Error resizing the proxy", err.Error())
			return
		}
		// The resize was accepted: record that the proxy is busy, keeping the size it still has.
		busy := state
		busy.Status = types.StringValue("resizing")
		busy.Timeouts = plan.Timeouts
		response.Diagnostics.Append(response.State.Set(ctx, &busy)...)
		if response.Diagnostics.HasError() {
			return
		}
		if err := r.api.waitProxyResized(ctx, vpcId, id, plan.NodeCpu.ValueInt64(), plan.NodeRam.ValueInt64(), timeoutFrom(plan.Timeouts, "update", proxyUpdateTimeout)); err != nil {
			response.Diagnostics.AddError("Error waiting for the resize", err.Error())
			return
		}
	}

	detail, err := r.api.getProxy(id)
	if err != nil {
		response.Diagnostics.AddError("Error reading the proxy", err.Error())
		return
	}
	plan.VpcId = state.VpcId
	plan.setFromDetail(detail)
	plan.FlavorId = types.StringValue(flavor.FlavorId)
	applied := databaseProxyModel{NodeCpu: plan.NodeCpu, NodeRam: plan.NodeRam, Flavor: types.StringValue(flavor.FlavorName)}
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
	response.Diagnostics.Append(saveDriftSnapshot(ctx, response.Private, proxySnapshot(&applied))...)
}

// checkOperable returns an error when the proxy is not in a status the portal takes a resize or a restart in,
// or its VMs are not synced yet. The portal does not check this itself: it takes the call and loses it.
func (r *resourceDatabaseProxy) checkOperable(ctx context.Context, clusterId, vpcId, id, verb string) error {
	for attempt := 0; attempt < 2; attempt++ {
		items, err := r.api.listProxies(clusterId)
		if err != nil {
			return err
		}
		for _, p := range items {
			if p.LbClusterId != id {
				continue
			}
			if !proxyCanChange(string(p.Status)) {
				return fmt.Errorf("The proxy is %s; it must be running before it can be %s. Start it on the portal first.", p.Status, verb)
			}
			if p.VmSync {
				return nil
			}
			syncVpcInfrastructure(ctx, r.api.Client, vpcId)
		}
	}
	return fmt.Errorf("the virtual machines of the proxy %s are not synced with the portal yet; try again in a few minutes", id)
}

func (r *resourceDatabaseProxy) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state databaseProxyModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	id := state.Id.ValueString()
	vpcId := state.VpcId.ValueString()
	waitTimeout := timeoutFrom(state.Timeouts, "delete", proxyDeleteTimeout)

	_, err := r.api.getProxy(id)
	if errors.Is(err, errProxyNotFound) {
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Error reading the proxy", err.Error())
		return
	}
	for attempt := 1; ; attempt++ {
		// A delete sent while the proxy is busy is lost: wait until it rests. A stopped proxy can be deleted.
		detail, err := r.api.waitProxyIdle(ctx, vpcId, id, waitTimeout)
		if errors.Is(err, errProxyNotFound) {
			return
		}
		if err != nil {
			response.Diagnostics.AddError("Error waiting for the proxy", err.Error())
			return
		}
		if !strings.EqualFold(string(detail.Status), proxyStatusDeleting) {
			if err := r.api.deleteProxy(id); err != nil {
				response.Diagnostics.AddError("Error deleting the proxy", err.Error())
				return
			}
		}
		err = r.api.waitProxyDeleted(ctx, vpcId, state.DbClusterId.ValueString(), id, waitTimeout)
		if err == nil {
			return
		}
		// The portal sometimes puts the proxy back to "failed" instead of deleting it: send the delete again.
		if errors.Is(err, errProxyDeleteFellBack) && attempt < proxyDeleteAttempts {
			tflog.Warn(ctx, fmt.Sprintf("The proxy %s is failed again after the delete (attempt %d of %d), deleting it again", id, attempt, proxyDeleteAttempts))
			continue
		}
		response.Diagnostics.AddError("Error waiting for the delete", err.Error())
		return
	}
}

func (r *resourceDatabaseProxy) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	id := strings.TrimSpace(request.ID)
	detail, err := r.api.getProxy(id)
	if err != nil {
		response.Diagnostics.AddError("Error reading the proxy", fmt.Sprintf("proxy %s: %v", id, err))
		return
	}
	cluster, err := r.api.getClusterData(string(detail.DbClusterId))
	if err != nil {
		response.Diagnostics.AddError("Error reading the database cluster of the proxy", err.Error())
		return
	}
	var state databaseProxyModel
	state.VpcId = types.StringValue(cluster.VpcId)
	state.setFromDetail(detail)
	r.fillFlavorId(ctx, &state, types.StringNull(), types.StringNull())
	state.Timeouts = types.ObjectNull(timeoutsObjectTypes())
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

// ModifyPlan fails a plan that cannot be applied: a change to something fixed after creation, a size no flavor has,
// a lb_type the cluster does not take, or a resize of a proxy that is not running. Nothing here replaces the proxy.
func (r *resourceDatabaseProxy) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	if request.Plan.Raw.IsNull() || r.api == nil {
		return
	}
	var plan databaseProxyModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	if request.State.Raw.IsNull() {
		if known(plan.DbClusterId) && known(plan.LbType) {
			if d := r.checkType(plan.DbClusterId.ValueString(), plan.LbType.ValueString()); d != "" {
				response.Diagnostics.AddAttributeError(path.Root("lb_type"), "Invalid lb_type", d)
			}
		}
		if known(plan.VpcId) && known(plan.NodeCpu) && known(plan.NodeRam) {
			if _, err := r.flavorFor(plan.VpcId.ValueString(), &plan); err != nil {
				response.Diagnostics.AddAttributeError(path.Root("node_cpu"), "Invalid flavor", err.Error())
			}
		}
		return
	}

	var state databaseProxyModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	// The state holds what the portal returned on the last Read, so the message gives the value to put in the configuration.
	for _, c := range []immutableCheck{
		{name: "db_cluster_id", plan: plan.DbClusterId, state: state.DbClusterId},
		{name: "vpc_id", plan: plan.VpcId, state: state.VpcId},
		{name: "name", plan: plan.Name, state: state.Name},
		{name: "lb_type", plan: plan.LbType, state: state.LbType},
		{name: "high_availability", plan: plan.HighAvailability, state: state.HighAvailability},
		{name: "storage_profile", plan: plan.StorageProfile, state: state.StorageProfile},
	} {
		if c.changed() {
			response.Diagnostics.AddAttributeError(path.Root(c.name), "Cannot change "+c.name,
				fmt.Sprintf("`%s` cannot be changed after the proxy is created; it is `%s` on the portal. Set it to `%s`, or create another proxy.",
					c.name, attrText(c.state), attrText(c.state)))
		}
	}
	if response.Diagnostics.HasError() {
		return
	}

	// A resize: the size, or the flavor, differs from what the portal has.
	var configFlavor, configFlavorId types.String
	response.Diagnostics.Append(request.Config.GetAttribute(ctx, path.Root("flavor"), &configFlavor)...)
	response.Diagnostics.Append(request.Config.GetAttribute(ctx, path.Root("flavor_id"), &configFlavorId)...)
	if !known(plan.NodeCpu) || !known(plan.NodeRam) {
		return
	}
	sizeChanged := plan.NodeCpu.ValueInt64() != state.NodeCpu.ValueInt64() || plan.NodeRam.ValueInt64() != state.NodeRam.ValueInt64()
	flavorChanged := (!configFlavor.IsNull() && !configFlavor.IsUnknown() && configFlavor.ValueString() != state.Flavor.ValueString()) ||
		(!configFlavorId.IsNull() && !configFlavorId.IsUnknown() && configFlavorId.ValueString() != state.FlavorId.ValueString())
	if !sizeChanged && !flavorChanged {
		return
	}

	if !proxyCanChange(state.Status.ValueString()) {
		response.Diagnostics.AddAttributeError(path.Root("node_cpu"), "The proxy cannot be resized",
			fmt.Sprintf("The proxy is %s; it must be running before it can be resized. Start it on the portal first.", state.Status.ValueString()))
		return
	}
	// The flavor follows the size when it is not set.
	if configFlavor.IsNull() {
		response.Diagnostics.Append(response.Plan.SetAttribute(ctx, path.Root("flavor"), types.StringUnknown())...)
	}
	if configFlavorId.IsNull() {
		response.Diagnostics.Append(response.Plan.SetAttribute(ctx, path.Root("flavor_id"), types.StringUnknown())...)
	}
	target := plan
	target.Flavor, target.FlavorId = configFlavor, configFlavorId
	flavor, err := r.flavorFor(state.VpcId.ValueString(), &target)
	if err != nil {
		response.Diagnostics.AddAttributeError(path.Root("node_cpu"), "Invalid flavor", err.Error())
		return
	}
	if configFlavor.IsNull() {
		response.Diagnostics.Append(response.Plan.SetAttribute(ctx, path.Root("flavor"), types.StringValue(flavor.FlavorName))...)
	}
	if configFlavorId.IsNull() {
		response.Diagnostics.Append(response.Plan.SetAttribute(ctx, path.Root("flavor_id"), types.StringValue(flavor.FlavorId))...)
	}
}

func known(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
}
