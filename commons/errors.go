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

func DecodeError(err error) error {
	var urlErr *url.Error
	var netErr net.Error

	if errors.As(err, &urlErr) {
		if errors.As(urlErr.Err, &netErr) {
			return TimeoutError.Wrap(fmt.Errorf("we found a problem connecting against the API: %w", err))
		}
	}

	return UnknownError.Wrap(err)
}
