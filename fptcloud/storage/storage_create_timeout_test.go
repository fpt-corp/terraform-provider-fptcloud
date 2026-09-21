package fptcloud_storage

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	common "terraform-provider-fptcloud/commons"
)

type timeoutNetErr struct{}

func (timeoutNetErr) Error() string   { return "i/o timeout" }
func (timeoutNetErr) Timeout() bool   { return true }
func (timeoutNetErr) Temporary() bool { return true }

func TestIsCreateRequestTimeout(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"client timeout", &url.Error{Op: "Post", URL: "x", Err: timeoutNetErr{}}, true},
		{"gateway 504", common.HTTPError{Code: http.StatusGatewayTimeout}, true},
		{"gateway 502 (measured on production 2026-09-17)", common.HTTPError{Code: http.StatusBadGateway}, true},
		{"validation 400 is a real rejection", common.HTTPError{Code: http.StatusBadRequest}, false},
		{"server 500 is a real failure", common.HTTPError{Code: http.StatusInternalServerError}, false},
		{"other error", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isCreateRequestTimeout(tc.err))
		})
	}
}

type lookupStep struct {
	result StorageNameLookup
	err    error
}

func scriptedLookup(steps ...lookupStep) (func() (StorageNameLookup, error), *int) {
	calls := 0
	return func() (StorageNameLookup, error) {
		step := steps[len(steps)-1]
		if calls < len(steps) {
			step = steps[calls]
		}
		calls++
		return step.result, step.err
	}, &calls
}

func TestWaitForNewStorageByName_ReturnsNewIdOnceItAppears(t *testing.T) {
	lookup, calls := scriptedLookup(
		lookupStep{result: StorageNameLookup{}},
		lookupStep{result: StorageNameLookup{Found: true, Id: "new-id"}},
	)

	id, err := waitForNewStorageByName(context.Background(), lookup, "", time.Second, time.Millisecond)

	require.NoError(t, err)
	assert.Equal(t, "new-id", id)
	assert.Equal(t, 2, *calls)
}

func TestWaitForNewStorageByName_IgnoresTheStorageThatExistedBeforeThePost(t *testing.T) {
	// A deleted storage keeps its row (DISABLED) and the lookup answers the
	// latest row with that name, so the old id must not be adopted.
	lookup, _ := scriptedLookup(
		lookupStep{result: StorageNameLookup{Found: true, Id: "old-id", Status: "DISABLED"}},
		lookupStep{result: StorageNameLookup{Found: true, Id: "new-id"}},
	)

	id, err := waitForNewStorageByName(context.Background(), lookup, "old-id", time.Second, time.Millisecond)

	require.NoError(t, err)
	assert.Equal(t, "new-id", id)
}

func TestWaitForNewStorageByName_AcceptsANewRowAlreadyDisabledBySync(t *testing.T) {
	// Measured on production: a concurrent sync marks an in-flight row
	// DISABLED before it becomes ENABLED. It is still the new storage.
	lookup, _ := scriptedLookup(lookupStep{result: StorageNameLookup{Found: true, Id: "new-id", Status: "DISABLED"}})

	id, err := waitForNewStorageByName(context.Background(), lookup, "old-id", time.Second, time.Millisecond)

	require.NoError(t, err)
	assert.Equal(t, "new-id", id)
}

func TestWaitForNewStorageByName_KeepsPollingThroughLookupErrors(t *testing.T) {
	lookup, _ := scriptedLookup(
		lookupStep{err: errors.New("HTTP 502")},
		lookupStep{result: StorageNameLookup{Found: true, Id: "new-id"}},
	)

	id, err := waitForNewStorageByName(context.Background(), lookup, "", time.Second, time.Millisecond)

	require.NoError(t, err)
	assert.Equal(t, "new-id", id)
}

func TestWaitForNewStorageByName_FailsAfterTimeoutWhenNoNewStorage(t *testing.T) {
	lookup, calls := scriptedLookup(lookupStep{result: StorageNameLookup{Found: true, Id: "old-id"}})

	start := time.Now()
	_, err := waitForNewStorageByName(context.Background(), lookup, "old-id", 50*time.Millisecond, 5*time.Millisecond)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "50ms")
	assert.Less(t, time.Since(start), time.Second)
	assert.Greater(t, *calls, 1)
}
