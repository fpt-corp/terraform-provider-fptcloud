package commons

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// APIResponse represents the standard API response structure
type APIResponse struct {
	Status    bool   `json:"status"`
	ErrorCode *int   `json:"error_code"`
	Data      any    `json:"data"`
	Message   string `json:"message"`
}

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
		if httpErr.Code == 400 {
			return HttpError.WrapString(httpErr.Reason)
		}
	}
	return UnknownError.WrapString("System error, please try again !")
}

func BaseError(err error) error {
	if err == nil {
		return nil
	}

	// Handle URL/Network errors
	if err := handleNetworkError(err); err != nil {
		return err
	}

	// Handle HTTP errors
	if err := handleHTTPError(err); err != nil {
		return err
	}

	return UnknownError.WrapString("System error, please try again !")
}

// handleNetworkError handles URL and network-related errors
func handleNetworkError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		var netErr net.Error
		if errors.As(urlErr.Err, &netErr) {
			return TimeoutError.Wrap(fmt.Errorf("we found a problem connecting against the API: %w", err))
		}
	}
	return nil
}

// handleHTTPError handles HTTP errors and extracts meaningful messages
func handleHTTPError(err error) error {
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		return nil
	}

	if httpErr.Code != 200 {
		if message := extractMessageFromReason(httpErr.Reason); message != "" {
			return fmt.Errorf("%s", message)
		}
	}

	return nil
}

// extractMessageFromReason parses JSON from Reason field and extracts the message
func extractMessageFromReason(reason string) string {
	if reason == "" {
		return ""
	}

	var apiResp APIResponse
	if err := json.Unmarshal([]byte(reason), &apiResp); err != nil {
		return ""
	}

	// Return message only if it's an error response with a message
	if !apiResp.Status && apiResp.Message != "" {
		return apiResp.Message
	}

	return ""
}
