package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// shootResponseFixture is a trimmed response of the cluster detail endpoint: a
// public OSP cluster with a CPU pool and a GPU pool using time slicing. Unlike
// a list item, its workers carry their tags, and worker labels are an array.
const shootResponseFixture = `{
	"data": {
		"id": "m3m164up",
		"metadata": {
			"creationTimestamp": "2026-10-23T15:09:10Z",
			"labels": {
				"extensions.extensions.gardener.cloud/acl": "true",
				"fptcloud.com/flavor_pool_worker-pckhpezz": "09175bc8-c5ed-4fce-9f23-3e5e5a47feb4",
				"fptcloud.com/flavor_pool_worker-gpu": "2e495965-3e9e-4474-9aba-e0c934529deb"
			},
			"name": "v2-newui-m3m164up"
		},
		"spec": {
			"addons": {"gpuOperator": {"timeSliceConfig": {"maxClient": ["worker-gpu:4"]}}},
			"autoUpgrade": null,
			"extensions": [
				{"providerConfig": {"rule": {"action": "ALLOW", "cidrs": ["0.0.0.0/0"], "type": "remote_ip"}}, "type": "acl"}
			],
			"hibernation": null,
			"kubernetes": {"kubelet": {"maxPods": 110}, "version": "1.34.6"},
			"networking": {"nodes": "192.168.120.0/24", "pods": "100.96.0.0/11", "services": "100.64.0.0/13", "type": "calico"},
			"provider": {
				"infrastructureConfig": {"networks": {"id": "d146b97f-caa7-45aa-96c8-8f9bb021fddf"}},
				"workers": [
					{
						"annotations": {"worker.fptcloud.com/node-auto-repair": "true"},
						"labels": [],
						"machine": {"image": {"driverInstallationType": null, "gpuDriverVersion": null}, "type": "2C4G"},
						"maximum": 1,
						"minimum": 1,
						"name": "worker-pckhpezz",
						"providerConfig": {"networkName": null, "vGpuId": null},
						"systemComponents": {"allow": true},
						"tags": [{"id": "tag-cluster"}],
						"volume": {"size": "40Gi", "type": "Premium-SSD-8000_vsfshare01"}
					},
					{
						"annotations": {},
						"labels": [{"name": "worker.fptcloud/type", "value": "gpu"}],
						"machine": {"image": {"driverInstallationType": "managed-install", "gpuDriverVersion": "managed-550.163.01"}, "type": "8C24G_L4_24G"},
						"maximum": 2,
						"minimum": 1,
						"name": "worker-gpu",
						"providerConfig": {"networkName": null, "vGpuId": "e1c4c0eb-781b-45ff-8811-1d17bf4ceb3d"},
						"systemComponents": {"allow": false},
						"tags": [{"id": "tag-cluster"}, {"id": "tag-pool"}],
						"volume": {"size": "100Gi", "type": "Premium-SSD-8000_vsfshare01"}
					}
				]
			},
			"seedSelector": {"matchLabels": {"gardener_cloud_purpose": "public-VSFX2C1"}}
		},
		"tags": [{"id": "tag-cluster"}]
	},
	"error": false,
	"mess": ["Get Shoot Specific Successfully"]
}`

func TestToManagedKubernetesEngineDetail(t *testing.T) {
	shoot := decodeShoot(t, shootResponseFixture)
	networkIds := map[string]string{"worker-pckhpezz": "net-1", "worker-gpu": "net-1"}

	detail, diags := toManagedKubernetesEngineDetail(shoot, networkIds)

	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "v2-newui-m3m164up", detail.Id.ValueString())
	assert.Equal(t, "v2-newui-m3m164up", detail.ClusterName.ValueString())
	assert.Equal(t, "1.34.6", detail.K8SVersion.ValueString())
	assert.Equal(t, "public", detail.Purpose.ValueString())
	assert.True(t, detail.IsRunning.ValueBool())
	accessType, cidrs := endpointAccess(t, detail.ClusterEndpointAccess)
	assert.Equal(t, "public", accessType)
	assert.Equal(t, []string{"0.0.0.0/0"}, cidrs)
	assert.Equal(t, "192.168.120.0", detail.NodeNetwork.ValueString())
	assert.Equal(t, "24", detail.NodePrefix.ValueString())
	assert.Equal(t, "2026-10-23T15:09:10Z", detail.CreatedAt.ValueString())
	assert.Len(t, detail.Tags.Elements(), 1)

	require.Len(t, detail.Pools, 2)
	cpu, gpu := detail.Pools[0], detail.Pools[1]

	assert.Equal(t, "worker-pckhpezz", cpu.Name.ValueString())
	assert.Equal(t, "09175bc8-c5ed-4fce-9f23-3e5e5a47feb4", cpu.WorkerType.ValueString())
	assert.Equal(t, "net-1", cpu.NetworkID.ValueString())
	assert.Len(t, cpu.PoolTags.Elements(), 1)
	assert.Equal(t, "", cpu.DriverInstallationType.ValueString())
	assert.Equal(t, int64(0), cpu.MaxClient.ValueInt64())
	assert.Equal(t, "", cpu.GpuSharingClient.ValueString())

	assert.Equal(t, "worker-gpu", gpu.Name.ValueString())
	assert.Equal(t, "8C24G_L4_24G", gpu.MachineType.ValueString())
	assert.Equal(t, int64(100), gpu.WorkerDiskSize.ValueInt64())
	assert.Equal(t, int64(2), gpu.ScaleMax.ValueInt64())
	assert.Equal(t, "e1c4c0eb-781b-45ff-8811-1d17bf4ceb3d", gpu.VGpuID.ValueString())
	assert.Len(t, gpu.PoolTags.Elements(), 2, "pool_tags is the effective set, inherited cluster tags included")
	assert.Equal(t, "managed-install", gpu.DriverInstallationType.ValueString())
	assert.Equal(t, "managed-550.163.01", gpu.GpuDriverVersion.ValueString())
	assert.Equal(t, int64(4), gpu.MaxClient.ValueInt64())
	assert.Equal(t, "timeSlicing", gpu.GpuSharingClient.ValueString())
}

func TestPoolNetworkIds_OspUsesClusterNetwork(t *testing.T) {
	d := &datasourceManagedKubernetesEngine{}
	shoot := decodeShoot(t, shootResponseFixture)

	ids := d.poolNetworkIds(context.Background(), "vpc-1", "osp", shoot)

	assert.Equal(t, map[string]string{
		"worker-pckhpezz": "d146b97f-caa7-45aa-96c8-8f9bb021fddf",
		"worker-gpu":      "d146b97f-caa7-45aa-96c8-8f9bb021fddf",
	}, ids)
}

func TestGetCluster_FallsBackToV2On404(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&calls, 1)
		assert.Equal(t, "OSP", req.Header.Get("infra-type"))
		if !strings.Contains(req.URL.Path, "/v2/get-shoot-specific/shoots/v2-newui-m3m164up") {
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write([]byte(`{"error": true, "mess": ["Get Shoot Specific Failed"]}`))
			return
		}
		_, _ = rw.Write([]byte(shootResponseFixture))
	}))
	defer server.Close()
	client, err := commons.NewClientForTestingWithServer(server)
	require.NoError(t, err)

	shoot, err := newMfkeApiClient(client).getCluster("vpc-1", "osp", "v2-newui-m3m164up")

	require.NoError(t, err)
	assert.Equal(t, "v2-newui-m3m164up", shoot.Metadata.Name)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestGetCluster_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte(`{"error": true, "mess": ["Get Shoot Specific Failed"]}`))
	}))
	defer server.Close()
	client, err := commons.NewClientForTestingWithServer(server)
	require.NoError(t, err)

	_, err = newMfkeApiClient(client).getCluster("vpc-1", "osp", "missing-ab12cd34")

	require.Error(t, err)
}

// TestManagedKubernetesEngine_StateMatchesSchema sets a populated model into
// state, which fails if a tfsdk tag and a schema attribute disagree — the
// mismatch that broke this data source when it shared the resource's model.
func TestManagedKubernetesEngine_StateMatchesSchema(t *testing.T) {
	ctx := context.Background()
	d := NewDataSourceManagedKubernetesEngine()

	schemaResponse := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResponse)
	require.False(t, schemaResponse.Diagnostics.HasError(), schemaResponse.Diagnostics)

	detail, diags := toManagedKubernetesEngineDetail(decodeShoot(t, shootResponseFixture), map[string]string{})
	require.False(t, diags.HasError(), diags)
	detail.VpcId = types.StringValue("vpc-1")

	state := tfsdk.State{Schema: schemaResponse.Schema}
	diags = state.Set(ctx, &detail)
	require.False(t, diags.HasError(), diags)

	var roundTrip managedKubernetesEngineDetail
	diags = state.Get(ctx, &roundTrip)
	require.False(t, diags.HasError(), diags)
	assert.Len(t, roundTrip.Pools, 2)
}

func decodeShoot(t *testing.T, body string) managedKubernetesEngineShoot {
	t.Helper()
	var response managedKubernetesEngineShootResponse
	require.NoError(t, json.Unmarshal([]byte(body), &response))
	return response.Data
}
