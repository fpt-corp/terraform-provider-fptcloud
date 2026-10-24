package fptcloud_database

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &datasourceDatabaseRestorePoints{}
	_ datasource.DataSourceWithConfigure = &datasourceDatabaseRestorePoints{}
	_ datasource.DataSource              = &datasourceDatabaseRestoredClusters{}
	_ datasource.DataSourceWithConfigure = &datasourceDatabaseRestoredClusters{}
)

type datasourceDatabaseRestorePoints struct {
	api *databaseApiClient
}

type datasourceDatabaseRestoredClusters struct {
	api *databaseApiClient
}

func NewDataSourceDatabaseRestorePoints() datasource.DataSource {
	return &datasourceDatabaseRestorePoints{}
}

func NewDataSourceDatabaseRestoredClusters() datasource.DataSource {
	return &datasourceDatabaseRestoredClusters{}
}

type databaseRestorePointsModel struct {
	Id            types.String `tfsdk:"id"`
	ClusterId     types.String `tfsdk:"cluster_id"`
	RestorePoints types.List   `tfsdk:"restore_points"`
}

type databaseRestoredClustersModel struct {
	Id        types.String `tfsdk:"id"`
	ClusterId types.String `tfsdk:"cluster_id"`
	Clusters  types.List   `tfsdk:"clusters"`
}

func restorePointAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"backup_part":  types.StringType,
		"backup_type":  types.StringType,
		"backup_state": types.StringType,
		"backup_size":  types.StringType,
		"start_time":   types.StringType,
		"finish_time":  types.StringType,
	}
}

func restoredClusterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cluster_id":    types.StringType,
		"cluster_name":  types.StringType,
		"version":       types.StringType,
		"type_db":       types.StringType,
		"status":        types.StringType,
		"created_at":    types.StringType,
		"restore_point": types.StringType,
	}
}

func (d *datasourceDatabaseRestorePoints) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_restore_points"
}

func (d *datasourceDatabaseRestorePoints) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	d.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (d *datasourceDatabaseRestorePoints) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Lists the restore points (backups) of a FPT Cloud database cluster. A restore point with `backup_state = \"successful\"` can be restored with `fptcloud_database_restore`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Id of the database cluster.",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster.",
			},
			"restore_points": schema.ListAttribute{
				Computed:    true,
				Description: "Restore points, newest first.",
				ElementType: types.ObjectType{AttrTypes: restorePointAttrTypes()},
			},
		},
	}
}

func (d *datasourceDatabaseRestorePoints) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state databaseRestorePointsModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	clusterId := strings.TrimSpace(state.ClusterId.ValueString())
	if clusterId == "" {
		response.Diagnostics.AddError("Missing cluster_id", "cluster_id must be a non empty string")
		return
	}

	points, err := d.api.listRestorePoints(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading restore points", err.Error())
		return
	}
	elems := make([]attr.Value, 0, len(points))
	for _, p := range points {
		elems = append(elems, types.ObjectValueMust(restorePointAttrTypes(), map[string]attr.Value{
			"backup_part":  types.StringValue(string(p.BackupPart)),
			"backup_type":  types.StringValue(string(p.BackupType)),
			"backup_state": types.StringValue(string(p.BackupState)),
			"backup_size":  types.StringValue(string(p.BackupSize)),
			"start_time":   types.StringValue(string(p.StartTime)),
			"finish_time":  types.StringValue(string(p.FinishTime)),
		}))
	}
	state.Id = types.StringValue(clusterId)
	state.RestorePoints = types.ListValueMust(types.ObjectType{AttrTypes: restorePointAttrTypes()}, elems)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (d *datasourceDatabaseRestoredClusters) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_restored_clusters"
}

func (d *datasourceDatabaseRestoredClusters) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	d.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (d *datasourceDatabaseRestoredClusters) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Lists the clusters that were restored from a FPT Cloud database cluster, including deleted ones (status `deleted`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Id of the source database cluster.",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the source database cluster.",
			},
			"clusters": schema.ListAttribute{
				Computed:    true,
				Description: "Restored clusters.",
				ElementType: types.ObjectType{AttrTypes: restoredClusterAttrTypes()},
			},
		},
	}
}

func (d *datasourceDatabaseRestoredClusters) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state databaseRestoredClustersModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	clusterId := strings.TrimSpace(state.ClusterId.ValueString())
	if clusterId == "" {
		response.Diagnostics.AddError("Missing cluster_id", "cluster_id must be a non empty string")
		return
	}

	clusters, err := d.api.listRestoredClusters(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading restored clusters", err.Error())
		return
	}
	elems := make([]attr.Value, 0, len(clusters))
	for _, c := range clusters {
		elems = append(elems, types.ObjectValueMust(restoredClusterAttrTypes(), map[string]attr.Value{
			"cluster_id":    types.StringValue(c.ClusterId),
			"cluster_name":  types.StringValue(string(c.ClusterName)),
			"version":       types.StringValue(string(c.Version)),
			"type_db":       types.StringValue(string(c.TypeDb)),
			"status":        types.StringValue(string(c.Status)),
			"created_at":    types.StringValue(string(c.CreatedAt)),
			"restore_point": types.StringValue(string(c.RestorePoint)),
		}))
	}
	state.Id = types.StringValue(clusterId)
	state.Clusters = types.ListValueMust(types.ObjectType{AttrTypes: restoredClusterAttrTypes()}, elems)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}
