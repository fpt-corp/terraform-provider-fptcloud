package fptcloud_database

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	common "terraform-provider-fptcloud/commons"

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
	_ resource.Resource                   = &resourceDatabaseRestore{}
	_ resource.ResourceWithConfigure      = &resourceDatabaseRestore{}
	_ resource.ResourceWithValidateConfig = &resourceDatabaseRestore{}
)

var restoreNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)

// resourceDatabaseRestore is an action resource: creating it restores a restore point into a new cluster.
type resourceDatabaseRestore struct {
	api *databaseApiClient
}

func NewResourceDatabaseRestore() resource.Resource {
	return &resourceDatabaseRestore{}
}

type databaseRestoreModel struct {
	Id                     types.String `tfsdk:"id"`
	ClusterId              types.String `tfsdk:"cluster_id"`
	RestoreName            types.String `tfsdk:"restore_name"`
	BackupPart             types.String `tfsdk:"backup_part"`
	BackupType             types.String `tfsdk:"backup_type"`
	FlavorId               types.String `tfsdk:"flavor_id"`
	NodeCpu                types.Int64  `tfsdk:"node_cpu"`
	NodeRam                types.Int64  `tfsdk:"node_ram"`
	StorageProfile         types.String `tfsdk:"storage_profile"`
	DeleteClusterOnDestroy types.Bool   `tfsdk:"delete_cluster_on_destroy"`
	RestoredClusterId      types.String `tfsdk:"restored_cluster_id"`
	RestoredStatus         types.String `tfsdk:"restored_status"`
	Timeouts               types.Object `tfsdk:"timeouts"`
}

func (r *resourceDatabaseRestore) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_restore"
}

func (r *resourceDatabaseRestore) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (r *resourceDatabaseRestore) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Restores a restore point of a database cluster into a new cluster. Changing an argument later only updates the state and warns; " +
			"to restore again, use `terraform apply -replace=...`. Destroying it keeps the restored cluster unless `delete_cluster_on_destroy` is true.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "The Id of the restored cluster.",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the source database cluster.",
			},
			"restore_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the new cluster: starts with a letter, then letters, digits and `-`. It must not be used by another cluster in the VPC.",
			},
			"backup_part": schema.StringAttribute{
				Required:    true,
				Description: "The restore point, `backup_part` of `fptcloud_database_restore_points`.",
			},
			"backup_type": schema.StringAttribute{
				Required:    true,
				Description: "Type of the restore point, `full`, `diff` or `incr`.",
			},
			"flavor_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Flavor of the new cluster. Instead of it you can set `node_cpu` and `node_ram`. When none is set, the flavor of the source cluster is used. Ignored for an OpenSearch HA cluster, which restores with the compute of the source.",
			},
			"node_cpu": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Number of CPUs of the new cluster, used with `node_ram` to pick a flavor.",
			},
			"node_ram": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "RAM in GB of the new cluster, used with `node_cpu` to pick a flavor.",
			},
			"storage_profile": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Storage profile of the new cluster. Defaults to the one of the source cluster.",
			},
			"delete_cluster_on_destroy": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Delete the restored cluster when this resource is destroyed. Default `false`: the cluster is kept and only a warning is shown.",
			},
			"restored_cluster_id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "The Id of the restored cluster.",
			},
			"restored_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the restored cluster, refreshed on every read.",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeoutsBlock(false),
		},
	}
}

func (r *resourceDatabaseRestore) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var config databaseRestoreModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}
	validateTimeouts(config.Timeouts, &response.Diagnostics)
	if !config.RestoreName.IsUnknown() && !config.RestoreName.IsNull() && !restoreNameRe.MatchString(config.RestoreName.ValueString()) {
		response.Diagnostics.AddAttributeError(path.Root("restore_name"), "Invalid restore_name",
			"`restore_name` must start with a letter and contain only letters, digits and `-`.")
	}
	if config.NodeCpu.IsNull() != config.NodeRam.IsNull() {
		response.Diagnostics.AddAttributeError(path.Root("node_cpu"), "node_cpu and node_ram go together", "Set both `node_cpu` and `node_ram`, or neither.")
	}
	if !config.FlavorId.IsNull() && !config.NodeCpu.IsNull() {
		response.Diagnostics.AddAttributeError(path.Root("flavor_id"), "Use flavor_id or node_cpu and node_ram", "Set `flavor_id`, or `node_cpu` with `node_ram`, not both.")
	}
}

type restoreFlavor struct {
	FlavorId string    `json:"flavor_id"`
	Name     apiString `json:"flavor_name"`
	Vcpu     flexInt   `json:"flavor_vcpu"`
	Ram      flexInt   `json:"flavor_ram"` // MB
}

// resolveRestoreFlavor picks the flavor by id, or by CPU and RAM (GB).
func resolveRestoreFlavor(flavors []restoreFlavor, flavorId string, cpu, ramGb int64) (*restoreFlavor, error) {
	for i, f := range flavors {
		if flavorId != "" && f.FlavorId == flavorId {
			return &flavors[i], nil
		}
		if flavorId == "" && int64(f.Vcpu) == cpu && int64(f.Ram) == ramGb*1024 {
			return &flavors[i], nil
		}
	}
	names := make([]string, 0, len(flavors))
	for _, f := range flavors {
		names = append(names, fmt.Sprintf("%s (%d CPU, %d GB)", f.Name, f.Vcpu, int64(f.Ram)/1024))
	}
	sort.Strings(names)
	if flavorId != "" {
		return nil, fmt.Errorf("flavor %s is not available in this VPC. Available flavors: %s", flavorId, strings.Join(names, ", "))
	}
	return nil, fmt.Errorf("no flavor with %d CPU and %d GB RAM in this VPC. Available flavors: %s", cpu, ramGb, strings.Join(names, ", "))
}

// listRestoreFlavors reads the flavors of a VPC; the portal lists them per platform, so both are tried.
func (r *resourceDatabaseRestore) listRestoreFlavors(vpcId string) ([]restoreFlavor, error) {
	var all []restoreFlavor
	for _, isOps := range []string{"yes", "no"} {
		path := common.ApiPath.DatabaseFlavor(vpcId, isOps)
		body, err := r.api.sendGet(path)
		if err != nil {
			return nil, fmt.Errorf("failed calling path %s: %v", path, err)
		}
		var resp struct {
			Data []restoreFlavor `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed parsing %s: %v, body=%s", path, err, truncateBody(body))
		}
		all = append(all, resp.Data...)
	}
	return all, nil
}

// clusterNameTaken reports whether a cluster with this name exists in the VPC; deleted clusters do not count.
func (r *resourceDatabaseRestore) clusterNameTaken(vpcId, name string) (bool, error) {
	type item struct {
		ClusterName apiString `json:"cluster_name"`
		Status      apiString `json:"status"`
	}
	clusters, err := getPaged[item](r.api, func(page, size int) string {
		return common.ApiPath.DatabaseList(vpcId, page, size)
	}, 0)
	if err != nil {
		return false, err
	}
	for _, c := range clusters {
		if strings.EqualFold(string(c.ClusterName), name) && !strings.EqualFold(string(c.Status), "deleted") {
			return true, nil
		}
	}
	return false, nil
}

// findRestoredCluster finds the new cluster in the restored clusters of the source by its name.
func (r *resourceDatabaseRestore) findRestoredCluster(sourceId, name string) (*restoredCluster, error) {
	clusters, err := r.api.listRestoredClusters(sourceId)
	if err != nil {
		return nil, err
	}
	for i := range clusters {
		if string(clusters[i].ClusterName) == name && !strings.EqualFold(string(clusters[i].Status), "deleted") {
			return &clusters[i], nil
		}
	}
	return nil, nil
}

func (r *resourceDatabaseRestore) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan databaseRestoreModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	sourceId := strings.TrimSpace(plan.ClusterId.ValueString())
	name := plan.RestoreName.ValueString()
	timeout := timeoutFrom(plan.Timeouts, "create", restoreCreateTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	source, err := r.api.getClusterData(sourceId)
	if err != nil {
		response.Diagnostics.AddError("Error reading the source cluster", err.Error())
		return
	}
	if source.Status != "running" {
		response.Diagnostics.AddError("The source cluster is not running",
			fmt.Sprintf("Cluster %s is %q; a restore needs it to be running.", sourceId, source.Status))
		return
	}
	if err := checkBackupEngine(source.TypeDb, false); err != nil {
		response.Diagnostics.AddError("Restore is not available for this cluster", err.Error())
		return
	}

	points, err := r.api.listRestorePoints(sourceId)
	if err != nil {
		response.Diagnostics.AddError("Error reading the restore points", err.Error())
		return
	}
	found := false
	for _, p := range points {
		if string(p.BackupPart) == plan.BackupPart.ValueString() && strings.EqualFold(string(p.BackupType), plan.BackupType.ValueString()) {
			if !strings.EqualFold(string(p.BackupState), backupRestoreStateSuccess) {
				response.Diagnostics.AddError("The restore point cannot be used",
					fmt.Sprintf("Restore point %s (%s) is %q; only a successful backup can be restored.", p.BackupPart, p.BackupType, p.BackupState))
				return
			}
			found = true
		}
	}
	if !found {
		response.Diagnostics.AddError("The restore point does not exist",
			fmt.Sprintf("Cluster %s has no restore point %q of type %q. List them with the fptcloud_database_restore_points data source.",
				sourceId, plan.BackupPart.ValueString(), plan.BackupType.ValueString()))
		return
	}

	if taken, err := r.clusterNameTaken(source.VpcId, name); err != nil {
		response.Diagnostics.AddError("Error listing the clusters of the VPC", err.Error())
		return
	} else if taken {
		response.Diagnostics.AddError("The restore name is already used",
			fmt.Sprintf("A cluster named %q already exists in the VPC. Choose another restore_name, or delete that cluster first.", name))
		return
	}

	// Compute of the new cluster. An OpenSearch HA restore takes the compute of the source and has no flavor name.
	body := map[string]interface{}{
		"need_restore": "yes",
		"cluster_id":   sourceId,
		"backup_part":  plan.BackupPart.ValueString(),
		"backup_type":  plan.BackupType.ValueString(),
		"restore_name": name,
	}
	storage := source.StorageProfile
	if !plan.StorageProfile.IsNull() && !plan.StorageProfile.IsUnknown() && plan.StorageProfile.ValueString() != "" {
		storage = plan.StorageProfile.ValueString()
	}
	body["restore_storage_profile"] = storage
	plan.StorageProfile = types.StringValue(storage)

	if strings.EqualFold(source.TypeDb, "opensearch") && strings.EqualFold(source.IsCluster, "yes") {
		body["node_cpu"], body["node_ram"] = source.NodeCpu, source.NodeRam
		body["flavor"], body["flavor_id"] = "", source.FlavorId
		plan.FlavorId, plan.NodeCpu, plan.NodeRam = types.StringValue(source.FlavorId), types.Int64Value(int64(source.NodeCpu)), types.Int64Value(int64(source.NodeRam))
	} else {
		flavors, err := r.listRestoreFlavors(source.VpcId)
		if err != nil {
			response.Diagnostics.AddError("Error listing the flavors", err.Error())
			return
		}
		flavorId := plan.FlavorId.ValueString()
		if plan.FlavorId.IsNull() || plan.FlavorId.IsUnknown() {
			flavorId = ""
		}
		cpu, ram := plan.NodeCpu.ValueInt64(), plan.NodeRam.ValueInt64()
		if flavorId == "" && (plan.NodeCpu.IsNull() || plan.NodeCpu.IsUnknown()) {
			flavorId = source.FlavorId // nothing chosen: same compute as the source
		}
		flavor, err := resolveRestoreFlavor(flavors, flavorId, cpu, ram)
		if err != nil {
			response.Diagnostics.AddError("Invalid flavor", err.Error())
			return
		}
		// The API needs the flavor name; an empty one makes the restore fail.
		body["node_cpu"], body["node_ram"] = int64(flavor.Vcpu), int64(flavor.Ram)/1024
		body["flavor"], body["flavor_id"] = string(flavor.Name), flavor.FlavorId
		plan.FlavorId, plan.NodeCpu, plan.NodeRam = types.StringValue(flavor.FlavorId), types.Int64Value(int64(flavor.Vcpu)), types.Int64Value(int64(flavor.Ram)/1024)
	}

	if err := r.api.mutate("POST", common.ApiPath.DatabaseRestoreCreate(), body); err != nil {
		response.Diagnostics.AddError("Error restoring", err.Error())
		return
	}

	// restore/create does not return the new cluster: it is found by name.
	var created *restoredCluster
	if err := pollUntil(ctx, restoreAppearTimeout, "the restored cluster "+name+" to appear", func() (bool, error) {
		c, err := r.findRestoredCluster(sourceId, name)
		if err != nil {
			return false, nil
		}
		created = c
		return c != nil, nil
	}); err != nil {
		response.Diagnostics.AddError("The restored cluster was not found",
			err.Error()+". The restore may still have started: look for a cluster named "+name+" in the portal.")
		return
	}

	plan.Id = types.StringValue(created.ClusterId)
	plan.RestoredClusterId = types.StringValue(created.ClusterId)
	plan.RestoredStatus = created.statusValue()
	if plan.Timeouts.IsUnknown() {
		plan.Timeouts = types.ObjectNull(restoreTimeoutsObjectTypes())
	}
	// The cluster exists now: record it before waiting.
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	err = pollUntil(ctx, timeout, "the restored cluster "+created.ClusterId+" to be running", func() (bool, error) {
		c, err := r.findRestoredCluster(sourceId, name)
		if err != nil || c == nil {
			return false, nil
		}
		plan.RestoredStatus = c.statusValue()
		switch strings.ToLower(string(c.Status)) {
		case "running":
			return true, nil
		case "failed":
			return false, fmt.Errorf("the restore of cluster %s failed (status failed)", c.ClusterId)
		}
		return false, nil
	})
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
	if err != nil {
		response.Diagnostics.AddError("Error waiting for the restore", err.Error())
	}
}

func (c restoredCluster) statusValue() types.String {
	return types.StringValue(string(c.Status))
}

func (r *resourceDatabaseRestore) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state databaseRestoreModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	// Only the status is refreshed. The id is kept whatever happens to the source or to the restored cluster:
	// the resource is not removed from the state because of a change outside of Terraform.
	if id := state.RestoredClusterId.ValueString(); id != "" {
		if c, err := r.api.getClusterData(id); err != nil {
			response.Diagnostics.AddWarning("Could not read the restored cluster", err.Error())
		} else {
			state.RestoredStatus = types.StringValue(c.Status)
		}
	}
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *resourceDatabaseRestore) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state databaseRestoreModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	changed := !plan.ClusterId.Equal(state.ClusterId) || !plan.RestoreName.Equal(state.RestoreName) ||
		!plan.BackupPart.Equal(state.BackupPart) || !plan.BackupType.Equal(state.BackupType) ||
		!plan.FlavorId.Equal(state.FlavorId) || !plan.NodeCpu.Equal(state.NodeCpu) ||
		!plan.NodeRam.Equal(state.NodeRam) || !plan.StorageProfile.Equal(state.StorageProfile)
	if changed {
		tflog.Info(ctx, "Restore arguments changed; only the state is updated")
		response.Diagnostics.AddWarning("The restore was not run again",
			"An argument of the restore changed, so only the state was updated and the restored cluster is untouched. To restore again, use `terraform apply -replace=...`.")
	}
	plan.Id = state.Id
	plan.RestoredClusterId = state.RestoredClusterId
	plan.RestoredStatus = state.RestoredStatus
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *resourceDatabaseRestore) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state databaseRestoreModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	id := state.RestoredClusterId.ValueString()
	if id == "" {
		return
	}
	if !state.DeleteClusterOnDestroy.ValueBool() {
		response.Diagnostics.AddWarning("The restored cluster was kept",
			fmt.Sprintf("Cluster %s (%s) still exists. Set `delete_cluster_on_destroy = true` to delete it with this resource, or delete it on the portal.", id, state.RestoreName.ValueString()))
		return
	}

	timeout := timeoutFrom(state.Timeouts, "delete", backupDeleteTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := r.api.mutate("DELETE", common.ApiPath.DatabaseDelete(id), nil); err != nil {
		response.Diagnostics.AddError("Error deleting the restored cluster", err.Error())
		return
	}
	err := pollUntil(ctx, timeout, "the restored cluster "+id+" to be deleted", func() (bool, error) {
		c, err := r.api.getClusterData(id)
		if err != nil {
			return true, nil // no longer there
		}
		return strings.EqualFold(c.Status, "deleted"), nil
	})
	if err != nil {
		response.Diagnostics.AddError("Error waiting for the restored cluster to be deleted", err.Error())
	}
}
