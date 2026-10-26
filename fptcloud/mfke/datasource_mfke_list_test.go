package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"terraform-provider-fptcloud/commons"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// listResponseFixture is a trimmed response of the cluster list endpoint: a
// public v1 cluster and a hibernated private v2 cluster with a GPU pool, whose
// worker labels come back as an object.
const listResponseFixture = `{
	"data": {
		"items": [
			{
				"id": "rv14j9yn",
				"metadata": {
					"creationTimestamp": "2026-09-18T15:25:06Z",
					"deletionTimestamp": "2026-10-05T16:02:03Z",
					"labels": {
						"fptcloud.com/flavor_pool_worker-7newkb39": "09175bc8-c5ed-4fce-9f23-3e5e5a47feb4",
						"extensions.extensions.gardener.cloud/acl": "true",
						"shoot.gardener.cloud/status": "unhealthy"
					},
					"name": "data-usage-v1-rv14j9yn"
				},
				"spec": {
					"extensions": [
						{"providerConfig": {"rule": {"action": "ALLOW", "cidrs": ["10.0.0.0/8", "1.2.3.4/32"], "type": "remote_ip"}}, "type": "acl"}
					],
					"hibernation": {},
					"kubernetes": {"kubelet": {"maxPods": 110}, "version": "1.32.5"},
					"networking": {"nodes": "192.168.120.0/24", "pods": "100.96.0.0/11", "services": "100.64.0.0/13", "type": "calico"},
					"provider": {
						"workers": [
							{
								"annotations": {"worker.fptcloud.com/node-auto-repair": "true"},
								"labels": null,
								"machine": {"type": "2C4G"},
								"maximum": 3,
								"minimum": 1,
								"name": "worker-7newkb39",
								"providerConfig": {"vGpuId": null},
								"systemComponents": {"allow": true},
								"volume": {"size": "40Gi", "type": "Premium-SSD-8000_vsfshare01"}
							}
						]
					},
					"seedSelector": {"matchLabels": {"gardener_cloud_purpose": "public-VSFX2C1"}}
				},
				"status": {
					"advertisedAddresses": [
						{"name": "external", "url": "https://api.rv14j9yn.prd.m.fke.fptcloud.com"},
						{"name": "internal", "url": "https://api.internal.example"}
					],
					"extraData": {"action": "Error", "errorMessage": "FPT Cloud encountered an error"},
					"lastOperation": {"state": "Failed", "type": "Delete"}
				},
				"tags": []
			},
			{
				"id": "wmf9lqbr",
				"isV2": true,
				"metadata": {
					"creationTimestamp": "2026-10-05T14:53:41Z",
					"deletionTimestamp": null,
					"labels": {
						"fptcloud.com/flavor_pool_worker-dp93jayz": "e9702c7c-c4e0-4cea-9166-d95397dadd6e",
						"extensions.extensions.gardener.cloud/private-network": "true",
						"shoot.gardener.cloud/status": "healthy"
					},
					"name": "l4-v2-wmf9lqbr"
				},
				"spec": {
					"extensions": [
						{"providerConfig": {"privateCluster": true, "allowCIDRs": ["172.16.0.0/12"]}, "type": "private-network"}
					],
					"hibernation": {"enabled": true},
					"kubernetes": {"kubelet": {"maxPods": 110}, "version": "1.34.6"},
					"networking": {"nodes": "10.10.0.0/16", "pods": "100.96.0.0/11", "services": "100.64.0.0/13", "type": "cilium"},
					"provider": {
						"workers": [
							{
								"annotations": {},
								"labels": {"worker.fptcloud/type": "gpu"},
								"machine": {"type": "8C40G_L4_24G"},
								"maximum": 1,
								"minimum": 1,
								"name": "worker-dp93jayz",
								"providerConfig": {"vGpuId": "e1c4c0eb-781b-45ff-8811-1d17bf4ceb3d"},
								"systemComponents": {"allow": false},
								"volume": {"size": "40Gi", "type": "Premium-SSD-8000_vsfshare01"}
							}
						]
					},
					"seedSelector": {"matchLabels": {"gardener_cloud_purpose": "private-VSFX2C1"}}
				},
				"status": {
					"advertisedAddresses": [],
					"extraData": {"action": "Hibernated", "errorMessage": null},
					"lastOperation": {"state": "Succeeded", "type": "Reconcile"}
				},
				"tags": [{"id": "tag-1", "key": "env", "value": "dev"}]
			}
		]
	}
}`

// newListTestClient serves the cluster list endpoint through handle, which
// receives the requested page number.
func newListTestClient(t *testing.T, handle func(page int) string) (*MfkeApiClient, *int32) {
	t.Helper()
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&calls, 1)
		assert.Contains(t, req.URL.Path, "/v1/xplat/fke/vpc/vpc-1/m-fke/osp/get-shoot-cluster/shoots")
		assert.Equal(t, strconv.Itoa(managedKubernetesEngineListPageSize), req.URL.Query().Get("page_size"))
		assert.Equal(t, "OSP", req.Header.Get("infra-type"))

		page, err := strconv.Atoi(req.URL.Query().Get("page"))
		require.NoError(t, err)
		_, _ = rw.Write([]byte(handle(page)))
	}))
	t.Cleanup(server.Close)

	client, err := commons.NewClientForTestingWithServer(server)
	require.NoError(t, err)

	return newMfkeApiClient(client), &calls
}

// pageOf builds a list response holding clusters named <prefix>-0 ... <prefix>-(n-1).
func pageOf(prefix string, n int) string {
	items := make([]string, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, fmt.Sprintf(`{"id": "%d", "metadata": {"name": "%s-%d"}}`, i, prefix, i))
	}
	return `{"data": {"items": [` + strings.Join(items, ",") + `]}}`
}

func TestListClusters_SinglePage(t *testing.T) {
	m, calls := newListTestClient(t, func(page int) string { return listResponseFixture })

	items, err := m.listClusters("vpc-1", "osp")

	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, int32(1), *calls)
}

func TestListClusters_EmptyVpc(t *testing.T) {
	m, calls := newListTestClient(t, func(page int) string { return `{"data": {"items": []}}` })

	items, err := m.listClusters("vpc-1", "osp")

	require.NoError(t, err)
	assert.Empty(t, items)
	assert.NotNil(t, items)
	assert.Equal(t, int32(1), *calls)
}

func TestListClusters_FollowsPages(t *testing.T) {
	m, calls := newListTestClient(t, func(page int) string {
		if page == 1 {
			return pageOf("a", managedKubernetesEngineListPageSize)
		}
		return pageOf("b", 3)
	})

	items, err := m.listClusters("vpc-1", "osp")

	require.NoError(t, err)
	assert.Len(t, items, managedKubernetesEngineListPageSize+3)
	assert.Equal(t, int32(2), *calls)
}

func TestListClusters_StopsWhenApiIgnoresPage(t *testing.T) {
	m, calls := newListTestClient(t, func(page int) string {
		return pageOf("a", managedKubernetesEngineListPageSize)
	})

	items, err := m.listClusters("vpc-1", "osp")

	require.NoError(t, err)
	assert.Len(t, items, managedKubernetesEngineListPageSize)
	assert.Equal(t, int32(2), *calls)
}

func TestListClusters_ApiError(t *testing.T) {
	m, _ := newListTestClient(t, func(page int) string {
		return `{"error": true, "mess": ["vpc not found"], "data": {"items": []}}`
	})

	_, err := m.listClusters("vpc-1", "osp")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "vpc not found")
}

// endpointAccess unpacks a cluster_endpoint_access object into its type and
// allowed CIDR blocks.
func endpointAccess(t *testing.T, access types.Object) (string, []string) {
	t.Helper()
	require.False(t, access.IsNull())

	attrs := access.Attributes()
	cidrs := []string{}
	for _, v := range attrs["allow_cidr"].(types.List).Elements() {
		cidrs = append(cidrs, v.(types.String).ValueString())
	}
	return attrs["type"].(types.String).ValueString(), cidrs
}

func TestToManagedKubernetesEngineSummary(t *testing.T) {
	var response managedKubernetesEngineListResponse
	require.NoError(t, json.Unmarshal([]byte(listResponseFixture), &response))
	require.Len(t, response.Data.Items, 2)

	v1, diags := toManagedKubernetesEngineSummary(response.Data.Items[0])
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "data-usage-v1-rv14j9yn", v1.Id.ValueString())
	assert.Equal(t, "data-usage-v1-rv14j9yn", v1.ClusterName.ValueString(), "cluster_name is the full name, as in the single-cluster data source")
	assert.Equal(t, "1.32.5", v1.K8SVersion.ValueString())
	assert.Equal(t, "public", v1.Purpose.ValueString())
	assert.True(t, v1.IsRunning.ValueBool(), "an empty hibernation spec means running")
	accessType, cidrs := endpointAccess(t, v1.ClusterEndpointAccess)
	assert.Equal(t, "public", accessType)
	assert.Equal(t, []string{"10.0.0.0/8", "1.2.3.4/32"}, cidrs)
	assert.Equal(t, "calico", v1.NetworkType.ValueString())
	assert.Equal(t, "192.168.120.0", v1.NodeNetwork.ValueString())
	assert.Equal(t, "24", v1.NodePrefix.ValueString())
	assert.Equal(t, "100.96.0.0", v1.PodNetwork.ValueString())
	assert.Equal(t, "11", v1.PodPrefix.ValueString())
	assert.Equal(t, "100.64.0.0", v1.ServiceNetwork.ValueString())
	assert.Equal(t, "13", v1.ServicePrefix.ValueString())
	assert.Equal(t, int64(110), v1.K8SMaxPod.ValueInt64())
	assert.Equal(t, "2026-09-18T15:25:06Z", v1.CreatedAt.ValueString())
	assert.Empty(t, v1.Tags.Elements())
	require.Len(t, v1.Pools, 1)
	pool := v1.Pools[0]
	assert.Equal(t, "worker-7newkb39", pool.Name.ValueString())
	assert.Equal(t, "09175bc8-c5ed-4fce-9f23-3e5e5a47feb4", pool.WorkerType.ValueString())
	assert.Equal(t, "2C4G", pool.MachineType.ValueString())
	assert.Equal(t, "Premium-SSD-8000_vsfshare01", pool.StorageProfile.ValueString())
	assert.Equal(t, int64(40), pool.WorkerDiskSize.ValueInt64())
	assert.Equal(t, int64(1), pool.ScaleMin.ValueInt64())
	assert.Equal(t, int64(3), pool.ScaleMax.ValueInt64())
	assert.True(t, pool.WorkerBase.ValueBool())
	assert.Equal(t, "", pool.VGpuID.ValueString())
	assert.True(t, pool.IsEnableAutoRepair.ValueBool())

	v2, diags := toManagedKubernetesEngineSummary(response.Data.Items[1])
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "l4-v2-wmf9lqbr", v2.Id.ValueString())
	assert.Equal(t, "private", v2.Purpose.ValueString())
	assert.False(t, v2.IsRunning.ValueBool(), "a hibernated cluster is not running")
	accessType, cidrs = endpointAccess(t, v2.ClusterEndpointAccess)
	assert.Equal(t, "private", accessType)
	assert.Equal(t, []string{"172.16.0.0/12"}, cidrs)
	assert.Equal(t, "cilium", v2.NetworkType.ValueString())
	assert.Equal(t, "10.10.0.0", v2.NodeNetwork.ValueString())
	assert.Equal(t, "16", v2.NodePrefix.ValueString())
	assert.Len(t, v2.Tags.Elements(), 1)
	require.Len(t, v2.Pools, 1)
	assert.Equal(t, "e1c4c0eb-781b-45ff-8811-1d17bf4ceb3d", v2.Pools[0].VGpuID.ValueString())
	assert.False(t, v2.Pools[0].WorkerBase.ValueBool())
	assert.False(t, v2.Pools[0].IsEnableAutoRepair.ValueBool())
}

// TestManagedKubernetesEngines_StateMatchesSchema sets a populated model into
// state, which fails if a tfsdk tag and a schema attribute disagree.
func TestManagedKubernetesEngines_StateMatchesSchema(t *testing.T) {
	ctx := context.Background()
	d := NewDataSourceManagedKubernetesEngines()

	schemaResponse := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResponse)
	require.False(t, schemaResponse.Diagnostics.HasError(), schemaResponse.Diagnostics)

	var response managedKubernetesEngineListResponse
	require.NoError(t, json.Unmarshal([]byte(listResponseFixture), &response))

	model := managedKubernetesEngines{VpcId: types.StringValue("vpc-1")}
	for _, item := range response.Data.Items {
		summary, diags := toManagedKubernetesEngineSummary(item)
		require.False(t, diags.HasError(), diags)
		model.Clusters = append(model.Clusters, summary)
	}

	state := tfsdk.State{Schema: schemaResponse.Schema}
	diags := state.Set(ctx, &model)
	require.False(t, diags.HasError(), diags)

	var roundTrip managedKubernetesEngines
	diags = state.Get(ctx, &roundTrip)
	require.False(t, diags.HasError(), diags)
	assert.Len(t, roundTrip.Clusters, 2)
}
