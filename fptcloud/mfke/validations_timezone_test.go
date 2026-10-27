package fptcloud_mfke

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validateTimezone(value types.String) *validator.StringResponse {
	response := &validator.StringResponse{}
	autoUpgradeTimezoneValidator{}.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("auto_upgrade_timezone"),
		ConfigValue: value,
	}, response)
	return response
}

func TestAutoUpgradeTimezoneValidator(t *testing.T) {
	for _, tz := range autoUpgradeTimezones {
		assert.False(t, validateTimezone(types.StringValue(tz)).Diagnostics.HasError(), tz)
	}

	// Null, unknown and empty are left to the other checks.
	assert.False(t, validateTimezone(types.StringNull()).Diagnostics.HasError())
	assert.False(t, validateTimezone(types.StringUnknown()).Diagnostics.HasError())
	assert.False(t, validateTimezone(types.StringValue("")).Diagnostics.HasError())

	// Valid IANA zones the API still rejects, as the docs used to suggest.
	for _, tz := range []string{"Asia/Bangkok", "Asia/Ho_Chi_Minh", "asia/saigon"} {
		response := validateTimezone(types.StringValue(tz))
		require.True(t, response.Diagnostics.HasError(), tz)
		assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "Asia/Saigon")
	}
}

// TestAutoUpgradeTimezoneIsValidatedBySchema makes sure the validator is
// attached to the resource schema, so a bad value fails at plan time.
func TestAutoUpgradeTimezoneIsValidatedBySchema(t *testing.T) {
	response := &resource.SchemaResponse{}
	(&resourceManagedKubernetesEngine{}).Schema(context.Background(), resource.SchemaRequest{}, response)

	attribute, ok := response.Schema.Attributes["auto_upgrade_timezone"]
	require.True(t, ok)
	validators := attribute.(interface{ StringValidators() []validator.String }).StringValidators()
	require.Len(t, validators, 1)
	assert.IsType(t, autoUpgradeTimezoneValidator{}, validators[0])
}
