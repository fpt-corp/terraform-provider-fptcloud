package fptcloud_backup_native

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	frequencyDaily   = "daily"
	frequencyWeekly  = "weekly"
	frequencyMonthly = "monthly"
)

// v2 backup_mode values. Only weekly and monthly use full_backup_date.
const (
	volumeModeDaily   = 0
	volumeModeWeekly  = 1
	volumeModeMonthly = 2
)

var scheduleTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// Schedule is the schedule block both job resources share.
type Schedule struct {
	FullBackupFrequency string
	FullBackupDay       int
	Time                string
	TimeZone            string
}

func scheduleSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		MaxItems:    1,
		Description: "When backups run.",
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"full_backup_frequency": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{frequencyDaily, frequencyWeekly, frequencyMonthly}, false),
				Description:  "`daily` runs a full backup every day. `weekly` and `monthly` run a full backup on `full_backup_day` and an incremental backup on the other days.",
			},
			"full_backup_day": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 31),
				Description:  "Day of the full backup. `weekly`: 0-6, Monday is 0. `monthly`: 1-31. Required for `weekly` and `monthly`, not allowed for `daily`.",
			},
			"time": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(scheduleTimePattern, "must be HH:MM in 24-hour time"),
				Description:  "Time of day the backup starts, `HH:MM` in `time_zone`.",
			},
			"time_zone": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "UTC",
				ValidateFunc: validateTimeZone,
				Description:  "IANA time zone of `time`, for example `Asia/Ho_Chi_Minh`.",
			},
		}},
	}
}

func validateTimeZone(v interface{}, key string) ([]string, []error) {
	zone, _ := v.(string)
	if zone == "" || zone == "Local" {
		return nil, []error{fmt.Errorf("%s must be an IANA time zone name, got %q", key, zone)}
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return nil, []error{fmt.Errorf("%s must be an IANA time zone name, got %q", key, zone)}
	}
	return nil, nil
}

func expandSchedule(raw interface{}) Schedule {
	list, _ := raw.([]interface{})
	if len(list) == 0 || list[0] == nil {
		return Schedule{TimeZone: "UTC"}
	}
	m := list[0].(map[string]interface{})
	s := Schedule{}
	s.FullBackupFrequency, _ = m["full_backup_frequency"].(string)
	s.FullBackupDay, _ = m["full_backup_day"].(int)
	s.Time, _ = m["time"].(string)
	s.TimeZone, _ = m["time_zone"].(string)
	if s.TimeZone == "" {
		s.TimeZone = "UTC"
	}
	return s
}

func flattenSchedule(s Schedule) []interface{} {
	day := s.FullBackupDay
	if s.FullBackupFrequency == frequencyDaily {
		day = 0
	}
	return []interface{}{map[string]interface{}{
		"full_backup_frequency": s.FullBackupFrequency,
		"full_backup_day":       day,
		"time":                  s.Time,
		"time_zone":             s.TimeZone,
	}}
}

// validateScheduleDay checks full_backup_day against the frequency. daySet
// says whether the configuration sets full_backup_day at all, because 0 is a
// real value (Monday) and cannot stand for "unset". dayKnown is false when the
// day comes from a value Terraform learns only at apply; s.FullBackupDay then
// reads 0, so only its presence is checked.
func validateScheduleDay(s Schedule, daySet bool, dayKnown bool) error {
	switch s.FullBackupFrequency {
	case frequencyDaily:
		if daySet {
			return fmt.Errorf("schedule.full_backup_day is not allowed when full_backup_frequency is daily")
		}
	case frequencyWeekly:
		if !daySet {
			return fmt.Errorf("schedule.full_backup_day is required when full_backup_frequency is weekly (0-6, Monday is 0)")
		}
		if dayKnown && (s.FullBackupDay < 0 || s.FullBackupDay > 6) {
			return fmt.Errorf("schedule.full_backup_day must be 0-6 (Monday is 0) when full_backup_frequency is weekly, got %d", s.FullBackupDay)
		}
	case frequencyMonthly:
		if !daySet {
			return fmt.Errorf("schedule.full_backup_day is required when full_backup_frequency is monthly (1-31)")
		}
		if dayKnown && (s.FullBackupDay < 1 || s.FullBackupDay > 31) {
			return fmt.Errorf("schedule.full_backup_day must be 1-31 when full_backup_frequency is monthly, got %d", s.FullBackupDay)
		}
	}
	return nil
}

// scheduleDaySet reads from the raw configuration whether
// schedule.full_backup_day is written and whether its value is known yet.
// known is false when the schedule block itself is not known yet; the check
// then waits for apply.
func scheduleDaySet(rawConfig cty.Value) (set bool, dayKnown bool, known bool) {
	if !rawConfig.IsKnown() || rawConfig.IsNull() || !rawConfig.Type().IsObjectType() || !rawConfig.Type().HasAttribute("schedule") {
		return false, false, false
	}
	sched := rawConfig.GetAttr("schedule")
	if !sched.IsKnown() || sched.IsNull() || sched.LengthInt() == 0 {
		return false, false, false
	}
	block := sched.Index(cty.NumberIntVal(0))
	if !block.IsKnown() || block.IsNull() {
		return false, false, false
	}
	day := block.GetAttr("full_backup_day")
	return !day.IsNull(), day.IsKnown(), true
}

// customizeSchedule is the CustomizeDiff step both job resources share.
func customizeSchedule(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	set, dayKnown, known := scheduleDaySet(d.GetRawConfig())
	if !known {
		return nil
	}
	return validateScheduleDay(expandSchedule(d.Get("schedule")), set, dayKnown)
}

// applyVmCreateSchedule fills the schedule part of a v1 create body the way
// the portal does: daily is sent as Quick, weekly and monthly as Advance with
// the chosen day (portal create/CreateBackupPage.js:107-134).
func applyVmCreateSchedule(p *VmJobCreatePayload, s Schedule) {
	p.TimeZone = s.TimeZone
	if s.FullBackupFrequency == frequencyDaily {
		p.BackupMode = "Quick"
		p.QuickBackupInput = &VmQuickBackupInput{
			QuickBackupFreq: VmQuickBackupFreq{Full: "Daily", Incremental: ""},
			QuickBackupTime: s.Time,
		}
		p.AdvancedBackupInput = nil
		return
	}
	day := s.FullBackupDay
	fullFreq := "Weekly"
	if s.FullBackupFrequency == frequencyMonthly {
		fullFreq = "Monthly"
	}
	p.BackupMode = "Advance"
	p.QuickBackupInput = nil
	p.AdvancedBackupInput = &VmAdvancedBackupInput{
		IncrementalBackupFreq: "daily",
		IncrementalBackupTime: s.Time,
		EnableFullBackup:      true,
		FullyBackupFreq:       fullFreq,
		FullyBackupDate:       &day,
	}
}

// applyVmUpdateSchedule fills the schedule part of a v1 update body.
// start_time is UTC with a Z suffix: the backend ignores any other offset
// (awx_handler.py:426-433). A daily schedule sends no day, as the portal does.
func applyVmUpdateSchedule(p *VmJobUpdatePayload, s Schedule, now time.Time) error {
	start, err := scheduleTimeToUTC(s.Time, s.TimeZone, now)
	if err != nil {
		return err
	}
	p.StartTime = start.Format("2006-01-02T15:04:05Z")
	p.FullBackupFreq = s.FullBackupFrequency
	if s.FullBackupFrequency == frequencyDaily {
		p.IncrementalBackupFreq = ""
		p.FullBackupDate = nil
		return nil
	}
	day := s.FullBackupDay
	p.IncrementalBackupFreq = "daily"
	p.FullBackupDate = &day
	return nil
}

// vmScheduleFromDetail rebuilds the schedule block from a v1 job. The API
// stores only the UTC start time, so the zone comes from state.
func vmScheduleFromDetail(job *VmJobDetail, zone string) (Schedule, error) {
	s := Schedule{FullBackupFrequency: strings.ToLower(job.FullBackupFreq), TimeZone: zone}
	if s.FullBackupFrequency != frequencyDaily && job.FullBackupDate != nil {
		s.FullBackupDay = *job.FullBackupDate
	}
	if job.StartTime == "" {
		return s, nil
	}
	start, err := parseAPITime(job.StartTime)
	if err != nil {
		return s, err
	}
	s.Time, err = utcToScheduleTime(start, zone)
	return s, err
}

func volumeBackupMode(frequency string) int {
	switch frequency {
	case frequencyWeekly:
		return volumeModeWeekly
	case frequencyMonthly:
		return volumeModeMonthly
	}
	return volumeModeDaily
}

func volumeFrequency(mode int) string {
	switch mode {
	case volumeModeWeekly:
		return frequencyWeekly
	case volumeModeMonthly:
		return frequencyMonthly
	}
	return frequencyDaily
}

// applyVolumeSchedule fills the schedule part of a v2 create or update body.
func applyVolumeSchedule(p *VolumeJobPayload, s Schedule) {
	p.BackupMode = volumeBackupMode(s.FullBackupFrequency)
	p.BackupTime = s.Time + ":00"
	p.BackupTimeZone = s.TimeZone
	p.FullBackupDate = nil
	if p.BackupMode != volumeModeDaily {
		day := s.FullBackupDay
		p.FullBackupDate = &day
	}
}

// volumeScheduleFromDetail rebuilds the schedule block from a v2 job. A daily
// job ignores the stored day: v2 keeps the previous day when a job changes to
// daily, and the daily mode does not use it (spec, volume job schedule mapping).
func volumeScheduleFromDetail(job *VolumeJobDetail) Schedule {
	s := Schedule{FullBackupFrequency: volumeFrequency(job.BackupMode), Time: job.BackupTime, TimeZone: job.BackupTimeZone}
	if len(s.Time) >= 5 {
		s.Time = s.Time[:5]
	}
	if s.TimeZone == "" {
		s.TimeZone = "UTC"
	}
	if job.BackupMode != volumeModeDaily && job.FullBackupDate != nil {
		s.FullBackupDay = *job.FullBackupDate
	}
	return s
}
