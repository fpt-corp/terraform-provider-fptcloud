package fptcloud

import (
	"context"
	"strings"

	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestProvider tests the provider configuration
func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err.Error())
	}
}

// TestProviderImp tests the provider implementation
func TestProviderImp(t *testing.T) {
	var _ *schema.Provider = Provider()
}

// TestConfig tests the configuration
func TestConfig(t *testing.T) {
	rawProvider := Provider()
	raw := map[string]interface{}{
		"token":       "example_token",
		"tenant_name": "example_tenant_name",
		"region":      "example_region",
	}

	diags := rawProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(raw))
	if diags.HasError() {
		t.Fatalf("provider configure failed: %s", diagnosticsToString(diags))
	}
}

func diagnosticsToString(diags diag.Diagnostics) string {
	diagsAsStrings := make([]string, len(diags))
	for i, diag := range diags {
		diagsAsStrings[i] = diag.Summary
	}

	return strings.Join(diagsAsStrings, "; ")
}
