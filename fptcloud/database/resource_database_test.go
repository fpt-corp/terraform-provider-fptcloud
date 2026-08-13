package fptcloud_database

import (
	"context"
	"strings"
	"testing"
)

func TestClusterIdFromCreateResponse(t *testing.T) {
	body := `{
        "message": "Create database successfully",
        "type": "success",
        "data": {"cluster_id": "owiiqtii", "status": "PROVISIONING"}
    }`

	clusterId, diagErr := clusterIdFromCreateResponse(context.Background(), []byte(body))
	if diagErr != nil {
		t.Fatalf("unexpected error: %s - %s", diagErr.Summary(), diagErr.Detail())
	}
	if clusterId != "owiiqtii" {
		t.Errorf("cluster id = %q, want %q", clusterId, "owiiqtii")
	}
}

// None of these may return an empty id together with a nil diagnostic: that combination
// used to leave the state empty without telling anybody the create had failed
func TestClusterIdFromCreateResponseErrors(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantSummary string
		wantDetail  string
	}{
		{
			name:        "type error",
			body:        `{"type": "error", "message": "Quota exceeded"}`,
			wantSummary: "Error creating database",
			wantDetail:  "Quota exceeded",
		},
		{
			name:        "error_code set while type says success",
			body:        `{"type": "success", "error_code": 4001, "message": "Cluster name already exists"}`,
			wantSummary: "Error creating database",
			wantDetail:  "Cluster name already exists",
		},
		{
			name:        "success without a cluster id",
			body:        `{"type": "success", "message": "ok", "data": {}}`,
			wantSummary: "Database created without a cluster id",
			wantDetail:  "cannot be tracked",
		},
		{
			name:        "success with a blank cluster id",
			body:        `{"type": "success", "data": {"cluster_id": "   "}}`,
			wantSummary: "Database created without a cluster id",
			wantDetail:  "cannot be tracked",
		},
		{
			name:        "unknown type",
			body:        `{"type": "warning", "message": "something else happened"}`,
			wantSummary: "Unexpected response when creating the database",
			wantDetail:  `type "warning"`,
		},
		{
			name:        "no type at all",
			body:        `{"data": {"cluster_id": "owiiqtii"}}`,
			wantSummary: "Unexpected response when creating the database",
			wantDetail:  `type ""`,
		},
		{
			name:        "unparsable body",
			body:        `<html>gateway timeout</html>`,
			wantSummary: "Error unmarshalling response",
			wantDetail:  "gateway timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterId, diagErr := clusterIdFromCreateResponse(context.Background(), []byte(tt.body))
			if diagErr == nil {
				t.Fatalf("expected an error, got cluster id %q", clusterId)
			}
			if clusterId != "" {
				t.Errorf("cluster id = %q, want it empty when the create failed", clusterId)
			}
			if diagErr.Summary() != tt.wantSummary {
				t.Errorf("summary = %q, want %q", diagErr.Summary(), tt.wantSummary)
			}
			if !strings.Contains(diagErr.Detail(), tt.wantDetail) {
				t.Errorf("detail = %q, want it to contain %q", diagErr.Detail(), tt.wantDetail)
			}
		})
	}
}
