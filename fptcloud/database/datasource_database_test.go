package fptcloud_database

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	common "terraform-provider-fptcloud/commons"
)

const clusterDetailResponse = `{
    "code": "200",
    "message": "Successful",
    "data": {
        "cluster": {
            "vpc_id": "86c33d49-8bd4-4712-8d65-3962d68fb933",
            "org_name": "XPLAT-FDE-STG-ORG",
            "vcd_url": "https://stg10.fptcloud.net",
            "network_id": "urn:vcloud:network:a76d6b4e-7d83-4821-b725-75d08c9e7723",
            "vm_network": "subnet-db",
            "storage_profile": "Premium-SSD",
            "edge_id": "2fdf8547-0fb7-44dd-92ab-3382e0f175de",
            "edge_gateway_id": "urn:vcloud:gateway:0ba98f75-e294-4690-8b88-9ef1546232b0",
            "cluster_id": "owiiqtii",
            "cluster_name": "Hatest-chiHaxoa",
            "version": "7.0.39",
            "is_beta_version": 1,
            "type_config": "short-config",
            "type_db": "mongodb_replicaset",
            "engine_db": null,
            "port_db": "27017",
            "end_point": "mongodb://172.26.47.15:27017/?authSource=admin",
            "master_count": 1,
            "worker_count": 0,
            "is_cluster": "no",
            "is_monitor": false,
            "is_backup": false,
            "node_cpu": 2,
            "node_core": 0,
            "node_ram": 4,
            "data_disk_size": 20,
            "ip_public": null,
            "engine_active_pitr": 0,
            "ip_vip": "172.26.47.15",
            "status": "running",
            "database_name": "db_default",
            "admin_password": "********",
            "source_cluster_id": null,
            "engine_edition": "Community",
            "is_new_version": true,
            "created_at": "2026-07-28 17:00:34",
            "is_alert": true,
            "is_autoscaling": false,
            "list_service": ["overview", "backup_restore"],
            "backup_service_status": 0,
            "need_reboot": false,
            "engine_name": "Mongodb",
            "is_beta": 0,
            "is_default_version": 0,
            "is_deprecated_version": 0,
            "is_eos_version": 0,
            "end_of_support": "",
            "flavor_id": "b2e61365-cb8b-441b-a22f-89a1427745cd",
            "limit_disk_size": 16384,
            "need_restart": 0,
            "vm_sync": true
        },
        "nodes": {
            "total": 1,
            "items": [
                {
                    "id": "b2c95b56-7039-41e1-9a8c-7c7b052d6f1f",
                    "vdc_name": "FDE-VMW-STG-INTERNAL-VPC",
                    "org_name": "XPLAT-FDE-STG-ORG",
                    "vpc_id": "86c33d49-8bd4-4712-8d65-3962d68fb933",
                    "cluster_id": "owiiqtii",
                    "cluster_name": "Hatest-chiHaxoa",
                    "vmw_id": "9aa9b0e0-fd2c-4e07-8696-1bba209ab229",
                    "name": "fde-mongodb-owiiqtii-node1",
                    "status": "POWERED_ON",
                    "guest_os": "Ubuntu Linux (64-bit)",
                    "ip_address": "172.26.47.15",
                    "number_of_cpus": 2,
                    "data_disk_size": 20,
                    "memory_mb": 4096,
                    "network_name": "172.26.47.15",
                    "database_role": "true",
                    "is_deployed": true,
                    "platform": "VMW",
                    "ip_private": "169.220.0.40",
                    "node_type": "data_node",
                    "created_at": "2026-07-28T17:00:34",
                    "updated_at": "2026-07-28T17:07:15",
                    "shard_id": null,
                    "replica_set": null
                }
            ]
        }
    },
    "type": "success"
}`

const flavorListResponse = `{
    "code": "200",
    "message": "Successful",
    "data": [
        {"flavor_id": "e30cafce-fd93-43c9-abca-dd5c3748678b", "flavor_name": "Small-2", "flavor_vcpu": 1, "flavor_ram": 2048, "flavor_site": "VMW"},
        {"flavor_id": "b2e61365-cb8b-441b-a22f-89a1427745cd", "flavor_name": "Medium-4", "flavor_vcpu": 2, "flavor_ram": 4096, "flavor_site": "VMW"}
    ]
}`

// newTestDataSource spins up a server answering the cluster detail call with the given
// status and body, and the flavor list call with an empty list
func newTestDataSource(t *testing.T, statusCode int, body string) (*datasourceDatabase, func()) {
	t.Helper()
	return newTestDataSourceWithFlavors(t, statusCode, body, `{"code": "200", "data": []}`)
}

// newTestDataSourceWithFlavors dispatches on the requested path so the flavor lookup can
// be exercised alongside the cluster detail call
func newTestDataSourceWithFlavors(t *testing.T, statusCode int, body string, flavorBody string) (*datasourceDatabase, func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		status, payload := statusCode, body
		if strings.Contains(req.URL.Path, "get_list_flavor_v2") {
			status, payload = http.StatusOK, flavorBody
		}

		rw.WriteHeader(status)
		if _, err := rw.Write([]byte(payload)); err != nil {
			t.Errorf("failed writing test response: %v", err)
		}
	}))

	client, err := common.NewClientForTestingWithServer(server)
	if err != nil {
		server.Close()
		t.Fatalf("failed creating test client: %v", err)
	}

	return &datasourceDatabase{client: client, dataBaseClient: newDatabaseApiClient(client)}, server.Close
}

func TestGetDatabaseDetail(t *testing.T) {
	d, closeServer := newTestDataSourceWithFlavors(t, http.StatusOK, clusterDetailResponse, flavorListResponse)
	defer closeServer()

	detail, diagErr := d.getDatabaseDetail(context.Background(), "owiiqtii")
	if diagErr != nil {
		t.Fatalf("unexpected error: %s - %s", diagErr.Summary(), diagErr.Detail())
	}

	var state databaseDataSourceModel
	d.mapDetailToState(context.Background(), detail, &state)

	if got := state.Id.ValueString(); got != "owiiqtii" {
		t.Errorf("id = %q, want %q", got, "owiiqtii")
	}
	if got := state.ClusterName.ValueString(); got != "Hatest-chiHaxoa" {
		t.Errorf("cluster_name = %q, want %q", got, "Hatest-chiHaxoa")
	}
	if got := state.Edition.ValueString(); got != "Community" {
		t.Errorf("edition = %q, want %q", got, "Community")
	}
	// Not returned by the cluster payload, taken from the first node
	if got := state.VdcName.ValueString(); got != "FDE-VMW-STG-INTERNAL-VPC" {
		t.Errorf("vdc_name = %q, want %q", got, "FDE-VMW-STG-INTERNAL-VPC")
	}
	// Not returned by the cluster payload, derived from the platform of the nodes
	if got := state.IsOps.ValueString(); got != "no" {
		t.Errorf("is_ops = %q, want %q", got, "no")
	}
	// Not returned by the cluster payload, resolved through the flavor list
	if got := state.Flavor.ValueString(); got != "Medium-4" {
		t.Errorf("flavor = %q, want %q", got, "Medium-4")
	}
	if got := state.FlavorId.ValueString(); got != "b2e61365-cb8b-441b-a22f-89a1427745cd" {
		t.Errorf("flavor_id = %q", got)
	}
	if got := state.Status.ValueString(); got != "running" {
		t.Errorf("status = %q, want %q", got, "running")
	}
	if got := state.EndPoint.ValueString(); got != "mongodb://172.26.47.15:27017/?authSource=admin" {
		t.Errorf("end_point = %q", got)
	}
	if got := state.NodeRam.ValueInt64(); got != 4 {
		t.Errorf("node_ram = %d, want 4", got)
	}
	if got := state.LimitDiskSize.ValueInt64(); got != 16384 {
		t.Errorf("limit_disk_size = %d, want 16384", got)
	}
	if !state.VmSync.ValueBool() {
		t.Error("vm_sync = false, want true")
	}
	// Fields returned as JSON null must land as null, not as an empty string
	if !state.EngineDb.IsNull() {
		t.Errorf("engine_db = %q, want null", state.EngineDb.ValueString())
	}
	if !state.IpPublic.IsNull() {
		t.Errorf("ip_public = %q, want null", state.IpPublic.ValueString())
	}
	// Fields absent from the payload must stay null as well
	if !state.VhostName.IsNull() {
		t.Errorf("vhost_name = %q, want null", state.VhostName.ValueString())
	}
	if got := len(state.ListService.Elements()); got != 2 {
		t.Errorf("list_service has %d elements, want 2", got)
	}
	if got := state.NodesTotal.ValueInt64(); got != 1 {
		t.Errorf("nodes_total = %d, want 1", got)
	}
	if got := len(state.Nodes.Elements()); got != 1 {
		t.Fatalf("nodes has %d elements, want 1", got)
	}

	node := state.Nodes.Elements()[0].String()
	for _, want := range []string{"fde-mongodb-owiiqtii-node1", "POWERED_ON", "data_node", "169.220.0.40"} {
		if !strings.Contains(node, want) {
			t.Errorf("node object is missing %q: %s", want, node)
		}
	}
}

// The flavor name is extra information, a failing lookup must leave it null
// instead of breaking the whole read
func TestFlavorLookupFailureIsNotFatal(t *testing.T) {
	tests := []struct {
		name       string
		flavorBody string
	}{
		{name: "empty flavor list", flavorBody: `{"code": "200", "data": []}`},
		{name: "flavor id not in the list", flavorBody: `{"code": "200", "data": [{"flavor_id": "other-id", "flavor_name": "Small-2"}]}`},
		{name: "unparsable flavor list", flavorBody: `not a json payload`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, closeServer := newTestDataSourceWithFlavors(t, http.StatusOK, clusterDetailResponse, tt.flavorBody)
			defer closeServer()

			detail, diagErr := d.getDatabaseDetail(context.Background(), "owiiqtii")
			if diagErr != nil {
				t.Fatalf("unexpected error: %s - %s", diagErr.Summary(), diagErr.Detail())
			}

			var state databaseDataSourceModel
			d.mapDetailToState(context.Background(), detail, &state)

			if !state.Flavor.IsNull() {
				t.Errorf("flavor = %q, want null", state.Flavor.ValueString())
			}
			// The rest of the cluster must still be mapped
			if got := state.Id.ValueString(); got != "owiiqtii" {
				t.Errorf("id = %q, want %q", got, "owiiqtii")
			}
		})
	}
}

func TestSyncVpcInfrastructure(t *testing.T) {
	var mu sync.Mutex
	called := map[string]int{}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		mu.Lock()
		called[req.Method+" "+req.URL.Path]++
		mu.Unlock()

		// One endpoint answers with an error on purpose: it must not stop the others
		if strings.HasSuffix(req.URL.Path, "/storages/sync") {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := common.NewClientForTestingWithServer(server)
	if err != nil {
		t.Fatalf("failed creating test client: %v", err)
	}

	syncVpcInfrastructure(context.Background(), client, "vpc-1")

	// The calls are fired in parallel, so only the set matters, not the order
	want := []string{
		"POST /v1/vmware/vpc/vpc-1/compute/instances/async",
		"POST /v1/vmware/vpc/vpc-1/storages/sync",
		"POST /v1/vmware/vpc/vpc-1/storages/sync/v2",
	}

	mu.Lock()
	defer mu.Unlock()
	for _, path := range want {
		if called[path] != 1 {
			t.Errorf("%q was called %d times, want 1 (all calls: %v)", path, called[path], called)
		}
	}
	if len(called) != len(want) {
		t.Errorf("called %v, want exactly %v", called, want)
	}
}

// Endpoints that never answer must not hold the read hostage: all three are still fired,
// but the read stops waiting for them after syncWaitTimeout
func TestSyncVpcInfrastructureDoesNotWaitForSlowEndpoints(t *testing.T) {
	unblock := make(chan struct{})

	var mu sync.Mutex
	var served int

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		mu.Lock()
		served++
		mu.Unlock()

		select {
		case <-unblock:
		case <-req.Context().Done():
		}
		rw.WriteHeader(http.StatusOK)
	}))
	// Deferred calls run last in first out: the handlers have to be released before
	// Close, which waits for every outstanding request
	defer server.Close()
	defer close(unblock)

	client, err := common.NewClientForTestingWithServer(server)
	if err != nil {
		t.Fatalf("failed creating test client: %v", err)
	}

	start := time.Now()
	syncVpcInfrastructure(context.Background(), client, "vpc-1")
	elapsed := time.Since(start)

	if elapsed > syncWaitTimeout+2*time.Second {
		t.Errorf("syncVpcInfrastructure blocked for %s, want it to give up around %s", elapsed, syncWaitTimeout)
	}

	// Every endpoint must have been triggered even though none of them answered
	mu.Lock()
	defer mu.Unlock()
	if served != 3 {
		t.Errorf("the server was called %d times, want 3", served)
	}
}

// The sync calls have to survive the read that started them, otherwise nothing would
// ever reach BSS once the read stops waiting
func TestSyncVpcInfrastructureOutlivesTheRead(t *testing.T) {
	finished := make(chan string, 3)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Answer later than the read is willing to wait
		select {
		case <-time.After(syncWaitTimeout + 500*time.Millisecond):
		case <-req.Context().Done():
			return
		}
		rw.WriteHeader(http.StatusOK)
		finished <- req.URL.Path
	}))
	defer server.Close()

	client, err := common.NewClientForTestingWithServer(server)
	if err != nil {
		t.Fatalf("failed creating test client: %v", err)
	}

	// The read context is cancelled right after the sync returns
	ctx, cancel := context.WithCancel(context.Background())
	syncVpcInfrastructure(ctx, client, "vpc-1")
	cancel()

	for i := 0; i < 3; i++ {
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Fatalf("only %d of 3 sync calls survived the read being cancelled", i)
		}
	}
}

func TestSyncVpcInfrastructureSkipped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		t.Error("no request should be sent without a vpc id")
		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := common.NewClientForTestingWithServer(server)
	if err != nil {
		t.Fatalf("failed creating test client: %v", err)
	}

	syncVpcInfrastructure(context.Background(), client, "")
	syncVpcInfrastructure(context.Background(), nil, "vpc-1")
}

func TestIsOpsFromNodes(t *testing.T) {
	osp := "OSP"
	vmw := "VMW"
	empty := ""

	tests := []struct {
		name  string
		items []databaseDetailNodeItem
		want  string
	}{
		{name: "openstack", items: []databaseDetailNodeItem{{Platform: &osp}}, want: "yes"},
		{name: "vmware", items: []databaseDetailNodeItem{{Platform: &vmw}}, want: "no"},
		{name: "skips nodes without a platform", items: []databaseDetailNodeItem{{Platform: &empty}, {Platform: &osp}}, want: "yes"},
		{name: "no nodes", items: nil, want: ""},
		{name: "no platform at all", items: []databaseDetailNodeItem{{}}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOpsFromNodes(tt.items); got != tt.want {
				t.Errorf("isOpsFromNodes = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetDatabaseDetailErrors(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		body        string
		wantSummary string
		wantDetail  string
	}{
		{
			name:        "http 404",
			statusCode:  http.StatusNotFound,
			body:        `{"code": "404", "message": "Cluster not found", "type": "error"}`,
			wantSummary: "Database cluster not found",
			wantDetail:  "Cluster not found",
		},
		{
			name:        "http 401",
			statusCode:  http.StatusUnauthorized,
			body:        `{"message": "Unauthenticated."}`,
			wantSummary: "Unauthorized when calling the database API",
			wantDetail:  "Unauthenticated.",
		},
		{
			name:        "http 500",
			statusCode:  http.StatusInternalServerError,
			body:        `{"message": "Internal server error"}`,
			wantSummary: "Database API server error",
			wantDetail:  "Internal server error",
		},
		{
			name:        "http 429",
			statusCode:  http.StatusTooManyRequests,
			body:        `Too many requests`,
			wantSummary: "Rate limited by the database API",
			wantDetail:  "Too many requests",
		},
		{
			name:        "http 400 without a parsable body",
			statusCode:  http.StatusBadRequest,
			body:        `<html>bad request</html>`,
			wantSummary: errorCallingApi,
			wantDetail:  "bad request",
		},
		{
			// What the staging API really answers for an unknown cluster id
			name:        "http 400 with an internal error message",
			statusCode:  http.StatusBadRequest,
			body:        `{"code": "400", "message": "'NoneType' object has no attribute 'vpc_id'", "type": "error"}`,
			wantSummary: errorCallingApi,
			wantDetail:  "Check that the database cluster owiiqtii exists and belongs to the configured region and VPC",
		},
		{
			// What the staging API really answers for an invalid token
			name:        "http 422 with a detail field",
			statusCode:  http.StatusUnprocessableEntity,
			body:        `{"detail":"Could not validate credentials"}`,
			wantSummary: errorCallingApi,
			wantDetail:  "Could not validate credentials",
		},
		{
			name:        "http 200 with an error code in the body",
			statusCode:  http.StatusOK,
			body:        `{"code": "400", "message": "Cluster is being deleted", "type": "error"}`,
			wantSummary: "Error getting detail of database cluster owiiqtii",
			wantDetail:  "Cluster is being deleted",
		},
		{
			name:        "http 200 with a numeric error code in the body",
			statusCode:  http.StatusOK,
			body:        `{"code": 403, "message": "Permission denied", "type": "error"}`,
			wantSummary: "Error getting detail of database cluster owiiqtii",
			wantDetail:  "Permission denied",
		},
		{
			name:        "http 200 with type error only",
			statusCode:  http.StatusOK,
			body:        `{"message": "Something went wrong", "type": "error"}`,
			wantSummary: "Error getting detail of database cluster owiiqtii",
			wantDetail:  "Something went wrong",
		},
		{
			name:        "http 200 with an empty cluster",
			statusCode:  http.StatusOK,
			body:        `{"code": "200", "message": "Successful", "type": "success", "data": {"cluster": {}, "nodes": {"total": 0, "items": []}}}`,
			wantSummary: "Database cluster not found",
			wantDetail:  "No database cluster with id owiiqtii was found",
		},
		{
			name:        "malformed json body",
			statusCode:  http.StatusOK,
			body:        `not a json payload`,
			wantSummary: "Error unmarshalling response",
			wantDetail:  "not a json payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, closeServer := newTestDataSource(t, tt.statusCode, tt.body)
			defer closeServer()

			detail, diagErr := d.getDatabaseDetail(context.Background(), "owiiqtii")
			if diagErr == nil {
				t.Fatalf("expected an error, got detail %+v", detail)
			}
			if diagErr.Summary() != tt.wantSummary {
				t.Errorf("summary = %q, want %q", diagErr.Summary(), tt.wantSummary)
			}
			if !strings.Contains(diagErr.Detail(), tt.wantDetail) {
				t.Errorf("detail = %q, want it to contain %q", diagErr.Detail(), tt.wantDetail)
			}
		})
	}
}

func TestTruncateBody(t *testing.T) {
	if got := truncateBody([]byte("  hello  ")); got != "hello" {
		t.Errorf("truncateBody = %q, want %q", got, "hello")
	}

	long := strings.Repeat("a", maxErrorBodyLength+50)
	got := truncateBody([]byte(long))
	if !strings.HasSuffix(got, "... (truncated)") {
		t.Errorf("truncateBody did not truncate a %d byte body", len(long))
	}
	if len(got) != maxErrorBodyLength+len("... (truncated)") {
		t.Errorf("truncated body has length %d", len(got))
	}
}

// The nodes of a freshly created cluster show up a little after the cluster itself, so
// the data source keeps re-reading until they do
// shortenNodeWait makes the polling interval negligible so tests do not sit through it
func shortenNodeWait(t *testing.T) {
	t.Helper()
	original := nodeWaitInterval
	nodeWaitInterval = time.Millisecond
	t.Cleanup(func() { nodeWaitInterval = original })
}

func TestWaitForNodes(t *testing.T) {
	shortenNodeWait(t)
	noNodes := strings.Replace(clusterDetailResponse, `"total": 1`, `"total": 0`, 1)

	var mu sync.Mutex
	detailCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "cluster/detail") {
			mu.Lock()
			detailCalls++
			// The nodes only appear on the third attempt
			payload := noNodes
			if detailCalls >= 3 {
				payload = clusterDetailResponse
			}
			mu.Unlock()

			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(payload))
			return
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(`{"code": "200", "data": []}`))
	}))
	defer server.Close()

	client, err := common.NewClientForTestingWithServer(server)
	if err != nil {
		t.Fatalf("failed creating test client: %v", err)
	}
	d := &datasourceDatabase{client: client, dataBaseClient: newDatabaseApiClient(client)}

	first, diagErr := d.getDatabaseDetail(context.Background(), "owiiqtii")
	if diagErr != nil {
		t.Fatalf("unexpected error: %s - %s", diagErr.Summary(), diagErr.Detail())
	}
	if first.Nodes.Total != 0 {
		t.Fatalf("the first read reported %d nodes, want 0", first.Nodes.Total)
	}

	detail, diagErr := d.waitForNodes(context.Background(), "owiiqtii", "vpc-1", first, time.Minute)
	if diagErr != nil {
		t.Fatalf("unexpected error: %s - %s", diagErr.Summary(), diagErr.Detail())
	}
	if detail.Nodes.Total != 1 {
		t.Errorf("nodes total = %d, want 1", detail.Nodes.Total)
	}
	if len(detail.Nodes.Items) != 1 {
		t.Errorf("got %d node items, want 1", len(detail.Nodes.Items))
	}
}

// Waiting has to end at the deadline, and hand back the last readable answer instead of
// failing: a cluster that is simply not ready yet is not an error
func TestWaitForNodesGivesUpAtTheDeadline(t *testing.T) {
	shortenNodeWait(t)

	noNodes := strings.Replace(clusterDetailResponse, `"total": 1`, `"total": 0`, 1)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		if strings.Contains(req.URL.Path, "cluster/detail") {
			_, _ = rw.Write([]byte(noNodes))
			return
		}
		_, _ = rw.Write([]byte(`{"code": "200", "data": []}`))
	}))
	defer server.Close()

	client, err := common.NewClientForTestingWithServer(server)
	if err != nil {
		t.Fatalf("failed creating test client: %v", err)
	}
	d := &datasourceDatabase{client: client, dataBaseClient: newDatabaseApiClient(client)}

	first, diagErr := d.getDatabaseDetail(context.Background(), "owiiqtii")
	if diagErr != nil {
		t.Fatalf("unexpected error: %s - %s", diagErr.Summary(), diagErr.Detail())
	}

	start := time.Now()
	detail, diagErr := d.waitForNodes(context.Background(), "owiiqtii", "vpc-1", first, 20*time.Millisecond)
	elapsed := time.Since(start)

	if diagErr != nil {
		t.Fatalf("waiting must not fail when the cluster is only slow: %s", diagErr.Detail())
	}
	if detail == nil || detail.Nodes.Total != 0 {
		t.Error("the last readable answer should have been handed back")
	}
	if elapsed > 30*time.Second {
		t.Errorf("waiting took %s, it should have stopped at the deadline", elapsed)
	}
}
