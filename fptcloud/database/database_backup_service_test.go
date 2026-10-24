package fptcloud_database

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	backupPollInterval = time.Millisecond
	restoreAppearTimeout = 2 * time.Second
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".json")
	require.NoError(t, err)
	return b
}

// newTestApi starts a server with the given handler and returns a client for it.
func newTestApi(t *testing.T, handler http.Handler) *databaseApiClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := common.NewClientWithURL("key", server.URL, "VN/SGN2", "tenant", 10)
	require.NoError(t, err)
	return newDatabaseApiClient(client)
}

func TestCheckBusinessError(t *testing.T) {
	assert.NoError(t, checkBusinessError(nil))
	assert.NoError(t, checkBusinessError([]byte("  ")))
	assert.NoError(t, checkBusinessError(fixture(t, "job_create_response")))
	assert.NoError(t, checkBusinessError(fixture(t, "restore_create_response")), "restore/create answers with code, not error_code")

	err := checkBusinessError(fixture(t, "capacity_error_notification"))
	var be *businessError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, int64(160009), be.Code)
	assert.Equal(t, "warning", be.Type)
	assert.Contains(t, err.Error(), `alert_service "backup"`)

	err = checkBusinessError(fixture(t, "capacity_error_decrease"))
	require.ErrorAs(t, err, &be)
	assert.Equal(t, int64(160006), be.Code)
	assert.Contains(t, err.Error(), "must be greater than the current size")

	assert.ErrorContains(t, checkBusinessError([]byte(`{"type":"error","message":"boom"}`)), "boom")
	assert.ErrorContains(t, checkBusinessError([]byte("<html>")), "unexpected response")
}

func TestParseBackupFixtures(t *testing.T) {
	var d backupDetail
	require.NoError(t, json.Unmarshal(fixture(t, "backup_detail_active_pitr"), &d))
	assert.Equal(t, int64(205), int64(d.BackupServiceId))
	assert.Equal(t, int64(1), int64(d.IsArchived))
	assert.Equal(t, int64(5), int64(d.BackupEnabled))
	assert.Equal(t, "quantity", string(d.BackupRetentionType))
	assert.InDelta(t, 0.072, d.usedGb(), 0.001, "used_gb is a percentage of the capacity")

	var inactive backupDetail
	require.NoError(t, json.Unmarshal(fixture(t, "backup_detail_inactive"), &inactive))
	assert.Equal(t, int64(0), int64(inactive.BackupServiceId), "a null service id reads as 0")
	assert.Equal(t, "inactive", backupStatusName(int64(inactive.BackupEnabled)))

	var l struct {
		Jobs []backupJob `json:"jobs"`
	}
	require.NoError(t, json.Unmarshal(fixture(t, "job_list_after_update"), &l))
	require.Len(t, l.Jobs, 2)
	diff := findBackupJob(l.Jobs, "diff")
	require.NotNil(t, diff)
	assert.Equal(t, []int64{6, 10}, diff.hours())
	assert.Equal(t, []int64{1, 2, 3}, diff.days())
	assert.Equal(t, int64(15), diff.minute())
	assert.Equal(t, int64(7), int64(diff.Retention))
	assert.Nil(t, findBackupJob(l.Jobs, "other"))
}

func TestBackupStatusHelpers(t *testing.T) {
	assert.True(t, backupIsReady(1))
	assert.True(t, backupIsReady(5))
	for _, code := range []int64{0, 2, 3, 4, 6, -1, -2, -3} {
		assert.False(t, backupIsReady(code), code)
	}
	assert.Equal(t, "error_updated", backupStatusName(-3))
	assert.Equal(t, "unknown(9)", backupStatusName(9))
}

func TestFlexTypes(t *testing.T) {
	var v struct {
		A flexInt   `json:"a"`
		B flexInt   `json:"b"`
		C flexInt   `json:"c"`
		D flexFloat `json:"d"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"a": 5, "b": "7", "c": null, "d": "0.5"}`), &v))
	assert.Equal(t, flexInt(5), v.A)
	assert.Equal(t, flexInt(7), v.B)
	assert.Equal(t, flexInt(0), v.C)
	assert.Equal(t, flexFloat(0.5), v.D)
	assert.Error(t, json.Unmarshal([]byte(`{"a": "x"}`), &v))
}

func TestMutateSendsMethodAndBody(t *testing.T) {
	type call struct{ method, path, body string }
	var got []call
	api := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf strings.Builder
		b := make([]byte, 1024)
		n, _ := r.Body.Read(b)
		buf.Write(b[:n])
		got = append(got, call{r.Method, r.URL.Path, buf.String()})
		assert.Equal(t, "saigon-02-vn", r.Header.Get("fpt-region"))
		_, _ = w.Write([]byte(`{"message":"ok","type":"success"}`))
	}))

	require.NoError(t, api.setBackupCapacity("c1", 20, 5, 1))
	require.NoError(t, api.updateBackupJob(backupJobRequest{JobId: 9, ClusterId: "c1", Name: "daily-full", Type: "full", Retention: 7, Hour: "1", Minute: "30", DayOfWeek: "1,2"}))
	require.NoError(t, api.createBackupJob(backupJobRequest{JobId: 9, ClusterId: "c1", Name: "daily-full", Type: "full", Retention: 7, Hour: "1", Minute: "30", DayOfWeek: "1,2"}))
	require.NoError(t, api.deleteBackupJob(9))
	require.NoError(t, api.runNowBackupJob(9))
	require.NoError(t, api.disableBackup("c1"))

	require.Len(t, got, 6)
	assert.Equal(t, call{"POST", "/v1/xplat/database/backup/service/capacity", `{"backup_enabled":5,"capacity":20,"cluster_id":"c1","is_archived":1,"storage_profile":null}`}, got[0])
	assert.Equal(t, "PUT", got[1].method)
	assert.Equal(t, "/v1/xplat/database/backup/service/job/update", got[1].path)
	assert.Contains(t, got[1].body, `"job_id":9`)
	assert.Contains(t, got[1].body, `"hour":"1","minute":"30","day_of_week":"1,2"`)
	assert.Equal(t, "POST", got[2].method)
	assert.NotContains(t, got[2].body, "job_id", "a new job has no id")
	assert.Equal(t, call{"DELETE", "/v1/xplat/database/backup/service/job/delete/9", ""}, got[3])
	assert.Equal(t, "/v1/xplat/database/backup/service/job/runnow/9", got[4].path)
	assert.Equal(t, call{"DELETE", "/v1/xplat/database/backup/service/delete/c1", ""}, got[5])
}

func TestMutateReportsBusinessAndHttpErrors(t *testing.T) {
	api := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/xplat/database/backup/service/capacity":
			_, _ = w.Write(fixture(t, "capacity_error_notification"))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"detail":"database down"}`))
		}
	}))
	err := api.setBackupCapacity("c1", 10, 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "160009")

	err = api.deleteBackupJob(3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status_code=500")
	assert.Contains(t, err.Error(), "database down")
}

func TestGetPagedReadsEveryPage(t *testing.T) {
	var calls int32
	api := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		page := r.URL.Query().Get("page")
		items := `[{"cluster_id":"a"},{"cluster_id":"b"}]`
		if page == "2" {
			items = `[{"cluster_id":"c"}]`
		}
		_, _ = w.Write([]byte(`{"code":"200","data":` + items + `,"metadata":{"total_items":3}}`))
	}))
	clusters, err := api.listRestoredClusters("src")
	require.NoError(t, err)
	assert.Len(t, clusters, 3)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestBusyBackupActions(t *testing.T) {
	api := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"task_name":"Run full backup","status":"pending"},{"task_name":"Create Backup Service","status":"successful"}],"metadata":{"total_items":2}}`))
	}))
	busy, err := api.busyBackupActions("c1")
	require.NoError(t, err)
	require.Len(t, busy, 1)
	assert.Equal(t, "Run full backup", string(busy[0].TaskName))

	idle := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture(t, "actions_log"))
	}))
	busy, err = idle.busyBackupActions("c1")
	require.NoError(t, err)
	assert.Empty(t, busy)
}

// serveSequence answers GETs with the bodies in turn, repeating the last one.
func serveSequence(bodies ...string) http.Handler {
	var n int32
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(atomic.AddInt32(&n, 1)) - 1
		if i >= len(bodies) {
			i = len(bodies) - 1
		}
		_, _ = w.Write([]byte(bodies[i]))
	})
}

func detailBody(status int, capacity int) string {
	b, _ := json.Marshal(map[string]interface{}{"cluster_id": "c1", "backup_enabled": status, "capacity": capacity})
	return string(b)
}

func TestWaitBackupReady(t *testing.T) {
	// processing (2) -> enabled (1) with the new capacity
	api := newTestApi(t, serveSequence(detailBody(2, 10), detailBody(4, 10), detailBody(1, 10), detailBody(1, 20)))
	err := api.waitBackupReady(context.Background(), "c1", 5*time.Second, func(d *backupDetail) bool { return int64(d.Capacity) >= 20 })
	require.NoError(t, err)

	// 5 (active) is ready as well
	api = newTestApi(t, serveSequence(detailBody(5, 10)))
	require.NoError(t, api.waitBackupReady(context.Background(), "c1", time.Second, nil))

	// an error state stops the wait
	api = newTestApi(t, serveSequence(detailBody(2, 0), detailBody(-1, 0)))
	err = api.waitBackupReady(context.Background(), "c1", 5*time.Second, nil)
	assert.ErrorContains(t, err, "error_enabled")

	// a service that never settles times out
	api = newTestApi(t, serveSequence(detailBody(2, 0)))
	err = api.waitBackupReady(context.Background(), "c1", 30*time.Millisecond, nil)
	assert.ErrorContains(t, err, "timed out")

	// a cancelled context stops the wait
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, api.waitBackupReady(ctx, "c1", time.Minute, nil), context.Canceled)
}

func TestWaitBackupInactive(t *testing.T) {
	api := newTestApi(t, serveSequence(detailBody(3, 10), detailBody(3, 10), detailBody(0, 0)))
	require.NoError(t, api.waitBackupInactive(context.Background(), "c1", 5*time.Second))

	api = newTestApi(t, serveSequence(detailBody(-2, 10)))
	assert.ErrorContains(t, api.waitBackupInactive(context.Background(), "c1", time.Second), "error_deleted")
}

func jobsBody(jobs ...string) string {
	return `{"jobs":[` + strings.Join(jobs, ",") + `]}`
}

func jobBody(id int, status string, runNow int) string {
	const name = "daily-full"
	return `{"job_id":` + itoa(id) + `,"backup_job_name":"` + name + `","backup_type":"full","backup_retention":7,"backup_schedule":{"minute":"30","hour":"3","day_of_week":"1,2"},"status":"` + status + `","is_run_now":` + itoa(runNow) + `}`
}

func itoa(i int) string { b, _ := json.Marshal(i); return string(b) }

func TestWaitBackupJob(t *testing.T) {
	// found by name, because job/create does not return the id
	api := newTestApi(t, serveSequence(jobsBody(), jobsBody(jobBody(7, "creating", 0)), jobsBody(jobBody(7, "ready", 0))))
	job, err := api.waitBackupJob(context.Background(), "c1", "daily-full", 0, 5*time.Second, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(7), int64(job.JobId))

	// by id, and only once the change is visible
	api = newTestApi(t, serveSequence(jobsBody(jobBody(7, "ready", 0))))
	_, err = api.waitBackupJob(context.Background(), "c1", "daily-full", 7, 40*time.Millisecond, func(j backupJob) bool { return j.minute() == 45 })
	assert.ErrorContains(t, err, "timed out")

	for _, status := range []string{"created_fail", "updated_fail", "deleted_fail"} {
		api = newTestApi(t, serveSequence(jobsBody(jobBody(7, status, 0))))
		_, err = api.waitBackupJob(context.Background(), "c1", "daily-full", 7, time.Second, nil)
		assert.ErrorContains(t, err, status)
	}
}

func TestWaitBackupJobGoneAndIdle(t *testing.T) {
	api := newTestApi(t, serveSequence(jobsBody(jobBody(7, "ready", 0)), jobsBody(jobBody(7, "deleting", 0)), jobsBody()))
	require.NoError(t, api.waitBackupJobGone(context.Background(), "c1", 7, 5*time.Second))

	api = newTestApi(t, serveSequence(jobsBody(jobBody(7, "deleted_fail", 0))))
	assert.ErrorContains(t, api.waitBackupJobGone(context.Background(), "c1", 7, time.Second), "failed")

	api = newTestApi(t, serveSequence(jobsBody(jobBody(7, "ready", 1)), jobsBody(jobBody(7, "ready", 1)), jobsBody(jobBody(7, "ready", 0))))
	require.NoError(t, api.waitBackupJobIdle(context.Background(), "c1", 7, 5*time.Second))
}

func TestJobEditable(t *testing.T) {
	var ready, failed, running, busy backupJob
	require.NoError(t, json.Unmarshal([]byte(jobBody(1, "ready", 0)), &ready))
	require.NoError(t, json.Unmarshal([]byte(jobBody(1, "updated_fail", 0)), &failed))
	require.NoError(t, json.Unmarshal([]byte(jobBody(1, "ready", 1)), &running))
	require.NoError(t, json.Unmarshal([]byte(jobBody(1, "updating", 0)), &busy))
	assert.NoError(t, jobEditable(ready))
	assert.NoError(t, jobEditable(failed), "a failed update can be retried")
	assert.ErrorContains(t, jobEditable(running), "is running")
	assert.ErrorContains(t, jobEditable(busy), "is updating")
}

func TestGetsAreRepeatedAfterATransientError(t *testing.T) {
	var calls int32
	api := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) <= 2 {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":"403","message":"403 FORBIDDEN"}`))
			return
		}
		_, _ = w.Write(fixture(t, "backup_detail_enabled"))
	}))
	d, err := api.getBackupDetail("d4ln5mv2")
	require.NoError(t, err)
	assert.Equal(t, int64(229), int64(d.BackupServiceId))
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))

	// A failure that lasts is still reported.
	always := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	_, err = always.getBackupDetail("c1")
	assert.ErrorContains(t, err, "403")
}

func TestBackupDetailWithoutAStateIsAnError(t *testing.T) {
	for _, body := range []string{`{"cluster_id":"c1","capacity":10}`, `{"cluster_id":"c1","backup_enabled":null}`} {
		api := newTestApi(t, serveSequence(body))
		_, err := api.getBackupDetail("c1")
		assert.ErrorContains(t, err, "backup state of cluster c1 is missing", body)
	}
}
