package fptcloud_database

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runNowFixture(t *testing.T) (*fakePortal, *resourceDatabaseBackupRunNow, resource.SchemaResponse, int64) {
	p := newFakePortal()
	api := p.start(t)
	br, bs := backupResource(api)
	_, created := createBackup(t, br, bs, planModel(10, jobAttrs("daily-full", 7, []int64{1}, 30, []int64{1}), nil))
	r := &resourceDatabaseBackupRunNow{api: api}
	var s resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &s)
	p.mu.Lock()
	p.calls = nil
	p.mu.Unlock()
	return p, r, s, created.FullJob.Attributes()["job_id"].(types.Int64).ValueInt64()
}

func runNowModel(jobId int64, triggers map[string]string) databaseBackupRunNowModel {
	elems := map[string]attr.Value{}
	for k, v := range triggers {
		elems[k] = types.StringValue(v)
	}
	return databaseBackupRunNowModel{
		Id: types.StringUnknown(), ClusterId: types.StringValue("c1"), JobId: types.Int64Value(jobId),
		Triggers: types.MapValueMust(types.StringType, elems), JobName: types.StringUnknown(), TriggeredAt: types.StringUnknown(),
	}
}

func rawTf(s resource.SchemaResponse) tftypes.Value {
	return tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)
}

func TestRunNowCreateAndRetrigger(t *testing.T) {
	p, r, s, jobId := runNowFixture(t)

	plan := tfsdk.Plan{Schema: s.Schema, Raw: rawTf(s)}
	require.False(t, plan.Set(context.Background(), runNowModel(jobId, map[string]string{"run": "1"})).HasError())
	created := &resource.CreateResponse{State: tfsdk.State{Schema: s.Schema, Raw: rawTf(s)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, created)
	require.False(t, created.Diagnostics.HasError(), created.Diagnostics)
	var state databaseBackupRunNowModel
	require.False(t, created.State.Get(context.Background(), &state).HasError())
	assert.Equal(t, "daily-full", state.JobName.ValueString())
	assert.Equal(t, "c1/"+itoa(int(jobId)), state.Id.ValueString())
	assert.NotEmpty(t, state.TriggeredAt.ValueString())
	assert.Len(t, p.callsLike("job/runnow"), 1)
	assert.Empty(t, p.callsLike("job/create"), "it does not wait for the run, and creates nothing")

	// While the job runs it cannot be started again.
	again := &resource.CreateResponse{State: tfsdk.State{Schema: s.Schema, Raw: rawTf(s)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, again)
	require.True(t, again.Diagnostics.HasError())
	assert.Contains(t, again.Diagnostics.Errors()[0].Detail(), "already running")
	assert.Len(t, p.callsLike("job/runnow"), 1)

	// The run ends. Same triggers: nothing happens. New triggers: it runs again.
	p.mu.Lock()
	p.jobs[0].runNow = 0
	p.mu.Unlock()

	update := func(triggers map[string]string, jobId int64) *resource.UpdateResponse {
		newPlan := tfsdk.Plan{Schema: s.Schema, Raw: rawTf(s)}
		m := runNowModel(jobId, triggers)
		m.Id, m.JobName, m.TriggeredAt = state.Id, state.JobName, state.TriggeredAt
		require.False(t, newPlan.Set(context.Background(), m).HasError())
		resp := &resource.UpdateResponse{State: created.State}
		r.Update(context.Background(), resource.UpdateRequest{Plan: newPlan, State: created.State}, resp)
		return resp
	}
	resp := update(map[string]string{"run": "1"}, jobId+1000)
	require.False(t, resp.Diagnostics.HasError())
	require.Len(t, resp.Diagnostics.Warnings(), 1, "a changed argument only warns")
	assert.Len(t, p.callsLike("job/runnow"), 1)

	resp = update(map[string]string{"run": "2"}, jobId)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Len(t, p.callsLike("job/runnow"), 2, "a new trigger runs the job again")

	// Destroy calls no API.
	p.mu.Lock()
	p.calls = nil
	p.mu.Unlock()
	del := &resource.DeleteResponse{State: created.State}
	r.Delete(context.Background(), resource.DeleteRequest{State: created.State}, del)
	assert.False(t, del.Diagnostics.HasError())
	assert.Empty(t, p.calls)
}

func TestRunNowRefusals(t *testing.T) {
	p, r, s, jobId := runNowFixture(t)
	create := func(id int64) *resource.CreateResponse {
		plan := tfsdk.Plan{Schema: s.Schema, Raw: rawTf(s)}
		require.False(t, plan.Set(context.Background(), runNowModel(id, nil)).HasError())
		resp := &resource.CreateResponse{State: tfsdk.State{Schema: s.Schema, Raw: rawTf(s)}}
		r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
		return resp
	}

	assert.Contains(t, create(99999).Diagnostics.Errors()[0].Detail(), "does not exist")

	p.mu.Lock()
	p.jobs[0].status = "updating"
	p.mu.Unlock()
	assert.Contains(t, create(jobId).Diagnostics.Errors()[0].Detail(), "can only run when it is ready")
	assert.Empty(t, p.callsLike("job/runnow"))
}
