package fptcloud_backup_veeam

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceBackupVeeamJobShape(t *testing.T) {
	ds := DataSourceBackupVeeamJob()
	assert.NotNil(t, ds.ReadContext)
	// A data source must not carry write methods.
	assert.Nil(t, ds.CreateContext)
	assert.Nil(t, ds.UpdateContext)
	assert.Nil(t, ds.DeleteContext)
	assert.Nil(t, ds.InternalValidate(nil, false))
}

// The data source reuses flattenJobDetail, so its field names and types have to
// stay compatible with the resource. If they drift, flattenJobDetail starts
// failing at runtime instead of at compile time - this catches that.
func TestDataSourceSchemaAcceptsFlattenJobDetail(t *testing.T) {
	d := schema.TestResourceDataRaw(t, dataSourceBackupVeeamJobSchema, map[string]interface{}{})
	detail := &JobDetail{
		Id:                    "job-1",
		Name:                  "job-1",
		Description:           "d",
		ScheduleEnabled:       true,
		IsCapacityTierEnabled: false,
		BackupObject: []BackupObject{
			{VmId: "vm-1", VmDisplayName: "VM One"},
			{VmId: "vm-2", VmDisplayName: "VM Two"},
		},
		BackupRetention:       RetentionPayload{Cycles: 3, LimitType: "Cycles"},
		NotificationMethodIds: []string{"nm-1"},
		BackupSchedule: &SchedulePayload{
			ScheduleType: "daily",
			// Veeam's casing, as the detail endpoint really answers it.
			DailySchedule: &DailySchedulePayload{Type: "WeekDays", RunAt: "03:00:00"},
		},
	}

	assert.Nil(t, flattenJobDetail(d, detail))

	assert.Equal(t, "job-1", d.Get("name"))
	assert.Equal(t, "d", d.Get("description"))
	assert.Equal(t, 2, d.Get("vm_ids").(*schema.Set).Len())
	assert.Equal(t, 2, d.Get("vm_display_names").(*schema.Set).Len())
	assert.Equal(t, 1, d.Get("notification_method_ids").(*schema.Set).Len())

	retention := d.Get("retention").([]interface{})[0].(map[string]interface{})
	assert.Equal(t, 3, retention["cycles"])
	assert.Equal(t, "Cycles", retention["limit_type"])

	schedule := d.Get("schedule").([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "daily", schedule["type"])
	daily := schedule["daily"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "03:00:00", daily["run_at"])
	// Canonicalised here too, because both paths share flattenJobDetail.
	assert.Equal(t, "weekDays", daily["type"])
}

// The schema must not expose status or enabled: the detail endpoint does not
// return them, so they would always read as empty and mislead.
func TestDataSourceSchemaOmitsFieldsDetailDoesNotReturn(t *testing.T) {
	_, hasStatus := dataSourceBackupVeeamJobSchema["status"]
	_, hasEnabled := dataSourceBackupVeeamJobSchema["enabled"]
	assert.False(t, hasStatus)
	assert.False(t, hasEnabled)
}

type stubJobLister struct {
	BackupVeeamService
	items []JobListItem
	err   error
}

func (s stubJobLister) ListJobs(_ string, _ string, _ string) (JobListResponse, error) {
	if s.err != nil {
		return JobListResponse{}, s.err
	}
	return JobListResponse{Items: s.items, TotalCount: len(s.items)}, nil
}

func TestResolveJobIdByNameExactMatch(t *testing.T) {
	// The server's name filter is a partial match, so "db" also returns
	// "db-daily". Only the exact name may be accepted.
	svc := stubJobLister{items: []JobListItem{
		{Id: "id-daily", Name: "db-daily"},
		{Id: "id-db", Name: "db"},
	}}

	id, err := resolveJobIdByName(svc, "vpc-1", "db")
	assert.Nil(t, err)
	assert.Equal(t, "id-db", id)
}

func TestResolveJobIdByNameNotFound(t *testing.T) {
	svc := stubJobLister{items: []JobListItem{{Id: "id-daily", Name: "db-daily"}}}

	_, err := resolveJobIdByName(svc, "vpc-1", "db")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no backup job named")
}

func TestResolveJobIdByNameEmptyName(t *testing.T) {
	_, err := resolveJobIdByName(stubJobLister{}, "vpc-1", "")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "job_id or name")
}

// Names are unique per VPC so this should be unreachable, but reporting beats
// picking one: acting on a job the caller did not choose is worse than failing.
func TestResolveJobIdByNameAmbiguousReportsInsteadOfGuessing(t *testing.T) {
	svc := stubJobLister{items: []JobListItem{
		{Id: "id-1", Name: "db"},
		{Id: "id-2", Name: "db"},
	}}

	_, err := resolveJobIdByName(svc, "vpc-1", "db")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "id-1")
	assert.Contains(t, err.Error(), "id-2")
	assert.Contains(t, err.Error(), "use job_id instead")
}
