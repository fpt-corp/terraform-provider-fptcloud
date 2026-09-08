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
	// Thứ tự Sunday trước - giống hệt portal.
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
	// Cùng một chuỗi cho cả 7 ngày.
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

// Bitmap toàn 0 nghĩa là job không bao giờ chạy - không map bừa, báo ok=false
// để resource để lộ diff thay vì ghi giá trị sai vào state.
func TestParsePeriodBitmapAllZeroIsNotOk(t *testing.T) {
	entries := []bkv.PeriodScheduleEntry{{Name: "Sunday", Value: strings.TrimRight(strings.Repeat("0,", 24), ",")}}
	_, _, ok := bkv.ParsePeriodBitmap(entries)
	assert.False(t, ok)
}

// start_hour > end_hour cho bitmap toàn 0 - chính là bug im lặng mà
// CustomizeDiff phải chặn ở plan-time.
func TestBuildPeriodBitmapWrappingWindowIsAllZero(t *testing.T) {
	entries := bkv.BuildPeriodBitmap(20, 6)
	_, _, ok := bkv.ParsePeriodBitmap(entries)
	assert.False(t, ok)
}

// Bitmap không liên tục: lấy index đầu và cuối có giá trị 1.
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
	// NOT_AVAILABLE là trạng thái BÌNH THƯỜNG của job vừa tạo, chưa chạy lần nào.
	for _, s := range []string{"NOT_AVAILABLE", "WORKING", "SUCCESS", "WARNING"} {
		assert.False(t, bkv.IsPendingStatus(s), s)
		assert.False(t, bkv.IsFailedStatus(s), s)
	}
}
