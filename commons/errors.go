package commons

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Errors raised
var (
	TimeoutError         = constError("TimeoutError")
	UnknownError         = constError("UnknownError")
	ZeroMatchesError     = constError("ZeroMatchesError")
	MultipleMatchesError = constError("MultipleMatchesError")
	HttpError            = constError("HttpError")
)

type constError string

func (err constError) Error() string {
	return string(err)
}

func (err constError) Is(target error) bool {
	ts := target.Error()
	es := string(err)
	return ts == es || strings.HasPrefix(ts, es+": ")
}

func (err constError) Wrap(inner error) error {
	return wrapError{msg: string(err), err: inner}
}

func (err constError) WrapString(errorString string) error {
	return wrapError{msg: string(err), err: errors.New(errorString)}
}

type wrapError struct {
	err error
	msg string
}

func (err wrapError) Error() string {
	if err.err != nil {
		return fmt.Sprintf("%s: %v", err.msg, err.err)
	}
	return err.msg
}

func (err wrapError) Unwrap() error {
	return err.err
}

func (err wrapError) Is(target error) bool {
	return constError(err.msg).Is(target)
}

// reasonError renders as the reason the API sent, so a wrapped error reads
// "HttpError: <reason>" and not the full "<code>: <status>, <reason>" dump,
// while still carrying the HTTPError for errors.As.
type reasonError struct {
	http HTTPError
}

func (err reasonError) Error() string {
	return err.http.Reason
}

func (err reasonError) Unwrap() error {
	return err.http
}

func DecodeError(err error) error {
	var urlErr *url.Error
	var netErr net.Error
	var httpErr HTTPError

	if errors.As(err, &urlErr) {
		if errors.As(urlErr.Err, &netErr) {
			return TimeoutError.Wrap(fmt.Errorf("we found a problem connecting against the API: %w", err))
		}
	}

	if errors.As(err, &httpErr) {
		// Every status keeps the reason the API sent. Only 400 used to: 404,
		// 409, 422 and 5xx all collapsed into "System error, please try again !",
		// so a message the API had already written for the user (e.g. "Security
		// group name already exists") never reached them.
		//
		// Wrapping the HTTPError itself -- rather than only its text -- also
		// keeps the status code reachable via errors.As. Callers already try
		// that (see fptcloud/storage/resource_storage.go), but it could never
		// match while the original error was dropped here.
		return HttpError.Wrap(reasonError{http: httpErr})
	}
	return UnknownError.WrapString("System error, please try again !")
}
