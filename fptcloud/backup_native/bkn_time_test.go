package fptcloud_backup_native

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAPITimeFormats(t *testing.T) {
	naive, err := parseAPITime("2026-07-01T02:00:00")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 7, 1, 2, 0, 0, 0, time.UTC), naive)

	micro, err := parseAPITime("2026-07-01T02:00:00.123456")
	require.NoError(t, err)
	assert.Equal(t, 123456000, micro.Nanosecond())

	offset, err := parseAPITime("2026-07-01T09:00:00+07:00")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 7, 1, 2, 0, 0, 0, time.UTC), offset)

	_, err = parseAPITime("yesterday")
	assert.Error(t, err)
}

func TestFormatAPITime(t *testing.T) {
	assert.Equal(t, "2026-07-01T02:00:00Z", formatAPITime("2026-07-01T02:00:00"))
	assert.Equal(t, "2026-07-01T02:00:00Z", formatAPITime("2026-07-01T09:00:00+07:00"))
	assert.Equal(t, "", formatAPITime(""))
	assert.Equal(t, "45", formatAPITime("45"), "unparsable values pass through")
}

func TestScheduleTimeToUTCCrossesMidnight(t *testing.T) {
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	got, err := scheduleTimeToUTC("02:00", "Asia/Ho_Chi_Minh", now)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 12, 19, 0, 0, 0, time.UTC), got)
}

func TestScheduleTimeRoundTrip(t *testing.T) {
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	for _, zone := range []string{"UTC", "Asia/Ho_Chi_Minh", "Asia/Tokyo", "America/New_York"} {
		utc, err := scheduleTimeToUTC("23:30", zone, now)
		require.NoError(t, err)
		back, err := utcToScheduleTime(utc, zone)
		require.NoError(t, err)
		assert.Equal(t, "23:30", back, zone)
	}
}

func TestScheduleTimeRejectsBadInput(t *testing.T) {
	now := time.Now()
	_, err := scheduleTimeToUTC("02:00", "Mars/Olympus", now)
	assert.ErrorContains(t, err, "invalid time_zone")
	_, err = scheduleTimeToUTC("2am", "UTC", now)
	assert.ErrorContains(t, err, "expected HH:MM")
}
