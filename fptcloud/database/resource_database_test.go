package fptcloud_database

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDatabaseModel() databaseResourceModel {
	return databaseResourceModel{
		Id:                   types.StringValue("cluster-1"),
		VpcId:                types.StringValue("vpc-1"),
		NetworkId:            types.StringValue("net-1"),
		VmNetwork:            types.StringValue("vmnet"),
		TypeConfig:           types.StringValue("short-config"),
		TypeDb:               types.StringValue("psql_patroni"),
		Version:              types.StringValue("17"),
		VdcName:              types.StringValue("my-vpc"),
		IsCluster:            types.StringValue("yes"),
		MasterCount:          types.Int64Value(1),
		WorkerCount:          types.Int64Value(2),
		NodeCpu:              types.Int64Value(2),
		NodeCore:             types.Int64Value(0),
		NodeRam:              types.Int64Value(4),
		DataDiskSize:         types.Int64Value(10),
		ClusterName:          types.StringValue("db"),
		DatabaseName:         types.StringValue("app"),
		VhostName:            types.StringValue("VHostDefault"),
		IsPublic:             types.StringValue("no"),
		AdminPassword:        types.StringValue("secret"),
		StorageProfile:       types.StringValue("SSD"),
		EdgeId:               types.StringValue("edge-1"),
		Edition:              types.StringValue("Community"),
		FlavorId:             types.StringValue("flavor-1"),
		IsOps:                types.StringValue("no"),
		Flavor:               types.StringValue("flavor"),
		NumberOfNode:         types.Int64Value(2),
		NumberOfShard:        types.Int64Value(2),
		DomainName:           types.StringValue(""),
		MaintenanceEmail:     types.StringValue("a@b.c"),
		DayOfWeekMaintenance: types.Int64Value(1),
		TimeMaintenance:      types.StringValue("01:00"),
		TagIds:               types.StringNull(),
		Nodes:                emptyNodesList(),
	}
}

func runModifyPlan(t *testing.T, plan, state databaseResourceModel) diag.Diagnostics {
	t.Helper()
	ctx := context.Background()
	r := &resourceDatabase{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	p := tfsdk.Plan{Schema: schemaResp.Schema}
	s := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, p.Set(ctx, &plan).HasError())
	require.False(t, s.Set(ctx, &state).HasError())

	var resp resource.ModifyPlanResponse
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: p, State: tfsdk.State{Schema: s.Schema, Raw: s.Raw}}, &resp)
	return resp.Diagnostics
}

func TestModifyPlanRejectsImmutableChanges(t *testing.T) {
	changes := map[string]func(m *databaseResourceModel){
		"vpc_id":          func(m *databaseResourceModel) { m.VpcId = types.StringValue("other") },
		"network_id":      func(m *databaseResourceModel) { m.NetworkId = types.StringValue("other") },
		"vm_network":      func(m *databaseResourceModel) { m.VmNetwork = types.StringValue("other") },
		"type_config":     func(m *databaseResourceModel) { m.TypeConfig = types.StringValue("other") },
		"type_db":         func(m *databaseResourceModel) { m.TypeDb = types.StringValue("other") },
		"version":         func(m *databaseResourceModel) { m.Version = types.StringValue("18") },
		"vdc_name":        func(m *databaseResourceModel) { m.VdcName = types.StringValue("other") },
		"is_cluster":      func(m *databaseResourceModel) { m.IsCluster = types.StringValue("no") },
		"master_count":    func(m *databaseResourceModel) { m.MasterCount = types.Int64Value(3) },
		"worker_count":    func(m *databaseResourceModel) { m.WorkerCount = types.Int64Value(5) },
		"node_core":       func(m *databaseResourceModel) { m.NodeCore = types.Int64Value(1) },
		"cluster_name":    func(m *databaseResourceModel) { m.ClusterName = types.StringValue("other") },
		"database_name":   func(m *databaseResourceModel) { m.DatabaseName = types.StringValue("other") },
		"vhost_name":      func(m *databaseResourceModel) { m.VhostName = types.StringValue("other") },
		"is_public":       func(m *databaseResourceModel) { m.IsPublic = types.StringValue("yes") },
		"edge_id":         func(m *databaseResourceModel) { m.EdgeId = types.StringValue("other") },
		"edition":         func(m *databaseResourceModel) { m.Edition = types.StringValue("Enterprise") },
		"is_ops":          func(m *databaseResourceModel) { m.IsOps = types.StringValue("yes") },
		"number_of_node":  func(m *databaseResourceModel) { m.NumberOfNode = types.Int64Value(9) },
		"number_of_shard": func(m *databaseResourceModel) { m.NumberOfShard = types.Int64Value(9) },
		"domain_name":     func(m *databaseResourceModel) { m.DomainName = types.StringValue("example.com") },
	}
	assert.Len(t, changes, 21)

	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			plan, state := testDatabaseModel(), testDatabaseModel()
			change(&plan)
			diags := runModifyPlan(t, plan, state)
			require.True(t, diags.HasError())
			assert.Contains(t, diags.Errors()[0].Detail(), fmt.Sprintf("`%s` cannot be changed after creation", name))
			assert.Contains(t, diags.Errors()[0].Detail(), "Set it to `")
		})
	}
}

func TestModifyPlanErrorShowsServerValue(t *testing.T) {
	plan, state := testDatabaseModel(), testDatabaseModel()
	state.WorkerCount = types.Int64Value(4) // the portal has 4 workers
	plan.WorkerCount = types.Int64Value(2)
	diags := runModifyPlan(t, plan, state)
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "Set it to `4`")
}

func TestModifyPlanAllowsInPlaceChanges(t *testing.T) {
	plan, state := testDatabaseModel(), testDatabaseModel()
	plan.NodeCpu = types.Int64Value(4)
	plan.NodeRam = types.Int64Value(8)
	plan.DataDiskSize = types.Int64Value(20)
	plan.FlavorId = types.StringValue("flavor-2")
	plan.TagIds = types.StringValue("tag-1")
	assert.False(t, runModifyPlan(t, plan, state).HasError())

	plan = testDatabaseModel()
	plan.StorageProfile = types.StringValue("HDD") // storage policy lookup is skipped without a client
	assert.False(t, runModifyPlan(t, plan, state).HasError())
}

func TestModifyPlanStillRejectsMixedChange(t *testing.T) {
	plan, state := testDatabaseModel(), testDatabaseModel()
	plan.NodeCpu = types.Int64Value(4)
	plan.StorageProfile = types.StringValue("HDD")
	diags := runModifyPlan(t, plan, state)
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), mixedChangeMessage)
}

func TestModifyPlanIgnoresValuesNotKnownYet(t *testing.T) {
	plan, state := testDatabaseModel(), testDatabaseModel()
	plan.VpcId = types.StringUnknown() // comes from another resource
	state.DomainName = types.StringNull()
	plan.DomainName = types.StringValue("")
	assert.False(t, runModifyPlan(t, plan, state).HasError())
}

func TestModifyPlanEditionAndVdcNameIgnoreCase(t *testing.T) {
	plan, state := testDatabaseModel(), testDatabaseModel()
	state.Edition = types.StringValue("community")
	state.VdcName = types.StringValue("MY-VPC")
	assert.False(t, runModifyPlan(t, plan, state).HasError())
}

func TestModifyPlanRefusesWhatTerraformCannotChange(t *testing.T) {
	changes := map[string]func(m *databaseResourceModel){
		"maintenance_email":       func(m *databaseResourceModel) { m.MaintenanceEmail = types.StringValue("new@example.com") },
		"day_of_week_maintenance": func(m *databaseResourceModel) { m.DayOfWeekMaintenance = types.Int64Value(5) },
		"time_maintenance":        func(m *databaseResourceModel) { m.TimeMaintenance = types.StringValue("23:30") },
		"admin_password":          func(m *databaseResourceModel) { m.AdminPassword = types.StringValue("another-secret") },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			plan, state := testDatabaseModel(), testDatabaseModel()
			change(&plan)
			diags := runModifyPlan(t, plan, state)
			require.True(t, diags.HasError())
			require.Len(t, diags.Errors(), 1)
			assert.Equal(t, "Cannot change "+name, diags.Errors()[0].Summary())
			assert.Contains(t, diags.Errors()[0].Detail(), "`"+name+"` cannot be changed by Terraform after the cluster is created")
			assert.Contains(t, diags.Errors()[0].Detail(), "apply -replace")
		})
	}

	// The message shows the recorded value, but never the password.
	plan, state := testDatabaseModel(), testDatabaseModel()
	plan.MaintenanceEmail = types.StringValue("new@example.com")
	diags := runModifyPlan(t, plan, state)
	assert.Contains(t, diags.Errors()[0].Detail(), "set it back to `a@b.c`")
	plan, state = testDatabaseModel(), testDatabaseModel()
	plan.AdminPassword = types.StringValue("another-secret")
	diags = runModifyPlan(t, plan, state)
	assert.NotContains(t, diags.Errors()[0].Detail(), "secret")

	// After an import the password in the state is the masked one: no error, the first apply records the configured one.
	plan, state = testDatabaseModel(), testDatabaseModel()
	state.AdminPassword = types.StringValue("********")
	assert.False(t, runModifyPlan(t, plan, state).HasError())

	// An imported cluster has no recorded value: nothing to compare with, so the first apply records the configured one.
	plan, state = testDatabaseModel(), testDatabaseModel()
	state.MaintenanceEmail, state.TimeMaintenance, state.DayOfWeekMaintenance = types.StringNull(), types.StringNull(), types.Int64Null()
	assert.False(t, runModifyPlan(t, plan, state).HasError())
}

func TestKeepIfSameText(t *testing.T) {
	assert.Equal(t, "Community", keepIfSameText(types.StringValue("Community"), "community").ValueString())
	assert.Equal(t, "Community", keepIfSameText(types.StringValue("Community"), "").ValueString())
	assert.Equal(t, "Enterprise", keepIfSameText(types.StringValue("Community"), "Enterprise").ValueString())
	assert.Equal(t, "Community", keepIfSameText(types.StringNull(), "Community").ValueString())
}

func detailResponse() string {
	return `{"code":"200","message":"ok","data":{"cluster":{"vpc_id":"vpc-1","network_id":"net-1","vm_network":"vmnet",
	"cluster_id":"cluster-1","cluster_name":"db","version":"17","type_config":"short-config","type_db":"psql_patroni",
	"master_count":1,"worker_count":2,"is_cluster":"yes","node_cpu":2,"node_core":0,"node_ram":4,"data_disk_size":10,
	"status":"running","database_name":"app","storage_profile":"SSD","edge_id":"edge-1","engine_edition":"community"},
	"nodes":{"total":1,"items":[{"id":"n1","vdc_name":"MY-VPC"}]}}}`
}

func TestInternalReadKeepsValuesTheServerDoesNotReturn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(detailResponse()))
	}))
	defer server.Close()
	client, err := common.NewClientWithURL("key", server.URL, "VN/HAN", "tenant", 10)
	require.NoError(t, err)
	r := &resourceDatabase{client: client, dataBaseClient: newDatabaseApiClient(client)}

	state := testDatabaseModel()
	state.DomainName = types.StringValue("db.example.com")
	require.NoError(t, r.internalRead(context.Background(), "cluster-1", &state))

	assert.Equal(t, "db.example.com", state.DomainName.ValueString(), "domain_name is not returned by the server")
	assert.Equal(t, "Community", state.Edition.ValueString(), "same text in another case keeps the configured value")
	assert.Equal(t, "my-vpc", state.VdcName.ValueString())
	assert.Equal(t, "no", state.IsPublic.ValueString())
	assert.Equal(t, int64(2), state.NumberOfNode.ValueInt64())

	// The planned config matches the refreshed state, so there is nothing to reject.
	assert.False(t, runModifyPlan(t, testDatabaseModel(), func() databaseResourceModel {
		s := state
		s.DomainName = types.StringValue("")
		return s
	}()).HasError())
}

type fakePrivate map[string][]byte

func (f fakePrivate) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return f[key], nil
}

func (f fakePrivate) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	f[key] = value
	return nil
}

func TestDriftSnapshotRoundTripAndWarnings(t *testing.T) {
	ctx := context.Background()
	private := fakePrivate{}

	// Nothing saved yet (a resource created before drift detection): no warnings.
	loaded, diags := loadDriftSnapshot(ctx, private)
	require.False(t, diags.HasError())
	assert.Nil(t, loaded)
	var none diag.Diagnostics
	addDriftWarnings(&none, loaded, databaseSnapshot(&databaseResourceModel{}))
	assert.Empty(t, none)

	applied := testDatabaseModel()
	require.False(t, saveDriftSnapshot(ctx, private, databaseSnapshot(&applied)).HasError())
	loaded, _ = loadDriftSnapshot(ctx, private)

	// Same values: no warning.
	var same diag.Diagnostics
	addDriftWarnings(&same, loaded, databaseSnapshot(&applied))
	assert.Empty(t, same)

	// Portal changed two attributes: one warning each, in a stable order.
	onPortal := testDatabaseModel()
	onPortal.DataDiskSize = types.Int64Value(30)
	onPortal.NodeCpu = types.Int64Value(8)
	var warnings diag.Diagnostics
	addDriftWarnings(&warnings, loaded, databaseSnapshot(&onPortal))
	require.Len(t, warnings, 2)
	assert.Equal(t, diag.SeverityWarning, warnings[0].Severity())
	assert.Equal(t,
		"`data_disk_size` was changed outside of Terraform (applied: 10, on the portal now: 30). The next apply will set it back to 10.",
		warnings[0].Detail())
	assert.Contains(t, warnings[1].Detail(), "`node_cpu` was changed outside of Terraform (applied: 2, on the portal now: 8)")
}

// clusterSequence answers the cluster detail with the given statuses in turn (the last one repeats).
func clusterSequence(statuses ...[2]interface{}) http.Handler {
	var n int32
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(atomic.AddInt32(&n, 1)) - 1
		if i >= len(statuses) {
			i = len(statuses) - 1
		}
		_, _ = fmt.Fprintf(w, `{"code":"200","data":{"cluster":{"cluster_id":"c1","status":%q,"node_cpu":%d,"node_ram":8,"data_disk_size":10},"nodes":{"total":1,"items":[]}}}`,
			statuses[i][0], statuses[i][1])
	})
}

func TestWaitForClusterAfterResize(t *testing.T) {
	clusterPollInterval = time.Millisecond
	t.Cleanup(func() { clusterPollInterval = 10 * time.Second })
	newCpu := func(c databaseData) bool { return c.NodeCpu == 4 }
	wait := func(h http.Handler) error {
		api := newTestApi(t, h)
		r := &resourceDatabase{dataBaseClient: api}
		return r.waitForCluster(context.Background(), "c1", newCpu)
	}
	s := func(status string, cpu int) [2]interface{} { return [2]interface{}{status, cpu} }

	// resizing -> running with the new values: done (a still-running cluster with old values is not enough).
	assert.NoError(t, wait(clusterSequence(s("running", 2), s("resizing", 2), s("running", 4))))
	// warning with the new values counts as done as well
	assert.NoError(t, wait(clusterSequence(s("resizing", 2), s("warning", 4))))
	// resizing -> running without the new values, twice: the resize failed
	err := wait(clusterSequence(s("resizing", 2), s("running", 2), s("running", 2)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the portal did not apply it")
	// same with warning
	err = wait(clusterSequence(s("resizing", 2), s("warning", 2), s("warning", 2)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is warning again")
	// one late answer is not enough to fail: the next read shows the values
	assert.NoError(t, wait(clusterSequence(s("resizing", 2), s("running", 2), s("running", 4))))
	// the cluster that never left running with the old values is still waited for, until the change appears
	assert.NoError(t, wait(clusterSequence(s("running", 2), s("running", 2), s("running", 2), s("resizing", 2), s("running", 4))))
	// failed stops at once
	assert.ErrorContains(t, wait(clusterSequence(s("resizing", 2), s("failed", 2))), "entered failed status")
}
