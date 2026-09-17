package fptcloud_storage_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	common "terraform-provider-fptcloud/commons"
	fptcloud_storage "terraform-provider-fptcloud/fptcloud/storage"
)

const policiesResponse = `{"data":[
	{"id":"policy-db-1","name":"Premium-SSD","isDefault":true,"infra_id":"policy-infra-1"},
	{"id":"policy-db-2","name":"Premium-SSD-3000","isDefault":false,"infra_id":"policy-infra-2"}
]}`

type recordedRequest struct {
	Method string
	Path   string
	Body   map[string]interface{}
}

// newAsyncCreateServer answers the two calls of the async create flow and
// records every request, so a test can assert what was (and was not) sent.
func newAsyncCreateServer(t *testing.T, createStatus int, createBody string) (*common.Client, *[]recordedRequest, func()) {
	t.Helper()
	var requests []recordedRequest
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rec := recordedRequest{Method: req.Method, Path: req.URL.Path}
		if raw, _ := io.ReadAll(req.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &rec.Body)
		}
		requests = append(requests, rec)

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v2/vpc/vpc_id/storage-policies":
			_, _ = rw.Write([]byte(policiesResponse))
		case req.Method == http.MethodPost && req.URL.Path == "/v1/vmware/vpc/vpc_id/create-storage":
			rw.WriteHeader(createStatus)
			_, _ = rw.Write([]byte(createBody))
		default:
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
	client, err := common.NewClientForTestingWithServer(server)
	require.NoError(t, err)
	return client, &requests, server.Close
}

func TestCreateStorageAsync_PostsInfraPolicyIdToAsyncEndpoint(t *testing.T) {
	client, requests, closeFn := newAsyncCreateServer(t, http.StatusCreated,
		`{"status":true,"error_code":null,"data":{"id":"storage-db-id"},"message":"Successfully"}`)
	defer closeFn()

	service := fptcloud_storage.NewStorageService(client)
	storageId, err := service.CreateStorageAsync(fptcloud_storage.StorageDTO{
		VpcId: "vpc_id", Name: "disk-01", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-db-2",
	})

	require.NoError(t, err)
	assert.Equal(t, "storage-db-id", storageId)
	require.Len(t, *requests, 2)
	post := (*requests)[1]
	assert.Equal(t, http.MethodPost, post.Method)
	assert.Equal(t, "/v1/vmware/vpc/vpc_id/create-storage", post.Path)
	assert.Equal(t, "disk-01", post.Body["name"])
	assert.Equal(t, float64(20), post.Body["size"])
	assert.Equal(t, "policy-infra-2", post.Body["storage_policy_id"], "the async endpoint takes the infra policy id, not the portal id")
	assert.Equal(t, "DEFAULT", post.Body["storage_type"])
	assert.Equal(t, "", post.Body["vm_id"])
	assert.Equal(t, "", post.Body["snapshot_id"])
	assert.Equal(t, "", post.Body["description"])
}

func TestCreateStorageAsync_SendsInstanceIdAsVmId(t *testing.T) {
	client, requests, closeFn := newAsyncCreateServer(t, http.StatusCreated,
		`{"status":true,"data":{"id":"storage-db-id"}}`)
	defer closeFn()

	instanceId := "instance-1"
	service := fptcloud_storage.NewStorageService(client)
	_, err := service.CreateStorageAsync(fptcloud_storage.StorageDTO{
		VpcId: "vpc_id", Name: "disk-01", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-db-1", InstanceId: &instanceId,
	})

	require.NoError(t, err)
	require.Len(t, *requests, 2)
	assert.Equal(t, "instance-1", (*requests)[1].Body["vm_id"])
}

func TestCreateStorageAsync_UnknownPolicyFailsWithoutPosting(t *testing.T) {
	client, requests, closeFn := newAsyncCreateServer(t, http.StatusCreated,
		`{"status":true,"data":{"id":"storage-db-id"}}`)
	defer closeFn()

	service := fptcloud_storage.NewStorageService(client)
	_, err := service.CreateStorageAsync(fptcloud_storage.StorageDTO{
		VpcId: "vpc_id", Name: "disk-01", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-missing",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy-missing")
	for _, r := range *requests {
		assert.NotEqual(t, http.MethodPost, r.Method, "no storage may be created when the policy cannot be resolved")
	}
}

func TestCreateStorageAsync_StatusFalseInSuccessfulResponseIsAnError(t *testing.T) {
	client, _, closeFn := newAsyncCreateServer(t, http.StatusOK,
		// data.id is present on purpose: only the status flag may reject this.
		`{"status":false,"error_code":"202099","data":{"id":"storage-db-id"},"message":"Some rejection"}`)
	defer closeFn()

	service := fptcloud_storage.NewStorageService(client)
	_, err := service.CreateStorageAsync(fptcloud_storage.StorageDTO{
		VpcId: "vpc_id", Name: "disk-01", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-db-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Some rejection")
}

func TestCreateStorageAsync_MissingIdIsAnError(t *testing.T) {
	client, _, closeFn := newAsyncCreateServer(t, http.StatusCreated, `{}`)
	defer closeFn()

	service := fptcloud_storage.NewStorageService(client)
	_, err := service.CreateStorageAsync(fptcloud_storage.StorageDTO{
		VpcId: "vpc_id", Name: "disk-01", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-db-1",
	})

	require.Error(t, err)
}

func TestCreateStorageAsync_HttpErrorCarriesApiMessage(t *testing.T) {
	client, _, closeFn := newAsyncCreateServer(t, http.StatusBadRequest,
		`{"status":false,"error_code":"202008","data":null,"message":"Storage policy not found"}`)
	defer closeFn()

	service := fptcloud_storage.NewStorageService(client)
	_, err := service.CreateStorageAsync(fptcloud_storage.StorageDTO{
		VpcId: "vpc_id", Name: "disk-01", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-db-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Storage policy not found")
}

func newFindByNameServer(t *testing.T, status int, body string) (*common.Client, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v2/vpc/vpc_id/storage" || req.URL.Query().Get("name") != "disk-01" {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		rw.WriteHeader(status)
		_, _ = rw.Write([]byte(body))
	}))
	client, err := common.NewClientForTestingWithServer(server)
	require.NoError(t, err)
	return client, server.Close
}

func TestCreateStorageAsync_ResolvesPolicyOncePerVpc(t *testing.T) {
	// The policy list is fetched from the infrastructure and took 9-23 s on
	// production; creating N disks must not pay for it N times.
	client, requests, closeFn := newAsyncCreateServer(t, http.StatusCreated,
		`{"status":true,"data":{"id":"storage-db-id"}}`)
	defer closeFn()

	service := fptcloud_storage.NewStorageService(client)
	for i := 0; i < 3; i++ {
		_, err := service.CreateStorageAsync(fptcloud_storage.StorageDTO{
			VpcId: "vpc_id", Name: "disk", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-db-1",
		})
		require.NoError(t, err)
	}

	policyCalls := 0
	for _, r := range *requests {
		if r.Path == "/v2/vpc/vpc_id/storage-policies" {
			policyCalls++
		}
	}
	assert.Equal(t, 1, policyCalls)
}

func TestCreateStorageAsync_GatewayTimeoutIsReportedAsCreateTimeout(t *testing.T) {
	client, _, closeFn := newAsyncCreateServer(t, http.StatusGatewayTimeout, `upstream timed out`)
	defer closeFn()

	_, err := fptcloud_storage.NewStorageService(client).CreateStorageAsync(fptcloud_storage.StorageDTO{
		VpcId: "vpc_id", Name: "disk-01", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-db-1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, fptcloud_storage.ErrCreateRequestTimeout))
}

func TestCreateStorageAsync_RejectionIsNotReportedAsCreateTimeout(t *testing.T) {
	client, _, closeFn := newAsyncCreateServer(t, http.StatusBadRequest, `{"status":false,"message":"Storage policy not found"}`)
	defer closeFn()

	_, err := fptcloud_storage.NewStorageService(client).CreateStorageAsync(fptcloud_storage.StorageDTO{
		VpcId: "vpc_id", Name: "disk-01", Type: "EXTERNAL", SizeGb: 20, StoragePolicyId: "policy-db-1",
	})

	require.Error(t, err)
	assert.False(t, errors.Is(err, fptcloud_storage.ErrCreateRequestTimeout))
}

func TestLookupStorageByName(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   fptcloud_storage.StorageNameLookup
		errors bool
	}{
		{"found", http.StatusOK, `{"id":"1","name":"disk-01","status":"ENABLED"}`, fptcloud_storage.StorageNameLookup{Found: true, Id: "1", Status: "ENABLED"}, false},
		{"found with null status", http.StatusOK, `{"id":"1","name":"disk-01","status":null}`, fptcloud_storage.StorageNameLookup{Found: true, Id: "1"}, false},
		{"not found (HTTP 500 + 1501002)", http.StatusInternalServerError, `{"message":"{'error_code': '1501002', 'message': 'Storage not found'}"}`, fptcloud_storage.StorageNameLookup{}, false},
		{"other server error", http.StatusInternalServerError, `{"message":"database unavailable"}`, fptcloud_storage.StorageNameLookup{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, closeFn := newFindByNameServer(t, tc.status, tc.body)
			defer closeFn()

			got, err := fptcloud_storage.NewStorageService(client).LookupStorageByName("vpc_id", "disk-01")

			if tc.errors {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
