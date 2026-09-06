package fptcloud_mgpu_cluster

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// forcesReplacement reports whether an attribute's plan modifiers include
// RequiresReplace. The modifier only flips its response flag when given a full
// plan/state context, which a unit test cannot fabricate, so identify it by
// the description it advertises instead.
func forcesReplacement(modifiers []planmodifier.String) bool {
	for _, m := range modifiers {
		if strings.Contains(m.Description(context.Background()), "destroy and recreate") {
			return true
		}
	}
	return false
}

// TestClusterScopedFieldsForceReplacement pins the cluster-scoped fields the
// backend cannot change in place. ssh_key_id in particular is a property of
// the cluster, not of a worker pool: there is no API to rotate it, so changing
// it must destroy and recreate the cluster rather than silently drift.
func TestClusterScopedFieldsForceReplacement(t *testing.T) {
	fields := TopFields()

	forceNew := []string{"vpc_id", "cluster_name", "network_id", "ssh_key_id"}
	for _, name := range forceNew {
		attr, ok := fields[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s: expected a StringAttribute, got %T", name, fields[name])
		}
		if !forcesReplacement(attr.PlanModifiers) {
			t.Errorf("%s must force replacement: it cannot be updated in place", name)
		}
	}

	// internal_subnet_lb is the counter-example: also required and
	// cluster-scoped, but it has a dedicated update endpoint, so changing it
	// must NOT recreate the cluster.
	lb, ok := fields["internal_subnet_lb"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("internal_subnet_lb: expected a StringAttribute, got %T", fields["internal_subnet_lb"])
	}
	if forcesReplacement(lb.PlanModifiers) {
		t.Error("internal_subnet_lb must not force replacement: updateInternalSubnetLb changes it in place")
	}
}

// TestComputedFieldsKeepStateInPlan guards against the plan noise that made
// unrelated updates unreadable: every computed attribute the user did not set
// showed as "(known after apply)", including the resource id, which cannot
// change for a cluster that already exists.
func TestComputedFieldsKeepStateInPlan(t *testing.T) {
	top := TopFields()
	pool := PoolFields()

	for _, tc := range []struct {
		where  string
		fields map[string]schema.Attribute
		names  []string
	}{
		{"cluster", top, []string{"k8s_version", "purpose", "network_type", "edge_gateway_id", "edge_gateway_name", "pod_network", "service_network"}},
		{"pool", pool, []string{"network_id", "network_name", "container_runtime", "hpc_flavor_name", "gpu_type"}},
	} {
		for _, name := range tc.names {
			attr, ok := tc.fields[name].(schema.StringAttribute)
			if !ok {
				t.Fatalf("%s %s: expected a StringAttribute, got %T", tc.where, name, tc.fields[name])
			}
			if !attr.Computed {
				continue
			}
			if len(attr.PlanModifiers) == 0 {
				t.Errorf("%s %s is computed but keeps no state across plans: it will show as \"(known after apply)\" on every unrelated change", tc.where, name)
			}
		}
	}
}
