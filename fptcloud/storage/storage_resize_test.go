package fptcloud_storage_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "terraform-provider-fptcloud/commons"
	fptcloud_storage "terraform-provider-fptcloud/fptcloud/storage"
)

func resizeServer(t *testing.T, status int, body string) (*common.Client, *httptest.Server, *[]recordedCall) {
	t.Helper()
	calls := &[]recordedCall{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		*calls = append(*calls, recordedCall{Method: r.Method, Path: r.URL.Path, Body: raw})
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))

	client, err := common.NewClientWithURL("token", server.URL, "VN/SGN", "tenant", 1)
	require.NoError(t, err)
	parsed, err := url.Parse(server.URL)
	require.NoError(t, err)
	client.BaseURL = parsed
	return client, server, calls
}

type recordedCall struct {
	Method string
	Path   string
	Body   []byte
}

func TestResizeStorage_PostsPortalPayload(t *testing.T) {
	client, server, calls := resizeServer(t, http.StatusOK, `{"status": true, "message": "ok"}`)
	defer server.Close()

	service := fptcloud_storage.NewStorageService(client)
	res, err := service.ResizeStorage("vpc-1", fptcloud_storage.ResizeStorageDTO{
		DiskId:          "disk-1",
		Size:            20,
		Name:            "storage-disk-2563",
		StoragePolicyId: "policy-portal-id",
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	require.Len(t, *calls, 1)

	call := (*calls)[0]
	assert.Equal(t, http.MethodPost, call.Method)
	assert.Equal(t, "/v1/vmware/vpc/vpc-1/resize-storage", call.Path)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(call.Body, &body))
	assert.Equal(t, map[string]interface{}{
		"disk_id":         "disk-1",
		"size":            float64(20),
		"name":            "storage-disk-2563",
		"storagePolicyId": "policy-portal-id",
	}, body)
}

func TestResizeStorage_ErrorsOnStatusFalse(t *testing.T) {
	client, server, _ := resizeServer(t, http.StatusOK,
		`{"status": false, "message": "size must be greater than current size"}`)
	defer server.Close()

	res, err := fptcloud_storage.NewStorageService(client).ResizeStorage("vpc-1",
		fptcloud_storage.ResizeStorageDTO{DiskId: "disk-1", Size: 5})

	assert.Nil(t, res)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "size must be greater than current size")
}

func TestResizeStorage_ErrorsOnHTTPError(t *testing.T) {
	client, server, _ := resizeServer(t, http.StatusInternalServerError,
		`{"status": false, "message": "change volume_type failed !!!"}`)
	defer server.Close()

	res, err := fptcloud_storage.NewStorageService(client).ResizeStorage("vpc-1",
		fptcloud_storage.ResizeStorageDTO{DiskId: "disk-1", Size: 20})

	assert.Nil(t, res)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "change volume_type failed")
}

func TestGetVpcPlatform_ReadsPlatformAndCaches(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1/vmware/user/tenants/enabled":
			_, _ = w.Write([]byte(`{"id": "user-1", "tenants": []}`))
		case "/v1/vmware/vpc/vpc-platform-1/user/user-1/vpc_user":
			_, _ = w.Write([]byte(`{"data": {"user_id": "user-1", "vpc_id": "vpc-platform-1", "platform": "osp"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := common.NewClientWithURL("token", server.URL, "VN/SGN", "tenant", 1)
	require.NoError(t, err)
	parsed, err := url.Parse(server.URL)
	require.NoError(t, err)
	client.BaseURL = parsed

	platform, err := fptcloud_storage.GetVpcPlatform(context.Background(), client, "vpc-platform-1")
	assert.NoError(t, err)
	assert.Equal(t, fptcloud_storage.PlatformOsp, platform)
	assert.Equal(t, []string{
		"/v1/vmware/user/tenants/enabled",
		"/v1/vmware/vpc/vpc-platform-1/user/user-1/vpc_user",
	}, paths)

	platform, err = fptcloud_storage.GetVpcPlatform(context.Background(), client, "vpc-platform-1")
	assert.NoError(t, err)
	assert.Equal(t, fptcloud_storage.PlatformOsp, platform)
	assert.Len(t, paths, 2, "the platform lookup should be cached")
}
