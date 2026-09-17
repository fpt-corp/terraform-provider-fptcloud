package fptcloud_storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	common "terraform-provider-fptcloud/commons"
)

const storageNotFoundBody = `{"status":false,"error_code":"500000","message":"{'status': False, 'error_code': '1501002', 'message': 'Storage not found'}"}`

// fakeStorageApi plays the API for one create: the name lookup answers
// "not found" until the create POST has been received and nameHitsBeforeRow
// more lookups have happened, then the new row.
type fakeStorageApi struct {
	mu                sync.Mutex
	createStatus      int
	createBody        string
	nameHitsBeforeRow int
	posted            bool
	nameHitsAfterPost int
	posts             int
}

func (f *fakeStorageApi) handler(rw http.ResponseWriter, req *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case req.Method == http.MethodGet && req.URL.Path == "/v2/vpc/vpc_id/storage-policies":
		_, _ = rw.Write([]byte(`{"data":[{"id":"policy-db","infra_id":"policy-infra"}]}`))
	case req.Method == http.MethodPost && req.URL.Path == "/v1/vmware/vpc/vpc_id/create-storage":
		f.posts++
		f.posted = true
		rw.WriteHeader(f.createStatus)
		_, _ = rw.Write([]byte(f.createBody))
	case req.Method == http.MethodGet && req.URL.Path == "/v2/vpc/vpc_id/storage" && req.URL.Query().Get("name") != "":
		if f.posted {
			f.nameHitsAfterPost++
		}
		if !f.posted || f.nameHitsAfterPost <= f.nameHitsBeforeRow {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(storageNotFoundBody))
			return
		}
		_, _ = rw.Write([]byte(`{"id":"new-id","name":"disk-01","status":null}`))
	case req.Method == http.MethodGet && req.URL.Path == "/v2/vpc/vpc_id/storage" && req.URL.Query().Get("id") == "new-id":
		_, _ = rw.Write([]byte(`{"id":"new-id","name":"disk-01","type":"EXTERNAL","size_gb":10,"status":"ENABLED","storage_policy_id":"policy-db","vpc_id":"vpc_id"}`))
	default:
		rw.WriteHeader(http.StatusNotFound)
	}
}

func runCreate(t *testing.T, api *fakeStorageApi, lookupTimeout time.Duration) (*schema.ResourceData, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(api.handler))
	t.Cleanup(server.Close)
	client, err := common.NewClientForTestingWithServer(server)
	require.NoError(t, err)

	previous := createLookupInterval
	createLookupInterval = 5 * time.Millisecond
	t.Cleanup(func() { createLookupInterval = previous })

	// ResourceData has no timeout setter: Resource.Data copies the resource's
	// Timeouts, so give it one configured like a user's timeouts block.
	resource := ResourceStorage()
	resource.Timeouts = &schema.ResourceTimeout{Create: &lookupTimeout}
	d := resource.Data(nil)
	for k, v := range map[string]interface{}{
		"vpc_id": "vpc_id", "name": "disk-01", "type": "EXTERNAL", "size_gb": 10, "storage_policy_id": "policy-db",
	} {
		require.NoError(t, d.Set(k, v))
	}
	require.Equal(t, lookupTimeout, d.Timeout(schema.TimeoutCreate))

	diags := resourceStorageCreate(context.Background(), d, client)
	if diags.HasError() {
		return d, &diagError{summary: diags[0].Summary}
	}
	return d, nil
}

type diagError struct{ summary string }

func (e *diagError) Error() string { return e.summary }

func TestResourceStorageCreate_GatewayTimeoutFallsBackToNameLookup(t *testing.T) {
	api := &fakeStorageApi{createStatus: http.StatusGatewayTimeout, createBody: "upstream timed out", nameHitsBeforeRow: 2}

	d, err := runCreate(t, api, time.Second)

	require.NoError(t, err)
	assert.Equal(t, "new-id", d.Id())
	assert.Equal(t, 1, api.posts, "a timed-out create must not be re-sent")
}

func TestResourceStorageCreate_GatewayTimeoutFailsWhenNoStorageAppears(t *testing.T) {
	api := &fakeStorageApi{createStatus: http.StatusBadGateway, createBody: "invalid upstream response", nameHitsBeforeRow: 1 << 30}

	d, err := runCreate(t, api, 60*time.Millisecond)

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no new storage appeared"), err.Error())
	assert.Equal(t, "", d.Id())
	assert.Equal(t, 1, api.posts)
}

func TestResourceStorageCreate_RejectionDoesNotLookUpByName(t *testing.T) {
	api := &fakeStorageApi{createStatus: http.StatusBadRequest, createBody: `{"status":false,"message":"Storage policy not found"}`}

	_, err := runCreate(t, api, time.Second)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Storage policy not found")
	assert.Equal(t, 0, api.nameHitsAfterPost, "a real rejection must fail at once, not wait for a storage")
}

func TestResourceStorage_CreateLookupTimeoutDefaultsTo15Minutes(t *testing.T) {
	timeouts := ResourceStorage().Timeouts
	require.NotNil(t, timeouts)
	require.NotNil(t, timeouts.Create)
	assert.Equal(t, 15*time.Minute, *timeouts.Create)
}
