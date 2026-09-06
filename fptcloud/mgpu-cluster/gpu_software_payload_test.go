package fptcloud_mgpu_cluster

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestGpuSoftwareWorkerPayload pins the worker_groups entry to the shape the
// console actually sends, captured from two real create flows: one pool with
// no MIG or sharing (v1), and one with both (v2).
func TestGpuSoftwareWorkerPayload(t *testing.T) {
	tests := []struct {
		name string
		pool *managedGpuClusterPool
		want string
	}{
		{
			// Real v1 request: no mig block, no gpu_sharing block. The console
			// omits mig_mode and sharing_client_type entirely and says MIG is
			// off through mig_profile alone.
			name: "no mig, no sharing",
			pool: &managedGpuClusterPool{
				WorkerPoolID: types.StringValue("worker-l4ig3pqf"),
				GpuType:      types.StringValue("H200"),
				GpuDriver: types.ObjectValueMust(gpuDriverAttrTypes, map[string]attr.Value{
					"installation_type": types.StringValue("PRE_INSTALL"),
					"version":           types.StringValue("550.90.07"),
				}),
				GpuSharing: types.ObjectNull(gpuSharingAttrTypes),
				Mig:        types.ObjectNull(migAttrTypes),
			},
			want: `{"name":"worker-l4ig3pqf","mig_profile":"all-disabled","max_client":0,"gpu_scheduler":"NONE","driver_type":"PRE_INSTALL","driver_version":"550.90.07","enable_operand":true,"gpu_type":"H200"}`,
		},
		{
			// Real v2 request: both blocks set, so every field is present.
			name: "mig and sharing",
			pool: &managedGpuClusterPool{
				WorkerPoolID: types.StringValue("worker-686lx0a0"),
				GpuType:      types.StringValue("H200"),
				GpuDriver: types.ObjectValueMust(gpuDriverAttrTypes, map[string]attr.Value{
					"installation_type": types.StringValue("PRE_INSTALL"),
					"version":           types.StringValue("550.90.07"),
				}),
				GpuSharing: types.ObjectValueMust(gpuSharingAttrTypes, map[string]attr.Value{
					"client_type": types.StringValue("TIMESLICING"),
					"max_client":  types.Int64Value(2),
				}),
				Mig: types.ObjectValueMust(migAttrTypes, map[string]attr.Value{
					"strategy": types.StringValue("SINGLE"),
					"profile":  types.StringValue("all-1g.35gb"),
				}),
			},
			want: `{"name":"worker-686lx0a0","mig_mode":"SINGLE","mig_profile":"all-1g.35gb","sharing_client_type":"TIMESLICING","max_client":2,"gpu_scheduler":"NONE","driver_type":"PRE_INSTALL","driver_version":"550.90.07","enable_operand":true,"gpu_type":"H200"}`,
		},
		{
			// USER_INSTALL: the user brings their own driver, so the platform
			// manages neither MIG nor sharing (SRS 2.3).
			name: "user install",
			pool: &managedGpuClusterPool{
				WorkerPoolID: types.StringValue("worker-userinstall"),
				GpuType:      types.StringValue("H200"),
				GpuDriver: types.ObjectValueMust(gpuDriverAttrTypes, map[string]attr.Value{
					"installation_type": types.StringValue("USER_INSTALL"),
					"version":           types.StringValue(""),
				}),
				GpuSharing: types.ObjectNull(gpuSharingAttrTypes),
				Mig:        types.ObjectNull(migAttrTypes),
			},
			want: `{"name":"worker-userinstall","mig_mode":"NONE","mig_profile":"NONE","sharing_client_type":"all-disabled","max_client":0,"gpu_scheduler":"NONE","driver_type":"USER_INSTALL","driver_version":"","enable_operand":true,"gpu_type":"H200"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(gpuSoftwareWorkerFromPool(tt.pool))
			if err != nil {
				t.Fatalf("marshalling worker: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("worker payload mismatch\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

// TestGpuSoftwareRequestWithoutSoftware pins the cluster-level body for a
// cluster that installs no operator, matching the real v1 request: an empty
// operator_version map and a null mig_strategy.
func TestGpuSoftwareRequestWithoutSoftware(t *testing.T) {
	state := &managedGpuCluster{
		K8SVersion: types.StringValue("1.31.4"),
		Software: types.ObjectNull(map[string]attr.Type{
			"software_type":        types.StringType,
			"software_version":     types.StringType,
			"cluster_mig_strategy": types.StringType,
		}),
		Pools: []*managedGpuClusterPool{},
	}

	body := buildGpuSoftwareRequest(state, "mycluster-h12kg0f2", "osp", "tokyo-jp", "afb02abc-1e90-4bf6-a8a1-283c69e37032", false)

	got, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshalling request: %v", err)
	}

	want := `{"isV2":false,"name":"mycluster-h12kg0f2","infra_type":"OSP","region":"tokyo-jp","tenant_id":"afb02abc-1e90-4bf6-a8a1-283c69e37032","kubernetes_version":"1.31.4","operator_version":{},"cluster_type":"BM","mig_strategy":null,"worker_groups":[],"status":"NOTREADY"}`
	if string(got) != want {
		t.Errorf("request payload mismatch\n got: %s\nwant: %s", got, want)
	}
}
