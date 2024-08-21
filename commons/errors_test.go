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
