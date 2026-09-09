package fptcloud_backup_veeam

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestExpandJobPayloadDaily(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamJobSchema, map[string]interface{}{
		"vpc_id":           "vpc-1",
		"name":             "job-1",
		"description":      "d",
		"vm_ids":           []interface{}{"vm-1", "vm-2"},
		"schedule_enabled": true,
		"retention":        []interface{}{map[string]interface{}{"cycles": 7, "limit_type": "Days"}},
		"schedule": []interface{}{map[string]interface{}{
			"type":  "daily",
			"daily": []interface{}{map[string]interface{}{"type": "weekDays", "run_at": "22:00:00"}},
		}},
	})

	payload := expandJobPayload(d)

	assert.Equal(t, "job-1", payload.Name)
	assert.Len(t, payload.VmIds, 2)
	assert.Equal(t, 7, payload.Retention.Cycles)
	assert.Equal(t, "Days", payload.Retention.LimitType)
	// enabled is always sent as true - the portal hardcodes it too, and the
	// update endpoint overwrites it from the database, so it is harmless.
	assert.True(t, payload.Enabled)
	assert.Equal(t, "daily", payload.Schedule.ScheduleType)
	assert.NotNil(t, payload.Schedule.DailySchedule)
	assert.Nil(t, payload.Schedule.MonthlySchedule)
	assert.Nil(t, payload.Schedule.PeriodSchedule)
	assert.Equal(t, "weekDays", payload.Schedule.DailySchedule.Type)
	// days always carries all seven entries; the user never picks them.
	assert.Len(t, payload.Schedule.DailySchedule.Days, 7)
}

func TestExpandJobPayloadTrimsName(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamJobSchema, map[string]interface{}{
		"vpc_id":    "vpc-1",
		"name":      "  job-1  ",
		"vm_ids":    []interface{}{"vm-1"},
		"retention": []interface{}{map[string]interface{}{"cycles": 7, "limit_type": "Days"}},
	})

	assert.Equal(t, "job-1", expandJobPayload(d).Name)
}

func TestExpandJobPayloadPeriodBuildsBitmap(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamJobSchema, map[string]interface{}{
		"vpc_id":    "vpc-1",
		"name":      "job-1",
		"vm_ids":    []interface{}{"vm-1"},
		"retention": []interface{}{map[string]interface{}{"cycles": 7, "limit_type": "Days"}},
		"schedule": []interface{}{map[string]interface{}{
			"type":   "period",
			"period": []interface{}{map[string]interface{}{"full_period": 4, "start_hour": 20, "end_hour": 23}},
		}},
	})

	payload := expandJobPayload(d)

	assert.Equal(t, "period", payload.Schedule.ScheduleType)
	assert.Equal(t, "Hours", payload.Schedule.PeriodSchedule.Type)
	assert.Equal(t, 4, payload.Schedule.PeriodSchedule.FullPeriod)
	assert.Len(t, payload.Schedule.PeriodSchedule.Schedules, 7)

	start, end, ok := ParsePeriodBitmap(payload.Schedule.PeriodSchedule.Schedules)
	assert.True(t, ok)
	assert.Equal(t, 20, start)
	assert.Equal(t, 23, end)
}

// day_of_month is only sent when day_number_in_month is onDay.
func TestExpandJobPayloadMonthlyOnDayOnlySendsDayOfMonth(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamJobSchema, map[string]interface{}{
		"vpc_id":    "vpc-1",
		"name":      "job-1",
		"vm_ids":    []interface{}{"vm-1"},
		"retention": []interface{}{map[string]interface{}{"cycles": 7, "limit_type": "Days"}},
		"schedule": []interface{}{map[string]interface{}{
			"type": "monthly",
			"monthly": []interface{}{map[string]interface{}{
				"day_number_in_month": "onDay", "day_of_month": 32, "run_at": "22:00:00",
			}},
		}},
	})

	payload := expandJobPayload(d)

	assert.Equal(t, 32, payload.Schedule.MonthlySchedule.DayOfMonth)
	assert.Equal(t, "", payload.Schedule.MonthlySchedule.DayOfWeek)
	assert.Len(t, payload.Schedule.MonthlySchedule.Months, 12)
}

func TestExpandJobPayloadMonthlyWeekPositionOnlySendsDayOfWeek(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamJobSchema, map[string]interface{}{
		"vpc_id":    "vpc-1",
		"name":      "job-1",
		"vm_ids":    []interface{}{"vm-1"},
		"retention": []interface{}{map[string]interface{}{"cycles": 7, "limit_type": "Days"}},
		"schedule": []interface{}{map[string]interface{}{
			"type": "monthly",
			"monthly": []interface{}{map[string]interface{}{
				"day_number_in_month": "fourth", "day_of_week": "saturday",
			}},
		}},
	})

	payload := expandJobPayload(d)

	assert.Equal(t, "saturday", payload.Schedule.MonthlySchedule.DayOfWeek)
	assert.Equal(t, 0, payload.Schedule.MonthlySchedule.DayOfMonth)
}

func TestFlattenJobDetailRoundTrip(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamJobSchema, map[string]interface{}{})
	detail := &JobDetail{
		Id:              "job-1",
		Name:            "job-1",
		Description:     "d",
		ScheduleEnabled: true,
		BackupObject:    []BackupObject{{VmId: "vm-1", VmDisplayName: "VM One"}},
		BackupRetention: RetentionPayload{Cycles: 7, LimitType: "Days"},
		BackupSchedule: &SchedulePayload{
			ScheduleType:  "daily",
			DailySchedule: &DailySchedulePayload{Type: "weekDays", RunAt: "22:00:00"},
		},
	}

	err := flattenJobDetail(d, detail)
	assert.Nil(t, err)
	assert.Equal(t, "job-1", d.Get("name"))
	assert.Equal(t, "d", d.Get("description"))
	assert.Equal(t, 1, d.Get("vm_ids").(*schema.Set).Len())
	assert.Equal(t, 1, d.Get("vm_display_names").(*schema.Set).Len())

	schedule := d.Get("schedule").([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "daily", schedule["type"])
	daily := schedule["daily"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "weekDays", daily["type"])
}

// An all-zero bitmap must not be mapped to start_hour/end_hour; let the plan show a diff.
func TestFlattenJobDetailSkipsBrokenPeriodBitmap(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamJobSchema, map[string]interface{}{})
	detail := &JobDetail{
		Id:   "job-1",
		Name: "job-1",
		BackupSchedule: &SchedulePayload{
			ScheduleType: "period",
			PeriodSchedule: &PeriodSchedulePayload{
				FullPeriod: 4,
				Schedules:  []PeriodScheduleEntry{{Name: "Sunday", Value: "0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0"}},
			},
		},
		BackupRetention: RetentionPayload{Cycles: 7, LimitType: "Days"},
	}

	err := flattenJobDetail(d, detail)
	assert.Nil(t, err)
	schedule := d.Get("schedule").([]interface{})[0].(map[string]interface{})
	period := schedule["period"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, 0, period["start_hour"])
	assert.Equal(t, 0, period["end_hour"])
}

// The backend reads day_of_month back as 1 when the job does not use onDay;
// writing that into state would create a phantom diff, so it must be skipped.
func TestFlattenJobDetailOmitsDayOfMonthWhenNotOnDay(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBackupVeeamJobSchema, map[string]interface{}{})
	detail := &JobDetail{
		Id:   "job-1",
		Name: "job-1",
		BackupSchedule: &SchedulePayload{
			ScheduleType: "monthly",
			MonthlySchedule: &MonthlySchedulePayload{
				RunAt:            "22:00:00",
				DayNumberInMonth: "fourth",
				DayOfWeek:        "saturday",
				DayOfMonth:       1, // the backend returns junk in this field
			},
		},
		BackupRetention: RetentionPayload{Cycles: 7, LimitType: "Days"},
	}

	err := flattenJobDetail(d, detail)
	assert.Nil(t, err)
	schedule := d.Get("schedule").([]interface{})[0].(map[string]interface{})
	monthly := schedule["monthly"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "saturday", monthly["day_of_week"])
	assert.Equal(t, 0, monthly["day_of_month"])
}

func TestResourceBackupVeeamJobShape(t *testing.T) {
	res := ResourceBackupVeeamJob()
	assert.NotNil(t, res.CreateContext)
	assert.NotNil(t, res.ReadContext)
	assert.NotNil(t, res.UpdateContext)
	assert.NotNil(t, res.DeleteContext)
	assert.NotNil(t, res.CustomizeDiff)
	assert.NotNil(t, res.Importer)
	assert.NotNil(t, res.Timeouts)
	assert.Nil(t, res.InternalValidate(nil, true))
}
