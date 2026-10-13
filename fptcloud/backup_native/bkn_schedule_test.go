package fptcloud_backup_native

import (
	"testing"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scheduleConfig(day cty.Value) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"schedule": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"full_backup_frequency": cty.StringVal("weekly"),
			"full_backup_day":       day,
			"time":                  cty.StringVal("02:00"),
			"time_zone":             cty.StringVal("UTC"),
		})}),
	})
}

// Review focus 1: Monday is 0 and must count as set.
func TestScheduleDaySetTreatsZeroAsSet(t *testing.T) {
	set, known := scheduleDaySet(scheduleConfig(cty.NumberIntVal(0)))
	assert.True(t, known)
	assert.True(t, set)

	set, known = scheduleDaySet(scheduleConfig(cty.NullVal(cty.Number)))
	assert.True(t, known)
	assert.False(t, set)

	_, known = scheduleDaySet(scheduleConfig(cty.UnknownVal(cty.Number)))
	assert.True(t, known, "an unknown day is still a set day")

	_, known = scheduleDaySet(cty.UnknownVal(cty.DynamicPseudoType))
	assert.False(t, known)
}

func TestValidateScheduleDay(t *testing.T) {
	assert.NoError(t, validateScheduleDay(Schedule{FullBackupFrequency: "daily"}, false))
	assert.ErrorContains(t, validateScheduleDay(Schedule{FullBackupFrequency: "daily"}, true), "not allowed")
	assert.ErrorContains(t, validateScheduleDay(Schedule{FullBackupFrequency: "weekly"}, false), "required")
	assert.NoError(t, validateScheduleDay(Schedule{FullBackupFrequency: "weekly", FullBackupDay: 0}, true))
	assert.ErrorContains(t, validateScheduleDay(Schedule{FullBackupFrequency: "weekly", FullBackupDay: 7}, true), "0-6")
	assert.ErrorContains(t, validateScheduleDay(Schedule{FullBackupFrequency: "monthly"}, false), "required")
	assert.ErrorContains(t, validateScheduleDay(Schedule{FullBackupFrequency: "monthly", FullBackupDay: 0}, true), "1-31")
	assert.NoError(t, validateScheduleDay(Schedule{FullBackupFrequency: "monthly", FullBackupDay: 31}, true))
}

func TestExpandAndFlattenSchedule(t *testing.T) {
	raw := []interface{}{map[string]interface{}{
		"full_backup_frequency": "monthly", "full_backup_day": 15, "time": "02:30", "time_zone": "Asia/Tokyo",
	}}
	s := expandSchedule(raw)
	assert.Equal(t, Schedule{FullBackupFrequency: "monthly", FullBackupDay: 15, Time: "02:30", TimeZone: "Asia/Tokyo"}, s)
	assert.Equal(t, raw, flattenSchedule(s))

	daily := flattenSchedule(Schedule{FullBackupFrequency: "daily", FullBackupDay: 5, Time: "01:00", TimeZone: "UTC"})
	assert.Equal(t, 0, daily[0].(map[string]interface{})["full_backup_day"], "daily never carries a day")

	assert.Equal(t, "UTC", expandSchedule(nil).TimeZone)
}

func TestApplyVmCreateScheduleDailyIsQuick(t *testing.T) {
	var p VmJobCreatePayload
	applyVmCreateSchedule(&p, Schedule{FullBackupFrequency: "daily", Time: "02:00", TimeZone: "Asia/Ho_Chi_Minh"})
	assert.Equal(t, "Quick", p.BackupMode)
	require.NotNil(t, p.QuickBackupInput)
	assert.Equal(t, VmQuickBackupFreq{Full: "Daily", Incremental: ""}, p.QuickBackupInput.QuickBackupFreq)
	assert.Equal(t, "02:00", p.QuickBackupInput.QuickBackupTime)
	assert.Nil(t, p.AdvancedBackupInput)
	assert.Equal(t, "Asia/Ho_Chi_Minh", p.TimeZone)
}

func TestApplyVmCreateScheduleWeeklyIsAdvance(t *testing.T) {
	var p VmJobCreatePayload
	applyVmCreateSchedule(&p, Schedule{FullBackupFrequency: "weekly", FullBackupDay: 0, Time: "03:15", TimeZone: "UTC"})
	assert.Equal(t, "Advance", p.BackupMode)
	assert.Nil(t, p.QuickBackupInput)
	require.NotNil(t, p.AdvancedBackupInput)
	a := p.AdvancedBackupInput
	assert.Equal(t, "daily", a.IncrementalBackupFreq)
	assert.Equal(t, "03:15", a.IncrementalBackupTime)
	assert.True(t, a.EnableFullBackup)
	assert.Equal(t, "Weekly", a.FullyBackupFreq)
	require.NotNil(t, a.FullyBackupDate)
	assert.Equal(t, 0, *a.FullyBackupDate)
}

func TestApplyVmUpdateSchedule(t *testing.T) {
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

	var daily VmJobUpdatePayload
	require.NoError(t, applyVmUpdateSchedule(&daily, Schedule{FullBackupFrequency: "daily", Time: "02:00", TimeZone: "Asia/Ho_Chi_Minh"}, now))
	assert.Equal(t, "daily", daily.FullBackupFreq)
	assert.Equal(t, "", daily.IncrementalBackupFreq)
	assert.Nil(t, daily.FullBackupDate)
	assert.Equal(t, "2026-10-12T19:00:00Z", daily.StartTime)

	var monthly VmJobUpdatePayload
	require.NoError(t, applyVmUpdateSchedule(&monthly, Schedule{FullBackupFrequency: "monthly", FullBackupDay: 31, Time: "02:00", TimeZone: "UTC"}, now))
	assert.Equal(t, "monthly", monthly.FullBackupFreq)
	assert.Equal(t, "daily", monthly.IncrementalBackupFreq)
	require.NotNil(t, monthly.FullBackupDate)
	assert.Equal(t, 31, *monthly.FullBackupDate)
}

func TestVmScheduleFromDetail(t *testing.T) {
	day := 5
	weekly, err := vmScheduleFromDetail(&VmJobDetail{FullBackupFreq: "Weekly", IncrementalBackupFreq: "daily", FullBackupDate: &day, StartTime: "2026-10-12T19:00:00"}, "Asia/Ho_Chi_Minh")
	require.NoError(t, err)
	assert.Equal(t, Schedule{FullBackupFrequency: "weekly", FullBackupDay: 5, Time: "02:00", TimeZone: "Asia/Ho_Chi_Minh"}, weekly)

	stale := 3
	daily, err := vmScheduleFromDetail(&VmJobDetail{FullBackupFreq: "daily", FullBackupDate: &stale, StartTime: "2026-10-12T19:00:00"}, "UTC")
	require.NoError(t, err)
	assert.Equal(t, 0, daily.FullBackupDay, "a daily job ignores the stored day")
	assert.Equal(t, "19:00", daily.Time)
}

func TestVolumeScheduleMapping(t *testing.T) {
	var p VolumeJobPayload
	applyVolumeSchedule(&p, Schedule{FullBackupFrequency: "monthly", FullBackupDay: 15, Time: "02:00", TimeZone: "Asia/Ho_Chi_Minh"})
	assert.Equal(t, 2, p.BackupMode)
	require.NotNil(t, p.FullBackupDate)
	assert.Equal(t, 15, *p.FullBackupDate)
	assert.Equal(t, "02:00:00", p.BackupTime)
	assert.Equal(t, "Asia/Ho_Chi_Minh", p.BackupTimeZone)

	applyVolumeSchedule(&p, Schedule{FullBackupFrequency: "daily", Time: "02:00", TimeZone: "UTC"})
	assert.Equal(t, 0, p.BackupMode)
	assert.Nil(t, p.FullBackupDate, "daily sends no day, as the portal does")
}

// A weekly job changed to daily keeps its old day in the backend (spec,
// volume job schedule mapping); Read must not report it.
func TestVolumeScheduleFromDetailIgnoresStaleDayForDaily(t *testing.T) {
	stale := 5
	s := volumeScheduleFromDetail(&VolumeJobDetail{BackupMode: 0, FullBackupDate: &stale, BackupTime: "02:00:00", BackupTimeZone: "Asia/Tokyo"})
	assert.Equal(t, Schedule{FullBackupFrequency: "daily", FullBackupDay: 0, Time: "02:00", TimeZone: "Asia/Tokyo"}, s)

	weekly := volumeScheduleFromDetail(&VolumeJobDetail{BackupMode: 1, FullBackupDate: &stale, BackupTime: "23:45:00"})
	assert.Equal(t, Schedule{FullBackupFrequency: "weekly", FullBackupDay: 5, Time: "23:45", TimeZone: "UTC"}, weekly)
}
