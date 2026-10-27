package fptcloud_backup_native

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The statuses a one-shot resource stores when the API has no status to give.
// Otherwise status holds the API's own value. The APIs already use "Failed"
// (v1) and "failed" (v2), so these carry a prefix.
const (
	// runStatusTimeout: the provider stopped waiting before the action ended,
	// because the timeout ran out or the API failed too many reads in a row.
	runStatusTimeout = "tf_timeout"
	// runStatusInterrupted: the apply was interrupted (Ctrl+C) while the
	// provider waited. The action goes on without Terraform.
	runStatusInterrupted = "tf_interrupted"
	// runStatusFailed: the action failed and the API gave no status for it,
	// for example a job deleted while its backup ran.
	runStatusFailed = "tf_failed"
)

// runStatusesDescription ends the description of every one-shot status.
const runStatusesDescription = "The provider stores its own value when the API has none to give: " +
	"`tf_timeout` when the provider stopped waiting before the action ended, " +
	"`tf_interrupted` when the apply was interrupted while it waited, " +
	"`tf_failed` when the action failed without an API status. " +
	"A plan or apply checks `tf_timeout` and `tf_interrupted` again and replaces them with the API's status once the action ends."

// runMessageSchema is the message attribute of every one-shot resource.
func runMessageSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Why the action did not succeed, or why the provider stopped waiting for it. Empty after a success.",
	}
}

// runFailedError reports an action that ended without success. status is the
// API's status for it, or "" when the API gave none.
type runFailedError struct {
	status string
	err    error
}

func (e runFailedError) Error() string { return e.err.Error() }
func (e runFailedError) Unwrap() error { return e.err }

func runFailed(status string, err error) error { return runFailedError{status, err} }

// runOutcome turns the error that ended the wait on a started action into the
// status and message to store. Once an action has started, the resource is
// saved whatever the outcome, so that a later apply does not run it again.
func runOutcome(ctx context.Context, err error) (status, message string) {
	var failed runFailedError
	switch {
	case errors.As(err, &failed):
		if failed.status == "" {
			return runStatusFailed, failed.Error()
		}
		return failed.status, failed.Error()
	case errors.Is(ctx.Err(), context.Canceled):
		return runStatusInterrupted, fmt.Sprintf("the apply was interrupted while waiting; the action goes on without Terraform (%v)", err)
	}
	return runStatusTimeout, err.Error()
}

// runInProgress reports whether status belongs to an action the provider
// stopped watching before it ended, which a refresh checks again.
func runInProgress(status string) bool {
	return status == runStatusTimeout || status == runStatusInterrupted
}

// runWarning reports an action that did not end in success. It is a warning,
// not an error: an error would taint the resource and run the action again.
func runWarning(status, message string) diag.Diagnostics {
	detail := "The resource is saved with this status and is not run again. Change a value in triggers to run the action again."
	if runInProgress(status) {
		detail = "The resource is saved with this status and is not run again. " +
			"The next plan or apply checks the API and stores the result once the action ends. " +
			"Change a value in triggers to run the action again."
	}
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("status %s: %s", status, message),
		Detail:   detail,
	}}
}

// saveRunOutcome stores how the wait on a started action ended. A nil err
// stores nothing: the caller sets the success values.
func saveRunOutcome(ctx context.Context, d *schema.ResourceData, err error) diag.Diagnostics {
	status, message := runOutcome(ctx, err)
	if diags := setValues(d, map[string]interface{}{"status": status, "message": message}); diags.HasError() {
		return diags
	}
	return runWarning(status, message)
}

// refreshRun checks once on an action the provider stopped watching, and
// stores its result when it has ended. check returns the values to store, or
// nil while the action still runs. A failed check keeps the state and warns,
// so that the plan can go on.
func refreshRun(d *schema.ResourceData, check func() (map[string]interface{}, error)) diag.Diagnostics {
	if !runInProgress(d.Get("status").(string)) {
		return nil
	}
	values, err := check()
	if err != nil {
		return diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("could not check the action of %s: %v", d.Id(), err),
			Detail:   "The stored status is kept; the next plan or apply checks again.",
		}}
	}
	if values == nil {
		return nil
	}
	return setValues(d, values)
}

// idTime reads the Unix time that ends a one-shot resource's id: when the
// provider sent the action.
func idTime(id string) (time.Time, error) {
	seconds, err := strconv.ParseInt(id[strings.LastIndex(id, "/")+1:], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("the id %q does not end in a Unix time", id)
	}
	return time.Unix(seconds, 0).UTC(), nil
}

// restoreRowValues returns what a restore stores from its history row: the
// start alone while it runs, and the outcome once ended. err is the failure
// the row reports.
func restoreRowValues(status, start, end string, ended bool, err error) map[string]interface{} {
	values := map[string]interface{}{"started_at": formatAPITime(start)}
	if !ended {
		return values
	}
	values["status"], values["message"], values["finished_at"] = status, "", formatAPITime(end)
	if err != nil {
		values["message"] = err.Error()
	}
	return values
}
