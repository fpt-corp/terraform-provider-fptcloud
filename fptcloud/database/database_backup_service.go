package fptcloud_database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"
)

// Status of the backup service, as reported by backup/detail.
const (
	backupStatusInactive      = 0
	backupStatusEnabled       = 1
	backupStatusProcessing    = 2
	backupStatusDeleting      = 3
	backupStatusUpdating      = 4
	backupStatusActive        = 5
	backupStatusDeactivating  = 6
	backupStatusErrorEnabled  = -1
	backupStatusErrorDeleted  = -2
	backupStatusErrorUpdated  = -3
	jobStatusReady            = "ready"
	jobStatusCreatedFail      = "created_fail"
	jobStatusUpdatedFail      = "updated_fail"
	jobStatusDeletedFail      = "deleted_fail"
	backupJobFull             = "full"
	backupJobDiff             = "diff"
	backupRestoreStateSuccess = "successful"
)

// Intervals are variables so that tests do not have to wait.
var (
	backupPollInterval    = 10 * time.Second
	backupCreateTimeout   = 45 * time.Minute
	backupUpdateTimeout   = 45 * time.Minute
	backupDeleteTimeout   = 30 * time.Minute
	restoreCreateTimeout  = 60 * time.Minute
	restoreAppearTimeout  = 3 * time.Minute
	backupActionBusyNames = map[string]bool{"pending": true, "processing": true, "updating": true, "deleting": true, "creating": true, "running": true}
)

// backupStatusName is the name of a backup_enabled code.
func backupStatusName(code int64) string {
	switch code {
	case backupStatusInactive:
		return "inactive"
	case backupStatusEnabled:
		return "enabled"
	case backupStatusProcessing:
		return "processing"
	case backupStatusDeleting:
		return "deleting"
	case backupStatusUpdating:
		return "updating"
	case backupStatusActive:
		return "active"
	case backupStatusDeactivating:
		return "deactivating"
	case backupStatusErrorEnabled:
		return "error_enabled"
	case backupStatusErrorDeleted:
		return "error_deleted"
	case backupStatusErrorUpdated:
		return "error_updated"
	}
	return "unknown(" + strconv.FormatInt(code, 10) + ")"
}

// backupIsReady reports whether the service can take changes: enabled (1) and active (5) are both normal.
func backupIsReady(code int64) bool {
	return code == backupStatusEnabled || code == backupStatusActive
}

// flexInt reads a JSON number, a numeric string or null.
type flexInt int64

func (n *flexInt) UnmarshalJSON(b []byte) error {
	raw := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if raw == "" || raw == "null" {
		*n = 0
		return nil
	}
	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		*n = flexInt(i)
		return nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("%q is not a number", raw)
	}
	*n = flexInt(math.Round(f))
	return nil
}

// flexFloat reads a JSON number, a numeric string or null.
type flexFloat float64

func (n *flexFloat) UnmarshalJSON(b []byte) error {
	raw := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if raw == "" || raw == "null" {
		*n = 0
		return nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("%q is not a number", raw)
	}
	*n = flexFloat(f)
	return nil
}

// businessError is an error the API reports inside an HTTP 200 body: {error_code, message, type}.
type businessError struct {
	Code    int64
	Message string
	Type    string
}

func (e *businessError) Error() string {
	msg := fmt.Sprintf("API returned error_code %d (%s): %s", e.Code, e.Type, e.Message)
	if e.Code == 160009 {
		msg += ". Add an alert with alert_service \"backup\" and at least one recipient to the cluster (Alert tab of the portal), then apply again"
	}
	return msg
}

// checkBusinessError returns a *businessError when the body of a mutation reports a failure.
// An empty body counts as success; a body that is not JSON is an error.
func checkBusinessError(body []byte) error {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil
	}
	var result struct {
		ErrorCode flexInt   `json:"error_code"`
		Message   apiString `json:"message"`
		Type      apiString `json:"type"`
	}
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		return fmt.Errorf("unexpected response from the API: %s", truncateBody(body))
	}
	if result.ErrorCode != 0 || strings.EqualFold(string(result.Type), "error") {
		return &businessError{Code: int64(result.ErrorCode), Message: apiMessage(result.Message, body), Type: string(result.Type)}
	}
	return nil
}

// mutate sends a POST, PUT or DELETE and turns both HTTP errors and errors inside the body into a Go error.
func (m *databaseApiClient) mutate(method, path string, params interface{}) error {
	var (
		answer databaseApiAnswer
		err    error
	)
	switch method {
	case "POST":
		answer, err = m.sendPostForAnswer(path, params)
	case "PUT":
		answer, err = m.sendPutForAnswer(path, params)
	case "DELETE":
		var body []byte
		body, err = m.sendDelete(path)
		answer = databaseApiAnswer{Body: body}
		var httpErr common.HTTPError
		if errors.As(err, &httpErr) {
			answer = databaseApiAnswer{StatusCode: httpErr.Code, Body: []byte(httpErr.Reason)}
		}
	default:
		return fmt.Errorf("unsupported method %s", method)
	}
	if err != nil {
		if answer.StatusCode == 0 {
			return fmt.Errorf("failed calling %s %s: %v", method, path, err)
		}
		return fmt.Errorf("failed calling %s %s: status_code=%d, body=%s", method, path, answer.StatusCode, truncateBody(answer.Body))
	}
	return checkBusinessError(answer.Body)
}

type backupDetail struct {
	ClusterId           string    `json:"cluster_id"`
	ClusterName         apiString `json:"cluster_name"`
	TypeDb              apiString `json:"type_db"`
	Status              apiString `json:"status"`
	BackupEnabled       flexInt   `json:"backup_enabled"`
	StoragePolicy       apiString `json:"storage_policy"`
	Capacity            flexInt   `json:"capacity"`
	BackupServiceId     flexInt   `json:"backup_service_id"`
	BackupRetentionType apiString `json:"backup_retention_type"`
	IsArchived          flexInt   `json:"is_archived"`
	PercentBackupUsed   flexFloat `json:"percent_backup_used"`
}

// usedGb is the used part of the capacity; the API reports it as a percentage.
func (d backupDetail) usedGb() float64 {
	return math.Round(float64(d.Capacity)*float64(d.PercentBackupUsed)/100*1000) / 1000
}

type backupJob struct {
	JobId     flexInt   `json:"job_id"`
	Name      apiString `json:"backup_job_name"`
	Type      apiString `json:"backup_type"`
	Retention flexInt   `json:"backup_retention"`
	Schedule  struct {
		Minute    apiString `json:"minute"`
		Hour      apiString `json:"hour"`
		DayOfWeek apiString `json:"day_of_week"`
	} `json:"backup_schedule"`
	Status   apiString `json:"status"`
	IsRunNow flexInt   `json:"is_run_now"`
}

func (j backupJob) hours() []int64 { return parseIntList(string(j.Schedule.Hour)) }
func (j backupJob) days() []int64  { return parseIntList(string(j.Schedule.DayOfWeek)) }
func (j backupJob) minute() int64 {
	v, _ := strconv.ParseInt(strings.TrimSpace(string(j.Schedule.Minute)), 10, 64)
	return v
}

// parseIntList turns "1,5,9" into [1 5 9]; anything that is not a number is skipped.
func parseIntList(s string) []int64 {
	out := []int64{}
	for _, part := range strings.Split(s, ",") {
		if v, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// joinIntList turns [5 1 9] into "1,5,9", the form the API takes.
func joinIntList(values []int64) string {
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	parts := make([]string, len(sorted))
	for i, v := range sorted {
		parts[i] = strconv.FormatInt(v, 10)
	}
	return strings.Join(parts, ",")
}

// backupJobRequest is the body of job/create and job/update.
type backupJobRequest struct {
	JobId     int64  `json:"job_id,omitempty"`
	ClusterId string `json:"cluster_id"`
	Name      string `json:"backup_job_name"`
	Type      string `json:"backup_type"`
	Retention int64  `json:"backup_retention"`
	Hour      string `json:"hour"`
	Minute    string `json:"minute"`
	DayOfWeek string `json:"day_of_week"`
}

// getWithRetry reads a path, trying again after a failure: the API sometimes answers a single request with a
// transient error (for example 403) while the next one works. Reads change nothing, so they can be repeated.
func (m *databaseApiClient) getWithRetry(path string) ([]byte, error) {
	var (
		body []byte
		err  error
	)
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(backupPollInterval)
		}
		if body, err = m.sendGet(path); err == nil {
			return body, nil
		}
	}
	return nil, err
}

func (m *databaseApiClient) getBackupDetail(clusterId string) (*backupDetail, error) {
	path := common.ApiPath.DatabaseBackupDetail(clusterId)
	body, err := m.getWithRetry(path)
	if err != nil {
		return nil, fmt.Errorf("failed calling path %s: %v", path, err)
	}
	if berr := checkBusinessError(body); berr != nil {
		return nil, berr
	}
	var d backupDetail
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, fmt.Errorf("failed parsing %s: %v, body=%s", path, err, truncateBody(body))
	}
	if d.ClusterId == "" {
		return nil, fmt.Errorf("no backup information for cluster %s: %s", clusterId, truncateBody(body))
	}
	// A missing state would read as 0 (inactive) and make the service look deleted.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || strings.TrimSpace(string(raw["backup_enabled"])) == "" || strings.TrimSpace(string(raw["backup_enabled"])) == "null" {
		return nil, fmt.Errorf("the backup state of cluster %s is missing in the answer of %s: %s", clusterId, path, truncateBody(body))
	}
	return &d, nil
}

func (m *databaseApiClient) listBackupJobs(clusterId string) ([]backupJob, error) {
	path := common.ApiPath.DatabaseBackupJobList(clusterId)
	body, err := m.getWithRetry(path)
	if err != nil {
		return nil, fmt.Errorf("failed calling path %s: %v", path, err)
	}
	if berr := checkBusinessError(body); berr != nil {
		return nil, berr
	}
	var l struct {
		Jobs []backupJob `json:"jobs"`
	}
	if err := json.Unmarshal(body, &l); err != nil {
		return nil, fmt.Errorf("failed parsing %s: %v, body=%s", path, err, truncateBody(body))
	}
	return l.Jobs, nil
}

// findBackupJob returns the full or diff job of a list, or nil.
func findBackupJob(jobs []backupJob, jobType string) *backupJob {
	for i := range jobs {
		if strings.EqualFold(string(jobs[i].Type), jobType) {
			return &jobs[i]
		}
	}
	return nil
}

func (m *databaseApiClient) setBackupCapacity(clusterId string, capacity int64, currentStatus int64, isArchived int64) error {
	return m.mutate("POST", common.ApiPath.DatabaseBackupCapacity(), map[string]interface{}{
		"cluster_id":      clusterId,
		"capacity":        capacity,
		"backup_enabled":  currentStatus,
		"storage_profile": nil,
		"is_archived":     isArchived,
	})
}

func (m *databaseApiClient) disableBackup(clusterId string) error {
	return m.mutate("DELETE", common.ApiPath.DatabaseBackupDelete(clusterId), nil)
}

func (m *databaseApiClient) createBackupJob(req backupJobRequest) error {
	req.JobId = 0
	return m.mutate("POST", common.ApiPath.DatabaseBackupJobCreate(), req)
}

func (m *databaseApiClient) updateBackupJob(req backupJobRequest) error {
	return m.mutate("PUT", common.ApiPath.DatabaseBackupJobUpdate(), req)
}

func (m *databaseApiClient) deleteBackupJob(jobId int64) error {
	return m.mutate("DELETE", common.ApiPath.DatabaseBackupJobDelete(jobId), nil)
}

func (m *databaseApiClient) runNowBackupJob(jobId int64) error {
	return m.mutate("POST", common.ApiPath.DatabaseBackupJobRunNow(jobId), nil)
}

type backupAction struct {
	TaskId      flexInt   `json:"task_id"`
	TaskName    apiString `json:"task_name"`
	Status      apiString `json:"status"`
	CreatedTime apiString `json:"created_time"`
}

// busyBackupActions lists the tasks of the action log that have not finished.
func (m *databaseApiClient) busyBackupActions(clusterId string) ([]backupAction, error) {
	actions, err := getPaged[backupAction](m, func(page, size int) string {
		return common.ApiPath.DatabaseBackupActionsLog(clusterId, page, size)
	}, 1)
	if err != nil {
		return nil, err
	}
	var busy []backupAction
	for _, a := range actions {
		if backupActionBusyNames[strings.ToLower(strings.TrimSpace(string(a.Status)))] {
			busy = append(busy, a)
		}
	}
	return busy, nil
}

type restorePoint struct {
	BackupPart  apiString `json:"backup_part"`
	BackupType  apiString `json:"backup_type"`
	BackupState apiString `json:"backup_state"`
	BackupSize  apiString `json:"backup_size"`
	StartTime   apiString `json:"start_time"`
	FinishTime  apiString `json:"finish_time"`
}

type restoredCluster struct {
	ClusterId    string    `json:"cluster_id"`
	ClusterName  apiString `json:"cluster_name"`
	Version      apiString `json:"version"`
	TypeDb       apiString `json:"type_db"`
	Status       apiString `json:"status"`
	CreatedAt    apiString `json:"created_at"`
	RestorePoint apiString `json:"restore_point"`
}

func (m *databaseApiClient) listRestorePoints(clusterId string) ([]restorePoint, error) {
	return getPaged[restorePoint](m, func(page, size int) string {
		return common.ApiPath.DatabaseRestorePoints(clusterId, page, size)
	}, 0)
}

func (m *databaseApiClient) listRestoredClusters(clusterId string) ([]restoredCluster, error) {
	return getPaged[restoredCluster](m, func(page, size int) string {
		return common.ApiPath.DatabaseRestoredClusters(clusterId, page, size)
	}, 0)
}

// getPaged reads every page of a list endpoint ({data: [...], metadata: {total_items}}).
// maxPages 0 means all pages.
func getPaged[T any](m *databaseApiClient, pathFor func(page, size int) string, maxPages int) ([]T, error) {
	const pageSize = 50
	var all []T
	for page := 1; ; page++ {
		path := pathFor(page, pageSize)
		body, err := m.getWithRetry(path)
		if err != nil {
			return nil, fmt.Errorf("failed calling path %s: %v", path, err)
		}
		if berr := checkBusinessError(body); berr != nil {
			return nil, berr
		}
		var resp struct {
			Data     []T `json:"data"`
			Metadata struct {
				TotalItems flexInt `json:"total_items"`
			} `json:"metadata"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed parsing %s: %v, body=%s", path, err, truncateBody(body))
		}
		all = append(all, resp.Data...)
		if len(resp.Data) == 0 || int64(len(all)) >= int64(resp.Metadata.TotalItems) ||
			(maxPages > 0 && page >= maxPages) || page >= 200 {
			return all, nil
		}
	}
}

// getClusterData reads a cluster; the error says so when it does not exist.
func (m *databaseApiClient) getClusterData(clusterId string) (*databaseData, error) {
	path := common.ApiPath.DatabaseGet(clusterId)
	body, err := m.getWithRetry(path)
	if err != nil {
		return nil, fmt.Errorf("failed calling path %s: %v", path, err)
	}
	var d databaseReadResponse
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, fmt.Errorf("failed parsing %s: %v, body=%s", path, err, truncateBody(body))
	}
	if d.Data.Cluster.ClusterId == "" {
		return nil, fmt.Errorf("cluster %s not found: %s", clusterId, apiMessage(apiString(d.Message), body))
	}
	return &d.Data.Cluster, nil
}

// pollUntil calls check every backupPollInterval until it reports done, fails, or the timeout passes.
func pollUntil(ctx context.Context, timeout time.Duration, what string, check func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for {
		done, err := check()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for %s", timeout, what)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backupPollInterval):
		}
	}
}

// waitBackupStatus waits until the backup service is enabled/active and applied reports that the change is visible.
func (m *databaseApiClient) waitBackupReady(ctx context.Context, clusterId string, timeout time.Duration, applied func(*backupDetail) bool) error {
	return pollUntil(ctx, timeout, "the backup service of cluster "+clusterId, func() (bool, error) {
		d, err := m.getBackupDetail(clusterId)
		if err != nil {
			return false, nil // a failed poll is retried until the timeout
		}
		switch int64(d.BackupEnabled) {
		case backupStatusErrorEnabled, backupStatusErrorDeleted, backupStatusErrorUpdated:
			return false, fmt.Errorf("the backup service of cluster %s is in error state %s", clusterId, backupStatusName(int64(d.BackupEnabled)))
		}
		return backupIsReady(int64(d.BackupEnabled)) && (applied == nil || applied(d)), nil
	})
}

// waitBackupInactive waits until the backup service is gone.
func (m *databaseApiClient) waitBackupInactive(ctx context.Context, clusterId string, timeout time.Duration) error {
	return pollUntil(ctx, timeout, "the backup service of cluster "+clusterId+" to be removed", func() (bool, error) {
		d, err := m.getBackupDetail(clusterId)
		if err != nil {
			return false, nil
		}
		switch int64(d.BackupEnabled) {
		case backupStatusErrorEnabled, backupStatusErrorDeleted, backupStatusErrorUpdated:
			return false, fmt.Errorf("the backup service of cluster %s is in error state %s", clusterId, backupStatusName(int64(d.BackupEnabled)))
		}
		return int64(d.BackupEnabled) == backupStatusInactive, nil
	})
}

// waitBackupJob waits until the job is ready and applied reports that the change is visible.
// The job is looked up by name when jobId is 0, because job/create does not return the id.
func (m *databaseApiClient) waitBackupJob(ctx context.Context, clusterId, name string, jobId int64, timeout time.Duration, applied func(backupJob) bool) (*backupJob, error) {
	var found *backupJob
	err := pollUntil(ctx, timeout, "backup job "+name, func() (bool, error) {
		jobs, err := m.listBackupJobs(clusterId)
		if err != nil {
			return false, nil
		}
		for i := range jobs {
			match := jobId != 0 && int64(jobs[i].JobId) == jobId || jobId == 0 && string(jobs[i].Name) == name
			if !match {
				continue
			}
			switch strings.ToLower(string(jobs[i].Status)) {
			case jobStatusCreatedFail, jobStatusUpdatedFail, jobStatusDeletedFail:
				return false, fmt.Errorf("backup job %s ended in status %s", name, jobs[i].Status)
			case jobStatusReady:
				if applied == nil || applied(jobs[i]) {
					found = &jobs[i]
					return true, nil
				}
			}
		}
		return false, nil
	})
	return found, err
}

// waitBackupJobGone waits until the job disappears from job/list.
func (m *databaseApiClient) waitBackupJobGone(ctx context.Context, clusterId string, jobId int64, timeout time.Duration) error {
	return pollUntil(ctx, timeout, fmt.Sprintf("backup job %d to be deleted", jobId), func() (bool, error) {
		jobs, err := m.listBackupJobs(clusterId)
		if err != nil {
			return false, nil
		}
		for _, j := range jobs {
			if int64(j.JobId) == jobId {
				if strings.EqualFold(string(j.Status), jobStatusDeletedFail) {
					return false, fmt.Errorf("deleting backup job %d failed (%s)", jobId, j.Status)
				}
				return false, nil
			}
		}
		return true, nil
	})
}

// waitBackupJobIdle waits until no job is running right now (is_run_now = 0).
func (m *databaseApiClient) waitBackupJobIdle(ctx context.Context, clusterId string, jobId int64, timeout time.Duration) error {
	return pollUntil(ctx, timeout, fmt.Sprintf("backup job %d to finish its run", jobId), func() (bool, error) {
		jobs, err := m.listBackupJobs(clusterId)
		if err != nil {
			return false, nil
		}
		for _, j := range jobs {
			if int64(j.JobId) == jobId {
				return int64(j.IsRunNow) != 1, nil
			}
		}
		return true, nil
	})
}

// jobEditable says whether a job accepts a change now: ready or updated_fail, and not running.
func jobEditable(j backupJob) error {
	status := strings.ToLower(string(j.Status))
	if status != jobStatusReady && status != jobStatusUpdatedFail {
		return fmt.Errorf("backup job %s is %s; wait until it is ready", j.Name, j.Status)
	}
	if int64(j.IsRunNow) == 1 {
		return fmt.Errorf("backup job %s is running; wait until the run finishes", j.Name)
	}
	return nil
}
