package fptcloud_backup_native

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForTaskPendingThenSuccess(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/task/task-1/progress": {
			{Body: `{"state":"PENDING","message":"PENDING","status_code":200}`},
			{Body: `{"state":"STARTED","message":"STARTED","status_code":200}`},
			{Body: `{"state":"SUCCESS","message":"Create backup successfully","status_code":201}`},
		},
	})
	msg, err := waitForTask(context.Background(), NewVmService(client), "vpc-1", "task-1", "creating", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Create backup successfully", msg)
}

// status_code is 200 while a task is pending, so a 200 alone is not "done".
func TestWaitForTaskDoesNotStopOnPending200(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/task/task-1/progress": {{Body: `{"state":"PENDING","message":"PENDING","status_code":200}`}},
	})
	_, err := waitForTask(context.Background(), NewVmService(client), "vpc-1", "task-1", "creating", 300*time.Millisecond)
	assert.ErrorContains(t, err, "timeout")
}

func TestWaitForTaskSuccessStateWithErrorCodeFails(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/task/task-1/progress": {{Body: `{"state":"SUCCESS","message":"Instance is registered to a backup","status_code":400}`}},
	})
	_, err := waitForTask(context.Background(), NewVmService(client), "vpc-1", "task-1", "creating the VM backup job", time.Minute)
	assert.ErrorContains(t, err, "creating the VM backup job failed (status 400): Instance is registered to a backup (a VM can belong")
}

func TestWaitForTaskFailureState(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/task/task-1/progress": {{Body: `{"state":"FAILURE","message":"FAILURE","status_code":200}`}},
	})
	_, err := waitForTask(context.Background(), NewVmService(client), "vpc-1", "task-1", "updating", time.Minute)
	assert.ErrorContains(t, err, "ended in state FAILURE")
}

func TestWaitVmJobSettled(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {
			{Body: `{"data":{"id":"job-1","status":"45"}}`},
			{Body: `{"data":{"id":"job-1","status":"Updating"}}`},
			{Body: `{"data":{"id":"job-1","status":"Success"}}`},
		},
	})
	job, err := waitVmJobSettled(context.Background(), NewVmService(client), "vpc-1", "job-1", settleToChange, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Success", job.Status)
}

func TestWaitVmJobGone(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {
			{Body: `{"data":{"id":"job-1","status":"Deleting"}}`},
			{Body: `{}`},
		},
	})
	assert.NoError(t, waitVmJobGone(context.Background(), NewVmService(client), "vpc-1", "job-1", time.Minute))
}

func TestWaitVmJobGoneFailsOnError(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{"data":{"id":"job-1","status":"error_deleting"}}`}},
	})
	assert.ErrorContains(t, waitVmJobGone(context.Background(), NewVmService(client), "vpc-1", "job-1", time.Minute), "error_deleting")
}

// A job whose last backup failed sits in "error"; that is not a failed delete
// until the job has moved on from the status it had when the delete started.
func TestWaitVmJobGoneIgnoresErrorFromBeforeTheDelete(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {
			{Body: `{"data":{"id":"job-1","status":"Error"}}`},
			{Body: `{"data":{"id":"job-1","status":"Error"}}`},
			{Body: `{}`},
		},
	})
	assert.NoError(t, waitVmJobGone(context.Background(), NewVmService(client), "vpc-1", "job-1", time.Minute))
}

func TestWaitVmJobGoneFailsWhenTheDeleteEndsInError(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {
			{Body: `{"data":{"id":"job-1","status":"Deleting"}}`},
			{Body: `{"data":{"id":"job-1","status":"Error"}}`},
		},
	})
	assert.ErrorContains(t, waitVmJobGone(context.Background(), NewVmService(client), "vpc-1", "job-1", time.Minute), "state Error")
}

func TestWaitVolumeJobGoneIgnoresErrorFromBeforeTheDelete(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9": {
			{Body: `{"data":{"id":"job-9","status":"error"}}`},
			{Status: 400, Body: `{"message":"Backup job not found with id job-9"}`},
		},
	})
	assert.NoError(t, waitVolumeJobGone(context.Background(), NewVolumeService(client), "vpc-1", "job-9", time.Minute))
}

func TestWaitVolumeJobGone(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9": {
			{Body: `{"data":{"id":"job-9","status":"deleting"}}`},
			{Status: 400, Body: `{"message":"Backup job not found with id job-9"}`},
		},
	})
	assert.NoError(t, waitVolumeJobGone(context.Background(), NewVolumeService(client), "vpc-1", "job-9", time.Minute))
}

// Review focus 5.
func TestFindVmHistoryRowIgnoresOlderRows(t *testing.T) {
	since := time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)
	rows := []VmHistory{
		{VmId: "vm-1", Type: "Restore", Status: "Success", StartTime: "2026-10-13T07:00:00"},
		{VmId: "vm-2", Type: "Restore", Status: "Restoring", StartTime: "2026-10-13T08:00:30"},
		{VmId: "vm-1", Type: "Restore keep", Status: "Restoring keep", StartTime: "2026-10-13T08:00:30"},
	}
	assert.Nil(t, findVmHistoryRow(rows, "vm-1", "Restore", since, nil))

	rows = append(rows, VmHistory{VmId: "vm-1", Type: "Restore", Status: "Restoring", StartTime: "2026-10-13T07:59:00"})
	row := findVmHistoryRow(rows, "vm-1", "Restore", since, nil)
	require.NotNil(t, row, "a row inside the clock-skew window counts")
	assert.Equal(t, "Restoring", row.Status)
}

func TestWaitVmRestoreSuccessAndFailure(t *testing.T) {
	setFastPolling(t)
	since := time.Now().UTC()
	start := since.Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {
			{Body: `{"total":0,"data":[]}`},
			{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"Restoring","start_time":"` + start + `"}]}`},
			{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"Success","start_time":"` + start + `","end_time":"` + start + `"}]}`},
		},
	})
	row, err := waitVmRestore(context.Background(), NewVmService(client), "vpc-1", "vm-1", "Restore", since, nil, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Success", row.Status)

	_, failing := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"Failed","reason":"flavor quota exceeded","start_time":"` + start + `"}]}`}},
	})
	_, err = waitVmRestore(context.Background(), NewVmService(failing), "vpc-1", "vm-1", "Restore", since, nil, time.Minute)
	assert.ErrorContains(t, err, "flavor quota exceeded")
}

// A restore that finished just before this one was submitted can still sit
// inside the clock-skew window; rows seen before submitting never match.
func TestFindVmHistoryRowSkipsRowsSeenBeforeSubmit(t *testing.T) {
	since := time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)
	rows := []VmHistory{{VmId: "vm-1", Type: "Restore", Status: "Success", StartTime: "2026-10-13T07:59:30"}}
	seen := map[string]bool{"2026-10-13T07:59:30": true}
	assert.Nil(t, findVmHistoryRow(rows, "vm-1", "Restore", since, seen))

	rows = append(rows, VmHistory{VmId: "vm-1", Type: "Restore", Status: "Restoring", StartTime: "2026-10-13T08:00:10"})
	row := findVmHistoryRow(rows, "vm-1", "Restore", since, seen)
	require.NotNil(t, row)
	assert.Equal(t, "Restoring", row.Status)
}

func TestWaitVmRestoreStatusIgnoresCase(t *testing.T) {
	setFastPolling(t)
	since := time.Now().UTC()
	start := since.Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"SUCCESS","start_time":"` + start + `"}]}`}},
	})
	_, err := waitVmRestore(context.Background(), NewVmService(client), "vpc-1", "vm-1", "Restore", since, nil, 2*time.Second)
	require.NoError(t, err)

	_, failing := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore","status":"failed","reason":"disk full","start_time":"` + start + `"}]}`}},
	})
	_, err = waitVmRestore(context.Background(), NewVmService(failing), "vpc-1", "vm-1", "Restore", since, nil, 2*time.Second)
	assert.ErrorContains(t, err, "disk full")
}

func TestFindVolumeRestoreRowSkipsRowsSeenBeforeSubmit(t *testing.T) {
	since := time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/history/combined-history": {{Body: `{"items":[
			{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"pending","start_time":"2026-10-13T08:00:20"},
			{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"success","start_time":"2026-10-13T07:59:40"}],"has_more":false}`}},
	})
	svc := NewVolumeService(client)
	row, err := findVolumeRestoreRow(svc, "vpc-1", "vol-1", since, map[string]bool{"2026-10-13T08:00:20": true})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "2026-10-13T07:59:40", row.StartTime, "the newest row was seen before submit, so the next one matches")

	row, err = findVolumeRestoreRow(svc, "vpc-1", "vol-1", since, map[string]bool{"2026-10-13T08:00:20": true, "2026-10-13T07:59:40": true})
	require.NoError(t, err)
	assert.Nil(t, row)
}

// Review focus 5: rows are newest first, so reading stops at the first row
// older than the cutoff.
func TestFindVolumeRestoreRowStopsAtCutoff(t *testing.T) {
	since := time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/history/combined-history": {
			{Body: `{"items":[
				{"source_type":"instance","vm_id":"vm-1","type":"Restore","status":"success","start_time":"2026-10-13T08:05:00"},
				{"source_type":"volume","object_id":"vol-2","type":"Restore keep","status":"pending","start_time":"2026-10-13T08:04:00"}],"has_more":true}`},
			{Body: `{"items":[
				{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"success","start_time":"2026-10-13T07:00:00"}],"has_more":true}`},
		},
	})
	row, err := findVolumeRestoreRow(NewVolumeService(client), "vpc-1", "vol-1", since, nil)
	require.NoError(t, err)
	assert.Nil(t, row, "the 07:00 row belongs to an earlier restore")
	assert.Equal(t, 2, api.count("GET "+volBase+"/history/combined-history"), "stops at the cutoff instead of reading every page")
}

func TestWaitVolumeRestoreFailureUsesExtractInfo(t *testing.T) {
	setFastPolling(t)
	since := time.Now().UTC()
	start := since.Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/history/combined-history": {{Body: `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"failed","start_time":"` + start + `","history_metadata":{"extract_info":{"status":"no space left"}}}],"has_more":false}`}},
	})
	_, err := waitVolumeRestore(context.Background(), NewVolumeService(client), "vpc-1", "vol-1", since, nil, time.Minute)
	assert.ErrorContains(t, err, "no space left")
}

// A dev apply got one 401 from the history list while a submitted restore
// ran; the wait must keep polling through a passing API error.
func TestWaitVmRestoreSurvivesAPassingAPIError(t *testing.T) {
	setFastPolling(t)
	since := time.Now().UTC()
	start := since.Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {
			{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore keep","status":"Restoring","start_time":"` + start + `"}]}`},
			{Status: 401, Body: `{"message":"Unauthorized Access"}`},
			{Status: 502, Body: `<html>bad gateway</html>`},
			{Body: `{"total":1,"data":[{"vm_id":"vm-1","type":"Restore keep","status":"Success","start_time":"` + start + `"}]}`},
		},
	})
	row, err := waitVmRestore(context.Background(), NewVmService(client), "vpc-1", "vm-1", "Restore keep", since, nil, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Success", row.Status)
}

func TestWaitVolumeRestoreSurvivesAPassingAPIError(t *testing.T) {
	setFastPolling(t)
	since := time.Now().UTC()
	start := since.Format(apiTimeLayout)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/history/combined-history": {
			{Status: 401, Body: `{"message":"Unauthorized Access"}`},
			{Body: `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"success","start_time":"` + start + `"}],"has_more":false}`},
		},
	})
	row, err := waitVolumeRestore(context.Background(), NewVolumeService(client), "vpc-1", "vol-1", since, nil, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "success", row.Status)
}

// The VM job create path: the task was accepted, then one progress read fails.
func TestWaitForTaskSurvivesAPassingAPIError(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/task/task-1/progress": {
			{Status: 401, Body: `{"message":"Unauthorized Access"}`},
			{Body: `{"state":"SUCCESS","message":"Create backup successfully","status_code":201}`},
		},
	})
	msg, err := waitForTask(context.Background(), NewVmService(client), "vpc-1", "task-1", "creating", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Create backup successfully", msg)
}

// An error that does not pass still ends the wait, and says it was retried.
func TestWaitGivesUpAfterRepeatedAPIErrors(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/histories/list": {{Status: 401, Body: `{"message":"Unauthorized Access"}`}},
	})
	_, err := waitVmRestore(context.Background(), NewVmService(client), "vpc-1", "vm-1", "Restore", time.Now().UTC(), nil, time.Minute)
	assert.ErrorContains(t, err, "Unauthorized Access")
	assert.ErrorContains(t, err, "6 reads in a row")
	assert.Equal(t, maxPollErrors+1, api.count("GET "+vmBase+"/histories/list"))
}

func TestWaitVmJobSettledFailsAtOnceOnFailedBeforeAChange(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{"data":{"id":"job-1","status":"Failed"}}`}},
	})
	_, err := waitVmJobSettled(context.Background(), NewVmService(client), "vpc-1", "job-1", settleToChange, time.Minute)
	assert.ErrorContains(t, err, "Failed")
	assert.ErrorContains(t, err, "until another run starts")
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"))
}

func TestWaitVmJobSettledBeforeAnUpdateWaitsOnlyForUpdating(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": jobStatuses("job-1", "Updating", "Backing up"),
	})
	job, err := waitVmJobSettled(context.Background(), NewVmService(client), "vpc-1", "job-1", settleToUpdate, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Backing up", job.Status)
}

func TestWaitVmJobSettledFailsAtOnceOnFailedBeforeAnUpdate(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{"data":{"id":"job-1","status":"Failed"}}`}},
	})
	_, err := waitVmJobSettled(context.Background(), NewVmService(client), "vpc-1", "job-1", settleToUpdate, time.Minute)
	assert.ErrorContains(t, err, "until another run starts")
	assert.Equal(t, 1, api.count("GET "+vmBase+"/detail"))
}

func TestWaitVmJobSettledTreatsFailedAsSettledBeforeADelete(t *testing.T) {
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + vmBase + "/detail": {{Body: `{"data":{"id":"job-1","status":"Failed"}}`}},
	})
	job, err := waitVmJobSettled(context.Background(), NewVmService(client), "vpc-1", "job-1", settleToDelete, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Failed", job.Status)
}

// jobStatuses scripts a job detail route that answers with each status in
// turn; the last one repeats.
func jobStatuses(jobId string, statuses ...string) []fakeResponse {
	out := make([]fakeResponse, len(statuses))
	for i, status := range statuses {
		out[i] = fakeResponse{Body: `{"data":{"id":"` + jobId + `","status":"` + status + `"}}`}
	}
	return out
}

func waitVmRun(t *testing.T, routes []fakeResponse, timeout time.Duration) (string, error) {
	t.Helper()
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{"GET " + vmBase + "/detail": routes})
	return waitVmJobRun(context.Background(), NewVmService(client), "vpc-1", "job-1", timeout)
}

func waitVolumeRun(t *testing.T, routes []fakeResponse) (string, error) {
	t.Helper()
	setFastPolling(t)
	_, client := newFakeAPI(t, map[string][]fakeResponse{"GET " + volBase + "/backup-jobs/job-9": routes})
	return waitVolumeJobRun(context.Background(), NewVolumeService(client), "vpc-1", "job-9", time.Minute)
}

func TestWaitVmJobRunSuccess(t *testing.T) {
	status, err := waitVmRun(t, jobStatuses("job-1", "Pending", "Backing up", "Success"), time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Success", status)
}

func TestWaitVmJobRunFailsOnceFailedHolds(t *testing.T) {
	_, err := waitVmRun(t, jobStatuses("job-1", "Pending", "Failed"), time.Minute)
	assert.ErrorContains(t, err, "Failed")
	assert.ErrorContains(t, err, "may still be retried")
}

func TestWaitVmJobRunTreatsErrorAsFailed(t *testing.T) {
	_, err := waitVmRun(t, jobStatuses("job-1", "Error"), time.Minute)
	assert.ErrorContains(t, err, "in state Error")
}

// The error streak lasts one poll, shorter than the hold, so the retry that
// follows decides the outcome.
func TestWaitVolumeJobRunErrorThenRetryEndsInSuccess(t *testing.T) {
	status, err := waitVolumeRun(t, jobStatuses("job-9", "backing up", "error", "pending", "backing up", "success"))
	require.NoError(t, err)
	assert.Equal(t, "success", status)
}

func TestWaitVolumeJobRunTreatsUnknownStatusAsRunning(t *testing.T) {
	status, err := waitVolumeRun(t, jobStatuses("job-9", "pending", "failed", "success"))
	require.NoError(t, err)
	assert.Equal(t, "success", status)
}

// Review focus 4.
func TestWaitJobRunFailsWhenTheJobIsDeleted(t *testing.T) {
	routes := append(jobStatuses("job-1", "Backing up"), fakeResponse{Body: `{}`})
	_, err := waitVmRun(t, routes, time.Minute)
	assert.ErrorContains(t, err, "deleted")
}

// Review focus 5.
func TestWaitJobRunSurvivesAPassingAPIError(t *testing.T) {
	routes := jobStatuses("job-1", "Backing up", "Success")
	routes = []fakeResponse{routes[0], {Status: 502, Body: `{"message":"bad gateway"}`}, routes[1]}
	status, err := waitVmRun(t, routes, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "Success", status)
}

func TestWaitJobRunTimesOutOnAStuckPending(t *testing.T) {
	_, err := waitVmRun(t, jobStatuses("job-1", "Pending"), 200*time.Millisecond)
	assert.ErrorContains(t, err, "stuck in Pending")
}

func TestWaitVolumeJobSettledAcceptsErrorAtOnce(t *testing.T) {
	setFastPolling(t)
	api, client := newFakeAPI(t, map[string][]fakeResponse{"GET " + volBase + "/backup-jobs/job-9": jobStatuses("job-9", "error")})
	require.NoError(t, waitVolumeJobSettled(context.Background(), NewVolumeService(client), "vpc-1", "job-9", time.Minute))
	assert.Equal(t, 1, api.count("GET "+volBase+"/backup-jobs/job-9"))
}

func TestWaitJobRunTimeoutNamesTheLastStatus(t *testing.T) {
	_, err := waitVmRun(t, jobStatuses("job-1", "Backing up"), 200*time.Millisecond)
	assert.ErrorContains(t, err, "last in state Backing up")
	assert.NotContains(t, err.Error(), "stuck in Pending")
}
