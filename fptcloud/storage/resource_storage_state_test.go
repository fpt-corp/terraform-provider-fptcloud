package fptcloud_storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateWaitState(t *testing.T) {
	cases := []struct {
		name         string
		storage      Storage
		wantInstance string
		want         string
	}{
		{"null status right after the async create is still creating", Storage{Status: ""}, "", "CREATING"},
		{"disabled placeholder is still creating", Storage{Status: "DISABLED"}, "", "DISABLED"},
		{"enabled without instance is done", Storage{Status: "ENABLED"}, "", "ENABLED"},
		{"enabled and attached to the wanted instance is done", Storage{Status: "ENABLED", InstanceId: "vm-1"}, "vm-1", "ENABLED"},
		{"enabled but not yet attached keeps waiting", Storage{Status: "ENABLED"}, "vm-1", "ATTACHING"},
		{"enabled but attached elsewhere keeps waiting", Storage{Status: "ENABLED", InstanceId: "vm-2"}, "vm-1", "ATTACHING"},
		{"unknown status is passed through so the wait fails on it", Storage{Status: "ERROR"}, "", "ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.storage
			assert.Equal(t, tc.want, createWaitState(&s, tc.wantInstance))
		})
	}
}
