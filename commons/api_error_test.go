package commons

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiErrorCode_ReadsTheTopLevelErrorCode(t *testing.T) {
	err := HTTPError{Code: 400, Reason: `{"status":false,"error_code":"1503002","message":"Instance not found"}`}
	assert.Equal(t, "1503002", ApiErrorCode(err))
	assert.True(t, IsApiErrorCode(err, "1503002"))
	assert.False(t, IsApiErrorCode(err, "1501002"))
}

func TestApiErrorCode_ReadsTheCodeOfAnUncaughtSystemError(t *testing.T) {
	// an uncaught SystemError answers 500/500000 and keeps its own code inside the message
	err := HTTPError{Code: 500, Reason: `{"status": false, "error_code": "500000", "data": null, "details": null, ` +
		`"message": "{'status': False, 'error_code': '1501002', 'message': 'Storage not found'}. System need to verified, please contact support admin."}`}
	assert.Equal(t, "1501002", ApiErrorCode(err))
}

func TestApiErrorCode_SeesThroughWrapping(t *testing.T) {
	err := fmt.Errorf("reading: %w", HTTPError{Code: 400, Reason: `{"error_code":"1503002"}`})
	assert.Equal(t, "1503002", ApiErrorCode(err))
}

func TestApiErrorCode_IsEmptyWithoutAnApiCode(t *testing.T) {
	for _, err := range []error{
		HTTPError{Code: 403, Reason: `{"error": "403 Forbidden: You don't have the permission"}`},
		HTTPError{Code: 502, Reason: `<html>502 Bad Gateway</html>`},
		HTTPError{Code: 500, Reason: `{"error_code": "500000", "message": "System need to verified"}`},
		errors.New("boom"),
		nil,
	} {
		assert.Equal(t, "", ApiErrorCode(err), fmt.Sprint(err))
	}
}

func TestHttpStatus(t *testing.T) {
	assert.Equal(t, 403, HttpStatus(fmt.Errorf("x: %w", HTTPError{Code: 403})))
	assert.Equal(t, 0, HttpStatus(errors.New("boom")))
}

func TestDescribeApiError(t *testing.T) {
	transport := &url.Error{Op: "Get", URL: "https://api/x", Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}
	cases := map[error]string{
		HTTPError{Code: 401}: "not authorized (HTTP 401): check the provider token",
		HTTPError{Code: 403}: "permission denied (HTTP 403): the token has no access to this resource or VPC",
		HTTPError{Code: 502, Reason: "<html>bad gateway</html>"}:                 "HTTP 502",
		HTTPError{Code: 400, Reason: `{"error_code":"1","message":"Bad thing"}`}: "HTTP 400: Bad thing",
		transport: "could not reach the API: Get \"https://api/x\": dial: connection refused",
	}
	for err, want := range cases {
		assert.Equal(t, want, DescribeApiError(err))
	}
}
