package fptcloud_mgpu_cluster

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

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
		if len(attr.PlanModifiers) == 0 {
			t.Errorf("%s must force replacement: it cannot be updated in place, but carries no plan modifier", name)
		}
	}

	// internal_subnet_lb is the counter-example: also required and
	// cluster-scoped, but it has a dedicated update endpoint, so changing it
	// must NOT recreate the cluster.
	lb, ok := fields["internal_subnet_lb"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("internal_subnet_lb: expected a StringAttribute, got %T", fields["internal_subnet_lb"])
	}
	if len(lb.PlanModifiers) != 0 {
		t.Error("internal_subnet_lb must not force replacement: updateInternalSubnetLb changes it in place")
	}
}
