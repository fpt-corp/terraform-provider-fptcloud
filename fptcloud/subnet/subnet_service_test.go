package fptcloud_subnet

import (
	"net/http"
	"net/http/httptest"
	"testing"

	common "terraform-provider-fptcloud/commons"

	"github.com/stretchr/testify/assert"
)

func newSubnetServiceForTesting(t *testing.T, status int, body string) SubnetService {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(status)
		_, _ = rw.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client, err := common.NewClientForTestingWithServer(server)
	assert.NoError(t, err)
	return NewSubnetService(client)
}

// The create call returns before the worker has recorded the subnet, so the
// wait must treat "not found yet" as pending instead of failing the apply.
func TestFindSubnetByName_NotRecordedYetIsNil(t *testing.T) {
	service := newSubnetServiceForTesting(t, http.StatusBadRequest,
		`{"status":false,"error_code":"101006","data":null,"message":"The selected network no longer exists."}`)

	subnet, err := service.FindSubnetByName(FindSubnetDTO{VpcId: "vpc-1", NetworkName: "sn-abc"})

	assert.NoError(t, err)
	assert.Nil(t, subnet)
}

func TestFindSubnetByName_OtherBadRequestIsAnError(t *testing.T) {
	service := newSubnetServiceForTesting(t, http.StatusBadRequest,
		`{"status":false,"error_code":"999999","data":null,"message":"boom"}`)

	subnet, err := service.FindSubnetByName(FindSubnetDTO{VpcId: "vpc-1", NetworkName: "sn-abc"})

	assert.Error(t, err)
	assert.Nil(t, subnet)
}

func TestFindSubnetByName_ServerErrorIsAnError(t *testing.T) {
	service := newSubnetServiceForTesting(t, http.StatusInternalServerError,
		`{"status":false,"error_code":"101006"}`)

	_, err := service.FindSubnetByName(FindSubnetDTO{VpcId: "vpc-1", NetworkName: "sn-abc"})

	assert.Error(t, err)
}

func TestFindSubnetByName_Found(t *testing.T) {
	service := newSubnetServiceForTesting(t, http.StatusOK,
		`{"status":true,"data":{"id":"subnet-1","name":"sn","network_name":"sn-abc"}}`)

	subnet, err := service.FindSubnetByName(FindSubnetDTO{VpcId: "vpc-1", NetworkName: "sn-abc"})

	assert.NoError(t, err)
	assert.Equal(t, "subnet-1", subnet.ID)
}
