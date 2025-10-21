package fptcloud_object_storage_test

import (
	"errors"
	"testing"

	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"

	"github.com/stretchr/testify/assert"
)

func TestHandleAPIError(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		err       error
		expected  fptcloud_object_storage.CommonResponse
	}{
		{
			name:      "no error",
			operation: "test operation",
			err:       nil,
			expected: fptcloud_object_storage.CommonResponse{
				Status: true,
			},
		},
		{
			name:      "with error",
			operation: "create bucket",
			err:       errors.New("network timeout"),
			expected: fptcloud_object_storage.CommonResponse{
				Status:  false,
				Message: "failed to create bucket: network timeout",
			},
		},
		{
			name:      "empty operation",
			operation: "",
			err:       errors.New("test error"),
			expected: fptcloud_object_storage.CommonResponse{
				Status:  false,
				Message: "failed to : test error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fptcloud_object_storage.HandleAPIError(tt.operation, tt.err)
			assert.Equal(t, tt.expected.Status, result.Status)
			assert.Equal(t, tt.expected.Message, result.Message)
		})
	}
}

func TestHandleJSONError(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		err       error
		expected  fptcloud_object_storage.CommonResponse
	}{
		{
			name:      "no error",
			operation: "test operation",
			err:       nil,
			expected: fptcloud_object_storage.CommonResponse{
				Status: true,
			},
		},
		{
			name:      "with error",
			operation: "create bucket",
			err:       errors.New("invalid json"),
			expected: fptcloud_object_storage.CommonResponse{
				Status:  false,
				Message: "failed to unmarshal create bucket response: invalid json",
			},
		},
		{
			name:      "empty operation",
			operation: "",
			err:       errors.New("test error"),
			expected: fptcloud_object_storage.CommonResponse{
				Status:  false,
				Message: "failed to unmarshal  response: test error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fptcloud_object_storage.HandleJSONError(tt.operation, tt.err)
			assert.Equal(t, tt.expected.Status, result.Status)
			assert.Equal(t, tt.expected.Message, result.Message)
		})
	}
}
