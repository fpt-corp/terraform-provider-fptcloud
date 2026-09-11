package fptcloud_mgpu_cluster

import (
	"encoding/json"
	"reflect"
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
					"mig_strategy":        types.StringValue("SINGLE"),
					"mig_profile":         types.StringValue("all-1g.35gb"),
					"sharing_client_type": types.StringValue("TIMESLICING"),
					"max_client":          types.Int64Value(2),
				}),
			},
			want: `{"name":"worker-686lx0a0","mig_mode":"SINGLE","mig_profile":"all-1g.35gb","sharing_client_type":"TIMESLICING","max_client":2,"gpu_scheduler":"NONE","driver_type":"PRE_INSTALL","driver_version":"550.90.07","enable_operand":true,"gpu_type":"H200"}`,
		},
		{
			// USER_INSTALL: the user brings their own driver, so the platform
			// manages neither MIG nor sharing.
			//
			// Note this departs from SRS 2.3, which says sharing_client_type
			// is "all-disabled" here — the API rejects that with HTTP 422
			// ("must be one of: ['MPS', 'TIMESLICING', 'NONE', '']"). Only
			// mig_profile takes all-disabled.
			name: "user install",
			pool: &managedGpuClusterPool{
				WorkerPoolID: types.StringValue("worker-userinstall"),
				GpuType:      types.StringValue("H200"),
				GpuDriver: types.ObjectValueMust(gpuDriverAttrTypes, map[string]attr.Value{
					"installation_type": types.StringValue("USER_INSTALL"),
					"version":           types.StringValue(""),
				}),
				GpuSharing: types.ObjectNull(gpuSharingAttrTypes),
			},
			want: `{"name":"worker-userinstall","mig_mode":"NONE","mig_profile":"all-disabled","sharing_client_type":"NONE","max_client":0,"gpu_scheduler":"NONE","driver_type":"USER_INSTALL","driver_version":"","enable_operand":true,"gpu_type":"H200"}`,
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
		Software:   types.SetNull(types.ObjectType{AttrTypes: softwareAttrTypes}),
		Pools:      []*managedGpuClusterPool{},
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

// softwareSet builds a software set from {type: version} pairs, giving the GPU
// operator the supplied MIG strategy.
func softwareSet(t *testing.T, migStrategy string, versions map[string]string) types.Set {
	t.Helper()

	elements := make([]attr.Value, 0, len(versions))
	for name, version := range versions {
		strategy := ""
		if name == softwareTypeGpuOperator {
			strategy = migStrategy
		}
		elements = append(elements, types.ObjectValueMust(softwareAttrTypes, map[string]attr.Value{
			"software_type":        types.StringValue(name),
			"software_version":     types.StringValue(version),
			"cluster_mig_strategy": types.StringValue(strategy),
		}))
	}
	return types.SetValueMust(types.ObjectType{AttrTypes: softwareAttrTypes}, elements)
}

// TestSoftwareToOperatorVersion covers the mapping onto the API's
// operator_version map, including the upper-casing of mig_strategy that only
// the GPU operator carries.
func TestSoftwareToOperatorVersion(t *testing.T) {
	got, migStrategy := softwareToOperatorVersion(softwareSet(t, "single", map[string]string{
		softwareTypeGpuOperator: "v25.10.1",
		"network_operator":      "v24.10.1",
	}))

	want := map[string]string{"gpu_operator": "v25.10.1", "network_operator": "v24.10.1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("operator_version mismatch\n got: %v\nwant: %v", got, want)
	}
	if migStrategy == nil || *migStrategy != "SINGLE" {
		t.Errorf("mig_strategy: expected SINGLE, got %v", migStrategy)
	}

	// Without a GPU operator there is no MIG strategy to send.
	_, migStrategy = softwareToOperatorVersion(softwareSet(t, "", map[string]string{"slurm_operator": "2.0.5"}))
	if migStrategy != nil {
		t.Errorf("mig_strategy: expected nil without a gpu_operator, got %q", *migStrategy)
	}
}

// TestGpuSharingObjectValueAllOff covers the ambiguity at the heart of reading
// gpu_sharing back: the API reports a pool that never configured sharing and
// one that configured NONE everywhere identically, so the previous state value
// has to break the tie. Getting it wrong either fails the apply with "was
// object, but now null" or leaves an unconfigured pool diffing forever.
func TestGpuSharingObjectValueAllOff(t *testing.T) {
	// The API's "off" values, after normalizeGpuNone has blanked them.
	const offMig, offProfile, offClient = "", "", ""

	t.Run("never configured stays null", func(t *testing.T) {
		got := gpuSharingObjectValue(types.ObjectNull(gpuSharingAttrTypes), offMig, offProfile, offClient, 0)
		if !got.IsNull() {
			t.Errorf("expected null so an unset block does not diff, got %v", got)
		}
	})

	t.Run("configured as NONE keeps the object", func(t *testing.T) {
		prior := types.ObjectValueMust(gpuSharingAttrTypes, map[string]attr.Value{
			"mig_strategy":        types.StringValue("NONE"),
			"mig_profile":         types.StringNull(),
			"sharing_client_type": types.StringValue("NONE"),
			"max_client":          types.Int64Value(0),
		})

		got := gpuSharingObjectValue(prior, offMig, offProfile, offClient, 0)
		if got.IsNull() {
			t.Fatal("expected an object: collapsing to null breaks the plan's promise and fails the apply")
		}

		attrs := got.Attributes()
		if v := attrs["mig_strategy"].(types.String).ValueString(); v != "NONE" {
			t.Errorf("mig_strategy: got %q, want NONE", v)
		}
		if v := attrs["sharing_client_type"].(types.String).ValueString(); v != "NONE" {
			t.Errorf("sharing_client_type: got %q, want NONE", v)
		}
	})

	t.Run("actual settings win over prior", func(t *testing.T) {
		got := gpuSharingObjectValue(types.ObjectNull(gpuSharingAttrTypes), "SINGLE", "all-1g.35gb", "TIMESLICING", 2)
		attrs := got.Attributes()
		if v := attrs["mig_strategy"].(types.String).ValueString(); v != "SINGLE" {
			t.Errorf("mig_strategy: got %q, want SINGLE", v)
		}
		if v := attrs["max_client"].(types.Int64).ValueInt64(); v != 2 {
			t.Errorf("max_client: got %d, want 2", v)
		}
	})
}

// TestOperatorVersionsCatalogParsing pins the shape of the operator-versions
// response the software validation reads its allowed types and versions from.
func TestOperatorVersionsCatalogParsing(t *testing.T) {
	// Captured from a real call to
	// GET .../fke-gpu/common/vpc/{vpcId}/operator-versions
	body := []byte(`{
		"gpu_operator":      {"software_version": ["v25.10.1", "v24.9.1", "v24.3.0"]},
		"network_operator":  {"software_version": ["v24.10.1", "v23.4.0"]},
		"slurm_operator":    {"software_version": ["2.0.5", "1.15.3", "1.14.10"]},
		"vgpu_scheduler":    {"software_version": ["2.8.0", "2.5.2", "2.5.0"]}
	}`)

	var resp map[string]operatorVersionEntry
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshalling operator-versions: %v", err)
	}

	catalog := make(map[string][]string, len(resp))
	for name, entry := range resp {
		catalog[name] = entry.SoftwareVersion
	}

	want := map[string][]string{
		"gpu_operator":     {"v25.10.1", "v24.9.1", "v24.3.0"},
		"network_operator": {"v24.10.1", "v23.4.0"},
		"slurm_operator":   {"2.0.5", "1.15.3", "1.14.10"},
		"vgpu_scheduler":   {"2.8.0", "2.5.2", "2.5.0"},
	}
	if !reflect.DeepEqual(catalog, want) {
		t.Errorf("catalog mismatch\n got: %v\nwant: %v", catalog, want)
	}

	// Error messages list the types in a stable order regardless of map
	// iteration order.
	names := operatorTypeNames(catalog)
	wantNames := []string{"gpu_operator", "network_operator", "slurm_operator", "vgpu_scheduler"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Errorf("operator names: got %v, want %v", names, wantNames)
	}
}

// TestSoftwareSetValueSkipsUninstalled pins how an uninstalled operator reads
// back: the API blanks its version rather than dropping the key, so a blank
// entry must not surface as an installed operator in state.
func TestSoftwareSetValueSkipsUninstalled(t *testing.T) {
	got := softwareSetValue(map[string]string{
		"gpu_operator":     "v25.10.1",
		"network_operator": "",
		"slurm_operator":   "",
		"vgpu_scheduler":   "",
		"service_mesh":     "",
	}, "SINGLE")

	if len(got.Elements()) != 1 {
		t.Fatalf("expected only the installed operator, got %d entries: %v", len(got.Elements()), got)
	}

	attrs := got.Elements()[0].(types.Object).Attributes()
	if v := attrs["software_type"].(types.String).ValueString(); v != "gpu_operator" {
		t.Errorf("software_type: got %q, want gpu_operator", v)
	}
	// The API reports the strategy upper-cased; config spells it lower-case,
	// so state has to match config or every plan shows a diff.
	if v := attrs["cluster_mig_strategy"].(types.String).ValueString(); v != "single" {
		t.Errorf("cluster_mig_strategy: got %q, want single", v)
	}

	// A cluster with no operators at all reads back as null, not an empty set,
	// so a config that never set software sees no diff.
	if empty := softwareSetValue(map[string]string{"gpu_operator": ""}, ""); !empty.IsNull() {
		t.Errorf("expected null set when no operator is installed, got %v", empty)
	}
}
