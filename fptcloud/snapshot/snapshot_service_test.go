package fptcloud_snapshot_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	common "terraform-provider-fptcloud/commons"
	fptcloud_snapshot "terraform-provider-fptcloud/fptcloud/snapshot"

	"github.com/stretchr/testify/assert"
)

const (
	vpcId      = "vpc-1"
	instanceId = "instance-1"
	snapshotId = "snapshot-1"

	collectionPath = "/v2/vmware/vpc/vpc-1/instance-snapshots"
	detailPath     = "/v2/vmware/vpc/vpc-1/instance-snapshots/snapshot-1"

	pendingSnapshot = `{"status": true, "data": {
		"id": "snapshot-1",
		"vpc_id": "vpc-1",
		"vm_id": "instance-1",
		"name": "before-upgrade",
		"status": "CREATING",
		"infra_snapshot_id": null
	}}`

	settledSnapshot = `{"status": true, "data": {
		"id": "snapshot-1",
		"vpc_id": "vpc-1",
		"vm_id": "instance-1",
		"vm_name": "server",
		"name": "before-upgrade",
		"status": "ACTIVE",
		"snapshot_type": "image_snapshot",
		"size_gb": 40.0,
		"infra_snapshot_id": "11111111-1111-1111-1111-111111111111",
		"created_by": "someone@fpt.com",
		"created_at": "2026-09-04T00:00:00",
		"volume": [{"snapshot_id": "vol-snap-1", "volume_id": "vol-1", "is_root": true, "volume_size": 40}]
	}}`
)

// newClientWithStatus serves one body and status code for every request, which
// NewClientForTesting cannot do — it always answers 200.
func newClientWithStatus(t *testing.T, status int, body string) (*common.Client, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(status)
		_, _ = rw.Write([]byte(body))
	}))

	client, err := common.NewClientForTestingWithServer(server)
	assert.NoError(t, err)

	return client, server
}

func TestCreateSnapshot_ReturnsTheRecordedSnapshot(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		collectionPath: pendingSnapshot,
	})
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	snapshot, err := service.CreateSnapshot(vpcId, fptcloud_snapshot.CreateSnapshotDTO{
		InstanceId:   instanceId,
		SnapshotName: "before-upgrade",
	})

	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.Equal(t, "snapshot-1", snapshot.Id)
	assert.Equal(t, "CREATING", snapshot.Status)
}

// The identifier is what makes the snapshot a resource; without it there is
// nothing to record and the caller must be told rather than left polling.
func TestCreateSnapshot_ReturnsErrorWhenNoSnapshotComesBack(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		collectionPath: `{"status": true, "message": "accepted"}`,
	})
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	snapshot, err := service.CreateSnapshot(vpcId, fptcloud_snapshot.CreateSnapshotDTO{
		InstanceId:   instanceId,
		SnapshotName: "before-upgrade",
	})

	assert.Error(t, err)
	assert.Nil(t, snapshot)
}

// The API's own error code survives, rather than being flattened into the
// generic message common.DecodeError produces for anything but a 400.
func TestCreateSnapshot_KeepsTheApiErrorCode(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		errorCode string
	}{
		{"quota exceeded", 429, `{"status": false, "error_code": "QUOTA_ERROR", "message": "CREATE_FAIL"}`, "QUOTA_ERROR"},
		{"name taken", 409, `{"status": false, "error_code": "NAME_ERROR", "message": "CREATE_FAIL"}`, "NAME_ERROR"},
		{"instance busy", 409, `{"status": false, "error_code": "102528", "message": "Another operation is in progress"}`, "102528"},
		{"bad vm state", 409, `{"status": false, "error_code": "102529", "message": "Instance is not in a valid state"}`, "102529"},
		{"instance missing", 404, `{"status": false, "error_code": "180004", "message": "Instance not found"}`, "180004"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, server := newClientWithStatus(t, tc.status, tc.body)
			defer server.Close()

			service := fptcloud_snapshot.NewSnapshotService(client)
			_, err := service.CreateSnapshot(vpcId, fptcloud_snapshot.CreateSnapshotDTO{
				InstanceId:   instanceId,
				SnapshotName: "before-upgrade",
			})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errorCode)
		})
	}
}

func TestGetSnapshot_ReturnsSnapshot(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{detailPath: settledSnapshot})
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	snapshot, err := service.GetSnapshot(vpcId, snapshotId)

	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.Equal(t, "snapshot-1", snapshot.Id)
	assert.Equal(t, "before-upgrade", snapshot.Name)
	assert.Equal(t, "ACTIVE", snapshot.Status)
	assert.Equal(t, 40.0, snapshot.SizeGb)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", snapshot.InfraSnapshotId)
	assert.Len(t, snapshot.Volume, 1)
	assert.True(t, snapshot.Volume[0].IsRoot)
}

// A snapshot that is gone is not an error: Read turns this into a cleared
// resource id so the next plan recreates it.
func TestGetSnapshot_ReturnsNilOn404(t *testing.T) {
	client, server := newClientWithStatus(t, 404, `{"status": false, "error_code": "SNAPSHOT_NOT_FOUND"}`)
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	snapshot, err := service.GetSnapshot(vpcId, snapshotId)

	assert.NoError(t, err)
	assert.Nil(t, snapshot)
}

// Any other failure must surface: treating it as "gone" would silently delete
// the resource from state.
func TestGetSnapshot_ReturnsErrorOnServerFailure(t *testing.T) {
	client, server := newClientWithStatus(t, 500, `{"status": false, "error_code": "INTERNAL_ERROR"}`)
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	snapshot, err := service.GetSnapshot(vpcId, snapshotId)

	assert.Error(t, err)
	assert.Nil(t, snapshot)
	assert.Contains(t, err.Error(), "INTERNAL_ERROR")
}

func TestGetSnapshot_ReturnsErrorOnMalformedResponse(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{detailPath: `not json`})
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	snapshot, err := service.GetSnapshot(vpcId, snapshotId)

	assert.Error(t, err)
	assert.Nil(t, snapshot)
}

func TestDeleteSnapshot_ReturnsNoErrorWhenAccepted(t *testing.T) {
	client, server := newClientWithStatus(t, 202, `{"status": true, "message": "Instance snapshot is being deleted"}`)
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)

	assert.NoError(t, service.DeleteSnapshot(vpcId, snapshotId))
}

// Destroying something that is already gone has reached the desired state.
func TestDeleteSnapshot_TreatsA404AsSuccess(t *testing.T) {
	client, server := newClientWithStatus(t, 404, `{"status": false, "error_code": "SNAPSHOT_NOT_FOUND"}`)
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)

	assert.NoError(t, service.DeleteSnapshot(vpcId, snapshotId))
}

func TestDeleteSnapshot_ReturnsErrorWhenRefused(t *testing.T) {
	client, server := newClientWithStatus(t, 409, `{"status": false, "error_code": "SNAPSHOT_DELETE_REFUSED", "message": "attached to an image"}`)
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	err := service.DeleteSnapshot(vpcId, snapshotId)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SNAPSHOT_DELETE_REFUSED")
}

// The create body has to carry everything both platforms need: the name and
// tags OpenStack uses, and the memory flag VMware uses.
func TestCreateSnapshot_SendsEveryPlatformField(t *testing.T) {
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, _ = io.ReadAll(req.Body)
		_, _ = rw.Write([]byte(pendingSnapshot))
	}))
	defer server.Close()

	client, err := common.NewClientForTestingWithServer(server)
	assert.NoError(t, err)

	service := fptcloud_snapshot.NewSnapshotService(client)
	_, err = service.CreateSnapshot(vpcId, fptcloud_snapshot.CreateSnapshotDTO{
		InstanceId:   instanceId,
		SnapshotName: "before-upgrade",
		IncludeRam:   true,
		TagIds:       []string{"tag-1"},
	})

	assert.NoError(t, err)

	var sent map[string]interface{}
	assert.NoError(t, json.Unmarshal(body, &sent))
	assert.Equal(t, "instance-1", sent["instance_id"])
	assert.Equal(t, "before-upgrade", sent["snapshot_name"])
	assert.Equal(t, true, sent["include_ram"])
	assert.Equal(t, []interface{}{"tag-1"}, sent["tag_ids"])
}

// A VMware snapshot has no name and needs no tags, so the optional fields drop
// out of the body rather than being sent as empty values.
func TestCreateSnapshot_OmitsFieldsThatWereNotSet(t *testing.T) {
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, _ = io.ReadAll(req.Body)
		_, _ = rw.Write([]byte(pendingSnapshot))
	}))
	defer server.Close()

	client, err := common.NewClientForTestingWithServer(server)
	assert.NoError(t, err)

	service := fptcloud_snapshot.NewSnapshotService(client)
	_, err = service.CreateSnapshot(vpcId, fptcloud_snapshot.CreateSnapshotDTO{
		InstanceId: instanceId,
	})

	assert.NoError(t, err)

	var sent map[string]interface{}
	assert.NoError(t, json.Unmarshal(body, &sent))
	assert.Equal(t, "instance-1", sent["instance_id"])
	assert.NotContains(t, sent, "snapshot_name")
	assert.NotContains(t, sent, "include_ram")
	assert.NotContains(t, sent, "tag_ids")
}

func TestGetSnapshot_ReadsTags(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		detailPath: `{"status": true, "data": {"id": "snapshot-1", "tags": [{"id": "tag-1", "key": "env", "value": "prod"}]}}`,
	})
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	snapshot, err := service.GetSnapshot(vpcId, snapshotId)

	assert.NoError(t, err)
	assert.Len(t, snapshot.Tags, 1)
	assert.Equal(t, "env", snapshot.Tags[0].Key)
}

// An empty 200 is a broken response, not a deleted snapshot. Reading it as
// "gone" would quietly drop the resource from Terraform state.
func TestGetSnapshot_ReturnsErrorOnAnEmptyBody(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		detailPath: `{"status": true}`,
	})
	defer server.Close()

	service := fptcloud_snapshot.NewSnapshotService(client)
	snapshot, err := service.GetSnapshot(vpcId, snapshotId)

	assert.Error(t, err)
	assert.Nil(t, snapshot)
}
