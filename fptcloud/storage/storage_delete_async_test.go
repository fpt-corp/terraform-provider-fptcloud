package fptcloud_storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "terraform-provider-fptcloud/commons"
)

type deleteRequest struct {
	Method string
	Path   string
	Body   string
}

// newDeleteServer answers the platform lookup, the storage read, the queued
// delete and the blocking delete; the storage turns DISABLED once a delete
// of either kind has been received.
func newDeleteServer(t *testing.T, platform string, deleteStatus int, deleteBody string) (*common.Client, func() []deleteRequest) {
	t.Helper()
	var mu sync.Mutex
	var requests []deleteRequest
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		raw, _ := io.ReadAll(req.Body)
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, deleteRequest{Method: req.Method, Path: req.URL.Path, Body: string(raw)})

		switch {
		case req.URL.Path == "/v1/vmware/user/tenants/enabled":
			_, _ = rw.Write([]byte(`{"id": "user-1", "tenants": []}`))
		case req.URL.Path == "/v1/vmware/vpc/vpc-1/user/user-1/vpc_user":
			_, _ = rw.Write([]byte(`{"data": {"user_id": "user-1", "vpc_id": "vpc-1", "platform": "` + platform + `"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/v2/vpc/vpc-1/storage":
			status := "ENABLED"
			if deleted {
				status = storageDisabled
			}
			_, _ = rw.Write([]byte(`{"id": "disk-1", "type": "EXTERNAL", "status": "` + status + `"}`))
		case req.Method == http.MethodPost && req.URL.Path == "/v1/vmware/vpc/vpc-1/storage/disk-1/delete-storage":
			rw.WriteHeader(deleteStatus)
			_, _ = rw.Write([]byte(deleteBody))
			deleted = deleteStatus < 300
		case req.Method == http.MethodDelete && req.URL.Path == "/v2/vpc/vpc-1/storage/disk-1":
			_, _ = rw.Write([]byte(`{"status": true}`))
			deleted = true
		default:
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	client, err := common.NewClientForTestingWithServer(server)
	require.NoError(t, err)
	return client, func() []deleteRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]deleteRequest(nil), requests...)
	}
}

func storageResourceData(t *testing.T, storageType string) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, ResourceStorage().Schema, map[string]interface{}{
		"vpc_id":            "vpc-1",
		"name":              "data-disk",
		"type":              storageType,
		"size_gb":           10,
		"storage_policy_id": "policy-1",
	})
	d.SetId("disk-1")
	return d
}

func deletePaths(requests []deleteRequest, method string) []string {
	var paths []string
	for _, r := range requests {
		if r.Method == method {
			paths = append(paths, r.Path)
		}
	}
	return paths
}

func TestDeleteStorageAsync_PostsToTheConsoleEndpoint(t *testing.T) {
	client, requests := newDeleteServer(t, "osp", http.StatusCreated, `{"status": true, "message": "Delete storage successfully"}`)

	err := NewStorageService(client).DeleteStorageAsync("vpc-1", "disk-1")

	require.NoError(t, err)
	calls := requests()
	require.Len(t, calls, 1)
	assert.Equal(t, http.MethodPost, calls[0].Method)
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/storage/disk-1/delete-storage", calls[0].Path)
	assert.Equal(t, "{}", calls[0].Body)
}

func TestDeleteStorageAsync_Errors(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		refused bool
		message string
	}{
		{"refused with 400", http.StatusBadRequest, `{"status": false, "message": "Unable to delete storage with associated snapshots"}`, true, "associated snapshots"},
		{"refused with status false", http.StatusOK, `{"status": false, "message": "Unable to delete storage"}`, true, "Unable to delete storage"},
		{"server error is retried", http.StatusInternalServerError, `{"status": false, "message": "System error"}`, false, "HTTP 500"},
		{"too many requests is retried", http.StatusTooManyRequests, `{"message": "slow down"}`, false, "HTTP 429"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newDeleteServer(t, "osp", tc.status, tc.body)

			err := NewStorageService(client).DeleteStorageAsync("vpc-1", "disk-1")

			require.Error(t, err)
			assert.Equal(t, tc.refused, errors.Is(err, ErrStorageDeleteRefused))
			assert.Contains(t, err.Error(), tc.message)
		})
	}
}

func TestResourceStorageDelete_ExternalDiskQueuesTheDelete(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	client, requests := newDeleteServer(t, "osp", http.StatusCreated, `{"status": true, "message": "Delete storage successfully"}`)

	diags := resourceStorageDelete(context.Background(), storageResourceData(t, External), client)

	require.False(t, diags.HasError(), "%v", diags)
	calls := requests()
	assert.Equal(t, []string{"/v1/vmware/vpc/vpc-1/storage/disk-1/delete-storage"}, deletePaths(calls, http.MethodPost))
	assert.Empty(t, deletePaths(calls, http.MethodDelete), "the blocking delete must not be used")
}

func TestResourceStorageDelete_VmwKeepsTheBlockingDelete(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	client, requests := newDeleteServer(t, "vmw", http.StatusCreated, `{"status": true}`)

	diags := resourceStorageDelete(context.Background(), storageResourceData(t, External), client)

	require.False(t, diags.HasError(), "%v", diags)
	calls := requests()
	assert.Equal(t, []string{"/v2/vpc/vpc-1/storage/disk-1"}, deletePaths(calls, http.MethodDelete))
	assert.Empty(t, deletePaths(calls, http.MethodPost))
}

func TestResourceStorageDelete_UnknownPlatformKeepsTheBlockingDelete(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	client, requests := newDeleteServer(t, "", http.StatusCreated, `{"status": true}`)
	// no tenant answer: the platform lookup fails
	client.BaseURL.Path = "/missing"

	diags := resourceStorageDelete(context.Background(), storageResourceData(t, External), client)

	assert.True(t, diags.HasError(), "the blocking delete is sent to the same broken base URL")
	assert.Empty(t, deletePaths(requests(), http.MethodPost), "the queued delete must not be used without a platform")
}

func TestResourceStorageDelete_LocalDiskKeepsTheBlockingDelete(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	client, requests := newDeleteServer(t, "osp", http.StatusCreated, `{"status": true}`)

	diags := resourceStorageDelete(context.Background(), storageResourceData(t, Local), client)

	require.False(t, diags.HasError(), "%v", diags)
	calls := requests()
	assert.Equal(t, []string{"/v2/vpc/vpc-1/storage/disk-1"}, deletePaths(calls, http.MethodDelete))
	assert.Empty(t, deletePaths(calls, http.MethodPost))
	for _, call := range calls {
		assert.NotContains(t, call.Path, "vpc_user", "the platform is only read for external disks")
	}
}
