package fptcloud_object_storage_test

import (
	"testing"

	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"

	"github.com/stretchr/testify/assert"
)

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		// Valid bucket names
		{"valid simple name", "mybucket", false},
		{"valid with numbers", "mybucket123", false},
		{"valid with hyphens", "my-bucket", false},
		{"valid with dots", "my.bucket", false},
		{"valid mixed", "my-bucket.123", false},
		{"valid minimum length", "abc", false},
		{"valid maximum length", "a123456789012345678901234567890123456789012345678901234567890bc", false},

		// Invalid bucket names
		{"too short", "ab", true},
		{"too long", "a1234567890123456789012345678901234567890123456789012345678901234", true},
		{"starts with uppercase", "MyBucket", true},
		{"starts with hyphen", "-mybucket", true},
		{"starts with dot", ".mybucket", true},
		{"ends with hyphen", "mybucket-", true},
		{"ends with dot", "mybucket.", true},
		{"contains uppercase", "myBucket", true},
		{"contains underscore", "my_bucket", true},
		{"consecutive dots", "my..bucket", true},
		{"consecutive hyphens", "my--bucket", false}, // This is actually valid
		{"ip address format", "192.168.1.1", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := fptcloud_object_storage.ValidateBucketName(tt.input, "name")

			if tt.hasError {
				assert.NotEmpty(t, errors, "Expected validation error for input: %s", tt.input)
			} else {
				assert.Empty(t, errors, "Expected no validation errors for input: %s, got: %v", tt.input, errors)
			}

			// Warnings should be empty for all cases
			assert.Empty(t, warnings, "Expected no warnings for input: %s", tt.input)
		})
	}
}

func TestValidateRegionName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		// Valid regions
		{"valid HCM-01", "HCM-01", false},
		{"valid HCM-02", "HCM-02", false},
		{"valid HN-01", "HN-01", false},
		{"valid HN-02", "HN-02", false},

		// Invalid regions
		{"invalid region", "INVALID", true},
		{"lowercase", "hcm-01", true},
		{"empty string", "", true},
		{"partial match", "HCM", true},
		{"extra characters", "HCM-01-extra", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := fptcloud_object_storage.ValidateRegionName(tt.input, "region_name")

			if tt.hasError {
				assert.NotEmpty(t, errors, "Expected validation error for input: %s", tt.input)
			} else {
				assert.Empty(t, errors, "Expected no validation errors for input: %s, got: %v", tt.input, errors)
			}

			// Warnings should be empty for all cases
			assert.Empty(t, warnings, "Expected no warnings for input: %s", tt.input)
		})
	}
}
