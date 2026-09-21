package fptcloud_object_storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeCommonResponse(t *testing.T) {
	const success = "done"

	tests := []struct {
		name        string
		body        string
		wantStatus  bool
		wantMessage string
	}{
		{
			// The case the hardcoded Status:true used to hide: a 2xx that reports a
			// logical failure in its body.
			name: "explicit false is a failure and keeps the message",
			body: `{"status":false,"message":"bucket is not empty"}`,

			wantStatus: false, wantMessage: "bucket is not empty",
		},
		{
			name: "explicit false without a message still fails",
			body: `{"status":false}`,

			wantStatus: false, wantMessage: "the API reported a failure without a message",
		},
		{
			name: "explicit true is a success",
			body: `{"status":true}`,

			wantStatus: true, wantMessage: success,
		},
		{
			// An absent status must not be read as false, or every endpoint that
			// answers without one would look like a failure.
			name: "absent status is a success",
			body: `{"message":"anything"}`,

			wantStatus: true, wantMessage: success,
		},
		{
			name: "empty body is a success",
			body: ``,

			wantStatus: true, wantMessage: success,
		},
		{
			name: "non-envelope body is a success",
			body: `["not","an","envelope"]`,

			wantStatus: true, wantMessage: success,
		},
		{
			name: "null body is a success",
			body: `null`,

			wantStatus: true, wantMessage: success,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeCommonResponse([]byte(tt.body), success)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, tt.wantMessage, got.Message)
		})
	}
}
