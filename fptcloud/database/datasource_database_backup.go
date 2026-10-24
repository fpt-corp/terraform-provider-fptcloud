package fptcloud_database

import (
	"context"
	"fmt"
	"strings"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &datasourceDatabaseBackup{}
	_ datasource.DataSourceWithConfigure = &datasourceDatabaseBackup{}
)

type datasourceDatabaseBackup struct {
	api *databaseApiClient
}

func NewDataSourceDatabaseBackup() datasource.DataSource {
	return &datasourceDatabaseBackup{}
}

type databaseBackupDataSourceModel struct {
	Id              types.String  `tfsdk:"id"`
	ClusterId       types.String  `tfsdk:"cluster_id"`
	BackupServiceId types.Int64   `tfsdk:"backup_service_id"`
	Status          types.String  `tfsdk:"status"`
	CapacityGb      types.Int64   `tfsdk:"capacity_gb"`
	UsedGb          types.Float64 `tfsdk:"used_gb"`
	RetentionType   types.String  `tfsdk:"retention_type"`
	Jobs            types.List    `tfsdk:"jobs"`
}

// configureDatabaseApi builds the API client from the provider data; it is shared by the backup data sources and resources.
func configureDatabaseApi(providerData interface{}, diags *diag.Diagnostics) *databaseApiClient {
	if providerData == nil {
		return nil
	}
	client, ok := providerData.(*common.Client)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}
	return newDatabaseApiClient(client)
}

func backupJobAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"job_id":       types.Int64Type,
		"name":         types.StringType,
		"type":         types.StringType,
		"retention":    types.Int64Type,
		"hours":        types.SetType{ElemType: types.Int64Type},
		"minute":       types.Int64Type,
		"days_of_week": types.SetType{ElemType: types.Int64Type},
		"status":       types.StringType,
		"is_run_now":   types.BoolType,
	}
}

func int64Set(values []int64) types.Set {
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.Int64Value(v)
	}
	return types.SetValueMust(types.Int64Type, elems)
}

// backupJobsList converts the jobs of job/list for the jobs attribute of the data source.
func backupJobsList(jobs []backupJob) types.List {
	elems := make([]attr.Value, 0, len(jobs))
	for _, j := range jobs {
		elems = append(elems, types.ObjectValueMust(backupJobAttrTypes(), map[string]attr.Value{
			"job_id":       types.Int64Value(int64(j.JobId)),
			"name":         types.StringValue(string(j.Name)),
			"type":         types.StringValue(string(j.Type)),
			"retention":    types.Int64Value(int64(j.Retention)),
			"hours":        int64Set(j.hours()),
			"minute":       types.Int64Value(j.minute()),
			"days_of_week": int64Set(j.days()),
			"status":       types.StringValue(string(j.Status)),
			"is_run_now":   types.BoolValue(int64(j.IsRunNow) == 1),
		}))
	}
	return types.ListValueMust(types.ObjectType{AttrTypes: backupJobAttrTypes()}, elems)
}

func (d *datasourceDatabaseBackup) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_backup"
}

func (d *datasourceDatabaseBackup) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	d.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (d *datasourceDatabaseBackup) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Retrieves the backup service of a FPT Cloud database cluster: its state, capacity, usage and backup jobs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Id of the database cluster.",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster.",
			},
			"backup_service_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The Id of the backup service. Null when backup is not enabled.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "State of the backup service: inactive, enabled, processing, deleting, updating, active, deactivating, error_enabled, error_deleted or error_updated.",
			},
			"capacity_gb": schema.Int64Attribute{
				Computed:    true,
				Description: "Backup capacity in GB.",
			},
			"used_gb": schema.Float64Attribute{
				Computed:    true,
				Description: "Used part of the backup capacity in GB.",
			},
			"retention_type": schema.StringAttribute{
				Computed:    true,
				Description: "How retention is counted, `quantity` or `day`.",
			},
			"jobs": schema.ListAttribute{
				Computed:    true,
				Description: "Backup jobs of the cluster.",
				ElementType: types.ObjectType{AttrTypes: backupJobAttrTypes()},
			},
		},
	}
}

func (d *datasourceDatabaseBackup) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state databaseBackupDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	clusterId := strings.TrimSpace(state.ClusterId.ValueString())
	if clusterId == "" {
		response.Diagnostics.AddError("Missing cluster_id", "cluster_id must be a non empty string")
		return
	}

	detail, err := d.api.getBackupDetail(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading backup", err.Error())
		return
	}
	jobs, err := d.api.listBackupJobs(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading backup jobs", err.Error())
		return
	}

	state.Id = types.StringValue(clusterId)
	state.BackupServiceId = types.Int64Null()
	if detail.BackupServiceId != 0 {
		state.BackupServiceId = types.Int64Value(int64(detail.BackupServiceId))
	}
	state.Status = types.StringValue(backupStatusName(int64(detail.BackupEnabled)))
	state.CapacityGb = types.Int64Value(int64(detail.Capacity))
	state.UsedGb = types.Float64Value(detail.usedGb())
	state.RetentionType = types.StringValue(string(detail.BackupRetentionType))
	state.Jobs = backupJobsList(jobs)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}
