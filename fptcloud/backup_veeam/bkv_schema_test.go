package fptcloud_backup_veeam

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// These tests call the pure validation function directly. Building a
// *schema.ResourceDiff through SDK internals breaks between versions, and all
// the logic worth testing lives in validateScheduleConfig anyway.

// start_hour > end_hour produces an all-zero bitmap, so the job NEVER RUNS and
// the API says nothing. It has to be rejected at plan time.
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

// day_of_month = 32 means the last day of the month, not a nonsense value.
func TestValidateAcceptsOnDayWithLastDayOfMonth(t *testing.T) {
	err := validateScheduleConfig(map[string]interface{}{
		"type":    "monthly",
		"monthly": []interface{}{map[string]interface{}{"day_number_in_month": "onDay", "day_of_month": 32}},
	})
	assert.Nil(t, err)
}

// The server strips the job name, so without normalising every plan proposes a change.
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
