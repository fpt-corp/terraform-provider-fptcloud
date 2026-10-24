package fptcloud_database

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// driftSnapshotKey is the private state key that holds the values Terraform
// last applied. Read compares them with what the portal returns now.
const driftSnapshotKey = "applied_snapshot"

// driftSnapshot maps an attribute name to the value as text.
type driftSnapshot map[string]string

// privateState is the part of the framework's private state the drift helpers
// need. Both the request and the response private state of a resource satisfy it.
type privateState interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// saveDriftSnapshot stores the applied values in the private state.
func saveDriftSnapshot(ctx context.Context, private privateState, snapshot driftSnapshot) diag.Diagnostics {
	if isNilPrivate(private) {
		return nil
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error saving applied values", err.Error())
		return diags
	}
	return private.SetKey(ctx, driftSnapshotKey, raw)
}

// isNilPrivate is true when there is no private state to write to, as when a resource method is called
// directly in a test. The framework always provides one.
func isNilPrivate(private privateState) bool {
	if private == nil {
		return true
	}
	v := reflect.ValueOf(private)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

// loadDriftSnapshot returns the applied values, or nil when none were saved
// (a resource created before drift detection existed).
func loadDriftSnapshot(ctx context.Context, private privateState) (driftSnapshot, diag.Diagnostics) {
	raw, diags := private.GetKey(ctx, driftSnapshotKey)
	if diags.HasError() || len(raw) == 0 {
		return nil, diags
	}
	var snapshot driftSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		// A snapshot nobody can read is treated as missing; it is rewritten on the next apply.
		return nil, diags
	}
	return snapshot, diags
}

// addDriftWarnings adds one warning per attribute whose value on the portal
// differs from the applied value. Attributes missing from either side are skipped.
func addDriftWarnings(diags *diag.Diagnostics, applied, current driftSnapshot) {
	names := make([]string, 0, len(applied))
	for name := range applied {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		now, ok := current[name]
		if !ok || now == applied[name] {
			continue
		}
		diags.AddWarning(
			fmt.Sprintf("%s was changed outside of Terraform", name),
			fmt.Sprintf("`%s` was changed outside of Terraform (applied: %s, on the portal now: %s). The next apply will set it back to %s.",
				name, applied[name], now, applied[name]),
		)
	}
}
