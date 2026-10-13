package commons

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

var nestedErrorCode = regexp.MustCompile(`'error_code': '(\d+)'`)

func ApiErrorCode(err error) string {
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		return ""
	}
	var body struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
	}
	if json.Unmarshal([]byte(httpErr.Reason), &body) != nil {
		return ""
	}
	if match := nestedErrorCode.FindStringSubmatch(body.Message); match != nil {
		return match[1]
	}
	if body.ErrorCode == "500000" {
		return ""
	}
	return body.ErrorCode
}

// IsApiErrorCode tells whether a failed request carries this portal error_code
func IsApiErrorCode(err error, code string) bool {
	return ApiErrorCode(err) == code
}

// HttpStatus returns the HTTP status of a failed request, or 0 when the request got no answer
func HttpStatus(err error) int {
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Code
	}
	return 0
}

// DescribeApiError turns a raw request error into a short, actionable reason
func DescribeApiError(err error) string {
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		return fmt.Sprintf("could not reach the API: %s", err)
	}
	switch httpErr.Code {
	case 401:
		return "not authorized (HTTP 401): check the provider token"
	case 403:
		return "permission denied (HTTP 403): the token has no access to this resource or VPC"
	}
	var body struct {
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(httpErr.Reason), &body) == nil && body.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", httpErr.Code, body.Message)
	}
	return fmt.Sprintf("HTTP %d", httpErr.Code)
}
