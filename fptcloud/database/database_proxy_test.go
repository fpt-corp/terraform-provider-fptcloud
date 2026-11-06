package fptcloud_database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type attrValue = attr.Value

func nullRawFor(t attr.Type) tftypes.Value {
	return tftypes.NewValue(t.TerraformType(context.Background()), nil)
}

func init() {
	proxyPollInterval = time.Millisecond
	proxyRestartGrace = 5 * time.Millisecond
	proxyAppearTimeout = 2 * time.Second
}

// proxyView is what the portal reports for a proxy at one read.
type proxyView struct {
	status   string
	cpu, ram int
	flavor   string
	endPoint string
	nodes    int
}

// fakeProxy is a proxy of the fake portal. Each read of it (detail or list) moves along queue; the last view stays.
type fakeProxy struct {
	id, name string
	ha       bool
	vmSync   bool
	queue    []proxyView
	removeIn int // list reads left before a deleted proxy is gone; -1 when not deleting
	// failDeletes is how many deletes the portal takes and then puts the proxy back to "failed" for (as it does when the
	// proxy was busy).
	failDeletes int
}

func (p *fakeProxy) current() proxyView {
	v := p.queue[0]
	if len(p.queue) > 1 {
		p.queue = p.queue[1:]
	}
	return v
}

// fakeProxyPortal imitates the load-balancing endpoints with the behaviour seen on the real portal (P0): create returns no
// id, the proxy is running before it has an end point, the portal checks nothing but "same flavor" on a resize,
// an unknown id answers HTTP 400.
type fakeProxyPortal struct {
	mu            sync.Mutex
	clusterStatus string
	lbTypes       []string
	flavors       []map[string]interface{}
	proxies       []*fakeProxy
	nextId        int
	calls         []string
	bodies        map[string][]string
}

func newFakeProxyPortal() *fakeProxyPortal {
	return &fakeProxyPortal{
		clusterStatus: "running",
		lbTypes:       []string{"maxscale"},
		bodies:        map[string][]string{},
		nextId:        1,
		flavors: []map[string]interface{}{
			{"flavor_id": "f-2c4g", "flavor_name": "2C4G", "flavor_vcpu": 2, "flavor_ram": 4096},
			{"flavor_id": "f-4c8g", "flavor_name": "4C8G", "flavor_vcpu": 4, "flavor_ram": 8192},
		},
	}
}

func (f *fakeProxyPortal) start(t *testing.T) *databaseApiClient {
	return newTestApi(t, http.HandlerFunc(f.serve))
}

// addRunning adds a settled proxy.
func (f *fakeProxyPortal) addRunning(id, name string, cpu, ram int, flavor string) *fakeProxy {
	p := &fakeProxy{id: id, name: name, vmSync: true, removeIn: -1,
		queue: []proxyView{{"running", cpu, ram, flavor, "R-W ://10.0.0.10:4006/, \nR-O ://10.0.0.10:4008/", 1}}}
	f.proxies = append(f.proxies, p)
	return p
}

func (f *fakeProxyPortal) callsLike(sub string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		if strings.Contains(c, sub) {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeProxyPortal) lastBody(call string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	b := f.bodies[call]
	if len(b) == 0 {
		return ""
	}
	return b[len(b)-1]
}

func (f *fakeProxyPortal) find(id string) *fakeProxy {
	for _, p := range f.proxies {
		if p.id == id {
			return p
		}
	}
	return nil
}

func (f *fakeProxyPortal) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	raw, _ := io.ReadAll(r.Body)
	call := r.Method + " " + r.URL.Path
	f.calls = append(f.calls, call)
	f.bodies[call] = append(f.bodies[call], string(raw))
	path := r.URL.Path
	const lb = "/v1/xplat/database/load-balancing/"

	switch {
	case strings.HasSuffix(path, "/async") || strings.Contains(path, "/storages"):
		_, _ = w.Write([]byte(`{}`))
	case strings.HasPrefix(path, "/v1/xplat/database/management/cluster/detail/"):
		fmt.Fprintf(w, `{"code":"200","data":{"cluster":{"cluster_id":"c1","vpc_id":"v1","status":%q},"nodes":{"total":3,"items":[]}}}`, f.clusterStatus)
	case strings.HasPrefix(path, "/v1/xplat/database/configure_management/get_list_flavor_v2"):
		writeJSON(w, map[string]interface{}{"code": "200", "data": f.flavors})
	case path == lb+"get-lb-type":
		writeJSON(w, map[string]interface{}{"code": "200", "type": "success", "data": f.lbTypes})
	case strings.HasPrefix(path, lb+"list/"):
		f.serveList(w)
	case strings.HasPrefix(path, lb+"detail/"):
		p := f.find(strings.TrimPrefix(path, lb+"detail/"))
		if p == nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"400","message":"'NoneType' object has no attribute 'master_count'"}`))
			return
		}
		v := p.current()
		ep := "null"
		if v.endPoint != "" {
			b, _ := json.Marshal(v.endPoint)
			ep = string(b)
		}
		fmt.Fprintf(w, `{"code":"200","message":"Successful","data":{"cluster":{"db_cluster_id":"c1","lb_cluster_id":%q,"lb_cluster_name":%q,"is_lb_cluster":%v,"lb_type":"maxscale","node_cpu":%d,"node_ram":%d,"storage_profile":"SSD","end_point":%s,"status":%q,"flavor":%q,"created_at":"2026-11-05T13:40:33"},"nodes":{"total":%d,"items":[]}},"type":"success"}`,
			p.id, p.name, p.ha, v.cpu, v.ram, ep, v.status, v.flavor, v.nodes)
	case path == lb+"create":
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		cpu, ram := int(body["node_cpu"].(float64)), int(body["node_ram"].(float64))
		p := &fakeProxy{id: fmt.Sprintf("LB%d", f.nextId), name: body["lb_cluster_name"].(string), ha: body["is_lb_cluster"] == "yes", vmSync: true, removeIn: -1}
		f.nextId++
		ep := "R-W ://10.0.0.10:4006/, \nR-O ://10.0.0.10:4008/"
		p.queue = []proxyView{{"processing", cpu, ram, "", "", 0}, {"processing", cpu, ram, "", "", 0},
			{"running", cpu, ram, body["flavor"].(string), "", 0}, {"running", cpu, ram, body["flavor"].(string), ep, 1}}
		f.proxies = append(f.proxies, p)
		_, _ = w.Write([]byte(`{"code":"200","message":"Successful","data":[],"type":"success"}`))
	case path == lb+"restart":
		var body map[string]string
		_ = json.Unmarshal(raw, &body)
		p := f.find(body["lb_cluster_id"])
		if p == nil {
			_, _ = w.Write([]byte(`{"error_code":180003,"message":"Load balancer is invalid for this action.","type":"error"}`))
			return
		}
		last := p.queue[len(p.queue)-1]
		busy, done := last, last
		busy.status = "restarting"
		p.queue = []proxyView{busy, busy, done}
		_, _ = w.Write([]byte(`{"code":"200","message":"LB Cluster : restart successful","data":null,"type":"success"}`))
	case strings.HasSuffix(path, "/resize"):
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		p := f.find(body["lb_cluster_id"].(string))
		if p == nil {
			_, _ = w.Write([]byte(`{"error_code":180003,"message":"Load balancer is invalid for this action.","type":"error"}`))
			return
		}
		last := p.queue[len(p.queue)-1]
		if last.flavor == body["flavor"] {
			_, _ = w.Write([]byte(`{"error_code":180004,"message":"To resize this load balancer, please choose a different flavor.","type":"error"}`))
			return
		}
		busy := last
		busy.status = "resizing"
		done := last
		done.cpu, done.ram, done.flavor = int(body["node_cpu"].(float64)), int(body["node_ram"].(float64)), body["flavor"].(string)
		p.queue = []proxyView{busy, busy, done}
		_, _ = w.Write([]byte(`{"code":"200","message":"LB Cluster : is resizing","data":null,"type":"success"}`))
	case strings.HasPrefix(path, lb+"delete/"):
		p := f.find(strings.TrimPrefix(path, lb+"delete/"))
		if p == nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"400","message":"'NoneType' object has no attribute 'db_cluster_id'"}`))
			return
		}
		last := p.queue[len(p.queue)-1]
		last.status = "deleting"
		if p.failDeletes > 0 {
			p.failDeletes--
			failed := last
			failed.status = "failed"
			p.queue = []proxyView{last, failed}
			p.removeIn = -1
		} else {
			p.queue = []proxyView{last}
			p.removeIn = 2
		}
		_, _ = w.Write([]byte(`{"code":"200","message":"This LB is being deleted !","data":[],"type":"success"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeProxyPortal) serveList(w http.ResponseWriter) {
	var rows []map[string]interface{}
	for i := 0; i < len(f.proxies); i++ {
		p := f.proxies[i]
		if p.removeIn == 0 {
			f.proxies = append(f.proxies[:i], f.proxies[i+1:]...)
			i--
			continue
		}
		if p.removeIn > 0 {
			p.removeIn--
		}
		v := p.current()
		ha := "False"
		slaves := 0
		if p.ha {
			ha, slaves = "True", 1
		}
		rows = append(rows, map[string]interface{}{
			"lb_cluster_id": p.id, "lb_cluster_name": p.name, "is_lb_cluster": ha, "lb_type": "maxscale",
			"node_cpu": v.cpu, "node_ram": v.ram, "storage_profile": "SSD", "master_count": 1, "slave_count": slaves,
			"vm_sync": p.vmSync, "status": v.status, "db_cluster_id": "c1", "created_at": "2026-11-05 13:40:33",
		})
	}
	writeJSON(w, map[string]interface{}{"code": "200", "data": rows, "metadata": map[string]interface{}{"total_items": len(rows), "page_size": 50, "current_page": 1}})
}

func proxyResource(api *databaseApiClient) (*resourceDatabaseProxy, resource.SchemaResponse) {
	r := &resourceDatabaseProxy{api: api}
	var s resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &s)
	return r, s
}

func proxyPlan() databaseProxyModel {
	return databaseProxyModel{
		Id: types.StringUnknown(), DbClusterId: types.StringValue("c1"), VpcId: types.StringValue("v1"),
		Name: types.StringValue("proxy-test-01"), LbType: types.StringValue("maxscale"), StorageProfile: types.StringValue("SSD"),
		NodeCpu: types.Int64Value(2), NodeRam: types.Int64Value(4), HighAvailability: types.BoolValue(true),
		Flavor: types.StringUnknown(), FlavorId: types.StringUnknown(), Status: types.StringUnknown(), EndPoint: types.StringUnknown(),
		NodeCount: types.Int64Unknown(), CreatedAt: types.StringUnknown(), Timeouts: types.ObjectNull(timeoutsObjectTypes()),
	}
}

// proxyState is the state of a settled 2C4G proxy as a Read would leave it.
func proxyState() databaseProxyModel {
	m := proxyPlan()
	m.Id = types.StringValue("LB1")
	m.Flavor, m.FlavorId = types.StringValue("2C4G"), types.StringValue("f-2c4g")
	m.Status = types.StringValue("running")
	m.EndPoint = types.StringValue("R-W ://10.0.0.10:4006/, \nR-O ://10.0.0.10:4008/")
	m.NodeCount = types.Int64Value(2)
	m.CreatedAt = types.StringValue("2026-11-05T13:40:33")
	return m
}

func nullRaw(s resource.SchemaResponse) tftypes.Value {
	return tftypes.NewValue(s.Schema.Type().TerraformType(context.Background()), nil)
}

func stateOf(t *testing.T, s resource.SchemaResponse, m databaseProxyModel) tfsdk.State {
	t.Helper()
	st := tfsdk.State{Schema: s.Schema, Raw: nullRaw(s)}
	require.False(t, st.Set(context.Background(), m).HasError())
	return st
}

func planOf(t *testing.T, s resource.SchemaResponse, m databaseProxyModel) tfsdk.Plan {
	t.Helper()
	p := tfsdk.Plan{Schema: s.Schema, Raw: nullRaw(s)}
	require.False(t, p.Set(context.Background(), m).HasError())
	return p
}

func errorText(d diag.Diagnostics) string {
	var out []string
	for _, e := range d.Errors() {
		out = append(out, e.Summary()+": "+e.Detail())
	}
	return strings.Join(out, "\n")
}

func createProxyResource(t *testing.T, r *resourceDatabaseProxy, s resource.SchemaResponse, m databaseProxyModel) (*resource.CreateResponse, databaseProxyModel) {
	t.Helper()
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s.Schema, Raw: nullRaw(s)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: planOf(t, s, m)}, resp)
	var got databaseProxyModel
	if !resp.State.Raw.IsNull() {
		require.False(t, resp.State.Get(context.Background(), &got).HasError())
	}
	return resp, got
}

func TestProxyCreate(t *testing.T) {
	p := newFakeProxyPortal()
	r, s := proxyResource(p.start(t))

	resp, got := createProxyResource(t, r, s, proxyPlan())
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))

	// create returns no id: the proxy is found by name in the list.
	assert.Equal(t, "LB1", got.Id.ValueString())
	// running is not enough: the state is written when the end point is there.
	assert.Equal(t, "running", got.Status.ValueString())
	assert.Contains(t, got.EndPoint.ValueString(), "4006")
	assert.Equal(t, int64(1), got.NodeCount.ValueInt64())
	assert.True(t, got.HighAvailability.ValueBool())
	assert.Equal(t, "2C4G", got.Flavor.ValueString())
	assert.Equal(t, "f-2c4g", got.FlavorId.ValueString())

	// The flavor goes by name, there is no flavor_id, and high availability is "yes".
	body := p.lastBody("POST /v1/xplat/database/load-balancing/create")
	assert.Contains(t, body, `"flavor":"2C4G"`)
	assert.Contains(t, body, `"is_lb_cluster":"yes"`)
	assert.Contains(t, body, `"node_ram":4`)
	assert.NotContains(t, body, "flavor_id")
	// The VPC is synced while waiting.
	assert.NotEmpty(t, p.callsLike("/compute/instances/async"))
}

func TestProxyCreateIsRefusedBeforeAnyCall(t *testing.T) {
	create := "POST /v1/xplat/database/load-balancing/create"
	cases := []struct {
		name   string
		setup  func(*fakeProxyPortal, *databaseProxyModel)
		expect string
	}{
		{"cluster not running", func(p *fakeProxyPortal, m *databaseProxyModel) { p.clusterStatus = "stopped" }, "must be running"},
		{"lb_type the cluster does not take", func(p *fakeProxyPortal, m *databaseProxyModel) { m.LbType = types.StringValue("mysqlrouter") }, "maxscale"},
		{"name already used", func(p *fakeProxyPortal, m *databaseProxyModel) { p.addRunning("OLD", "proxy-test-01", 2, 4, "2C4G") }, "already has a proxy named"},
		{"size no flavor has", func(p *fakeProxyPortal, m *databaseProxyModel) { m.NodeCpu = types.Int64Value(3) }, "no flavor has 3 vCPU"},
		{"flavor does not match the size", func(p *fakeProxyPortal, m *databaseProxyModel) { m.Flavor = types.StringValue("4C8G") }, "has 4 vCPU and 8 GB"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newFakeProxyPortal()
			plan := proxyPlan()
			tc.setup(p, &plan)
			r, s := proxyResource(p.start(t))
			resp, _ := createProxyResource(t, r, s, plan)
			require.True(t, resp.Diagnostics.HasError())
			assert.Contains(t, errorText(resp.Diagnostics), tc.expect)
			assert.Empty(t, p.callsLike(create), "no create call is sent")
		})
	}
}

func TestProxyCreateKeepsTheProxyWhenItFails(t *testing.T) {
	p := newFakeProxyPortal()
	api := p.start(t)
	r, s := proxyResource(api)
	_ = api
	// Rewrite the queue that create sets: the proxy ends failed.
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p.serve(w, req)
		if req.URL.Path == "/v1/xplat/database/load-balancing/create" {
			p.mu.Lock()
			pr := p.proxies[len(p.proxies)-1]
			pr.queue = []proxyView{{"processing", 2, 4, "", "", 0}, {"failed", 2, 4, "", "", 0}}
			p.mu.Unlock()
		}
	})
	r.api = newTestApi(t, handler)

	resp, got := createProxyResource(t, r, s, proxyPlan())
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, errorText(resp.Diagnostics), "entered failed status")
	// The proxy exists on the portal: it is in the state, so it is not lost and a destroy removes it.
	assert.Equal(t, "LB1", got.Id.ValueString())
}

func TestProxyUpdateResize(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").ha = true
	r, s := proxyResource(p.start(t))

	plan := proxyState()
	plan.NodeCpu, plan.NodeRam = types.Int64Value(4), types.Int64Value(8)
	plan.Flavor, plan.FlavorId = types.StringUnknown(), types.StringUnknown()
	resp := &resource.UpdateResponse{State: stateOf(t, s, proxyState())}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, s, plan), State: stateOf(t, s, proxyState())}, resp)
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))

	var got databaseProxyModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, int64(4), got.NodeCpu.ValueInt64())
	assert.Equal(t, int64(8), got.NodeRam.ValueInt64())
	assert.Equal(t, "4C8G", got.Flavor.ValueString())
	assert.Equal(t, "f-4c8g", got.FlavorId.ValueString())
	assert.Equal(t, "running", got.Status.ValueString())

	body := p.lastBody("POST /v1/xplat/database/load-balancing/LB1/resize")
	assert.Contains(t, body, `"flavor":"4C8G"`)
	assert.Contains(t, body, `"flavor_id":"f-4c8g"`)
	assert.Contains(t, body, `"node_cpu":4`)
	// Nothing but a resize: no delete, no create.
	assert.Empty(t, p.callsLike("create"))
	assert.Empty(t, p.callsLike("delete/"))
}

func TestProxyUpdateToTheSizeItHasCallsNoResize(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G")
	r, s := proxyResource(p.start(t))

	// Only flavor_id differs from the state (as when it was set in the configuration).
	plan := proxyState()
	plan.FlavorId = types.StringValue("f-2c4g")
	state := proxyState()
	state.FlavorId = types.StringNull()
	resp := &resource.UpdateResponse{State: stateOf(t, s, state)}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, s, plan), State: stateOf(t, s, state)}, resp)
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	assert.Empty(t, p.callsLike("/resize"), "the portal refuses a resize to the size the proxy has")
}

func TestProxyUpdateRefusesAProxyThatIsNotRunning(t *testing.T) {
	for _, status := range []string{"processing", "failed", "stopped", "resizing"} {
		t.Run(status, func(t *testing.T) {
			p := newFakeProxyPortal()
			p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").queue[0].status = status
			r, s := proxyResource(p.start(t))

			plan := proxyState()
			plan.NodeCpu, plan.NodeRam = types.Int64Value(4), types.Int64Value(8)
			plan.Flavor, plan.FlavorId = types.StringUnknown(), types.StringUnknown()
			resp := &resource.UpdateResponse{State: stateOf(t, s, proxyState())}
			r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, s, plan), State: stateOf(t, s, proxyState())}, resp)
			require.True(t, resp.Diagnostics.HasError())
			assert.Contains(t, errorText(resp.Diagnostics), "must be running before it can be resized")
			assert.Empty(t, p.callsLike("/resize"))
		})
	}
}

func TestProxyUpdateRefusesVMsThatAreNotSynced(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").vmSync = false
	r, s := proxyResource(p.start(t))

	plan := proxyState()
	plan.NodeCpu, plan.NodeRam = types.Int64Value(4), types.Int64Value(8)
	plan.Flavor, plan.FlavorId = types.StringUnknown(), types.StringUnknown()
	resp := &resource.UpdateResponse{State: stateOf(t, s, proxyState())}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, s, plan), State: stateOf(t, s, proxyState())}, resp)
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, errorText(resp.Diagnostics), "not synced")
	assert.NotEmpty(t, p.callsLike("/compute/instances/async"), "a sync is tried first")
	assert.Empty(t, p.callsLike("/resize"))
}

func modifyPlan(t *testing.T, r *resourceDatabaseProxy, s resource.SchemaResponse, state *databaseProxyModel, plan databaseProxyModel, config databaseProxyModel) *resource.ModifyPlanResponse {
	t.Helper()
	req := resource.ModifyPlanRequest{
		Plan:   planOf(t, s, plan),
		Config: tfsdk.Config{Schema: s.Schema, Raw: planOf(t, s, config).Raw},
		State:  tfsdk.State{Schema: s.Schema, Raw: nullRaw(s)},
	}
	if state != nil {
		req.State = stateOf(t, s, *state)
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	return resp
}

func TestProxyModifyPlanFixedAttributes(t *testing.T) {
	changes := map[string]func(*databaseProxyModel){
		"db_cluster_id":     func(m *databaseProxyModel) { m.DbClusterId = types.StringValue("other") },
		"vpc_id":            func(m *databaseProxyModel) { m.VpcId = types.StringValue("other") },
		"name":              func(m *databaseProxyModel) { m.Name = types.StringValue("another-name") },
		"lb_type":           func(m *databaseProxyModel) { m.LbType = types.StringValue("mysqlrouter") },
		"high_availability": func(m *databaseProxyModel) { m.HighAvailability = types.BoolValue(false) },
		"storage_profile":   func(m *databaseProxyModel) { m.StorageProfile = types.StringValue("HDD") },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			p := newFakeProxyPortal()
			r, s := proxyResource(p.start(t))
			state := proxyState()
			plan := proxyState()
			change(&plan)
			resp := modifyPlan(t, r, s, &state, plan, plan)
			require.True(t, resp.Diagnostics.HasError(), "a change must fail the plan, not replace the proxy")
			text := errorText(resp.Diagnostics)
			assert.Contains(t, text, "Cannot change "+name)
			assert.Contains(t, text, "on the portal", "the message gives the value the portal has")
			assert.Empty(t, resp.RequiresReplace, "nothing is ever replaced")
		})
	}
}

func TestProxyModifyPlanResize(t *testing.T) {
	resize := func() databaseProxyModel {
		plan := proxyState()
		plan.NodeCpu, plan.NodeRam = types.Int64Value(4), types.Int64Value(8)
		return plan
	}
	config := func() databaseProxyModel {
		c := resize()
		c.Flavor, c.FlavorId = types.StringNull(), types.StringNull()
		return c
	}

	t.Run("running proxy: the flavor follows the size", func(t *testing.T) {
		p := newFakeProxyPortal()
		r, s := proxyResource(p.start(t))
		state := proxyState()
		resp := modifyPlan(t, r, s, &state, resize(), config())
		require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
		var got databaseProxyModel
		require.False(t, resp.Plan.Get(context.Background(), &got).HasError())
		assert.Equal(t, "4C8G", got.Flavor.ValueString())
		assert.Equal(t, "f-4c8g", got.FlavorId.ValueString())
		assert.Empty(t, resp.RequiresReplace)
	})

	for _, status := range []string{"stopped", "processing", "failed", "resizing"} {
		t.Run("proxy "+status+" fails at plan", func(t *testing.T) {
			p := newFakeProxyPortal()
			r, s := proxyResource(p.start(t))
			state := proxyState()
			state.Status = types.StringValue(status)
			resp := modifyPlan(t, r, s, &state, resize(), config())
			require.True(t, resp.Diagnostics.HasError())
			assert.Contains(t, errorText(resp.Diagnostics), "The proxy is "+status+"; it must be running before it can be resized. Start it on the portal first.")
			assert.Empty(t, p.callsLike("/resize"), "no resize command is sent")
		})
	}

	t.Run("no flavor has the size", func(t *testing.T) {
		p := newFakeProxyPortal()
		r, s := proxyResource(p.start(t))
		state := proxyState()
		plan := resize()
		plan.NodeCpu = types.Int64Value(3)
		cfg := config()
		cfg.NodeCpu = types.Int64Value(3)
		resp := modifyPlan(t, r, s, &state, plan, cfg)
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, errorText(resp.Diagnostics), "no flavor has 3 vCPU")
	})

	t.Run("a stopped proxy is not a change when the size is the same", func(t *testing.T) {
		p := newFakeProxyPortal()
		r, s := proxyResource(p.start(t))
		state := proxyState()
		state.Status = types.StringValue("stopped")
		plan := proxyState()
		plan.Status = types.StringValue("stopped")
		resp := modifyPlan(t, r, s, &state, plan, plan)
		assert.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	})
}

func TestProxyModifyPlanCreateChecksTheType(t *testing.T) {
	p := newFakeProxyPortal()
	r, s := proxyResource(p.start(t))
	plan := proxyPlan()
	plan.LbType = types.StringValue("mysqlrouter")
	resp := modifyPlan(t, r, s, nil, plan, plan)
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, errorText(resp.Diagnostics), "maxscale")

	// Values that are not known yet (set by another resource) are not checked.
	plan = proxyPlan()
	plan.DbClusterId = types.StringUnknown()
	resp = modifyPlan(t, r, s, nil, plan, plan)
	assert.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
}

func TestProxyValidateConfig(t *testing.T) {
	r, s := proxyResource(nil)
	validate := func(m databaseProxyModel) string {
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: planOf(t, s, m).Raw}}, resp)
		return errorText(resp.Diagnostics)
	}
	ok := proxyPlan()
	assert.Empty(t, validate(ok))
	for _, name := range []string{"short", strings.Repeat("a", 26), "bad/name-01", "bad.name.01"} {
		m := proxyPlan()
		m.Name = types.StringValue(name)
		assert.Contains(t, validate(m), "Invalid name", name)
	}
	for _, name := range []string{"12345678", strings.Repeat("a", 25), "my proxy_01-a"} {
		m := proxyPlan()
		m.Name = types.StringValue(name)
		assert.Empty(t, validate(m), name)
	}
	m := proxyPlan()
	m.NodeCpu = types.Int64Value(0)
	assert.Contains(t, validate(m), "node_cpu")
	m = proxyPlan()
	m.Timeouts = types.ObjectValueMust(timeoutsObjectTypes(), map[string]attrValue{"create": types.StringValue("soon"), "update": types.StringNull(), "delete": types.StringNull()})
	assert.Contains(t, validate(m), "Invalid timeout")
}

func TestProxyRead(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 4, 8, "4C8G").ha = true
	r, s := proxyResource(p.start(t))

	// The portal has a bigger proxy than the state: Read writes the real values into the state.
	resp := &resource.ReadResponse{State: stateOf(t, s, proxyState())}
	r.Read(context.Background(), resource.ReadRequest{State: stateOf(t, s, proxyState())}, resp)
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	var got databaseProxyModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, int64(4), got.NodeCpu.ValueInt64())
	assert.Equal(t, int64(8), got.NodeRam.ValueInt64())
	assert.Equal(t, "4C8G", got.Flavor.ValueString())
	assert.Equal(t, "f-4c8g", got.FlavorId.ValueString(), "the flavor id follows the flavor")
	assert.Equal(t, "v1", got.VpcId.ValueString(), "the vpc is kept: the portal does not return it")
}

func TestProxyReadRemovesAProxyThatIsGone(t *testing.T) {
	p := newFakeProxyPortal()
	r, s := proxyResource(p.start(t))
	resp := &resource.ReadResponse{State: stateOf(t, s, proxyState())}
	r.Read(context.Background(), resource.ReadRequest{State: stateOf(t, s, proxyState())}, resp)
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	assert.True(t, resp.State.Raw.IsNull(), "the proxy is removed from the state")
}

func TestProxyStoppedOutsideTerraformIsNotAChange(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").ha = true
	p.proxies[0].queue[0].status = "stopped"
	r, s := proxyResource(p.start(t))

	resp := &resource.ReadResponse{State: stateOf(t, s, proxyState())}
	r.Read(context.Background(), resource.ReadRequest{State: stateOf(t, s, proxyState())}, resp)
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	var got databaseProxyModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, "stopped", got.Status.ValueString())
	assert.Empty(t, resp.Diagnostics.Warnings(), "a stopped proxy is neither an error nor a drift warning")

	// The plan has no change: status is not in the configuration, and the size is the same.
	mp := modifyPlan(t, r, s, &got, got, got)
	assert.False(t, mp.Diagnostics.HasError(), errorText(mp.Diagnostics))
}

func TestProxyDriftWarnings(t *testing.T) {
	applied := proxySnapshot(&databaseProxyModel{NodeCpu: types.Int64Value(2), NodeRam: types.Int64Value(4), Flavor: types.StringValue("2C4G")})
	onPortal := proxyState()
	onPortal.NodeCpu, onPortal.NodeRam, onPortal.Flavor = types.Int64Value(4), types.Int64Value(8), types.StringValue("4C8G")

	var warnings diag.Diagnostics
	addDriftWarnings(&warnings, applied, proxySnapshot(&onPortal))
	require.Len(t, warnings, 3, "one warning per attribute")
	text := ""
	for _, w := range warnings {
		text += w.Detail() + "\n"
	}
	assert.Contains(t, text, "`node_cpu` was changed outside of Terraform (applied: 2, on the portal now: 4). The next apply will set it back to 2.")
	assert.Contains(t, text, "`flavor` was changed outside of Terraform (applied: 2C4G, on the portal now: 4C8G)")

	var none diag.Diagnostics
	same := proxyState()
	addDriftWarnings(&none, applied, proxySnapshot(&same))
	assert.Empty(t, none)
}

func TestProxyDelete(t *testing.T) {
	deleteCall := "DELETE /v1/xplat/database/load-balancing/delete/LB1"
	t.Run("running", func(t *testing.T) {
		p := newFakeProxyPortal()
		p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G")
		r, s := proxyResource(p.start(t))
		resp := &resource.DeleteResponse{State: stateOf(t, s, proxyState())}
		r.Delete(context.Background(), resource.DeleteRequest{State: stateOf(t, s, proxyState())}, resp)
		require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
		assert.Len(t, p.callsLike(deleteCall), 1)
		assert.Empty(t, p.proxies, "the proxy is gone")
	})
	t.Run("stopped proxy is deleted too", func(t *testing.T) {
		p := newFakeProxyPortal()
		p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").queue[0].status = "stopped"
		r, s := proxyResource(p.start(t))
		resp := &resource.DeleteResponse{State: stateOf(t, s, proxyState())}
		r.Delete(context.Background(), resource.DeleteRequest{State: stateOf(t, s, proxyState())}, resp)
		require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
		assert.Len(t, p.callsLike(deleteCall), 1)
	})
	t.Run("busy proxy: waits until it rests before the delete is sent", func(t *testing.T) {
		p := newFakeProxyPortal()
		px := p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G")
		busy := px.queue[0]
		busy.status = "processing"
		px.queue = []proxyView{busy, busy, busy, px.queue[0]}
		r, s := proxyResource(p.start(t))
		resp := &resource.DeleteResponse{State: stateOf(t, s, proxyState())}
		r.Delete(context.Background(), resource.DeleteRequest{State: stateOf(t, s, proxyState())}, resp)
		require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
		assert.Len(t, p.callsLike(deleteCall), 1)
		assert.GreaterOrEqual(t, len(p.callsLike("detail/LB1")), 3, "the status was read until it rested")
	})
	t.Run("the portal puts the proxy back to failed: the delete is sent again", func(t *testing.T) {
		p := newFakeProxyPortal()
		p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").failDeletes = 2
		r, s := proxyResource(p.start(t))
		resp := &resource.DeleteResponse{State: stateOf(t, s, proxyState())}
		r.Delete(context.Background(), resource.DeleteRequest{State: stateOf(t, s, proxyState())}, resp)
		require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
		assert.Len(t, p.callsLike(deleteCall), 3, "two lost deletes, then the one that works")
		assert.Empty(t, p.proxies)
	})
	t.Run("the portal keeps putting it back: the error is reported after 3 deletes", func(t *testing.T) {
		p := newFakeProxyPortal()
		p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").failDeletes = 99
		r, s := proxyResource(p.start(t))
		resp := &resource.DeleteResponse{State: stateOf(t, s, proxyState())}
		r.Delete(context.Background(), resource.DeleteRequest{State: stateOf(t, s, proxyState())}, resp)
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, errorText(resp.Diagnostics), "could not be deleted")
		assert.Len(t, p.callsLike(deleteCall), proxyDeleteAttempts)
	})
	t.Run("already gone", func(t *testing.T) {
		p := newFakeProxyPortal()
		r, s := proxyResource(p.start(t))
		resp := &resource.DeleteResponse{State: stateOf(t, s, proxyState())}
		r.Delete(context.Background(), resource.DeleteRequest{State: stateOf(t, s, proxyState())}, resp)
		require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
		assert.Empty(t, p.callsLike(deleteCall))
	})
}

func TestProxyImport(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").ha = true
	r, s := proxyResource(p.start(t))
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: s.Schema, Raw: nullRaw(s)}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "LB1"}, resp)
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	var got databaseProxyModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, "LB1", got.Id.ValueString())
	assert.Equal(t, "c1", got.DbClusterId.ValueString())
	assert.Equal(t, "v1", got.VpcId.ValueString(), "the vpc comes from the parent cluster")
	assert.True(t, got.HighAvailability.ValueBool())
	assert.Equal(t, "f-2c4g", got.FlavorId.ValueString())

	resp = &resource.ImportStateResponse{State: tfsdk.State{Schema: s.Schema, Raw: nullRaw(s)}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "NOPE"}, resp)
	assert.True(t, resp.Diagnostics.HasError())
}

func TestProxyServiceGetProxy(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G")
	api := p.start(t)

	d, err := api.getProxy("LB1")
	require.NoError(t, err)
	assert.Equal(t, "proxy-test-01", string(d.LbClusterName))
	assert.Equal(t, int64(1), d.NodeCount)

	// An unknown id is HTTP 400 on the real portal, not 404.
	_, err = api.getProxy("NOPE")
	assert.ErrorIs(t, err, errProxyNotFound)
}

func TestProxyServiceBusinessErrors(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G")
	api := p.start(t)

	// Resize to the flavor the proxy has: the portal refuses with error_code 180004 inside an HTTP 200.
	err := api.resizeProxy(proxyResizeRequest{LbClusterId: "LB1", NodeCpu: 2, NodeRam: 4, Flavor: "2C4G", FlavorId: "f-2c4g"})
	var be *businessError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, int64(180004), be.Code)

	err = api.resizeProxy(proxyResizeRequest{LbClusterId: "NOPE", NodeCpu: 2, NodeRam: 4, Flavor: "2C4G"})
	require.ErrorAs(t, err, &be)
	assert.Equal(t, int64(180003), be.Code)

	assert.Error(t, api.deleteProxy("NOPE"), "HTTP 400 is an error")
}

func TestProxyFixturesParse(t *testing.T) {
	var resp struct {
		Data struct {
			Cluster proxyDetail `json:"cluster"`
			Nodes   struct {
				Total flexInt `json:"total"`
			} `json:"nodes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fixture(t, "proxy_detail_ha_running"), &resp))
	assert.True(t, proxyIsHA(resp.Data.Cluster.IsLbCluster))
	assert.Equal(t, int64(2), int64(resp.Data.Nodes.Total))
	assert.Contains(t, string(resp.Data.Cluster.EndPoint), "R-W")

	require.NoError(t, json.Unmarshal(fixture(t, "proxy_detail_failed"), &resp))
	assert.Empty(t, string(resp.Data.Cluster.EndPoint), "end_point is null while the proxy has none")
	assert.Equal(t, "failed", string(resp.Data.Cluster.Status))

	var list struct {
		Data []proxyListItem `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fixture(t, "proxy_list_running"), &list))
	require.NotEmpty(t, list.Data)
	assert.False(t, proxyIsHA(list.Data[0].IsLbCluster), `list returns "False" as a string`)
	assert.True(t, list.Data[0].VmSync)

	assert.Error(t, checkBusinessError(fixture(t, "proxy_error_resize_same_flavor")))
	assert.Error(t, checkBusinessError(fixture(t, "proxy_error_storage")))
	assert.NoError(t, checkBusinessError(fixture(t, "proxy_create_response")))
	assert.NoError(t, checkBusinessError(fixture(t, "proxy_restart_response")))
}

// proxySequence serves detail answers in order; the last one repeats.
func proxySequence(views ...proxyView) http.Handler {
	var mu sync.Mutex
	n := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/async") || strings.Contains(r.URL.Path, "/storages") {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		mu.Lock()
		v := views[min(n, len(views)-1)]
		n++
		mu.Unlock()
		ep := "null"
		if v.endPoint != "" {
			ep = `"` + v.endPoint + `"`
		}
		fmt.Fprintf(w, `{"code":"200","data":{"cluster":{"db_cluster_id":"c1","lb_cluster_id":"LB1","lb_cluster_name":"n","node_cpu":%d,"node_ram":%d,"end_point":%s,"status":%q,"flavor":%q},"nodes":{"total":%d}}}`,
			v.cpu, v.ram, ep, v.status, v.flavor, v.nodes)
	})
}

func TestWaitProxyReady(t *testing.T) {
	v := func(status, ep string) proxyView { return proxyView{status, 2, 4, "2C4G", ep, 1} }
	wait := func(views ...proxyView) (*proxyDetail, error) {
		api := newTestApi(t, proxySequence(views...))
		return api.waitProxyReady(context.Background(), "v1", "LB1", 5*time.Second)
	}

	// running without an end point is not ready.
	d, err := wait(v("processing", ""), v("running", ""), v("running", ""), v("running", "R-W"))
	require.NoError(t, err)
	assert.Equal(t, "R-W", string(d.EndPoint))

	for _, bad := range []string{"failed", "create_error"} {
		_, err = wait(v("processing", ""), v(bad, ""))
		require.Error(t, err, bad)
		assert.Contains(t, err.Error(), bad)
	}

	old := 20 * time.Millisecond
	api := newTestApi(t, proxySequence(v("processing", "")))
	_, err = api.waitProxyReady(context.Background(), "v1", "LB1", old)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestWaitProxyResized(t *testing.T) {
	v := func(status string, cpu int) proxyView { return proxyView{status, cpu, cpu * 2, "f", "ep", 1} }
	wait := func(views ...proxyView) error {
		api := newTestApi(t, proxySequence(views...))
		return api.waitProxyResized(context.Background(), "v1", "LB1", 4, 8, 5*time.Second)
	}

	// A proxy still running with the old size is not done; resizing then running with the new size is.
	assert.NoError(t, wait(v("running", 2), v("resizing", 2), v("running", 4)))
	assert.NoError(t, wait(v("resizing", 2), v("warning", 4)))
	// One late answer with the old size is not a failure.
	assert.NoError(t, wait(v("resizing", 2), v("running", 2), v("running", 4)))
	// Back to settled twice with the old size after being busy: the portal did not apply it.
	err := wait(v("resizing", 2), v("running", 2), v("running", 2))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not apply the resize")
	err = wait(v("resizing", 2), v("failed", 2))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed status")
}

func TestWaitProxyDeleted(t *testing.T) {
	p := newFakeProxyPortal()
	px := p.addRunning("LB1", "n", 2, 4, "2C4G")
	api := p.start(t)

	// Gone from the list: done.
	px.removeIn = 1
	assert.NoError(t, api.waitProxyDeleted(context.Background(), "v1", "c1", "LB1", 5*time.Second))
	// A proxy that is not in the list is done at once.
	assert.NoError(t, api.waitProxyDeleted(context.Background(), "v1", "c1", "LB1", 5*time.Second))

	for _, tc := range []struct {
		name  string
		views []proxyView
	}{
		{"delete_error", []proxyView{{"deleting", 2, 4, "", "", 0}, {"delete_error", 2, 4, "", "", 0}}},
		{"failed after deleting", []proxyView{{"deleting", 2, 4, "", "", 0}, {"failed", 2, 4, "", "", 0}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newFakeProxyPortal()
			px := p.addRunning("LB1", "n", 2, 4, "2C4G")
			px.queue = tc.views
			err := p.start(t).waitProxyDeleted(context.Background(), "v1", "c1", "LB1", 5*time.Second)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "could not be deleted")
			assert.Equal(t, tc.name == "failed after deleting", errors.Is(err, errProxyDeleteFellBack), "only the fall back to failed is retried")
		})
	}

	// A proxy that is only "failed" (never seen deleting) keeps being waited for: the delete may not have started yet.
	p2 := newFakeProxyPortal()
	p2.addRunning("LB1", "n", 2, 4, "2C4G").queue = []proxyView{{"failed", 2, 4, "", "", 0}}
	err := p2.start(t).waitProxyDeleted(context.Background(), "v1", "c1", "LB1", 30*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestCreateProxyIgnoresProxiesThatWereThere(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("OLD", "proxy-test-01", 2, 4, "2C4G")
	api := p.start(t)
	// The portal does not refuse a name that is used: the new proxy is the one that was not in the list before.
	id, err := api.createProxy(context.Background(), proxyCreateRequest{DbClusterId: "c1", LbClusterName: "proxy-test-01", IsLbCluster: "no", LbType: "maxscale",
		NodeCpu: 2, NodeRam: 4, Flavor: "2C4G", StorageProfile: "SSD"})
	require.NoError(t, err)
	assert.Equal(t, "LB1", id)
}

func TestResolveProxyFlavor(t *testing.T) {
	flavors := []proxyFlavor{
		{FlavorId: "f-2c4g", FlavorName: "2C4G", Vcpu: 2, RamMb: 4096},
		{FlavorId: "f-4c8g", FlavorName: "4C8G", Vcpu: 4, RamMb: 8192},
	}
	f, err := resolveProxyFlavor(flavors, 4, 8, "", "")
	require.NoError(t, err)
	assert.Equal(t, "4C8G", f.FlavorName)

	f, err = resolveProxyFlavor(flavors, 2, 4, "2C4G", "f-2c4g")
	require.NoError(t, err)
	assert.Equal(t, "f-2c4g", f.FlavorId)

	for name, args := range map[string][]interface{}{
		"no flavor of that size":      {int64(3), int64(4), "", ""},
		"unknown flavor name":         {int64(2), int64(4), "9C9G", ""},
		"unknown flavor id":           {int64(2), int64(4), "", "nope"},
		"name and id disagree":        {int64(2), int64(4), "4C8G", "f-2c4g"},
		"size does not match flavor":  {int64(2), int64(4), "4C8G", ""},
		"size does not match the id":  {int64(4), int64(8), "", "f-2c4g"},
		"ram does not match the flav": {int64(2), int64(8), "2C4G", ""},
	} {
		_, err = resolveProxyFlavor(flavors, args[0].(int64), args[1].(int64), args[2].(string), args[3].(string))
		assert.Error(t, err, name)
	}
}

func restartResource(api *databaseApiClient) (*resourceDatabaseProxyRestart, resource.SchemaResponse) {
	r := &resourceDatabaseProxyRestart{api: api}
	var s resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &s)
	return r, s
}

func restartModel(triggers string) databaseProxyRestartModel {
	return databaseProxyRestartModel{
		Id: types.StringUnknown(), ProxyId: types.StringValue("LB1"), VpcId: types.StringValue("v1"),
		Triggers:    types.MapValueMust(types.StringType, map[string]attrValue{"run": types.StringValue(triggers)}),
		RestartedAt: types.StringUnknown(), Timeouts: types.ObjectNull(timeoutsObjectTypes()),
	}
}

func TestProxyRestartCreateAndTrigger(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G")
	r, s := restartResource(p.start(t))
	restart := "POST /v1/xplat/database/load-balancing/restart"

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: s.Schema, Raw: nullRaw(s)}}
	plan := tfsdk.Plan{Schema: s.Schema, Raw: nullRaw(s)}
	require.False(t, plan.Set(context.Background(), restartModel("a")).HasError())
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	require.False(t, createResp.Diagnostics.HasError(), errorText(createResp.Diagnostics))
	var created databaseProxyRestartModel
	require.False(t, createResp.State.Get(context.Background(), &created).HasError())
	assert.Equal(t, "LB1", created.Id.ValueString())
	assert.NotEmpty(t, created.RestartedAt.ValueString())
	assert.Len(t, p.callsLike(restart), 1)
	assert.Equal(t, "running", mustProxy(t, p).queue[len(mustProxy(t, p).queue)-1].status, "the proxy runs again")

	update := func(triggers string) *resource.UpdateResponse {
		newPlan := tfsdk.Plan{Schema: s.Schema, Raw: nullRaw(s)}
		m := restartModel(triggers)
		m.Id, m.RestartedAt = created.Id, created.RestartedAt
		require.False(t, newPlan.Set(context.Background(), m).HasError())
		resp := &resource.UpdateResponse{State: createResp.State}
		r.Update(context.Background(), resource.UpdateRequest{Plan: newPlan, State: createResp.State}, resp)
		return resp
	}

	// The same triggers (another argument changed): the state is updated and nothing is restarted.
	resp := update("a")
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	assert.Len(t, p.callsLike(restart), 1)
	assert.NotEmpty(t, resp.Diagnostics.Warnings())

	// Changed triggers: restart again.
	resp = update("b")
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	assert.Len(t, p.callsLike(restart), 2)
}

func mustProxy(t *testing.T, p *fakeProxyPortal) *fakeProxy {
	t.Helper()
	require.NotEmpty(t, p.proxies)
	return p.proxies[0]
}

func TestProxyRestartRefusesAProxyThatIsNotRunning(t *testing.T) {
	for _, status := range []string{"stopped", "processing", "failed"} {
		t.Run(status, func(t *testing.T) {
			p := newFakeProxyPortal()
			p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").queue[0].status = status
			r, s := restartResource(p.start(t))
			resp := &resource.CreateResponse{State: tfsdk.State{Schema: s.Schema, Raw: nullRaw(s)}}
			plan := tfsdk.Plan{Schema: s.Schema, Raw: nullRaw(s)}
			require.False(t, plan.Set(context.Background(), restartModel("a")).HasError())
			r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
			require.True(t, resp.Diagnostics.HasError())
			assert.Contains(t, errorText(resp.Diagnostics), "must be running before it can be restarted")
			assert.Empty(t, p.callsLike("/restart"), "the portal does not check the status, so no restart is sent")
		})
	}
}

func TestProxyDeleteOfRestartCallsNoApi(t *testing.T) {
	p := newFakeProxyPortal()
	r, s := restartResource(p.start(t))
	st := tfsdk.State{Schema: s.Schema, Raw: nullRaw(s)}
	m := restartModel("a")
	m.Id, m.RestartedAt = types.StringValue("LB1"), types.StringValue("now")
	require.False(t, st.Set(context.Background(), m).HasError())
	resp := &resource.DeleteResponse{State: st}
	r.Delete(context.Background(), resource.DeleteRequest{State: st}, resp)
	assert.False(t, resp.Diagnostics.HasError())
	assert.Empty(t, p.calls)
}

func TestDatabaseProxiesDataSource(t *testing.T) {
	p := newFakeProxyPortal()
	p.addRunning("LB1", "proxy-test-01", 2, 4, "2C4G").ha = true
	d := &datasourceDatabaseProxies{api: p.start(t)}
	var sr datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &sr)

	st := tfsdk.State{Schema: sr.Schema, Raw: nullRawFor(sr.Schema.Type())}
	require.False(t, st.Set(context.Background(), databaseProxiesDataSourceModel{
		Id: types.StringNull(), ClusterId: types.StringValue("c1"), Proxies: types.ListNull(types.ObjectType{AttrTypes: databaseProxyAttrTypes()}),
	}).HasError())
	cfg := tfsdk.Config{Schema: sr.Schema, Raw: st.Raw}

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: sr.Schema, Raw: nullRawFor(sr.Schema.Type())}}
	d.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	var got databaseProxiesDataSourceModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, "c1", got.Id.ValueString())
	require.Len(t, got.Proxies.Elements(), 1)
	obj := got.Proxies.Elements()[0].(types.Object).Attributes()
	assert.Equal(t, types.StringValue("LB1"), obj["id"])
	assert.Equal(t, types.BoolValue(true), obj["high_availability"])
	assert.Equal(t, types.Int64Value(2), obj["node_count"])
	assert.Equal(t, types.BoolValue(true), obj["vm_sync"])
}

func TestDatabaseProxyTypesDataSource(t *testing.T) {
	p := newFakeProxyPortal()
	p.lbTypes = []string{"mysqlrouter"}
	d := &datasourceDatabaseProxyTypes{api: p.start(t)}
	var sr datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &sr)

	st := tfsdk.State{Schema: sr.Schema, Raw: nullRawFor(sr.Schema.Type())}
	require.False(t, st.Set(context.Background(), databaseProxyTypesDataSourceModel{
		Id: types.StringNull(), ClusterId: types.StringValue("c1"), Types: types.ListNull(types.StringType),
	}).HasError())
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: sr.Schema, Raw: nullRawFor(sr.Schema.Type())}}
	d.Read(context.Background(), datasource.ReadRequest{Config: tfsdk.Config{Schema: sr.Schema, Raw: st.Raw}}, resp)
	require.False(t, resp.Diagnostics.HasError(), errorText(resp.Diagnostics))
	var got databaseProxyTypesDataSourceModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, types.ListValueMust(types.StringType, []attrValue{types.StringValue("mysqlrouter")}), got.Types)
}
