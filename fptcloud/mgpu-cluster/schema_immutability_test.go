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

	forceNew := []string{
		"vpc_id", "cluster_name", "network_id", "ssh_key_id",
		// No backend endpoint changes these on a live cluster (SRS 11.3), so
		// a change has to recreate it rather than plan an in-place update.
		"network_type", "pod_network", "pod_prefix", "service_network", "service_prefix",
		// Bare metal has no working upgrade endpoint either: the console hides
		// the action, and the API that serves it cannot find bare-metal
		// clusters. A new version therefore means a new cluster.
		"k8s_version",
	}
	for _, name := range forceNew {
		attr, ok := fields[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s: expected a StringAttribute, got %T", name, fields[name])
		}
		if !forcesReplacement(attr.PlanModifiers) {
			t.Errorf("%s must force replacement: it cannot be updated in place", name)
		}
	}

	// Operations bare metal does not support must not be modelled at all:
	// offering a field the platform cannot honour is worse than omitting it.
	for _, name := range []string{
		"is_running", "hibernation_schedules",
		"is_enable_auto_upgrade", "auto_upgrade_expression", "auto_upgrade_timezone",
	} {
		if _, exists := fields[name]; exists {
			t.Errorf("%s must not be in the schema: the console hides it for bare metal and the endpoint cannot find bare-metal clusters", name)
		}
	}

	// Pool-level: the flavor of an existing pool cannot be changed, but
	// renaming a pool is the supported way to replace one.
	pool := PoolFields()
	flavor, ok := pool["hpc_flavor_id"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("hpc_flavor_id: expected a StringAttribute, got %T", pool["hpc_flavor_id"])
	}
	if !forcesReplacement(flavor.PlanModifiers) {
		t.Error("pools.hpc_flavor_id must force replacement: configure-worker-cluster ignores a new flavor on an existing pool")
	}

	poolName, ok := pool["name"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("pools.name: expected a StringAttribute, got %T", pool["name"])
	}
	if forcesReplacement(poolName.PlanModifiers) {
		t.Error("pools.name must not force replacement: renaming a pool replaces just that pool, not the cluster")
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
