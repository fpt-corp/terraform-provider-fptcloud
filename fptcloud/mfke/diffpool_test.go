package fptcloud_mfke

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// emptyPool builds a pool whose collection attributes are all set, so the only
// thing varying between cases is the name.
func emptyPool(name string) *managedKubernetesEnginePool {
	return &managedKubernetesEnginePool{
		WorkerPoolID: types.StringValue(name),
		PoolTags:     types.SetValueMust(types.StringType, []attr.Value{}),
		Kv:           types.SetValueMust(kvElementType(), []attr.Value{}),
		Taints:       types.SetValueMust(taintElementType(), []attr.Value{}),
	}
}

func kvElementType() attr.Type {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
	}}
}

func taintElementType() attr.Type {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"key":    types.StringType,
		"value":  types.StringType,
		"effect": types.StringType,
	}}
}

// Pools are keyed by name, so changing a pool name means the old pool is
// deleted and a new one created. The pool count stays the same, so the length
// check passes and the comparison loop then looks the old name up in the new
// map and gets nil. That dereference crashed the provider:
//
//	panic: runtime error: invalid memory address or nil pointer dereference
//	  ...mfke.(*resourceManagedKubernetesEngine).DiffPool(...utils.go:917)
func TestDiffPoolHandlesPoolNameChange(t *testing.T) {
	r := &resourceManagedKubernetesEngine{}

	from := &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{emptyPool("worker-old")}}
	to := &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{emptyPool("worker-new")}}

	if !r.DiffPool(context.Background(), from, to) {
		t.Error("deleting a pool and creating one with another name must be reported as a difference")
	}
}

// Swapping one pool for another, with the count unchanged, is the same trap.
func TestDiffPoolHandlesReplacedPool(t *testing.T) {
	r := &resourceManagedKubernetesEngine{}

	from := &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{
		emptyPool("pool-a"), emptyPool("pool-b"),
	}}
	to := &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{
		emptyPool("pool-a"), emptyPool("pool-c"),
	}}

	if !r.DiffPool(context.Background(), from, to) {
		t.Error("replacing a pool must be reported as a difference")
	}
}

// A nil entry in either slice must not crash either.
func TestDiffPoolHandlesNilPoolEntries(t *testing.T) {
	r := &resourceManagedKubernetesEngine{}

	cases := []struct {
		name string
		from *managedKubernetesEngine
		to   *managedKubernetesEngine
	}{
		{
			name: "nil in prior state",
			from: &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{nil}},
			to:   &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{emptyPool("worker-1")}},
		},
		{
			name: "nil in plan",
			from: &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{emptyPool("worker-1")}},
			to:   &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{nil}},
		},
		{
			name: "nil on both sides",
			from: &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{nil}},
			to:   &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{nil}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Only asserting it returns rather than panics.
			r.DiffPool(context.Background(), tc.from, tc.to)
		})
	}
}

// Identical pools must still compare equal, so the fix does not turn every plan
// into a worker-pool update.
func TestDiffPoolReportsNoChangeForIdenticalPools(t *testing.T) {
	r := &resourceManagedKubernetesEngine{}

	from := &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{emptyPool("worker-1")}}
	to := &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{emptyPool("worker-1")}}

	if r.DiffPool(context.Background(), from, to) {
		t.Error("identical pools must not be reported as different")
	}
}

// Each compared field, changed on its own, must be reported; and a changed
// field on one pool among several must not be masked by the others.
func TestDiffPoolDetectsFieldChanges(t *testing.T) {
	r := &resourceManagedKubernetesEngine{}

	kv := func(k, v string) types.Set {
		return types.SetValueMust(kvElementType(), []attr.Value{
			types.ObjectValueMust(kvElementType().(types.ObjectType).AttrTypes, map[string]attr.Value{
				"name": types.StringValue(k), "value": types.StringValue(v),
			}),
		})
	}
	taint := types.SetValueMust(taintElementType(), []attr.Value{
		types.ObjectValueMust(taintElementType().(types.ObjectType).AttrTypes, map[string]attr.Value{
			"key": types.StringValue("k"), "value": types.StringValue("v"), "effect": types.StringValue("NoSchedule"),
		}),
	})

	cases := map[string]func(p *managedKubernetesEnginePool){
		"scale_min":   func(p *managedKubernetesEnginePool) { p.ScaleMin = types.Int64Value(5) },
		"scale_max":   func(p *managedKubernetesEnginePool) { p.ScaleMax = types.Int64Value(9) },
		"max_client":  func(p *managedKubernetesEnginePool) { p.MaxClient = types.Int64Value(4) },
		"gpu_sharing": func(p *managedKubernetesEnginePool) { p.GpuSharingClient = types.StringValue("timeSlicing") },
		"pool_tags": func(p *managedKubernetesEnginePool) {
			p.PoolTags = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("x")})
		},
		"kv":     func(p *managedKubernetesEnginePool) { p.Kv = kv("a", "b") },
		"taints": func(p *managedKubernetesEnginePool) { p.Taints = taint },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := emptyPool("pool-b")
			mutate(changed)
			from := &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{emptyPool("pool-a"), emptyPool("pool-b")}}
			to := &managedKubernetesEngine{Pools: []*managedKubernetesEnginePool{emptyPool("pool-a"), changed}}
			if !r.DiffPool(context.Background(), from, to) {
				t.Errorf("change in %s not detected", name)
			}
		})
	}
}

// Pool order must not matter, and added/removed pools must be reported.
func TestDiffPoolOrderAndCount(t *testing.T) {
	r := &resourceManagedKubernetesEngine{}
	ab := []*managedKubernetesEnginePool{emptyPool("a"), emptyPool("b")}
	ba := []*managedKubernetesEnginePool{emptyPool("b"), emptyPool("a")}
	if r.DiffPool(context.Background(), &managedKubernetesEngine{Pools: ab}, &managedKubernetesEngine{Pools: ba}) {
		t.Error("reordering must not be a difference")
	}
	if !r.DiffPool(context.Background(), &managedKubernetesEngine{Pools: ab}, &managedKubernetesEngine{Pools: ab[:1]}) {
		t.Error("removed pool not detected")
	}
	if !r.DiffPool(context.Background(), &managedKubernetesEngine{Pools: ab[:1]}, &managedKubernetesEngine{Pools: ab}) {
		t.Error("added pool not detected")
	}
	if r.DiffPool(context.Background(), &managedKubernetesEngine{}, &managedKubernetesEngine{}) {
		t.Error("no pools on both sides must be equal")
	}
}
