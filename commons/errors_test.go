package commons

import (
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimeoutError_WrapsInnerError(t *testing.T) {
	innerErr := errors.New("inner error")
	err := TimeoutError.Wrap(innerErr)
	assert.Equal(t, "TimeoutError: inner error", err.Error())
	assert.True(t, errors.Is(err, TimeoutError))
	assert.True(t, errors.Is(err, innerErr))
}

func TestTimeoutError_WrapsString(t *testing.T) {
	err := TimeoutError.WrapString("wrapped error")
	assert.Equal(t, "TimeoutError: wrapped error", err.Error())
	assert.True(t, errors.Is(err, TimeoutError))
}

func TestDecodeError_HandlesUrlError(t *testing.T) {
	urlErr := &url.Error{Err: &net.DNSError{}}
	err := DecodeError(urlErr)
	assert.True(t, errors.Is(err, TimeoutError))
	assert.Contains(t, err.Error(), "we found a problem connecting against the API")
}

func TestDecodeError_HandlesHttpError(t *testing.T) {
	httpErr := HTTPError{Code: 400, Reason: "Bad Request"}
	err := DecodeError(httpErr)
	assert.True(t, errors.Is(err, HttpError))
	assert.Equal(t, "HttpError: Bad Request", err.Error())
}

func TestDecodeError_ReturnsUnknownError(t *testing.T) {
	otherErr := errors.New("some other error")
	err := DecodeError(otherErr)
	assert.True(t, errors.Is(err, UnknownError))
	assert.Equal(t, "UnknownError: System error, please try again !", err.Error())
}

// A 404/409/422/5xx used to become "System error, please try again !", hiding
// the message the API sent. Terraform users saw that for a duplicate security
// group name, a missing storage volume, and every operation failure.
func TestDecodeError_KeepsReasonForEveryStatus(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 409, 422, 500, 503} {
		httpErr := HTTPError{
			Code:   code,
			Status: "irrelevant",
			Reason: "Security group name already exists",
		}

		err := DecodeError(httpErr)

		assert.True(t, errors.Is(err, HttpError), "status %d", code)
		assert.Equal(t,
			"HttpError: Security group name already exists", err.Error(),
			"status %d", code)
	}
}

// Callers branch on the status code (resource_storage.go treats 404 as "gone").
// That could never work while DecodeError dropped the original error.
func TestDecodeError_KeepsStatusCodeReachable(t *testing.T) {
	err := DecodeError(HTTPError{Code: 404, Reason: "Storage not found"})

	var httpErr HTTPError
	assert.True(t, errors.As(err, &httpErr))
	assert.Equal(t, 404, httpErr.Code)
	assert.Equal(t, "Storage not found", httpErr.Reason)
}
