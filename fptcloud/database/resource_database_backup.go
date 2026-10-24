package fptcloud_database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &resourceDatabaseBackup{}
	_ resource.ResourceWithConfigure      = &resourceDatabaseBackup{}
	_ resource.ResourceWithImportState    = &resourceDatabaseBackup{}
	_ resource.ResourceWithModifyPlan     = &resourceDatabaseBackup{}
	_ resource.ResourceWithValidateConfig = &resourceDatabaseBackup{}
)

type resourceDatabaseBackup struct {
	api *databaseApiClient
}

func NewResourceDatabaseBackup() resource.Resource {
	return &resourceDatabaseBackup{}
}

type databaseBackupModel struct {
	Id              types.String  `tfsdk:"id"`
	ClusterId       types.String  `tfsdk:"cluster_id"`
	VpcId           types.String  `tfsdk:"vpc_id"`
	CapacityGb      types.Int64   `tfsdk:"capacity_gb"`
	FullJob         types.Object  `tfsdk:"full_job"`
	DiffJob         types.Object  `tfsdk:"diff_job"`
	BackupServiceId types.Int64   `tfsdk:"backup_service_id"`
	Status          types.String  `tfsdk:"status"`
	UsedGb          types.Float64 `tfsdk:"used_gb"`
	RetentionType   types.String  `tfsdk:"retention_type"`
	Timeouts        types.Object  `tfsdk:"timeouts"`
}

// databaseBackupTf is the model as the framework reads and writes it: full_job and diff_job are blocks, so lists
// of at most one object. The rest of the code uses databaseBackupModel, where they are single objects.
type databaseBackupTf struct {
	Id              types.String  `tfsdk:"id"`
	ClusterId       types.String  `tfsdk:"cluster_id"`
	VpcId           types.String  `tfsdk:"vpc_id"`
	CapacityGb      types.Int64   `tfsdk:"capacity_gb"`
	FullJob         types.List    `tfsdk:"full_job"`
	DiffJob         types.List    `tfsdk:"diff_job"`
	BackupServiceId types.Int64   `tfsdk:"backup_service_id"`
	Status          types.String  `tfsdk:"status"`
	UsedGb          types.Float64 `tfsdk:"used_gb"`
	RetentionType   types.String  `tfsdk:"retention_type"`
	Timeouts        types.Object  `tfsdk:"timeouts"`
}

func objectFromBlock(l types.List) types.Object {
	if l.IsUnknown() {
		return types.ObjectUnknown(backupJobObjectTypes())
	}
	if l.IsNull() || len(l.Elements()) == 0 {
		return nullJobObject()
	}
	return l.Elements()[0].(types.Object)
}

func blockFromObject(o types.Object) types.List {
	elemType := types.ObjectType{AttrTypes: backupJobObjectTypes()}
	if o.IsUnknown() {
		return types.ListUnknown(elemType)
	}
	if o.IsNull() {
		return types.ListValueMust(elemType, []attr.Value{})
	}
	return types.ListValueMust(elemType, []attr.Value{o})
}

func (t databaseBackupTf) model() databaseBackupModel {
	return databaseBackupModel{
		Id: t.Id, ClusterId: t.ClusterId, VpcId: t.VpcId, CapacityGb: t.CapacityGb,
		FullJob: objectFromBlock(t.FullJob), DiffJob: objectFromBlock(t.DiffJob),
		BackupServiceId: t.BackupServiceId, Status: t.Status, UsedGb: t.UsedGb, RetentionType: t.RetentionType,
		Timeouts: t.Timeouts,
	}
}

func (m databaseBackupModel) tf() databaseBackupTf {
	return databaseBackupTf{
		Id: m.Id, ClusterId: m.ClusterId, VpcId: m.VpcId, CapacityGb: m.CapacityGb,
		FullJob: blockFromObject(m.FullJob), DiffJob: blockFromObject(m.DiffJob),
		BackupServiceId: m.BackupServiceId, Status: m.Status, UsedGb: m.UsedGb, RetentionType: m.RetentionType,
		Timeouts: m.Timeouts,
	}
}

type tfGetter interface {
	Get(ctx context.Context, target interface{}) diag.Diagnostics
}

type tfSetter interface {
	Set(ctx context.Context, val interface{}) diag.Diagnostics
}

// getBackupModel reads a plan, state or config into the model.
func getBackupModel(ctx context.Context, from tfGetter, m *databaseBackupModel) diag.Diagnostics {
	var t databaseBackupTf
	diags := from.Get(ctx, &t)
	if !diags.HasError() {
		*m = t.model()
	}
	return diags
}

func setBackupModel(ctx context.Context, to tfSetter, m databaseBackupModel) diag.Diagnostics {
	return to.Set(ctx, m.tf())
}

type backupJobModel struct {
	Name       types.String `tfsdk:"name"`
	Retention  types.Int64  `tfsdk:"retention"`
	Hours      types.Set    `tfsdk:"hours"`
	Minute     types.Int64  `tfsdk:"minute"`
	DaysOfWeek types.Set    `tfsdk:"days_of_week"`
	JobId      types.Int64  `tfsdk:"job_id"`
	Status     types.String `tfsdk:"status"`
}

func backupJobObjectTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":         types.StringType,
		"retention":    types.Int64Type,
		"hours":        types.SetType{ElemType: types.Int64Type},
		"minute":       types.Int64Type,
		"days_of_week": types.SetType{ElemType: types.Int64Type},
		"job_id":       types.Int64Type,
		"status":       types.StringType,
	}
}

func timeoutsObjectTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"create": types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	}
}

// restoreTimeoutsObjectTypes are the timeouts of fptcloud_database_restore, which has no update.
func restoreTimeoutsObjectTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"create": types.StringType,
		"delete": types.StringType,
	}
}

// timeoutsBlock is the optional timeouts block shared by the backup and restore resources.
// Blocks are used instead of nested attributes because the provider is served over protocol 5.
func timeoutsBlock(withUpdate bool) schema.SingleNestedBlock {
	attrs := map[string]schema.Attribute{
		"create": schema.StringAttribute{Optional: true, Description: "How long to wait for the operation, as a duration such as `45m` or `1h`."},
		"delete": schema.StringAttribute{Optional: true, Description: "How long to wait for the operation, as a duration such as `30m`."},
	}
	if withUpdate {
		attrs["update"] = schema.StringAttribute{Optional: true, Description: "How long to wait for the operation, as a duration such as `45m` or `1h`."}
	}
	return schema.SingleNestedBlock{
		Description: "Timeouts for the long operations.",
		Attributes:  attrs,
	}
}

// timeoutFrom reads create, update or delete from a timeouts object; it returns def when unset or invalid.
func timeoutFrom(obj types.Object, key string, def time.Duration) time.Duration {
	if obj.IsNull() || obj.IsUnknown() {
		return def
	}
	if v, ok := obj.Attributes()[key].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		if d, err := time.ParseDuration(strings.TrimSpace(v.ValueString())); err == nil && d > 0 {
			return d
		}
	}
	return def
}

func validateTimeouts(obj types.Object, diags *diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return
	}
	for key, v := range obj.Attributes() {
		s, ok := v.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		if d, err := time.ParseDuration(strings.TrimSpace(s.ValueString())); err != nil || d <= 0 {
			diags.AddAttributeError(path.Root("timeouts").AtName(key), "Invalid timeout",
				fmt.Sprintf("%q is not a positive duration such as \"30m\" or \"1h\".", s.ValueString()))
		}
	}
}

func (r *resourceDatabaseBackup) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_backup"
}

func (r *resourceDatabaseBackup) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func backupJobAttributes(isDiff bool) map[string]schema.Attribute {
	retention := schema.Int64Attribute{
		Required:    true,
		Description: "Number of backups (or days, see `retention_type`) to keep: 1 to 31, 90, 180 or 365.",
	}
	if isDiff {
		retention = schema.Int64Attribute{
			Computed:    true,
			Description: "Set by the system to the retention of the full job.",
		}
	}
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Job name, 6 to 25 characters: letters, digits, spaces, `_` and `-`. It cannot be changed after the job is created.",
		},
		"retention": retention,
		"hours": schema.SetAttribute{
			Required:    true,
			ElementType: types.Int64Type,
			Description: "Hours of the day (0-23) the job runs. The full job and the diff job cannot share an hour.",
		},
		"minute": schema.Int64Attribute{
			Required:    true,
			Description: "Minute of the hour: 0, 5, 10, ..., 55 or 59.",
		},
		"days_of_week": schema.SetAttribute{
			Required:    true,
			ElementType: types.Int64Type,
			Description: "Days of the week the job runs, 1 (Monday) to 7 (Sunday).",
		},
		"job_id": schema.Int64Attribute{
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Description:   "The Id of the job.",
		},
		"status": schema.StringAttribute{
			Computed:    true,
			Description: "Status of the job, `ready` when it can be changed.",
		},
	}
}

func (r *resourceDatabaseBackup) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Manages the backup service of a FPT Cloud database cluster: its capacity and the full and diff backup jobs. " +
			"Nothing here is replaced: arguments that cannot be changed make the plan fail. Point-in-time recovery is not managed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "The Id of the database cluster.",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster. It cannot be changed after the backup is created.",
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "The VPC Id of the database cluster. It cannot be changed after the backup is created.",
			},
			"capacity_gb": schema.Int64Attribute{
				Required:    true,
				Description: "Backup capacity in GB, at least 10. It can only be increased.",
			},
			"backup_service_id": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "The Id of the backup service.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "State of the backup service: `enabled` or `active` when it is ready.",
			},
			"used_gb": schema.Float64Attribute{
				Computed:    true,
				Description: "Used part of the backup capacity in GB.",
			},
			"retention_type": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "How retention is counted, `quantity` or `day`. With `day` the jobs cannot be changed by Terraform.",
			},
		},
		Blocks: map[string]schema.Block{
			"full_job": schema.ListNestedBlock{
				Description:  "The full backup job. At most one block.",
				NestedObject: schema.NestedBlockObject{Attributes: backupJobAttributes(false)},
			},
			"diff_job": schema.ListNestedBlock{
				Description:  "The differential backup job. At most one block. It needs `full_job` and is not available for every engine.",
				NestedObject: schema.NestedBlockObject{Attributes: backupJobAttributes(true)},
			},
			"timeouts": timeoutsBlock(true),
		},
	}
}

// specFromObject reads a full_job or diff_job object. known is false when a configured value is not known yet.
func specFromObject(ctx context.Context, obj types.Object, isDiff bool) (spec *backupJobSpec, known bool, diags diag.Diagnostics) {
	if obj.IsNull() {
		return nil, true, nil
	}
	if obj.IsUnknown() {
		return nil, false, nil
	}
	var m backupJobModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, false, diags
	}
	spec = &backupJobSpec{Name: m.Name.ValueString(), Retention: m.Retention.ValueInt64(), Minute: m.Minute.ValueInt64(), JobId: m.JobId.ValueInt64()}
	known = !m.Name.IsUnknown() && !m.Minute.IsUnknown() && !m.Hours.IsUnknown() && !m.DaysOfWeek.IsUnknown() &&
		(isDiff || !m.Retention.IsUnknown())
	if !m.Hours.IsUnknown() && !m.Hours.IsNull() {
		diags.Append(m.Hours.ElementsAs(ctx, &spec.Hours, false)...)
	}
	if !m.DaysOfWeek.IsUnknown() && !m.DaysOfWeek.IsNull() {
		diags.Append(m.DaysOfWeek.ElementsAs(ctx, &spec.Days, false)...)
	}
	return spec, known, diags
}

// configFromModel reads the desired setup out of a plan or config. known is false when a value is not known yet.
func configFromModel(ctx context.Context, m *databaseBackupModel) (cfg backupConfig, known bool, diags diag.Diagnostics) {
	known = !m.CapacityGb.IsUnknown()
	cfg.CapacityGb = m.CapacityGb.ValueInt64()
	if m.CapacityGb.IsUnknown() {
		cfg.CapacityGb = backupMinCapacityGb
	}
	full, fullKnown, d := specFromObject(ctx, m.FullJob, false)
	diags.Append(d...)
	diff, diffKnown, d := specFromObject(ctx, m.DiffJob, true)
	diags.Append(d...)
	cfg.Full, cfg.Diff = full, diff
	return cfg, known && fullKnown && diffKnown, diags
}

// configFromServer is the setup that exists on the portal now.
func configFromServer(detail *backupDetail, jobs []backupJob) backupConfig {
	cfg := backupConfig{CapacityGb: int64(detail.Capacity)}
	if j := findBackupJob(jobs, backupJobFull); j != nil {
		cfg.Full = specFromJob(*j)
	}
	if j := findBackupJob(jobs, backupJobDiff); j != nil {
		cfg.Diff = specFromJob(*j)
	}
	return cfg
}

func specFromJob(j backupJob) *backupJobSpec {
	return &backupJobSpec{
		JobId:     int64(j.JobId),
		Name:      string(j.Name),
		Retention: int64(j.Retention),
		Hours:     j.hours(),
		Minute:    j.minute(),
		Days:      j.days(),
	}
}

func jobObject(j backupJob) types.Object {
	return types.ObjectValueMust(backupJobObjectTypes(), map[string]attr.Value{
		"name":         types.StringValue(string(j.Name)),
		"retention":    types.Int64Value(int64(j.Retention)),
		"hours":        int64Set(j.hours()),
		"minute":       types.Int64Value(j.minute()),
		"days_of_week": int64Set(j.days()),
		"job_id":       types.Int64Value(int64(j.JobId)),
		"status":       types.StringValue(string(j.Status)),
	})
}

func nullJobObject() types.Object {
	return types.ObjectNull(backupJobObjectTypes())
}

// modelFromServer builds the state from what the portal reports now. Settings that are not on the portal
// (vpc_id, timeouts) come from prior.
func modelFromServer(clusterId string, prior *databaseBackupModel, detail *backupDetail, jobs []backupJob) databaseBackupModel {
	m := databaseBackupModel{
		Id:              types.StringValue(clusterId),
		ClusterId:       types.StringValue(clusterId),
		VpcId:           prior.VpcId,
		CapacityGb:      types.Int64Value(int64(detail.Capacity)),
		FullJob:         nullJobObject(),
		DiffJob:         nullJobObject(),
		BackupServiceId: types.Int64Null(),
		Status:          types.StringValue(backupStatusName(int64(detail.BackupEnabled))),
		UsedGb:          types.Float64Value(detail.usedGb()),
		RetentionType:   types.StringValue(string(detail.BackupRetentionType)),
		Timeouts:        prior.Timeouts,
	}
	if m.Timeouts.IsUnknown() || m.Timeouts.AttributeTypes(context.Background()) == nil {
		m.Timeouts = types.ObjectNull(timeoutsObjectTypes())
	}
	if detail.BackupServiceId != 0 {
		m.BackupServiceId = types.Int64Value(int64(detail.BackupServiceId))
	}
	if j := findBackupJob(jobs, backupJobFull); j != nil {
		m.FullJob = jobObject(*j)
	}
	if j := findBackupJob(jobs, backupJobDiff); j != nil {
		m.DiffJob = jobObject(*j)
	}
	return m
}

// mergeApplied is the state after a successful apply: the planned settings, with what the portal computed
// (ids, status, usage, retention of the diff job) filled in.
func mergeApplied(plan, srv databaseBackupModel) databaseBackupModel {
	out := plan
	out.Id = srv.Id
	out.BackupServiceId = srv.BackupServiceId
	out.Status = srv.Status
	out.UsedGb = srv.UsedGb
	out.RetentionType = srv.RetentionType
	out.FullJob = mergeJobObject(plan.FullJob, srv.FullJob, false)
	out.DiffJob = mergeJobObject(plan.DiffJob, srv.DiffJob, true)
	if out.Timeouts.IsUnknown() {
		out.Timeouts = types.ObjectNull(timeoutsObjectTypes())
	}
	return out
}

func mergeJobObject(planObj, srvObj types.Object, retentionFromServer bool) types.Object {
	if planObj.IsNull() || srvObj.IsNull() {
		return srvObj
	}
	attrs := map[string]attr.Value{}
	for k, v := range planObj.Attributes() {
		attrs[k] = v
	}
	srv := srvObj.Attributes()
	attrs["job_id"] = srv["job_id"]
	attrs["status"] = srv["status"]
	if retentionFromServer {
		attrs["retention"] = srv["retention"]
	}
	return types.ObjectValueMust(backupJobObjectTypes(), attrs)
}

// backupSnapshot is the set of values compared against the portal on every Read.
func backupSnapshot(ctx context.Context, m *databaseBackupModel) driftSnapshot {
	snap := driftSnapshot{"capacity_gb": strconv.FormatInt(m.CapacityGb.ValueInt64(), 10)}
	add := func(label string, obj types.Object, isDiff bool) {
		spec, _, _ := specFromObject(ctx, obj, isDiff)
		if spec == nil {
			return
		}
		if !isDiff {
			snap[label+".retention"] = strconv.FormatInt(spec.Retention, 10)
		}
		snap[label+".hours"] = joinIntList(spec.Hours)
		snap[label+".minute"] = strconv.FormatInt(spec.Minute, 10)
		snap[label+".days_of_week"] = joinIntList(spec.Days)
	}
	add("full_job", m.FullJob, false)
	add("diff_job", m.DiffJob, true)
	return snap
}

func jobRequestFor(clusterId, jobType string, j backupJobSpec) backupJobRequest {
	return backupJobRequest{
		JobId:     j.JobId,
		ClusterId: clusterId,
		Name:      j.Name,
		Type:      jobType,
		Retention: j.Retention,
		Hour:      joinIntList(j.Hours),
		Minute:    strconv.FormatInt(j.Minute, 10),
		DayOfWeek: joinIntList(j.Days),
	}
}

// readServer reads the backup service and its jobs.
func (r *resourceDatabaseBackup) readServer(clusterId string) (*backupDetail, []backupJob, error) {
	detail, err := r.api.getBackupDetail(clusterId)
	if err != nil {
		return nil, nil, err
	}
	jobs, err := r.api.listBackupJobs(clusterId)
	if err != nil {
		return nil, nil, err
	}
	return detail, jobs, nil
}

// saveProgress writes the state as the portal has it now, so that a failure in a later step does not lose
// track of what was already created. When the portal cannot be read it writes the plan without unknown values.
func (r *resourceDatabaseBackup) saveProgress(ctx context.Context, state *tfsdk.State, clusterId string, plan databaseBackupModel) {
	detail, jobs, err := r.readServer(clusterId)
	var m databaseBackupModel
	if err == nil {
		m = mergeApplied(plan, modelFromServer(clusterId, &plan, detail, jobs))
		// What the portal has now wins over the plan: a step may not have happened yet.
		srv := modelFromServer(clusterId, &plan, detail, jobs)
		m.CapacityGb = srv.CapacityGb
		m.FullJob, m.DiffJob = srv.FullJob, srv.DiffJob
	} else {
		tflog.Warn(ctx, "Could not read the backup back to save progress: "+err.Error())
		m = plan
		m.Id = types.StringValue(clusterId)
		m.BackupServiceId = types.Int64Null()
		m.Status = types.StringNull()
		m.UsedGb = types.Float64Null()
		m.RetentionType = types.StringNull()
		m.FullJob, m.DiffJob = nullJobObject(), nullJobObject()
		if m.Timeouts.IsUnknown() {
			m.Timeouts = types.ObjectNull(timeoutsObjectTypes())
		}
	}
	setBackupModel(ctx, state, m)
}

func (r *resourceDatabaseBackup) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan databaseBackupModel
	response.Diagnostics.Append(getBackupModel(ctx, request.Plan, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	cfg, _, diags := configFromModel(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if problems := validateBackupConfig(cfg); len(problems) > 0 {
		response.Diagnostics.AddError("Invalid backup configuration", strings.Join(problems, "\n"))
	}
	if response.Diagnostics.HasError() {
		return
	}

	clusterId := strings.TrimSpace(plan.ClusterId.ValueString())
	timeout := timeoutFrom(plan.Timeouts, "create", backupCreateTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cluster, err := r.api.getClusterData(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading the database cluster", err.Error())
		return
	}
	if cluster.Status != "running" {
		response.Diagnostics.AddError("The database cluster is not running",
			fmt.Sprintf("Cluster %s is %q; backup can only be enabled while it is running.", clusterId, cluster.Status))
		return
	}
	if err := checkBackupEngine(cluster.TypeDb, cfg.Diff != nil); err != nil {
		response.Diagnostics.AddError("Backup is not available for this cluster", err.Error())
		return
	}
	detail, err := r.api.getBackupDetail(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading backup", err.Error())
		return
	}
	if int64(detail.BackupEnabled) != backupStatusInactive {
		response.Diagnostics.AddError("Backup already exists",
			fmt.Sprintf("The backup service of cluster %s is %s. Import it with `terraform import fptcloud_database_backup.<name> %s` instead of creating it.",
				clusterId, backupStatusName(int64(detail.BackupEnabled)), clusterId))
		return
	}
	busy, err := r.api.busyBackupActions(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading the action log", err.Error())
		return
	}
	if len(busy) > 0 {
		response.Diagnostics.AddError("The cluster is busy",
			fmt.Sprintf("Task %q is still %s on cluster %s. Apply again when it has finished.", busy[0].TaskName, busy[0].Status, clusterId))
		return
	}

	fail := func(summary string, err error) {
		r.saveProgress(ctx, &response.State, clusterId, plan)
		response.Diagnostics.AddError(summary, err.Error())
	}

	if err := r.api.setBackupCapacity(clusterId, cfg.CapacityGb, backupStatusInactive, 0); err != nil {
		response.Diagnostics.AddError("Error enabling backup", err.Error())
		return
	}
	// The service now exists on the portal: from here on the state must record it.
	if err := r.api.waitBackupReady(ctx, clusterId, timeout, func(d *backupDetail) bool { return int64(d.Capacity) >= cfg.CapacityGb }); err != nil {
		fail("Error waiting for the backup service", err)
		return
	}
	r.saveProgress(ctx, &response.State, clusterId, plan)

	for _, step := range []struct {
		kind backupStepKind
		typ  string
		job  *backupJobSpec
	}{
		{stepCreateFull, backupJobFull, cfg.Full},
		{stepCreateDiff, backupJobDiff, cfg.Diff},
	} {
		if step.job == nil {
			continue
		}
		job := *step.job
		if step.typ == backupJobDiff {
			job.Retention = cfg.Full.Retention
		}
		if err := r.runBackupStep(ctx, clusterId, backupStep{Kind: step.kind, Job: job}, timeout); err != nil {
			fail("Error during "+string(step.kind), err)
			return
		}
		r.saveProgress(ctx, &response.State, clusterId, plan)
	}

	r.finish(ctx, clusterId, plan, &response.State, response.Private, &response.Diagnostics)
}

// finish writes the final state after a successful apply and records the applied values for drift detection.
func (r *resourceDatabaseBackup) finish(ctx context.Context, clusterId string, plan databaseBackupModel, state *tfsdk.State, private privateState, diags *diag.Diagnostics) {
	detail, jobs, err := r.readServer(clusterId)
	if err != nil {
		diags.AddError("Error reading backup after the change", err.Error())
		return
	}
	final := mergeApplied(plan, modelFromServer(clusterId, &plan, detail, jobs))
	diags.Append(setBackupModel(ctx, state, final)...)
	diags.Append(saveDriftSnapshot(ctx, private, backupSnapshot(ctx, &final))...)
}

// runBackupStep makes one call to the portal and waits until the change is visible.
func (r *resourceDatabaseBackup) runBackupStep(ctx context.Context, clusterId string, step backupStep, timeout time.Duration) error {
	switch step.Kind {
	case stepDeleteDiff, stepDeleteFull:
		jobs, err := r.api.listBackupJobs(clusterId)
		if err != nil {
			return err
		}
		for _, j := range jobs {
			if int64(j.JobId) != step.Job.JobId {
				continue
			}
			if err := jobEditable(j); err != nil {
				return err
			}
			if err := r.api.deleteBackupJob(step.Job.JobId); err != nil {
				return err
			}
			return r.api.waitBackupJobGone(ctx, clusterId, step.Job.JobId, timeout)
		}
		return nil // already gone

	case stepUpdateFull, stepUpdateDiff:
		jobs, err := r.api.listBackupJobs(clusterId)
		if err != nil {
			return err
		}
		var current *backupJob
		for i := range jobs {
			if int64(jobs[i].JobId) == step.Job.JobId {
				current = &jobs[i]
			}
		}
		if current == nil {
			return fmt.Errorf("backup job %d (%s) does not exist on the portal any more", step.Job.JobId, step.Job.Name)
		}
		if err := jobEditable(*current); err != nil {
			return err
		}
		jobType := backupJobFull
		if step.Kind == stepUpdateDiff {
			jobType = backupJobDiff
		}
		if err := r.api.updateBackupJob(jobRequestFor(clusterId, jobType, step.Job)); err != nil {
			return err
		}
		_, err = r.api.waitBackupJob(ctx, clusterId, step.Job.Name, step.Job.JobId, timeout, func(j backupJob) bool {
			return specFromJob(j).sameSchedule(step.Job) && int64(j.Retention) == step.Job.Retention
		})
		return err

	case stepCreateFull, stepCreateDiff:
		jobs, err := r.api.listBackupJobs(clusterId)
		if err != nil {
			return err
		}
		for _, j := range jobs {
			if string(j.Name) == step.Job.Name {
				return fmt.Errorf("a backup job named %q already exists on cluster %s; use another name", step.Job.Name, clusterId)
			}
		}
		jobType := backupJobFull
		if step.Kind == stepCreateDiff {
			jobType = backupJobDiff
		}
		if err := r.api.createBackupJob(jobRequestFor(clusterId, jobType, step.Job)); err != nil {
			return err
		}
		// job/create does not return the id: the job is found again by its name.
		_, err = r.api.waitBackupJob(ctx, clusterId, step.Job.Name, 0, timeout, nil)
		return err

	case stepCapacity:
		detail, err := r.api.getBackupDetail(clusterId)
		if err != nil {
			return err
		}
		if !backupIsReady(int64(detail.BackupEnabled)) {
			return fmt.Errorf("the backup service is %s; wait until it is enabled or active", backupStatusName(int64(detail.BackupEnabled)))
		}
		// Sending the current is_archived back keeps point-in-time recovery as it is on the portal.
		if err := r.api.setBackupCapacity(clusterId, step.Capacity, int64(detail.BackupEnabled), int64(detail.IsArchived)); err != nil {
			return err
		}
		return r.api.waitBackupReady(ctx, clusterId, timeout, func(d *backupDetail) bool { return int64(d.Capacity) >= step.Capacity })
	}
	return fmt.Errorf("unknown step %q", step.Kind)
}

func (r *resourceDatabaseBackup) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state databaseBackupModel
	response.Diagnostics.Append(getBackupModel(ctx, request.State, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	clusterId := state.ClusterId.ValueString()
	if clusterId == "" {
		clusterId = state.Id.ValueString()
	}

	detail, jobs, err := r.readServer(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading backup", err.Error())
		return
	}
	if int64(detail.BackupEnabled) == backupStatusInactive {
		// The service looks removed outside of Terraform. Ask again before forgetting it: a resource dropped from the
		// state by one odd answer would be created again on the next apply.
		for attempt := 0; attempt < 2 && int64(detail.BackupEnabled) == backupStatusInactive; attempt++ {
			time.Sleep(backupPollInterval)
			if again, err := r.api.getBackupDetail(clusterId); err == nil {
				detail = again
			}
		}
		if int64(detail.BackupEnabled) == backupStatusInactive {
			response.State.RemoveResource(ctx)
			return
		}
	}
	if state.VpcId.IsNull() || state.VpcId.IsUnknown() {
		// An imported resource: the VPC comes from the cluster.
		if cluster, err := r.api.getClusterData(clusterId); err == nil {
			state.VpcId = types.StringValue(cluster.VpcId)
		}
	}

	applied, snapshotDiags := loadDriftSnapshot(ctx, request.Private)
	response.Diagnostics.Append(snapshotDiags...)

	fresh := modelFromServer(clusterId, &state, detail, jobs)
	if !backupIsReady(int64(detail.BackupEnabled)) {
		// While the service is changing, job/list answers with no jobs at all: keep the jobs of the state
		// instead of reporting them as deleted.
		fresh.FullJob, fresh.DiffJob = state.FullJob, state.DiffJob
	}
	for _, j := range []struct {
		label string
		prior types.Object
		now   types.Object
	}{{"full_job", state.FullJob, fresh.FullJob}, {"diff_job", state.DiffJob, fresh.DiffJob}} {
		if !j.prior.IsNull() && j.now.IsNull() {
			response.Diagnostics.AddWarning(j.label+" no longer exists on the portal",
				fmt.Sprintf("The %s was deleted outside of Terraform. The next apply will create it again.", j.label))
		}
	}
	addDriftWarnings(&response.Diagnostics, applied, backupSnapshot(ctx, &fresh))

	response.Diagnostics.Append(setBackupModel(ctx, &response.State, fresh)...)
}

func (r *resourceDatabaseBackup) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state databaseBackupModel
	response.Diagnostics.Append(getBackupModel(ctx, request.Plan, &plan)...)
	response.Diagnostics.Append(getBackupModel(ctx, request.State, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	desired, _, diags := configFromModel(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if problems := validateBackupConfig(desired); len(problems) > 0 {
		response.Diagnostics.AddError("Invalid backup configuration", strings.Join(problems, "\n"))
	}
	if response.Diagnostics.HasError() {
		return
	}

	clusterId := state.ClusterId.ValueString()
	timeout := timeoutFrom(plan.Timeouts, "update", backupUpdateTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The steps are worked out from what the portal has now, not from the state, so a change made on the portal is not missed.
	detail, jobs, err := r.readServer(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading backup", err.Error())
		return
	}
	if !backupIsReady(int64(detail.BackupEnabled)) {
		response.Diagnostics.AddError("The backup service is not ready",
			fmt.Sprintf("The backup service of cluster %s is %s; wait until it is enabled or active.", clusterId, backupStatusName(int64(detail.BackupEnabled))))
		return
	}
	steps, err := planBackupChanges(configFromServer(detail, jobs), desired)
	if err != nil {
		response.Diagnostics.AddError("Cannot apply the backup configuration", err.Error())
		return
	}
	if strings.EqualFold(string(detail.BackupRetentionType), "day") {
		for _, s := range steps {
			if s.Kind != stepCapacity {
				response.Diagnostics.AddError("Backup jobs cannot be changed", retentionDayMessage)
				return
			}
		}
	}
	if desired.Diff != nil {
		if cluster, err := r.api.getClusterData(clusterId); err == nil {
			if err := checkBackupEngine(cluster.TypeDb, true); err != nil {
				response.Diagnostics.AddError("Backup is not available for this cluster", err.Error())
				return
			}
		}
	}

	for _, step := range steps {
		tflog.Info(ctx, "Backup step: "+string(step.Kind))
		if err := r.runBackupStep(ctx, clusterId, step, timeout); err != nil {
			r.saveProgress(ctx, &response.State, clusterId, plan)
			response.Diagnostics.AddError("Error during "+string(step.Kind), err.Error())
			return
		}
		r.saveProgress(ctx, &response.State, clusterId, plan)
	}

	r.finish(ctx, clusterId, plan, &response.State, response.Private, &response.Diagnostics)
}

const retentionDayMessage = "The retention type of this backup is \"day\", so Terraform cannot change its jobs. " +
	"On the portal open the cluster, go to Backup and change the retention type back to the number of backups (quantity), " +
	"or change the jobs on the portal and set full_job and diff_job here to match."

func (r *resourceDatabaseBackup) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state databaseBackupModel
	response.Diagnostics.Append(getBackupModel(ctx, request.State, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	clusterId := state.ClusterId.ValueString()
	timeout := timeoutFrom(state.Timeouts, "delete", backupDeleteTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	detail, err := r.api.getBackupDetail(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading backup", err.Error())
		return
	}
	if int64(detail.BackupEnabled) == backupStatusInactive {
		return
	}
	// While the service is changing, job/list shows no jobs: wait until it is ready to see them.
	if !backupIsReady(int64(detail.BackupEnabled)) {
		if err := r.api.waitBackupReady(ctx, clusterId, timeout, nil); err != nil {
			response.Diagnostics.AddError("The backup service is not ready", err.Error())
			return
		}
	}
	jobs, err := r.api.listBackupJobs(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading backup jobs", err.Error())
		return
	}

	// Diff first, then full: the portal does not delete a full job while a diff job depends on it.
	for _, jobType := range []string{backupJobDiff, backupJobFull} {
		j := findBackupJob(jobs, jobType)
		if j == nil {
			continue
		}
		if int64(j.IsRunNow) == 1 {
			if err := r.api.waitBackupJobIdle(ctx, clusterId, int64(j.JobId), timeout); err != nil {
				response.Diagnostics.AddError("A backup is still running", err.Error())
				return
			}
		}
		kind := stepDeleteDiff
		if jobType == backupJobFull {
			kind = stepDeleteFull
		}
		if err := r.runBackupStep(ctx, clusterId, backupStep{Kind: kind, Job: *specFromJob(*j)}, timeout); err != nil {
			response.Diagnostics.AddError("Error during "+string(kind), err.Error())
			return
		}
	}

	if err := r.api.disableBackup(clusterId); err != nil {
		response.Diagnostics.AddError("Error disabling backup", err.Error())
		return
	}
	if err := r.api.waitBackupInactive(ctx, clusterId, timeout); err != nil {
		response.Diagnostics.AddError("Error waiting for the backup service to be removed", err.Error())
	}
}

func (r *resourceDatabaseBackup) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	id := strings.TrimSpace(request.ID)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), id)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("cluster_id"), id)...)
}

func (r *resourceDatabaseBackup) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var config databaseBackupModel
	response.Diagnostics.Append(getBackupModel(ctx, request.Config, &config)...)
	if response.Diagnostics.HasError() {
		return
	}
	validateTimeouts(config.Timeouts, &response.Diagnostics)

	cfg, known, diags := configFromModel(ctx, &config)
	response.Diagnostics.Append(diags...)
	if !known || response.Diagnostics.HasError() {
		// Values that come from other resources are checked at plan time, when they are known.
		if cfg.Diff != nil && cfg.Full == nil && !config.DiffJob.IsUnknown() && !config.FullJob.IsUnknown() {
			response.Diagnostics.AddAttributeError(path.Root("diff_job"), "diff_job needs full_job", "Add a full_job, or remove diff_job.")
		}
		return
	}
	for _, p := range validateBackupConfig(cfg) {
		response.Diagnostics.AddError("Invalid backup configuration", p)
	}
}

// ModifyPlan fails the plan for changes that cannot be made, instead of replacing the resource.
func (r *resourceDatabaseBackup) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	if request.Plan.Raw.IsNull() {
		return
	}
	var plan databaseBackupModel
	response.Diagnostics.Append(getBackupModel(ctx, request.Plan, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	cfg, known, diags := configFromModel(ctx, &plan)
	response.Diagnostics.Append(diags...)

	// The engine decides which jobs exist; the cluster may not exist yet when it is created in the same apply.
	if r.api != nil && !plan.ClusterId.IsUnknown() && plan.ClusterId.ValueString() != "" {
		if cluster, err := r.api.getClusterData(plan.ClusterId.ValueString()); err == nil {
			if err := checkBackupEngine(cluster.TypeDb, cfg.Diff != nil); err != nil {
				response.Diagnostics.AddError("Backup is not available for this cluster", err.Error())
			}
		}
	}

	if request.State.Raw.IsNull() {
		return
	}
	var state databaseBackupModel
	response.Diagnostics.Append(getBackupModel(ctx, request.State, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	for _, c := range []struct {
		name        string
		plan, state types.String
	}{{"cluster_id", plan.ClusterId, state.ClusterId}, {"vpc_id", plan.VpcId, state.VpcId}} {
		if !c.plan.IsUnknown() && !c.state.IsNull() && c.plan.ValueString() != c.state.ValueString() {
			response.Diagnostics.AddAttributeError(path.Root(c.name), "Cannot change "+c.name,
				fmt.Sprintf("`%s` cannot be changed after creation. Set it to `%s`. To back up another cluster, create another resource, or use `terraform apply -replace=...`.",
					c.name, c.state.ValueString()))
		}
	}
	if !plan.CapacityGb.IsUnknown() && !state.CapacityGb.IsNull() && plan.CapacityGb.ValueInt64() < state.CapacityGb.ValueInt64() {
		response.Diagnostics.AddAttributeError(path.Root("capacity_gb"), "Capacity cannot be decreased",
			fmt.Sprintf("`capacity_gb` cannot be decreased; it may have been increased on the portal. Set it to %d or more (the portal has %d now).",
				state.CapacityGb.ValueInt64(), state.CapacityGb.ValueInt64()))
	}

	current, _, d := configFromModel(ctx, &state)
	response.Diagnostics.Append(d...)
	for _, j := range []struct {
		label       string
		plan, state *backupJobSpec
	}{{"full_job", cfg.Full, current.Full}, {"diff_job", cfg.Diff, current.Diff}} {
		if j.plan != nil && j.state != nil && j.state.JobId != 0 && j.plan.Name != j.state.Name && known {
			response.Diagnostics.AddAttributeError(path.Root(j.label).AtName("name"), "Cannot change the job name",
				fmt.Sprintf("`%s.name` cannot be changed after the job is created. Set it to `%s`.", j.label, j.state.Name))
		}
	}

	if strings.EqualFold(state.RetentionType.ValueString(), "day") && known {
		changed := (cfg.Full == nil) != (current.Full == nil) || (cfg.Diff == nil) != (current.Diff == nil)
		if !changed && cfg.Full != nil {
			changed = !cfg.Full.sameSchedule(*current.Full) || cfg.Full.Retention != current.Full.Retention
		}
		if !changed && cfg.Diff != nil {
			changed = !cfg.Diff.sameSchedule(*current.Diff)
		}
		if changed {
			response.Diagnostics.AddError("Backup jobs cannot be changed", retentionDayMessage)
		}
	}
}
