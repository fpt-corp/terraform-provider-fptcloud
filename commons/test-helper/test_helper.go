package test_helper

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"terraform-provider-fptcloud/fptcloud"
)

var (
	TestProvider          *schema.Provider
	TestProviders         map[string]*schema.Provider
	TestProviderFactories map[string]func() (*schema.Provider, error)
	ENV                   = map[string]string{
		"VPC_ID": os.Getenv("VPC_ID"),
	}
)

func init() {
	TestProvider = fptcloud.Provider()
	TestProviders = map[string]*schema.Provider{
		"fptcloud": TestProvider,
	}
	TestProviderFactories = map[string]func() (*schema.Provider, error){
		"fptcloud": func() (*schema.Provider, error) {
			return TestProvider, nil
		},
	}
}

func TestProviderImpl(t *testing.T) {
	var _ = fptcloud.Provider()
}

func TestWithConfig(t *testing.T) {
	rawProvider := fptcloud.Provider()
	raw := map[string]interface{}{
		"token":       "example_token",
		"tenant_name": "example_tenant_name",
		"region":      "example_region",
	}

	diags := rawProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(raw))
	if diags.HasError() {
		t.Fatalf("provider configure failed: %s", DiagnosticsToString(diags))
	}
}

func DiagnosticsToString(diags diag.Diagnostics) string {
	diagsAsStrings := make([]string, len(diags))
	for i, diag := range diags {
		diagsAsStrings[i] = diag.Summary
	}

	return strings.Join(diagsAsStrings, "; ")
}

func TestPreCheck(t *testing.T) {
	if v := os.Getenv("FPTCLOUD_TOKEN"); v == "" {
		t.Fatal("FPTCLOUD_TOKEN must be set for tests")
	}
	if v := os.Getenv("FPTCLOUD_TENANT_NAME"); v == "" {
		t.Fatal("FPTCLOUD_TENANT_NAME must be set for tests")
	}
	if v := os.Getenv("FPTCLOUD_REGION"); v == "" {
		t.Fatal("FPTCLOUD_REGION must be set for tests")
	}
}
