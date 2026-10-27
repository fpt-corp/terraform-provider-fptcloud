package fptcloud_backup_native

import (
	"errors"
	"net/http"
	"testing"

	common "terraform-provider-fptcloud/commons"

	"github.com/stretchr/testify/assert"
)

func httpErr(code int, body string) error {
	return common.HTTPError{Code: code, Status: http.StatusText(code), Reason: body}
}

func TestAsAPIErrorReadsMessage(t *testing.T) {
	e := asAPIError(httpErr(400, `{"message":"Backup job not found with id x"}`))
	assert.Equal(t, 400, e.Code)
	assert.Equal(t, "Backup job not found with id x", e.Message)
}

func TestAsAPIErrorReadsIamError(t *testing.T) {
	e := asAPIError(httpErr(403, `{"error":"Permission denied"}`))
	assert.Equal(t, "Permission denied", e.Message)
}

func TestAsAPIErrorFlattensValidationErrors(t *testing.T) {
	e := asAPIError(httpErr(400, `{"validation_error":{"body_params":[{"loc":["metadata","name"],"msg":"Name must contain only letters, numbers, hyphens","type":"value_error"},{"loc":["retention"],"msg":"field required","type":"value_error.missing"}]}}`))
	assert.Equal(t, "metadata.name: Name must contain only letters, numbers, hyphens; retention: field required", e.Message)
}

func TestAsAPIErrorKeepsNonJSONBody(t *testing.T) {
	e := asAPIError(httpErr(502, "<html>bad gateway</html>"))
	assert.Equal(t, "<html>bad gateway</html>", e.Message)
	assert.Equal(t, []byte("<html>bad gateway</html>"), e.Body)
}

func TestAsAPIErrorIgnoresOtherErrors(t *testing.T) {
	assert.Nil(t, asAPIError(errors.New("dial tcp: connection refused")))
}

func TestIsNotFound(t *testing.T) {
	assert.True(t, isNotFound(httpErr(404, `{"message":"This backup is no longer exist"}`)))
	assert.True(t, isNotFound(httpErr(400, `{"message":"Backup job not found with id x"}`)))
	assert.False(t, isNotFound(httpErr(400, `{"message":"Resource is unavailable"}`)))
	assert.False(t, isNotFound(httpErr(500, `{"message":"not found"}`)))
	assert.False(t, isNotFound(errors.New("not found")))
}

func TestHintFor(t *testing.T) {
	cases := []struct {
		code    int
		message string
		want    string
	}{
		{200, "Tenant or vpc service is disabled, please try again later.", "not enabled for this VPC"},
		{403, "Tenant or vpc service is disabled, please try again later.", "not enabled for this VPC"},
		{403, "Quota exceeded", "backup quota is exhausted"},
		{400, "Instance is registered to a backup", "only one backup job; the fptcloud_backup_native_vm_jobs"},
		{400, "Volumes ['v'] is already registered.", "fptcloud_backup_native_volume_jobs"},
		{400, "Invalid fke volume", "pv-fke"},
		{400, "Resource is unavailable", "busy"},
		{400, "Resource is not available for delete", "a backup or restore of the VM or volume is running"},
		{500, "There is an unexpected error", "check that vpc_id is correct"},
		{400, "Backup schedule name has already existed", ""},
	}
	for _, c := range cases {
		if c.want == "" {
			assert.Equal(t, "", hintFor(c.code, c.message), c.message)
			continue
		}
		assert.Contains(t, hintFor(c.code, c.message), c.want, c.message)
	}
}

func TestWrapAPIErrorAddsActionAndHint(t *testing.T) {
	err := wrapAPIError("creating the VM backup job", httpErr(400, `{"message":"Instance is registered to a backup"}`))
	assert.Contains(t, err.Error(), "creating the VM backup job failed with HTTP 400: Instance is registered to a backup (a VM can belong to only one backup job")
}

func TestWrapAPIErrorKeepsTransportErrors(t *testing.T) {
	inner := errors.New("dial tcp: connection refused")
	err := wrapAPIError("listing VM backup jobs", inner)
	assert.ErrorIs(t, err, inner)
	assert.Contains(t, err.Error(), "listing VM backup jobs failed")
}
