package fptcloud_backup_veeam

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test gọi thẳng hàm validate thuần. Dựng *schema.ResourceDiff qua internals
// của SDK là cách test dễ vỡ theo version; toàn bộ logic cần kiểm nằm trong
// validateScheduleConfig nên test thẳng vào đó.

// start_hour > end_hour tạo bitmap toàn 0 -> job KHÔNG BAO GIỜ CHẠY, và API
// không báo lỗi. Phải chặn ở plan-time.
func TestValidateRejectsStartHourAfterEndHour(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type":   "period",
		"period": []interface{}{map[string]interface{}{"full_period": 4, "start_hour": 20, "end_hour": 6}},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "start_hour")
}

func TestValidateAcceptsValidPeriodWindow(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type":   "period",
		"period": []interface{}{map[string]interface{}{"full_period": 4, "start_hour": 20, "end_hour": 23}},
	})
	assert.Nil(t, err)
}

func TestValidateRejectsScheduleTypeMismatch(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type":    "daily",
		"monthly": []interface{}{map[string]interface{}{"day_number_in_month": "fourth", "day_of_week": "saturday"}},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "daily")
}

func TestValidateAcceptsMatchingDailyBlock(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type":  "daily",
		"daily": []interface{}{map[string]interface{}{"type": "weekDays", "run_at": "22:00:00"}},
	})
	assert.Nil(t, err)
}

func TestValidateRejectsOnDayWithoutDayOfMonth(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type":    "monthly",
		"monthly": []interface{}{map[string]interface{}{"day_number_in_month": "onDay"}},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "day_of_month")
}

func TestValidateRejectsDayOfMonthWhenNotOnDay(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type": "monthly",
		"monthly": []interface{}{map[string]interface{}{
			"day_number_in_month": "fourth", "day_of_week": "saturday", "day_of_month": 15,
		}},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "day_of_month")
}

func TestValidateRejectsWeekPositionWithoutDayOfWeek(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type":    "monthly",
		"monthly": []interface{}{map[string]interface{}{"day_number_in_month": "fourth"}},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "day_of_week")
}

// day_of_month = 32 nghĩa là ngày cuối tháng, không phải giá trị vô nghĩa.
func TestValidateAcceptsOnDayWithLastDayOfMonth(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type":    "monthly",
		"monthly": []interface{}{map[string]interface{}{"day_number_in_month": "onDay", "day_of_month": 32}},
	})
	assert.Nil(t, err)
}

// Server .strip() tên job -> nếu không chuẩn hoá thì mỗi plan đều đề nghị sửa.
func TestNameStateFuncTrimsWhitespace(t *testing.T) {
	stateFunc := resourceBackupVeeamJobSchema["name"].StateFunc
	assert.NotNil(t, stateFunc)
	assert.Equal(t, "job-db", stateFunc("  job-db  "))
}

func TestNameValidateFuncRejectsTooLong(t *testing.T) {
	validateFunc := resourceBackupVeeamJobSchema["name"].ValidateFunc
	_, errs := validateFunc(strings.Repeat("a", 51), "name")
	assert.NotEmpty(t, errs)
}

func TestNameValidateFuncRejectsInvalidCharacters(t *testing.T) {
	validateFunc := resourceBackupVeeamJobSchema["name"].ValidateFunc
	_, errs := validateFunc("job@db", "name")
	assert.NotEmpty(t, errs)
}

func TestNameValidateFuncAcceptsAllowedCharacters(t *testing.T) {
	validateFunc := resourceBackupVeeamJobSchema["name"].ValidateFunc
	_, errs := validateFunc("job-db_01 v1.2", "name")
	assert.Empty(t, errs)
}
