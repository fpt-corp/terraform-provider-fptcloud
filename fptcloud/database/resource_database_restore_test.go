package fptcloud_database

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func restoreResource(api *databaseApiClient) (*resourceDatabaseRestore, resource.SchemaResponse) {
	r := &resourceDatabaseRestore{api: api}
	var s resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &s)
	return r, s
}

func restorePlan() databaseRestoreModel {
	return databaseRestoreModel{
		Id: types.StringUnknown(), ClusterId: types.StringValue("src"), RestoreName: types.StringValue("restored-db"),
		BackupPart: types.StringValue("20261023-020002F"), BackupType: types.StringValue("full"),
		FlavorId: types.StringUnknown(), NodeCpu: types.Int64Unknown(), NodeRam: types.Int64Unknown(), StorageProfile: types.StringUnknown(),
		DeleteClusterOnDestroy: types.BoolValue(false), RestoredClusterId: types.StringUnknown(), RestoredStatus: types.StringUnknown(),
		Timeouts: types.ObjectNull(restoreTimeoutsObjectTypes()),
	}
}

func createRestore(t *testing.T, r *resourceDatabaseRestore, s resource.SchemaResponse, m databaseRestoreModel) (*resource.CreateResponse, databaseRestoreModel) {
	t.Helper()
	plan := tfsdk.Plan{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)}
	require.False(t, plan.Set(context.Background(), m).HasError())
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	var got databaseRestoreModel
	if !resp.State.Raw.IsNull() {
		require.False(t, resp.State.Get(context.Background(), &got).HasError())
	}
	return resp, got
}

func TestRestoreCreate(t *testing.T) {
	p := newFakePortal()
	r, s := restoreResource(p.start(t))

	resp, got := createRestore(t, r, s, restorePlan())
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

	// restore/create does not return the cluster: it is found by name.
	assert.Equal(t, "restored2", got.RestoredClusterId.ValueString())
	assert.Equal(t, "restored2", got.Id.ValueString())
	assert.Equal(t, "running", got.RestoredStatus.ValueString())

	// The source's compute and storage are used, and the flavor name is sent: an empty one makes the restore fail.
	body := p.lastBody("POST /v1/xplat/database/restore/create")
	assert.Contains(t, body, `"flavor":"4C8G"`)
	assert.Contains(t, body, `"flavor_id":"f-4c8g"`)
	assert.Contains(t, body, `"node_cpu":4`)
	assert.Contains(t, body, `"node_ram":8`)
	assert.Contains(t, body, `"restore_storage_profile":"SSD"`)
	assert.Contains(t, body, `"need_restore":"yes"`)
	assert.Contains(t, body, `"backup_part":"20261023-020002F"`)
	assert.Equal(t, "f-4c8g", got.FlavorId.ValueString())
	assert.Equal(t, "SSD", got.StorageProfile.ValueString())
}

func TestRestoreCreateWithCpuAndRam(t *testing.T) {
	p := newFakePortal()
	r, s := restoreResource(p.start(t))
	plan := restorePlan()
	plan.NodeCpu, plan.NodeRam = types.Int64Value(2), types.Int64Value(4)
	plan.StorageProfile = types.StringValue("HDD")

	resp, got := createRestore(t, r, s, plan)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	body := p.lastBody("POST /v1/xplat/database/restore/create")
	assert.Contains(t, body, `"flavor":"2C4G"`)
	assert.Contains(t, body, `"flavor_id":"f-2c4g"`)
	assert.Contains(t, body, `"restore_storage_profile":"HDD"`)
	assert.Equal(t, "f-2c4g", got.FlavorId.ValueString())
}

func TestRestoreCreateOpenSearchHA(t *testing.T) {
	p := newFakePortal()
	p.typeDb, p.isCluster = "opensearch", "yes"
	r, s := restoreResource(p.start(t))
	plan := restorePlan()
	plan.NodeCpu, plan.NodeRam = types.Int64Value(2), types.Int64Value(4) // ignored: the compute is the source's

	resp, _ := createRestore(t, r, s, plan)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	body := p.lastBody("POST /v1/xplat/database/restore/create")
	assert.Contains(t, body, `"flavor":""`)
	assert.Contains(t, body, `"flavor_id":"f-4c8g"`)
	assert.Contains(t, body, `"node_cpu":4`)
	assert.Empty(t, p.callsLike("get_list_flavor"), "no flavor lookup for OpenSearch HA")
}

func TestRestoreCreateRefusals(t *testing.T) {
	tests := []struct {
		name  string
		setup func(p *fakePortal, m *databaseRestoreModel)
		want  string
	}{
		{"source not running", func(p *fakePortal, m *databaseRestoreModel) { p.clusterStatus = "stopped" }, "needs it to be running"},
		{"mongodb sharding", func(p *fakePortal, m *databaseRestoreModel) { p.typeDb = "mongodb_sharding" }, "not supported by Terraform yet"},
		{"unknown restore point", func(p *fakePortal, m *databaseRestoreModel) { m.BackupPart = types.StringValue("nope") }, "has no restore point"},
		{"restore point not successful", func(p *fakePortal, m *databaseRestoreModel) { m.BackupPart = types.StringValue("20261022-020002F") }, `is "failed"`},
		{"name used in the VPC", func(p *fakePortal, m *databaseRestoreModel) {
			p.clusters = []map[string]string{{"cluster_name": "Restored-DB", "status": "running"}}
		}, "already exists in the VPC"},
		{"unknown flavor", func(p *fakePortal, m *databaseRestoreModel) {
			m.FlavorId = types.StringValue("f-nope")
		}, "is not available in this VPC"},
		{"no flavor with this size", func(p *fakePortal, m *databaseRestoreModel) {
			m.NodeCpu, m.NodeRam = types.Int64Value(64), types.Int64Value(256)
		}, "no flavor with 64 CPU"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newFakePortal()
			plan := restorePlan()
			tt.setup(p, &plan)
			r, s := restoreResource(p.start(t))
			resp, _ := createRestore(t, r, s, plan)
			require.True(t, resp.Diagnostics.HasError())
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tt.want)
			assert.Empty(t, p.callsLike("restore/create"))
			assert.True(t, resp.State.Raw.IsNull())
		})
	}

	// A deleted cluster with the same name does not block the restore.
	p := newFakePortal()
	p.clusters = []map[string]string{{"cluster_name": "restored-db", "status": "deleted"}}
	r, s := restoreResource(p.start(t))
	resp, _ := createRestore(t, r, s, restorePlan())
	assert.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
}

func TestRestoreFailedKeepsTheClusterId(t *testing.T) {
	p := newFakePortal()
	api := p.start(t)
	r, s := restoreResource(api)
	// The new cluster ends in failed.
	p.mu.Lock()
	p.restorePoints = p.restorePoints[:1]
	p.restoreEndsIn = "failed"
	p.mu.Unlock()
	resp, got := createRestore(t, r, s, restorePlan())
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "failed")
	assert.Equal(t, "restored2", got.RestoredClusterId.ValueString(), "the state keeps the id, so the cluster is not lost")
}

func stateFrom(t *testing.T, s resource.SchemaResponse, m databaseRestoreModel) tfsdk.State {
	t.Helper()
	st := tfsdk.State{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)}
	require.False(t, st.Set(context.Background(), m).HasError())
	return st
}

func TestRestoreReadNeverRemovesTheId(t *testing.T) {
	p := newFakePortal()
	r, s := restoreResource(p.start(t))
	_, created := createRestore(t, r, s, restorePlan())

	// The restored cluster is stopped: the status is refreshed.
	p.mu.Lock()
	p.clusterStatus = "stopped"
	p.mu.Unlock()
	st := stateFrom(t, s, created)
	resp := &resource.ReadResponse{State: st}
	r.Read(context.Background(), resource.ReadRequest{State: st}, resp)
	var got databaseRestoreModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, "stopped", got.RestoredStatus.ValueString())
	assert.Equal(t, "restored2", got.RestoredClusterId.ValueString())

	// The API fails (or the source or the cluster is gone): a warning, and the id stays.
	broken := &resourceDatabaseRestore{api: newTestApi(t, serveSequence(`{"code":"404"}`))}
	st = stateFrom(t, s, created)
	resp = &resource.ReadResponse{State: st}
	broken.Read(context.Background(), resource.ReadRequest{State: st}, resp)
	require.False(t, resp.State.Raw.IsNull(), "the resource is not removed")
	assert.NotEmpty(t, resp.Diagnostics.Warnings())
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, "restored2", got.RestoredClusterId.ValueString())
}

func TestRestoreUpdateOnlyWarns(t *testing.T) {
	p := newFakePortal()
	r, s := restoreResource(p.start(t))
	_, created := createRestore(t, r, s, restorePlan())
	p.mu.Lock()
	p.calls = nil
	p.mu.Unlock()

	plan := created
	plan.RestoreName = types.StringValue("another-name")
	plan.BackupPart = types.StringValue("other-part")
	planV := tfsdk.Plan{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)}
	require.False(t, planV.Set(context.Background(), plan).HasError())
	st := stateFrom(t, s, created)
	resp := &resource.UpdateResponse{State: st}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planV, State: st}, resp)

	require.False(t, resp.Diagnostics.HasError())
	require.Len(t, resp.Diagnostics.Warnings(), 1)
	assert.Contains(t, resp.Diagnostics.Warnings()[0].Detail(), "apply -replace")
	assert.Empty(t, p.calls, "no API call: the restore is not run again")
	var got databaseRestoreModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, "another-name", got.RestoreName.ValueString())
	assert.Equal(t, "restored2", got.RestoredClusterId.ValueString())

	// Turning on delete_cluster_on_destroy is a setting, not a change of the restore: no warning.
	plan = created
	plan.DeleteClusterOnDestroy = types.BoolValue(true)
	require.False(t, planV.Set(context.Background(), plan).HasError())
	st = stateFrom(t, s, created)
	resp = &resource.UpdateResponse{State: st}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planV, State: st}, resp)
	assert.Empty(t, resp.Diagnostics.Warnings())
}

func TestRestoreDelete(t *testing.T) {
	p := newFakePortal()
	r, s := restoreResource(p.start(t))
	_, created := createRestore(t, r, s, restorePlan())
	p.mu.Lock()
	p.calls = nil
	p.mu.Unlock()

	// By default the restored cluster is kept, with a warning.
	st := stateFrom(t, s, created)
	resp := &resource.DeleteResponse{State: st}
	r.Delete(context.Background(), resource.DeleteRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError())
	require.Len(t, resp.Diagnostics.Warnings(), 1)
	assert.Contains(t, resp.Diagnostics.Warnings()[0].Detail(), "still exists")
	assert.Empty(t, p.callsLike("provision/delete"))

	// With delete_cluster_on_destroy the cluster is deleted.
	created.DeleteClusterOnDestroy = types.BoolValue(true)
	st = stateFrom(t, s, created)
	resp = &resource.DeleteResponse{State: st}
	p.mu.Lock()
	p.clusterStatus = "deleted" // what the cluster detail reports once it is deleted
	p.mu.Unlock()
	r.Delete(context.Background(), resource.DeleteRequest{State: st}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	assert.Equal(t, []string{"DELETE /v1/xplat/database/provision/delete/restored2"}, p.callsLike("provision/delete"))
}

func TestRestoreValidateConfig(t *testing.T) {
	r, s := restoreResource(nil)
	validate := func(m databaseRestoreModel) (errs []string) {
		plan := tfsdk.Plan{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)}
		require.False(t, plan.Set(context.Background(), m).HasError())
		cfg := tfsdk.Config{Schema: s.Schema, Raw: plan.Raw}
		resp := resource.ValidateConfigResponse{}
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, &resp)
		for _, e := range resp.Diagnostics.Errors() {
			errs = append(errs, e.Summary())
		}
		return errs
	}
	ok := restorePlan()
	ok.FlavorId, ok.NodeCpu, ok.NodeRam, ok.StorageProfile = types.StringNull(), types.Int64Null(), types.Int64Null(), types.StringNull()
	assert.Empty(t, validate(ok))

	for _, name := range []string{"1abc", "has_underscore", "has space", "-lead"} {
		m := ok
		m.RestoreName = types.StringValue(name)
		assert.Equal(t, []string{"Invalid restore_name"}, validate(m), name)
	}
	for _, name := range []string{"a", "Restore-1", "abc-DEF-9"} {
		m := ok
		m.RestoreName = types.StringValue(name)
		assert.Empty(t, validate(m), name)
	}

	m := ok
	m.NodeCpu = types.Int64Value(2)
	assert.Equal(t, []string{"node_cpu and node_ram go together"}, validate(m))
	m.NodeRam = types.Int64Value(4)
	m.FlavorId = types.StringValue("f-4c8g")
	assert.Equal(t, []string{"Use flavor_id or node_cpu and node_ram"}, validate(m))
}

func TestResolveRestoreFlavor(t *testing.T) {
	flavors := []restoreFlavor{{FlavorId: "a", Name: "2C4G", Vcpu: 2, Ram: 4096}, {FlavorId: "b", Name: "4C8G", Vcpu: 4, Ram: 8192}}
	f, err := resolveRestoreFlavor(flavors, "b", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, "4C8G", string(f.Name))
	f, err = resolveRestoreFlavor(flavors, "", 2, 4)
	require.NoError(t, err)
	assert.Equal(t, "a", f.FlavorId)
	_, err = resolveRestoreFlavor(flavors, "", 3, 4)
	assert.ErrorContains(t, err, "2C4G (2 CPU, 4 GB), 4C8G (4 CPU, 8 GB)")
	_, err = resolveRestoreFlavor(nil, "x", 0, 0)
	assert.ErrorContains(t, err, "flavor x is not available")
}
