package fptcloud_backup_native

import (
	"fmt"
	"time"

	// Embed the time zone database so schedule.time_zone works on systems
	// without one installed.
	_ "time/tzdata"
)

// apiTimeLayout is how the backend writes naive UTC datetimes. When parsing,
// Go also accepts a fractional second after the seconds field.
const apiTimeLayout = "2006-01-02T15:04:05"

// parseAPITime reads a backend timestamp: naive UTC (the usual form), or
// RFC 3339 with an offset (the v2 list's next_run).
func parseAPITime(value string) (time.Time, error) {
	if t, err := time.ParseInLocation(apiTimeLayout, value, time.UTC); err == nil {
		return t, nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("unrecognised timestamp %q", value)
	}
	return t.UTC(), nil
}

// formatAPITime renders a backend timestamp as RFC 3339 UTC. A value it cannot
// parse is returned unchanged, so an unexpected format shows up in state
// instead of disappearing.
func formatAPITime(value string) string {
	if value == "" {
		return ""
	}
	t, err := parseAPITime(value)
	if err != nil {
		return value
	}
	return t.Format(time.RFC3339)
}

// scheduleTimeToUTC places "HH:MM" in zone on today's date in that zone and
// returns the instant in UTC. The backend converts the same way when it
// creates a v1 job (InputCreateBackupModel.py:139-161).
func scheduleTimeToUTC(hhmm string, zone string, now time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time_zone %q: %w", zone, err)
	}
	clock, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q, expected HH:MM: %w", hhmm, err)
	}
	today := now.In(loc)
	local := time.Date(today.Year(), today.Month(), today.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
	return local.UTC(), nil
}

// utcToScheduleTime renders a UTC instant as "HH:MM" in zone.
func utcToScheduleTime(t time.Time, zone string) (string, error) {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return "", fmt.Errorf("invalid time_zone %q: %w", zone, err)
	}
	return t.In(loc).Format("15:04"), nil
}
