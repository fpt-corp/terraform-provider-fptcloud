package fptcloud_database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &resourceDatabaseBackupRunNow{}
	_ resource.ResourceWithConfigure = &resourceDatabaseBackupRunNow{}
)

// resourceDatabaseBackupRunNow is an action resource: creating it, or changing `triggers`, runs a backup job once.
type resourceDatabaseBackupRunNow struct {
	api *databaseApiClient
}

func NewResourceDatabaseBackupRunNow() resource.Resource {
	return &resourceDatabaseBackupRunNow{}
}

type databaseBackupRunNowModel struct {
	Id          types.String `tfsdk:"id"`
	ClusterId   types.String `tfsdk:"cluster_id"`
	JobId       types.Int64  `tfsdk:"job_id"`
	Triggers    types.Map    `tfsdk:"triggers"`
	JobName     types.String `tfsdk:"job_name"`
	TriggeredAt types.String `tfsdk:"triggered_at"`
}

func (r *resourceDatabaseBackupRunNow) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_backup_run_now"
}

func (r *resourceDatabaseBackupRunNow) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (r *resourceDatabaseBackupRunNow) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Runs a backup job once, right away. Creating the resource, or changing `triggers`, starts the run and does not wait for it to finish. " +
			"Changing another argument only updates the state; to run again without changing `triggers`, use `terraform apply -replace=...`. Destroying it calls no API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "`<cluster_id>/<job_id>`.",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster.",
			},
			"job_id": schema.Int64Attribute{
				Required:    true,
				Description: "The Id of the backup job, for example `fptcloud_database_backup.db.full_job.job_id`.",
			},
			"triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arbitrary values; when one changes, the job runs again.",
			},
			"job_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the job.",
			},
			"triggered_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the job was last started by Terraform (UTC, RFC 3339).",
			},
		},
	}
}

// start checks that the job can run and starts it.
func (r *resourceDatabaseBackupRunNow) start(ctx context.Context, clusterId string, jobId int64) (jobName string, err error) {
	jobs, err := r.api.listBackupJobs(clusterId)
	if err != nil {
		return "", err
	}
	for _, j := range jobs {
		if int64(j.JobId) != jobId {
			continue
		}
		if !strings.EqualFold(string(j.Status), jobStatusReady) {
			return "", fmt.Errorf("backup job %s is %s; it can only run when it is ready", j.Name, j.Status)
		}
		if int64(j.IsRunNow) == 1 {
			return "", fmt.Errorf("backup job %s is already running", j.Name)
		}
		if err := r.api.runNowBackupJob(jobId); err != nil {
			return "", err
		}
		tflog.Info(ctx, fmt.Sprintf("Started backup job %d of cluster %s", jobId, clusterId))
		return string(j.Name), nil
	}
	return "", fmt.Errorf("backup job %d does not exist on cluster %s", jobId, clusterId)
}

func (r *resourceDatabaseBackupRunNow) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan databaseBackupRunNowModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	name, err := r.start(ctx, plan.ClusterId.ValueString(), plan.JobId.ValueInt64())
	if err != nil {
		response.Diagnostics.AddError("Error running the backup job", err.Error())
		return
	}
	plan.Id = types.StringValue(plan.ClusterId.ValueString() + "/" + strconv.FormatInt(plan.JobId.ValueInt64(), 10))
	plan.JobName = types.StringValue(name)
	plan.TriggeredAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *resourceDatabaseBackupRunNow) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state databaseBackupRunNowModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	// The run already happened; only the name is refreshed, and the resource stays even if the job is gone.
	jobs, err := r.api.listBackupJobs(state.ClusterId.ValueString())
	if err != nil {
		response.Diagnostics.AddWarning("Could not read the backup job", err.Error())
	} else {
		for _, j := range jobs {
			if int64(j.JobId) == state.JobId.ValueInt64() {
				state.JobName = types.StringValue(string(j.Name))
			}
		}
	}
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *resourceDatabaseBackupRunNow) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state databaseBackupRunNowModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	plan.Id = types.StringValue(plan.ClusterId.ValueString() + "/" + strconv.FormatInt(plan.JobId.ValueInt64(), 10))
	plan.JobName = state.JobName
	plan.TriggeredAt = state.TriggeredAt

	if !plan.Triggers.Equal(state.Triggers) {
		name, err := r.start(ctx, plan.ClusterId.ValueString(), plan.JobId.ValueInt64())
		if err != nil {
			response.Diagnostics.AddError("Error running the backup job", err.Error())
			return
		}
		plan.JobName = types.StringValue(name)
		plan.TriggeredAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	} else {
		response.Diagnostics.AddWarning("The backup job was not run again",
			"`cluster_id` or `job_id` changed, so only the state was updated. To run the job, change `triggers` or use `terraform apply -replace=...`.")
	}
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *resourceDatabaseBackupRunNow) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	// Nothing to undo: a backup that ran stays.
}
