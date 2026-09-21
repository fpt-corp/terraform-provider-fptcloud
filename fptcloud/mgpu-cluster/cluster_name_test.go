package fptcloud_mgpu_cluster

import "testing"

// TestHasRandomSuffix pins which names count as already-suffixed. The create
// path appends a suffix only when this returns false, so an inverted reading
// here either double-suffixes every name or never suffixes any.
func TestHasRandomSuffix(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"mycluster-abc12345", true},  // 8 alphanumeric after the dash
		{"my-cluster-0my4cgqe", true}, // extra dashes earlier are fine
		{"mycluster", false},          // no suffix at all
		{"mycluster-abc", false},      // too short
		{"mycluster-abc123456", false},
		{"mycluster-abc1234", false}, // 7 chars
		{"mycluster-ABC12345", true}, // upper case is alphanumeric too
	} {
		if got := hasRandomSuffix(tc.name); got != tc.want {
			t.Errorf("hasRandomSuffix(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestValidateClusterName covers the 3-20 character bound the console enforces.
// A name past it yields a shoot Gardener accepts but cannot mint an admin token
// for, which only surfaces later as an opaque failure from the GPU-software
// install — so it has to be rejected up front.
func TestValidateClusterName(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wantErr bool
	}{
		{"abc", false},                           // exactly the minimum
		{"mycluster", false},                     // typical
		{"12345678901234567890", false},          // exactly the maximum
		{"ab", true},                             // one under
		{"123456789012345678901", true},          // one over
		{"", true},                               // empty
		{"mycluster-abc12345", false},            // already suffixed: measured without it
		{"12345678901234567890-abc12345", false}, // max length plus a suffix still fits
		{"123456789012345678901-abc12345", true}, // over the limit even after stripping
	} {
		err := validateClusterName(tc.name)
		if tc.wantErr && err == nil {
			t.Errorf("validateClusterName(%q): expected an error", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("validateClusterName(%q): unexpected error: %s", tc.name, err.Detail())
		}
	}
}
