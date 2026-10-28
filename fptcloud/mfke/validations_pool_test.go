package fptcloud_mfke

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runtimePool(name, runtime string) *managedKubernetesEnginePool {
	p := emptyPool(name)
	p.ContainerRuntime = types.StringValue(runtime)
	return p
}

// Pools are matched by name, not by position: a reordered list must not make
// one pool look like it changed runtime, and a real change must still be caught.
func TestValidateImmutablePoolStringFieldMatchesByName(t *testing.T) {
	state := []*managedKubernetesEnginePool{runtimePool("a", "containerd"), runtimePool("b", "docker")}

	t.Run("reordered pools are not a runtime change", func(t *testing.T) {
		plan := []*managedKubernetesEnginePool{runtimePool("b", "docker"), runtimePool("a", "containerd")}
		if d := validateImmutablePoolStringField(plan, state); d != nil {
			t.Errorf("unexpected error: %v", d)
		}
	})
	t.Run("changing the runtime of a pool is rejected even when reordered", func(t *testing.T) {
		plan := []*managedKubernetesEnginePool{runtimePool("b", "containerd"), runtimePool("a", "containerd")}
		if d := validateImmutablePoolStringField(plan, state); d == nil {
			t.Error("runtime change of pool b not detected")
		}
	})
	t.Run("a pool with a new name is new, not a runtime change", func(t *testing.T) {
		plan := []*managedKubernetesEnginePool{runtimePool("c", "docker"), runtimePool("b", "docker")}
		if d := validateImmutablePoolStringField(plan, state); d != nil {
			t.Errorf("unexpected error: %v", d)
		}
	})
	t.Run("nil entries do not panic", func(t *testing.T) {
		validateImmutablePoolStringField([]*managedKubernetesEnginePool{nil, runtimePool("a", "containerd")}, []*managedKubernetesEnginePool{nil, runtimePool("a", "containerd")})
	})
}

func TestValidatePoolNamesUnique(t *testing.T) {
	if d := validatePoolNamesUnique([]*managedKubernetesEnginePool{emptyPool("a"), emptyPool("a")}); d == nil {
		t.Error("duplicate name not detected")
	}
	if d := validatePoolNamesUnique([]*managedKubernetesEnginePool{emptyPool("a"), nil, emptyPool("b")}); d != nil {
		t.Errorf("unexpected error: %v", d)
	}
}

func TestValidatePoolSkipsNilEntries(t *testing.T) {
	good := emptyPool("a")
	good.WorkerDiskSize = types.Int64Value(40)
	validatePool([]*managedKubernetesEnginePool{nil, good})
}

func TestValidateUpdateSkipsNilAndRejectsDuplicates(t *testing.T) {
	newEngine := func(pools ...*managedKubernetesEnginePool) *managedKubernetesEngine {
		return &managedKubernetesEngine{Pools: pools}
	}

	t.Run("nil pool entries do not panic", func(t *testing.T) {
		resp := &resource.UpdateResponse{}
		ValidateUpdate(newEngine(nil), newEngine(nil), resp)
	})
	t.Run("duplicate pool names are rejected", func(t *testing.T) {
		resp := &resource.UpdateResponse{}
		ok := ValidateUpdate(newEngine(emptyPool("a")), newEngine(emptyPool("a"), emptyPool("a")), resp)
		if ok || !resp.Diagnostics.HasError() {
			t.Error("duplicate names in the plan must fail validation")
		}
	})
}
