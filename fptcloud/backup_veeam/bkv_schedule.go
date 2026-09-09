package fptcloud_backup_veeam

import (
	"strconv"
	"strings"
)

// Sunday-first order, exactly like the portal's convertTimeToPeriodSchedule.
var periodDayOrder = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// allDays and allMonths are always sent in full, just as the portal does. The
// user never picks these - what drives a daily schedule is
// daily_schedule.type.
var allDays = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

var allMonths = []string{"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

// BuildPeriodBitmap builds a string of 24 zeroes and ones, one per hour, set
// for the hours inside [startHour, endHour], then repeats that same string for
// all seven days.
//
// If startHour > endHour every hour is 0 and the job NEVER RUNS. That case is
// rejected at plan time by validateScheduleConfig; this function does not
// silently repair it, because quietly rewriting the user's configuration is
// worse than refusing it.
func BuildPeriodBitmap(startHour int, endHour int) []PeriodScheduleEntry {
	hours := make([]string, 24)
	for i := 0; i < 24; i++ {
		if i >= startHour && i <= endHour {
			hours[i] = "1"
		} else {
			hours[i] = "0"
		}
	}
	value := strings.Join(hours, ",")

	entries := make([]PeriodScheduleEntry, 0, len(periodDayOrder))
	for _, day := range periodDayOrder {
		entries = append(entries, PeriodScheduleEntry{Name: day, Value: value})
	}
	return entries
}

// ParsePeriodBitmap maps a bitmap back to startHour/endHour by taking the
// first and last index set to 1.
//
// ok=false when no hour is enabled (an empty or all-zero bitmap). Callers must
// NOT guess a value in that case - leave it alone so Terraform shows the diff,
// because that is a broken configuration the user needs to fix.
func ParsePeriodBitmap(entries []PeriodScheduleEntry) (int, int, bool) {
	if len(entries) == 0 {
		return 0, 0, false
	}

	parts := strings.Split(entries[0].Value, ",")
	start := -1
	end := -1
	for i, part := range parts {
		flag, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || flag == 0 {
			continue
		}
		if start == -1 {
			start = i
		}
		end = i
	}

	if start == -1 {
		return 0, 0, false
	}
	return start, end, true
}
