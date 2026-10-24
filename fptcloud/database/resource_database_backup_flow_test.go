package fptcloud_database

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func backupResource(api *databaseApiClient) (*resourceDatabaseBackup, resource.SchemaResponse) {
	r := &resourceDatabaseBackup{api: api}
	var s resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &s)
	return r, s
}

func emptyBackupState(s resource.SchemaResponse) tfsdk.State {
	return tfsdk.State{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)}
}

func jobAttrs(name string, retention int64, hours []int64, minute int64, days []int64) map[string]attr.Value {
	ret := types.Int64Value(retention)
	if retention == 0 {
		ret = types.Int64Unknown()
	}
	return map[string]attr.Value{
		"name": types.StringValue(name), "retention": ret, "hours": int64Set(hours),
		"minute": types.Int64Value(minute), "days_of_week": int64Set(days),
		"job_id": types.Int64Unknown(), "status": types.StringUnknown(),
	}
}

// planModel is what Terraform hands to Create: the configured values, with the computed ones unknown.
func planModel(capacity int64, full, diff map[string]attr.Value) databaseBackupModel {
	m := databaseBackupModel{
		Id: types.StringUnknown(), ClusterId: types.StringValue("c1"), VpcId: types.StringValue("vpc-1"),
		CapacityGb: types.Int64Value(capacity), FullJob: nullJobObject(), DiffJob: nullJobObject(),
		BackupServiceId: types.Int64Unknown(), Status: types.StringUnknown(), UsedGb: types.Float64Unknown(),
		RetentionType: types.StringUnknown(), Timeouts: types.ObjectNull(timeoutsObjectTypes()),
	}
	if full != nil {
		m.FullJob = types.ObjectValueMust(backupJobObjectTypes(), full)
	}
	if diff != nil {
		m.DiffJob = types.ObjectValueMust(backupJobObjectTypes(), diff)
	}
	return m
}

func planFrom(t *testing.T, s resource.SchemaResponse, m databaseBackupModel) tfsdk.Plan {
	t.Helper()
	p := tfsdk.Plan{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)}
	require.False(t, p.Set(context.Background(), m.tf()).HasError())
	return p
}

// withPriorJobIds is the plan of an update: the ids of existing jobs are known from the state.
func withPriorJobIds(m databaseBackupModel, state databaseBackupModel) databaseBackupModel {
	for _, pair := range []struct {
		obj   *types.Object
		prior types.Object
	}{{&m.FullJob, state.FullJob}, {&m.DiffJob, state.DiffJob}} {
		if pair.obj.IsNull() || pair.prior.IsNull() {
			continue
		}
		attrs := pair.obj.Attributes()
		attrs["job_id"] = pair.prior.Attributes()["job_id"]
		*pair.obj = types.ObjectValueMust(backupJobObjectTypes(), attrs)
	}
	m.Id, m.BackupServiceId, m.RetentionType = state.Id, state.BackupServiceId, state.RetentionType
	return m
}

func createBackup(t *testing.T, r *resourceDatabaseBackup, s resource.SchemaResponse, plan databaseBackupModel) (*resource.CreateResponse, databaseBackupModel) {
	t.Helper()
	resp := &resource.CreateResponse{State: emptyBackupState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: planFrom(t, s, plan)}, resp)
	var got databaseBackupModel
	if !resp.State.Raw.IsNull() {
		require.False(t, getBackupModel(context.Background(), resp.State, &got).HasError())
	}
	return resp, got
}

func standardPlan() databaseBackupModel {
	return planModel(10,
		jobAttrs("daily-full", 7, []int64{1}, 30, []int64{1, 2, 3, 4, 5, 6, 7}),
		jobAttrs("hourly-diff", 0, []int64{5, 9}, 0, []int64{1, 2, 3, 4, 5}))
}

func TestBackupCreate(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))

	resp, got := createBackup(t, r, s, standardPlan())
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

	// The state has the ids and values the portal computed.
	assert.Equal(t, "c1", got.Id.ValueString())
	assert.Equal(t, int64(229), got.BackupServiceId.ValueInt64())
	assert.Equal(t, "enabled", got.Status.ValueString())
	assert.Equal(t, "quantity", got.RetentionType.ValueString())
	assert.Equal(t, int64(10), got.CapacityGb.ValueInt64())
	full, _, _ := specFromObject(context.Background(), got.FullJob, false)
	diff, _, _ := specFromObject(context.Background(), got.DiffJob, true)
	require.NotNil(t, full)
	require.NotNil(t, diff)
	assert.NotZero(t, full.JobId)
	assert.NotEqual(t, full.JobId, diff.JobId)
	assert.Equal(t, int64(7), diff.Retention, "the diff job takes the retention of the full job")
	assert.Equal(t, "ready", got.FullJob.Attributes()["status"].(types.String).ValueString())

	// The service is enabled without PITR, then the full job is created, then the diff job.
	assert.Equal(t, `{"backup_enabled":0,"capacity":10,"cluster_id":"c1","is_archived":0,"storage_profile":null}`,
		p.lastBody("POST /v1/xplat/database/backup/service/capacity"))
	creates := p.bodies["POST /v1/xplat/database/backup/service/job/create"]
	require.Len(t, creates, 2)
	assert.Contains(t, creates[0], `"backup_type":"full"`)
	assert.Contains(t, creates[0], `"hour":"1","minute":"30","day_of_week":"1,2,3,4,5,6,7"`)
	assert.Contains(t, creates[1], `"backup_type":"diff"`)
	assert.Contains(t, creates[1], `"backup_retention":7`)
	assert.Contains(t, creates[1], `"hour":"5,9"`)
}

func TestBackupCreateWithoutJobs(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	resp, got := createBackup(t, r, s, planModel(50, nil, nil))
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Equal(t, int64(50), got.CapacityGb.ValueInt64())
	assert.True(t, got.FullJob.IsNull())
	assert.True(t, got.DiffJob.IsNull())
	assert.Empty(t, p.callsLike("job/create"))
}

func TestBackupCreateRefusals(t *testing.T) {
	tests := []struct {
		name  string
		setup func(p *fakePortal)
		plan  databaseBackupModel
		want  string
	}{
		{"cluster not running", func(p *fakePortal) { p.clusterStatus = "stopped" }, standardPlan(), "is not running"},
		{"an unfinished task", func(p *fakePortal) { p.busy = true }, standardPlan(), "The cluster is busy"},
		{"backup already enabled", func(p *fakePortal) { p.backupEnabled, p.serviceId = 1, 5 }, standardPlan(), "terraform import"},
		{"mongodb enterprise", func(p *fakePortal) { p.typeDb = "mongodb_enterprise" }, planModel(10, jobAttrs("daily-full", 7, []int64{1}, 30, []int64{1}), nil), "not supported by Terraform yet"},
		{"mongodb sharding", func(p *fakePortal) { p.typeDb = "mongodb_sharding" }, planModel(10, nil, nil), "not supported by Terraform yet"},
		{"diff on an engine without it", func(p *fakePortal) { p.typeDb = "redis_replication" }, standardPlan(), "diff_job is not available"},
		{"invalid config", func(p *fakePortal) {}, planModel(5, nil, nil), "capacity_gb must be at least 10"},
		{"no alert for the backup", func(p *fakePortal) {
			p.capacityErr = string(fixture(t, "capacity_error_notification"))
		}, standardPlan(), "alert_service \"backup\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newFakePortal()
			tt.setup(p)
			r, s := backupResource(p.start(t))
			resp, _ := createBackup(t, r, s, tt.plan)
			require.True(t, resp.Diagnostics.HasError())
			assert.Contains(t, resp.Diagnostics.Errors()[0].Summary()+" "+resp.Diagnostics.Errors()[0].Detail(), tt.want)
			assert.True(t, resp.State.Raw.IsNull(), "nothing was created, so nothing is in the state")
			assert.Empty(t, p.callsLike("job/create"))
		})
	}
}

func TestBackupCreateKeepsTheServiceWhenAJobFails(t *testing.T) {
	p := newFakePortal()
	p.createFails = true
	r, s := backupResource(p.start(t))
	resp, got := createBackup(t, r, s, standardPlan())

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "created_fail")
	// The service exists on the portal, so the state must record it or the next apply would lose it.
	require.False(t, resp.State.Raw.IsNull())
	assert.Equal(t, int64(229), got.BackupServiceId.ValueInt64())
	assert.Equal(t, "c1", got.ClusterId.ValueString())
}

func TestBackupRead(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())

	read := func(prior databaseBackupModel) (*resource.ReadResponse, databaseBackupModel) {
		st := emptyBackupState(s)
		require.False(t, setBackupModel(context.Background(), &st, prior).HasError())
		resp := &resource.ReadResponse{State: st}
		r.Read(context.Background(), resource.ReadRequest{State: st}, resp)
		var got databaseBackupModel
		if !resp.State.Raw.IsNull() {
			require.False(t, getBackupModel(context.Background(), resp.State, &got).HasError())
		}
		return resp, got
	}

	// Unchanged on the portal: no warnings.
	resp, got := read(created)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Empty(t, resp.Diagnostics.Warnings())
	assert.Equal(t, created.FullJob, got.FullJob)

	// The portal has a different schedule and capacity: Read writes the real values.
	p.mu.Lock()
	p.capacity = 30
	p.jobs[0].hour = "8"
	p.mu.Unlock()
	_, got = read(created)
	assert.Equal(t, int64(30), got.CapacityGb.ValueInt64())
	full, _, _ := specFromObject(context.Background(), got.FullJob, false)
	assert.Equal(t, []int64{8}, full.Hours)

	// A job deleted on the portal becomes null, with a warning.
	p.mu.Lock()
	p.jobs = p.jobs[:1]
	p.mu.Unlock()
	resp, got = read(created)
	assert.True(t, got.DiffJob.IsNull())
	require.NotEmpty(t, resp.Diagnostics.Warnings())
	assert.Contains(t, resp.Diagnostics.Warnings()[0].Summary(), "diff_job no longer exists")

	// A service removed on the portal leaves the state.
	p.mu.Lock()
	p.backupEnabled, p.serviceId = 0, 0
	p.mu.Unlock()
	resp, _ = read(created)
	assert.True(t, resp.State.Raw.IsNull())
}

func TestBackupReadDoesNotForgetAServiceOnOneOddAnswer(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())

	// One answer says inactive, the following ones say the service is there.
	var reads int32
	odd := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "backup/detail/") && atomic.AddInt32(&reads, 1) == 1 {
			_, _ = w.Write([]byte(`{"cluster_id":"c1","backup_enabled":0,"capacity":0}`))
			return
		}
		p.serve(w, r)
	}))
	r.api = odd
	st := emptyBackupState(s)
	require.False(t, setBackupModel(context.Background(), &st, created).HasError())
	resp := &resource.ReadResponse{State: st}
	r.Read(context.Background(), resource.ReadRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.False(t, resp.State.Raw.IsNull(), "the resource stays")
}

func TestBackupReadFillsVpcOnImport(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())

	imported := databaseBackupModel{
		Id: types.StringValue("c1"), ClusterId: types.StringValue("c1"), VpcId: types.StringNull(), CapacityGb: types.Int64Null(),
		FullJob: nullJobObject(), DiffJob: nullJobObject(), BackupServiceId: types.Int64Null(), Status: types.StringNull(),
		UsedGb: types.Float64Null(), RetentionType: types.StringNull(), Timeouts: types.ObjectNull(timeoutsObjectTypes()),
	}
	st := emptyBackupState(s)
	require.False(t, setBackupModel(context.Background(), &st, imported).HasError())
	resp := &resource.ReadResponse{State: st}
	r.Read(context.Background(), resource.ReadRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	var got databaseBackupModel
	require.False(t, getBackupModel(context.Background(), resp.State, &got).HasError())
	assert.Equal(t, "vpc-1", got.VpcId.ValueString())
	assert.Equal(t, created.FullJob, got.FullJob)
	assert.Empty(t, resp.Diagnostics.Warnings(), "an import has no applied values to compare with")
}

func updateBackup(t *testing.T, r *resourceDatabaseBackup, s resource.SchemaResponse, prior databaseBackupModel, plan databaseBackupModel) (*resource.UpdateResponse, databaseBackupModel) {
	t.Helper()
	st := emptyBackupState(s)
	require.False(t, setBackupModel(context.Background(), &st, prior).HasError())
	plan = withPriorJobIds(plan, prior)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s.Schema, Raw: st.Raw.Copy()}}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planFrom(t, s, plan), State: st}, resp)
	var got databaseBackupModel
	require.False(t, getBackupModel(context.Background(), resp.State, &got).HasError())
	return resp, got
}

func TestBackupUpdateOrder(t *testing.T) {
	p := newFakePortal()
	p.isArchived = 0
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())

	p.mu.Lock()
	p.isArchived = 1 // PITR turned on on the portal
	p.calls = nil
	p.bodies = map[string][]string{}
	p.mu.Unlock()

	// New schedule and a longer retention for both jobs, and more capacity.
	plan := planModel(25,
		jobAttrs("daily-full", 14, []int64{2}, 15, []int64{1, 2}),
		jobAttrs("hourly-diff", 0, []int64{6, 10}, 45, []int64{3}))
	resp, got := updateBackup(t, r, s, created, plan)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

	var order []string
	for _, c := range p.calls {
		if strings.Contains(c, "/job/update") || strings.Contains(c, "service/capacity") || strings.Contains(c, "/job/create") || strings.Contains(c, "/job/delete") {
			order = append(order, c)
		}
	}
	assert.Equal(t, []string{
		"PUT /v1/xplat/database/backup/service/job/update",
		"PUT /v1/xplat/database/backup/service/job/update",
		"POST /v1/xplat/database/backup/service/capacity",
	}, order)
	updates := p.bodies["PUT /v1/xplat/database/backup/service/job/update"]
	assert.Contains(t, updates[0], `"backup_type":"full"`)
	assert.Contains(t, updates[0], `"backup_retention":14`)
	assert.Contains(t, updates[1], `"backup_type":"diff"`)
	assert.Contains(t, updates[1], `"backup_retention":14`, "the diff job is sent the new retention of the full job")

	// The capacity call sends the state of the service and the is_archived the portal has, so PITR is not turned off.
	assert.Equal(t, `{"backup_enabled":1,"capacity":25,"cluster_id":"c1","is_archived":1,"storage_profile":null}`,
		p.lastBody("POST /v1/xplat/database/backup/service/capacity"))

	assert.Equal(t, int64(25), got.CapacityGb.ValueInt64())
	full, _, _ := specFromObject(context.Background(), got.FullJob, false)
	assert.Equal(t, []int64{2}, full.Hours)
	assert.Equal(t, int64(15), full.Minute)
}

func TestBackupUpdateRemovesJobsDiffFirst(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())
	diffId := created.DiffJob.Attributes()["job_id"].(types.Int64).ValueInt64()
	fullId := created.FullJob.Attributes()["job_id"].(types.Int64).ValueInt64()
	p.mu.Lock()
	p.calls = nil
	p.mu.Unlock()

	// Removing full_job and diff_job from the configuration deletes the jobs; the resource stays.
	resp, got := updateBackup(t, r, s, created, planModel(10, nil, nil))
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Equal(t, []string{
		"DELETE /v1/xplat/database/backup/service/job/delete/" + itoa(int(diffId)),
		"DELETE /v1/xplat/database/backup/service/job/delete/" + itoa(int(fullId)),
	}, p.callsLike("job/delete"))
	assert.True(t, got.FullJob.IsNull())
	assert.True(t, got.DiffJob.IsNull())
	assert.Empty(t, p.callsLike("service/delete"), "the service stays")
}

func TestBackupUpdateRefusals(t *testing.T) {
	setup := func() (*fakePortal, *resourceDatabaseBackup, resource.SchemaResponse, databaseBackupModel) {
		p := newFakePortal()
		r, s := backupResource(p.start(t))
		_, created := createBackup(t, r, s, standardPlan())
		return p, r, s, created
	}
	changed := planModel(10,
		jobAttrs("daily-full", 7, []int64{2}, 30, []int64{1}),
		jobAttrs("hourly-diff", 0, []int64{5, 9}, 0, []int64{1, 2, 3, 4, 5}))

	t.Run("job is running", func(t *testing.T) {
		p, r, s, created := setup()
		p.mu.Lock()
		p.jobs[0].runNow = 1
		p.mu.Unlock()
		resp, _ := updateBackup(t, r, s, created, changed)
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "is running")
		assert.Empty(t, p.callsLike("job/update"))
	})
	t.Run("retention type day", func(t *testing.T) {
		p, r, s, created := setup()
		p.mu.Lock()
		p.retentionType = "day"
		p.calls = nil
		p.mu.Unlock()
		resp, _ := updateBackup(t, r, s, created, changed)
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "change the retention type back")
		assert.Empty(t, p.callsLike("job/create"), "no job call, and update-retention-type is never called")
		assert.Empty(t, p.callsLike("job/update"))
		assert.Empty(t, p.callsLike("job/delete"))
		assert.Empty(t, p.callsLike("retention"))
	})
	t.Run("capacity decreased on the portal side", func(t *testing.T) {
		p, r, s, created := setup()
		p.mu.Lock()
		p.capacity = 40
		p.mu.Unlock()
		resp, _ := updateBackup(t, r, s, created, planModel(10, nil, nil))
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "cannot be decreased")
	})
	t.Run("service not ready", func(t *testing.T) {
		p, r, s, created := setup()
		p.mu.Lock()
		p.backupEnabled = backupStatusUpdating
		p.mu.Unlock()
		resp, _ := updateBackup(t, r, s, created, changed)
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "not ready")
	})
	t.Run("a job that ended in updated_fail can be updated again", func(t *testing.T) {
		p, r, s, created := setup()
		p.mu.Lock()
		p.jobs[0].status = "updated_fail"
		p.mu.Unlock()
		resp, _ := updateBackup(t, r, s, created, changed)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	})
}

func TestBackupUpdateRecreatesAJobDeletedOnThePortal(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())
	p.mu.Lock()
	p.jobs = nil // both jobs deleted on the portal
	p.calls = nil
	p.mu.Unlock()

	resp, got := updateBackup(t, r, s, created, standardPlan())
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Len(t, p.callsLike("job/create"), 2)
	assert.False(t, got.FullJob.IsNull())
	assert.False(t, got.DiffJob.IsNull())
}

func TestBackupDelete(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())
	p.mu.Lock()
	p.calls = nil
	p.mu.Unlock()

	st := emptyBackupState(s)
	require.False(t, setBackupModel(context.Background(), &st, created).HasError())
	resp := &resource.DeleteResponse{State: st}
	r.Delete(context.Background(), resource.DeleteRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

	var order []string
	for _, c := range p.calls {
		if strings.HasPrefix(c, "DELETE") {
			order = append(order, c[:strings.LastIndex(c, "/")])
		}
	}
	assert.Equal(t, []string{
		"DELETE /v1/xplat/database/backup/service/job/delete",
		"DELETE /v1/xplat/database/backup/service/job/delete",
		"DELETE /v1/xplat/database/backup/service/delete",
	}, order)
	assert.Equal(t, 0, p.backupEnabled, "the service is inactive")
	assert.Empty(t, p.jobs)
}

func TestBackupDeleteWaitsForARunningJob(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, planModel(10, jobAttrs("daily-full", 7, []int64{1}, 30, []int64{1}), nil))
	p.mu.Lock()
	p.jobs[0].runNow = 1
	p.mu.Unlock()

	// The run ends after a few polls.
	go func() {
		for {
			p.mu.Lock()
			n := len(p.calls)
			p.mu.Unlock()
			if n > 40 {
				p.mu.Lock()
				p.jobs[0].runNow = 0
				p.mu.Unlock()
				return
			}
		}
	}()
	st := emptyBackupState(s)
	require.False(t, setBackupModel(context.Background(), &st, created).HasError())
	resp := &resource.DeleteResponse{State: st}
	r.Delete(context.Background(), resource.DeleteRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Equal(t, 0, p.backupEnabled)
}

func TestBackupDeleteWhenAlreadyGone(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	st := emptyBackupState(s)
	gone := planModel(10, nil, nil)
	gone.Id, gone.BackupServiceId, gone.Status, gone.UsedGb, gone.RetentionType = types.StringValue("c1"), types.Int64Value(1), types.StringValue("x"), types.Float64Value(0), types.StringValue("quantity")
	require.False(t, setBackupModel(context.Background(), &st, gone).HasError())
	resp := &resource.DeleteResponse{State: st}
	r.Delete(context.Background(), resource.DeleteRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError())
	assert.Empty(t, p.callsLike("DELETE"))
}

func TestBackupModifyPlan(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())

	modify := func(prior, plan databaseBackupModel) resource.ModifyPlanResponse {
		st := emptyBackupState(s)
		if prior.ClusterId.ValueString() != "" {
			require.False(t, setBackupModel(context.Background(), &st, prior).HasError())
		}
		resp := resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: planFrom(t, s, withPriorJobIds(plan, created)), State: st}, &resp)
		return resp
	}
	same := func() databaseBackupModel {
		return planModel(10,
			jobAttrs("daily-full", 7, []int64{1}, 30, []int64{1, 2, 3, 4, 5, 6, 7}),
			jobAttrs("hourly-diff", 0, []int64{5, 9}, 0, []int64{1, 2, 3, 4, 5}))
	}

	assert.False(t, modify(created, same()).Diagnostics.HasError())

	renamed := same()
	renamed.FullJob = types.ObjectValueMust(backupJobObjectTypes(), jobAttrs("other-name", 7, []int64{1}, 30, []int64{1, 2, 3, 4, 5, 6, 7}))
	resp := modify(created, renamed)
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "Set it to `daily-full`")

	smaller := same()
	smaller.CapacityGb = types.Int64Value(9)
	created.CapacityGb = types.Int64Value(30)
	resp = modify(created, smaller)
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "Set it to 30 or more")
	created.CapacityGb = types.Int64Value(10)

	moved := same()
	moved.ClusterId = types.StringValue("c2")
	resp = modify(created, moved)
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "Set it to `c1`")

	moved = same()
	moved.VpcId = types.StringValue("vpc-2")
	resp = modify(created, moved)
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "vpc_id")

	// retention type day: no job change
	byDay := created
	byDay.RetentionType = types.StringValue("day")
	other := same()
	other.DiffJob = nullJobObject()
	resp = modify(byDay, other)
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "retention type")
	assert.False(t, modify(byDay, same()).Diagnostics.HasError(), "no change, no error")

	// an engine without diff jobs
	p.mu.Lock()
	p.typeDb = "kafka"
	p.mu.Unlock()
	resp = modify(created, same())
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "diff_job is not available")
}

func TestBackupSnapshotAndMerge(t *testing.T) {
	m := standardPlan()
	snap := backupSnapshot(context.Background(), &m)
	assert.Equal(t, map[string]string{
		"capacity_gb":           "10",
		"full_job.retention":    "7",
		"full_job.hours":        "1",
		"full_job.minute":       "30",
		"full_job.days_of_week": "1,2,3,4,5,6,7",
		"diff_job.hours":        "5,9",
		"diff_job.minute":       "0",
		"diff_job.days_of_week": "1,2,3,4,5",
	}, map[string]string(snap))

	// Drift warnings name the attribute of the job.
	now := standardPlan()
	now.FullJob = types.ObjectValueMust(backupJobObjectTypes(), jobAttrs("daily-full", 7, []int64{4}, 30, []int64{1, 2, 3, 4, 5, 6, 7}))
	var diags resource.ReadResponse
	addDriftWarnings(&diags.Diagnostics, snap, backupSnapshot(context.Background(), &now))
	require.Len(t, diags.Diagnostics.Warnings(), 1)
	assert.Contains(t, diags.Diagnostics.Warnings()[0].Detail(), "`full_job.hours` was changed outside of Terraform (applied: 1, on the portal now: 4)")
}

func TestBackupValidateConfig(t *testing.T) {
	r, s := backupResource(nil)
	validate := func(m databaseBackupModel) (errs []string) {
		cfg := tfsdk.Config{Schema: s.Schema, Raw: planFrom(t, s, m).Raw}
		resp := resource.ValidateConfigResponse{}
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, &resp)
		for _, e := range resp.Diagnostics.Errors() {
			errs = append(errs, e.Summary()+" "+e.Detail())
		}
		return errs
	}
	assert.Empty(t, validate(standardPlan()))
	assert.NotEmpty(t, validate(planModel(10, nil, jobAttrs("hourly-diff", 0, []int64{5}, 0, []int64{1}))))

	bad := standardPlan()
	bad.Timeouts = types.ObjectValueMust(timeoutsObjectTypes(), map[string]attr.Value{
		"create": types.StringValue("soon"), "update": types.StringNull(), "delete": types.StringNull(),
	})
	errs := validate(bad)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "Invalid timeout")

	assert.Equal(t, "40m0s", timeoutFrom(types.ObjectValueMust(timeoutsObjectTypes(), map[string]attr.Value{
		"create": types.StringValue("40m"), "update": types.StringNull(), "delete": types.StringNull(),
	}), "create", 0).String())
	assert.Equal(t, backupUpdateTimeout, timeoutFrom(types.ObjectNull(timeoutsObjectTypes()), "update", backupUpdateTimeout))
}

func TestBackupReadKeepsJobsWhileTheServiceIsChanging(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())
	p.mu.Lock()
	p.backupEnabled = backupStatusUpdating // job/list is empty now
	p.mu.Unlock()

	st := emptyBackupState(s)
	require.False(t, setBackupModel(context.Background(), &st, created).HasError())
	resp := &resource.ReadResponse{State: st}
	r.Read(context.Background(), resource.ReadRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Empty(t, resp.Diagnostics.Warnings(), "the jobs are not reported as deleted")
	var got databaseBackupModel
	require.False(t, getBackupModel(context.Background(), resp.State, &got).HasError())
	assert.Equal(t, created.FullJob, got.FullJob)
	assert.Equal(t, created.DiffJob, got.DiffJob)
	assert.Equal(t, "updating", got.Status.ValueString())
}

func TestBackupDeleteWaitsForTheServiceToBeReady(t *testing.T) {
	p := newFakePortal()
	r, s := backupResource(p.start(t))
	_, created := createBackup(t, r, s, standardPlan())
	p.mu.Lock()
	p.backupEnabled, p.backupTicks = backupStatusUpdating, 2 // becomes enabled after two reads
	p.backupTarget.status, p.backupTarget.capacity = backupStatusEnabled, 10
	p.mu.Unlock()

	st := emptyBackupState(s)
	require.False(t, setBackupModel(context.Background(), &st, created).HasError())
	resp := &resource.DeleteResponse{State: st}
	r.Delete(context.Background(), resource.DeleteRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Len(t, p.callsLike("job/delete"), 2, "both jobs are deleted, not skipped")
	assert.Equal(t, 0, p.backupEnabled)
}
