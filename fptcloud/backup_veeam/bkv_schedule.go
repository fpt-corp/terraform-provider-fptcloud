package fptcloud_backup_veeam

import (
	"strconv"
	"strings"
)

// Thứ tự Sunday-first, giống hệt convertTimeToPeriodSchedule của portal.
var periodDayOrder = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// allDays và allMonths luôn được gửi đầy đủ, y như portal. Người dùng không
// chọn hai field này - thứ điều khiển lịch daily là daily_schedule.type.
var allDays = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

var allMonths = []string{"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

// BuildPeriodBitmap dựng chuỗi 24 số 0/1 (một số cho mỗi giờ), bật cho các giờ
// trong khoảng [startHour, endHour], rồi lặp cùng chuỗi đó cho cả 7 ngày.
//
// Nếu startHour > endHour thì mọi giờ đều là 0 và job KHÔNG BAO GIỜ CHẠY.
// Trường hợp đó bị chặn ở plan-time bởi validateScheduleConfig; hàm này không
// tự sửa vì im lặng sửa cấu hình của khách còn tệ hơn là báo lỗi.
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

// ParsePeriodBitmap map ngược bitmap về startHour/endHour bằng cách lấy index
// đầu tiên và cuối cùng có giá trị 1.
//
// ok=false khi không có giờ nào được bật (bitmap rỗng hoặc toàn 0). Gọi bên
// ngoài KHÔNG được đoán giá trị trong trường hợp này - để nguyên cho Terraform
// hiện diff, vì đó là cấu hình hỏng cần khách sửa.
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
