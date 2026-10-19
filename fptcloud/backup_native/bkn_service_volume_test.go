package fptcloud_backup_native

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	volBase = "/v2/vpc/vpc-1/backup-volume"

	volumeJobDetailBody = `{"data":{"id":"job-9","name":"pvc","description":"pv backups",
		"notifications":[{"id":"n-1","name":"ops"}],"retention":7,"full_backup_date":5,"backup_mode":1,
		"backup_time":"02:00:00","status":"initiated","created_at":"2026-10-13T08:00:00",
		"updated_at":"2026-10-13T08:00:00","is_deleted":false,"is_enabled":true,
		"backup_time_zone":"Asia/Ho_Chi_Minh",
		"targets":[{"id":"vol-1","name":"pv-fke-data","display_name":"pv-fke-data","is_deleted":false}]}}`

	volumeJobsEmptyBody = `{"total":0,"data":[],"page_index":1,"page_size":100,"has_more":false}`

	volumeJobsWithPvcBody = `{"total":2,"data":[
		{"id":"job-1","name":"web","job_type":"1","status":"Success","is_enabled":true,"instances":[]},
		{"id":"job-9","name":"pvc","job_type":"0","status":"initiated","is_enabled":true,
		 "next_run":"2026-10-14T02:00:00+07:00","created_at":"2026-10-13T08:00:00",
		 "instances":[{"id":"vol-1","name":"pv-fke-data","display_name":"pv-fke-data","is_deleted":false}]}],
		"page_index":1,"page_size":100,"has_more":false}`
)

func TestListVolumeJobsPagesAndFiltersVmJobs(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs": {
			{Body: `{"total":3,"data":[{"id":"job-1","name":"web","job_type":"1","instances":[]},{"id":"job-8","name":"a","job_type":"0","instances":[]}],"has_more":true}`},
			{Body: `{"total":3,"data":[{"id":"job-9","name":"b","job_type":0,"instances":[]}],"has_more":false}`},
		},
	})
	jobs, err := NewVolumeService(client).ListVolumeJobs("vpc-1")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "job-8", jobs[0].Id)
	assert.Equal(t, "job-9", jobs[1].Id, "a numeric job_type is accepted too")
	assert.Equal(t, 2, api.count("GET "+volBase+"/backup-jobs"))
	assert.Equal(t, "2", api.last("GET "+volBase+"/backup-jobs").Query.Get("page"))
}

// A job created between two page reads pushes job-8 onto page 2 as well.
func TestListVolumeJobsSkipsAJobShiftedOntoTheNextPage(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs": {
			{Body: `{"total":2,"data":[{"id":"job-9","name":"b","job_type":"0","instances":[]},{"id":"job-8","name":"a","job_type":"0","instances":[]}],"has_more":true}`},
			{Body: `{"total":3,"data":[{"id":"job-8","name":"a","job_type":"0","instances":[]},{"id":"job-7","name":"c","job_type":"0","instances":[]}],"has_more":false}`},
		},
	})
	jobs, err := NewVolumeService(client).ListVolumeJobs("vpc-1")
	require.NoError(t, err)
	ids := make([]string, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.Id)
	}
	assert.Equal(t, []string{"job-9", "job-8", "job-7"}, ids)
}

func TestCreateVolumeJobSuccess(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + volBase + "/backup-jobs": {{Status: 201, Body: `{"message":"Backup job created successfully","data":[],"status":true}`}},
	})
	failures, err := NewVolumeService(client).CreateVolumeJob("vpc-1", VolumeJobPayload{Name: "pvc", BackupObjects: []string{"vol-1"}})
	require.NoError(t, err)
	assert.Empty(t, failures)
	assert.Equal(t, []interface{}{"vol-1"}, api.last("POST " + volBase + "/backup-jobs").Body["backup_objects"])
}

func TestCreateVolumeJobPartialFailure(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + volBase + "/backup-jobs": {{Status: 202, Body: `{"message":"partial","status":true,"data":[{"volume_id":"vol-2","name":"pv-fke-b","created_resource":[],"error":"AWX failed"}]}`}},
	})
	failures, err := NewVolumeService(client).CreateVolumeJob("vpc-1", VolumeJobPayload{Name: "pvc"})
	require.NoError(t, err)
	assert.Equal(t, []VolumeFailure{{VolumeId: "vol-2", Name: "pv-fke-b", Error: "AWX failed"}}, failures)
}

func TestCreateVolumeJobAllFailedIsAnError(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + volBase + "/backup-jobs": {{Status: 500, Body: `{"message":"Failed to create backup job","status":true,"data":[{"volume_id":"vol-1","name":"pv-fke-a","error":"AWX failed"}]}`}},
	})
	_, err := NewVolumeService(client).CreateVolumeJob("vpc-1", VolumeJobPayload{Name: "pvc"})
	assert.ErrorContains(t, err, "failed for every volume: volume pv-fke-a (vol-1): AWX failed")
}

func TestGetVolumeJob(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9": {{Body: volumeJobDetailBody}},
	})
	job, err := NewVolumeService(client).GetVolumeJob("vpc-1", "job-9")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, 1, job.BackupMode)
	assert.Equal(t, "Asia/Ho_Chi_Minh", job.BackupTimeZone)
	assert.Equal(t, "vol-1", job.Targets[0].Id)
}

// v2 detail answers 400, not 404, for a missing job.
func TestGetVolumeJob400NotFoundMeansGone(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backup-jobs/job-9": {{Status: 400, Body: `{"message":"Backup job not found with id job-9"}`}},
	})
	job, err := NewVolumeService(client).GetVolumeJob("vpc-1", "job-9")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestDeleteVolumeJobTreatsNotFoundAsDone(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"DELETE " + volBase + "/backup-jobs/job-9": {{Status: 400, Body: `{"message":"Backup job not found with id job-9"}`}},
	})
	assert.NoError(t, NewVolumeService(client).DeleteVolumeJob("vpc-1", "job-9"))
}

func TestSetVolumeJobEnabledSendsLowercaseAction(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + volBase + "/backup-jobs/job-9/actions": {{Body: `{"message":"Request accepted"}`}},
	})
	require.NoError(t, NewVolumeService(client).SetVolumeJobEnabled("vpc-1", "job-9", false))
	assert.Equal(t, "disable", api.last("POST " + volBase + "/backup-jobs/job-9/actions").Body["action"])
}

func TestListVolumeRestorePoints(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/backups": {{Body: `{"data":{"volume_id":"vol-1","restore_points":[{"id":"rp-1","created_at":"2026-10-13T08:00:00","status":"available","is_available":true,"is_incremental":true,"size":0.4,"volume_id":"vol-1"}]}}`}},
	})
	points, err := NewVolumeService(client).ListVolumeRestorePoints("vpc-1", "vol-1")
	require.NoError(t, err)
	assert.Equal(t, []VolumeRestorePoint{{Id: "rp-1", CreatedAt: "2026-10-13T08:00:00", Status: "available", IsAvailable: true, IsIncremental: true, Size: 0.4}}, points)
}

func TestRestoreVolumeSendsMetadata(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + volBase + "/volumes/vol-1/restore": {{Status: 202, Body: `{"message":"Request accepted","status":true}`}},
	})
	require.NoError(t, NewVolumeService(client).RestoreVolume("vpc-1", "vol-1", VolumeRestorePayload{BackupId: "rp-1", Metadata: VolumeRestoreMetadata{Name: "restored", Size: 20}}))
	body := api.last("POST " + volBase + "/volumes/vol-1/restore").Body
	assert.Equal(t, "rp-1", body["backup_id"])
	assert.Equal(t, map[string]interface{}{"name": "restored", "size": float64(20)}, body["metadata"])
}

func TestListVolumeHistoryPage(t *testing.T) {
	_, client := newFakeAPI(t, map[string][]fakeResponse{
		"GET " + volBase + "/history/combined-history": {{Body: `{"items":[{"source_type":"volume","object_id":"vol-1","type":"Restore keep","status":"failed","start_time":"2026-10-13T08:00:00","end_time":null,"history_metadata":{"extract_info":{"status":"quota exceeded"}}}],"has_more":false,"total":1,"page":1,"page_size":50}`}},
	})
	page, err := NewVolumeService(client).ListVolumeHistoryPage("vpc-1", 1, 50)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "vol-1", page.Items[0].ObjectId)
	assert.Equal(t, "quota exceeded", page.Items[0].HistoryMetadata.ExtractInfo["status"])
}

func TestStartVolumeJobBackup(t *testing.T) {
	api, client := newFakeAPI(t, map[string][]fakeResponse{
		"POST " + volBase + "/backup-jobs/job-9/executions": {{Body: `{"message":"Backup job started successfully","status":true}`}},
	})
	require.NoError(t, NewVolumeService(client).StartVolumeJobBackup("vpc-1", "job-9"))
	assert.Empty(t, api.last("POST "+volBase+"/backup-jobs/job-9/executions").Body)
}
