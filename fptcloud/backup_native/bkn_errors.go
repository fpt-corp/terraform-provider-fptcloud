package fptcloud_backup_native

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	common "terraform-provider-fptcloud/commons"
)

// apiError is a non-2xx answer from the Backup Native API. Message is decoded
// from whichever of the three body shapes the backend used: {"message"},
// {"error"} (IAM denial) or {"validation_error"} (flask-pydantic).
type apiError struct {
	Code    int
	Message string
	Body    []byte
}

func (e *apiError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}

type errorBody struct {
	Message         string `json:"message"`
	Error           string `json:"error"`
	ValidationError *struct {
		BodyParams []struct {
			Loc []interface{} `json:"loc"`
			Msg string        `json:"msg"`
		} `json:"body_params"`
	} `json:"validation_error"`
}

// asAPIError returns the API error inside err, or nil when err did not come
// from an HTTP response.
func asAPIError(err error) *apiError {
	var httpErr common.HTTPError
	if !errors.As(err, &httpErr) {
		return nil
	}
	result := &apiError{Code: httpErr.Code, Body: []byte(httpErr.Reason), Message: strings.TrimSpace(httpErr.Reason)}

	var body errorBody
	if json.Unmarshal([]byte(httpErr.Reason), &body) != nil {
		return result
	}
	switch {
	case body.ValidationError != nil && len(body.ValidationError.BodyParams) > 0:
		parts := make([]string, 0, len(body.ValidationError.BodyParams))
		for _, param := range body.ValidationError.BodyParams {
			loc := make([]string, 0, len(param.Loc))
			for _, l := range param.Loc {
				loc = append(loc, fmt.Sprint(l))
			}
			parts = append(parts, fmt.Sprintf("%s: %s", strings.Join(loc, "."), param.Msg))
		}
		result.Message = strings.Join(parts, "; ")
	case body.Message != "":
		result.Message = body.Message
	case body.Error != "":
		result.Message = body.Error
	}
	return result
}

// isNotFound reports whether err means the object no longer exists. The
// backend answers 404 on some routes and 400 "... not found ..." on others
// (v2 detail, delete and actions).
func isNotFound(err error) bool {
	apiErr := asAPIError(err)
	if apiErr == nil {
		return false
	}
	if apiErr.Code == 404 {
		return true
	}
	return apiErr.Code == 400 && strings.Contains(strings.ToLower(apiErr.Message), "not found")
}

// msgNotAvailableForDelete is in the message of a delete the API refuses
// while a backup or restore of the VM runs (DeleteBackupController.py:513-522).
const msgNotAvailableForDelete = "not available for delete"

// hintFor returns advice for a failure the user can act on, or "".
func hintFor(code int, message string) string {
	m := strings.ToLower(message)
	switch {
	case strings.Contains(m, "service is disabled"):
		return "Backup Native is not enabled for this VPC; enable it in the FPT Cloud portal"
	case code == 403 && strings.Contains(m, "quota"):
		return "the backup quota is exhausted. When this happens the backend also disables every backup schedule in the VPC; raise the quota, then apply again"
	case strings.Contains(m, "registered to a backup"):
		return "a VM can belong to only one backup job; the fptcloud_backup_native_vm_jobs data source shows which job holds it"
	case strings.Contains(m, "already registered"):
		return "a volume can belong to only one backup job; the fptcloud_backup_native_volume_jobs data source shows which job holds it"
	case strings.Contains(m, "invalid fke volume"):
		return "only FKE persistent volumes (names starting with pv-fke) can be backed up"
	case strings.Contains(m, msgNotAvailableForDelete):
		return "a backup or restore of the VM is running, or its backup job is still being set up; wait for it to end and apply again"
	case strings.Contains(m, "unavailable") || strings.Contains(m, "busy"):
		return "the job or restore point is busy; wait for the running operation to finish and apply again. A volume restore point that stays busy may be stuck after a failed restore; use another restore point"
	case code >= 500:
		return "the API returned an internal error; check that vpc_id is correct and that Backup Native is enabled for this VPC"
	}
	return ""
}

// describe joins a message and its hint.
func describe(code int, message string) string {
	if hint := hintFor(code, message); hint != "" {
		return fmt.Sprintf("%s (%s)", message, hint)
	}
	return message
}

// wrapAPIError prefixes err with the action that failed and adds a hint.
// Errors that are not HTTP errors keep their chain.
func wrapAPIError(action string, err error) error {
	if apiErr := asAPIError(err); apiErr != nil {
		return fmt.Errorf("%s failed with HTTP %d: %s", action, apiErr.Code, describe(apiErr.Code, apiErr.Message))
	}
	return fmt.Errorf("%s failed: %w", action, err)
}
