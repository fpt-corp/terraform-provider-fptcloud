package fptcloud_subnet

import (
	"errors"
	"testing"

	common "terraform-provider-fptcloud/commons"
)

func fakeGet(responses ...func() ([]byte, error)) func(string) ([]byte, error) {
	calls := 0
	return func(string) ([]byte, error) {
		next := responses[calls]
		calls++
		return next()
	}
}

func notFoundYet() ([]byte, error) {
	return nil, common.HTTPError{Code: 400, Reason: `{"status": false, "error_code": "101006", "message": "The selected network no longer exists."}`}
}

func TestSubnetCreateRefreshWaitsWhileTheTaskHasNotWrittenTheSubnet(t *testing.T) {
	refresh := subnetCreateRefresh(fakeGet(notFoundYet), FindSubnetDTO{NetworkName: "sn-abc", VpcId: "vpc"})

	_, state, err := refresh()

	if err != nil || state != "PENDING" {
		t.Fatalf("want PENDING without error, got state=%q err=%v", state, err)
	}
}

func TestSubnetCreateRefreshCompletesOnceTheSubnetIsFound(t *testing.T) {
	found := func() ([]byte, error) {
		return []byte(`{"status": true, "data": {"id": "net-1", "name": "sn"}}`), nil
	}
	refresh := subnetCreateRefresh(fakeGet(found), FindSubnetDTO{NetworkName: "sn-abc", VpcId: "vpc"})

	result, state, err := refresh()

	if err != nil || state != "COMPLETE" || result.(*Subnet).ID != "net-1" {
		t.Fatalf("want COMPLETE with net-1, got state=%q result=%v err=%v", state, result, err)
	}
}

func TestSubnetCreateRefreshStopsOnAnyOtherError(t *testing.T) {
	forbidden := func() ([]byte, error) {
		return nil, common.HTTPError{Code: 403, Reason: `{"status": false, "error_code": "403000", "message": "Forbidden"}`}
	}
	refresh := subnetCreateRefresh(fakeGet(forbidden), FindSubnetDTO{NetworkName: "sn-abc", VpcId: "vpc"})

	_, _, err := refresh()

	if err == nil {
		t.Fatal("want an error for a 403")
	}
}

func TestSubnetCreateRefreshStopsOnANetworkFailure(t *testing.T) {
	broken := func() ([]byte, error) { return nil, errors.New("connection reset") }
	refresh := subnetCreateRefresh(fakeGet(broken), FindSubnetDTO{NetworkName: "sn-abc", VpcId: "vpc"})

	_, _, err := refresh()

	if err == nil {
		t.Fatal("want an error when the request itself fails")
	}
}
