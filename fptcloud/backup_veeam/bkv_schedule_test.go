package fptcloud_backup_veeam_test

import (
	"strings"
	"testing"

	bkv "terraform-provider-fptcloud/fptcloud/backup_veeam"

	"github.com/stretchr/testify/assert"
)

func TestBuildPeriodBitmapFullDay(t *testing.T) {
	entries := bkv.BuildPeriodBitmap(0, 23)
	assert.Len(t, entries, 7)
	// Sunday first - exactly like the portal.
	assert.Equal(t, "Sunday", entries[0].Name)
	assert.Equal(t, "Saturday", entries[6].Name)
	assert.Equal(t, strings.TrimRight(strings.Repeat("1,", 24), ","), entries[0].Value)
}

func TestBuildPeriodBitmapWindow(t *testing.T) {
	entries := bkv.BuildPeriodBitmap(20, 23)
	parts := strings.Split(entries[0].Value, ",")
	assert.Len(t, parts, 24)
	assert.Equal(t, "0", parts[0])
	assert.Equal(t, "0", parts[19])
	assert.Equal(t, "1", parts[20])
	assert.Equal(t, "1", parts[23])
	// The same string for all seven days.
	for _, e := range entries {
		assert.Equal(t, entries[0].Value, e.Value)
	}
}

func TestParsePeriodBitmapRoundTrip(t *testing.T) {
	start, end, ok := bkv.ParsePeriodBitmap(bkv.BuildPeriodBitmap(8, 17))
	assert.True(t, ok)
	assert.Equal(t, 8, start)
	assert.Equal(t, 17, end)
}

// An all-zero bitmap means the job never runs - do not guess, report ok=false
// so the resource shows a diff instead of writing a wrong value into state.
func TestParsePeriodBitmapAllZeroIsNotOk(t *testing.T) {
	entries := []bkv.PeriodScheduleEntry{{Name: "Sunday", Value: strings.TrimRight(strings.Repeat("0,", 24), ",")}}
	_, _, ok := bkv.ParsePeriodBitmap(entries)
	assert.False(t, ok)
}

// start_hour > end_hour yields an all-zero bitmap - exactly the silent bug
// CustomizeDiff has to reject at plan time.
func TestBuildPeriodBitmapWrappingWindowIsAllZero(t *testing.T) {
	entries := bkv.BuildPeriodBitmap(20, 6)
	_, _, ok := bkv.ParsePeriodBitmap(entries)
	assert.False(t, ok)
}

// Non-contiguous bitmap: take the first and last index set to 1.
func TestParsePeriodBitmapNonContiguousTakesFirstAndLast(t *testing.T) {
	value := "1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1"
	start, end, ok := bkv.ParsePeriodBitmap([]bkv.PeriodScheduleEntry{{Name: "Sunday", Value: value}})
	assert.True(t, ok)
	assert.Equal(t, 0, start)
	assert.Equal(t, 23, end)
}

func TestParsePeriodBitmapEmptyIsNotOk(t *testing.T) {
	_, _, ok := bkv.ParsePeriodBitmap(nil)
	assert.False(t, ok)
}

func TestStatusClassification(t *testing.T) {
	for _, s := range []string{"CREATING", "UPDATING", "DELETING", "STARTING", "DISABLING_SCHEDULE", "ENABLING_SCHEDULE"} {
		assert.True(t, bkv.IsPendingStatus(s), s)
		assert.False(t, bkv.IsFailedStatus(s), s)
	}
	for _, s := range []string{"ERROR", "CREATE_FAILED", "FAILED"} {
		assert.True(t, bkv.IsFailedStatus(s), s)
		assert.False(t, bkv.IsPendingStatus(s), s)
	}
	// NOT_AVAILABLE is the NORMAL state of a job just created that has not run yet.
	for _, s := range []string{"NOT_AVAILABLE", "WORKING", "SUCCESS", "WARNING"} {
		assert.False(t, bkv.IsPendingStatus(s), s)
		assert.False(t, bkv.IsFailedStatus(s), s)
	}
}
