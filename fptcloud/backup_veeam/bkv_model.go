package fptcloud_backup_veeam

// JobMutationResponse là dạng trả về chung của create / update / delete.
//
// CẢNH BÁO: API trả HTTP 200 KỂ CẢ KHI THẤT BẠI. Thất bại được báo bằng
// Status=false hoặc ErrorType khác rỗng. Nhánh lỗi nghiệp vụ (vd duplicateVm)
// KHÔNG set "status", nên zero value false của Go xử lý đúng luôn.
type JobMutationResponse struct {
	Status       bool   `json:"status"`
	Message      string `json:"message"`
	ErrorType    string `json:"error_type"`
	ResourceId   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	// BackupJobId ở response create mang UUID PHÍA PORTAL, không phải id Veeam.
	// Trùng tên với column backup_job_id (id Veeam) nhưng khác nghĩa.
	BackupJobId string `json:"backup_job_id"`
}

type RetentionPayload struct {
	Cycles    int    `json:"cycles"`
	LimitType string `json:"limit_type"`
}

type DailySchedulePayload struct {
	Enabled bool     `json:"enabled"`
	Type    string   `json:"type"`
	RunAt   string   `json:"run_at"`
	Days    []string `json:"days"`
}

type MonthlySchedulePayload struct {
	Enabled          bool     `json:"enabled"`
	RunAt            string   `json:"run_at"`
	DayNumberInMonth string   `json:"day_number_in_month"`
	DayOfWeek        string   `json:"day_of_week,omitempty"`
	DayOfMonth       int      `json:"day_of_month,omitempty"`
	Months           []string `json:"months"`
}

type PeriodScheduleEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PeriodSchedulePayload struct {
	Enabled    bool                  `json:"enabled"`
	Type       string                `json:"type"`
	FullPeriod int                   `json:"full_period"`
	Schedules  []PeriodScheduleEntry `json:"schedules"`
}

type SchedulePayload struct {
	ScheduleType    string                  `json:"schedule_type"`
	DailySchedule   *DailySchedulePayload   `json:"daily_schedule,omitempty"`
	MonthlySchedule *MonthlySchedulePayload `json:"monthly_schedule,omitempty"`
	PeriodSchedule  *PeriodSchedulePayload  `json:"period_schedule,omitempty"`
}

// CreateJobPayload dùng cho CẢ create lẫn update (API dùng chung
// InputCreateJobModel, không phải PATCH - luôn gửi full payload).
type CreateJobPayload struct {
	Name                  string           `json:"name"`
	Description           string           `json:"description"`
	Enabled               bool             `json:"enabled"`
	ScheduleEnabled       bool             `json:"schedule_enabled"`
	Retention             RetentionPayload `json:"retaintion"` // sai chính tả PHÍA API, giữ nguyên
	VmIds                 []string         `json:"vm_ids"`
	Schedule              *SchedulePayload `json:"schedule,omitempty"`
	NotificationMethodIds []string         `json:"notification_method_ids"`
	IsCapacityTierEnabled bool             `json:"is_capacity_tier_enabled"`
	IdempotencyKey        string           `json:"idempotency_key,omitempty"`
}

type BackupObject struct {
	VmId          string `json:"vm_id"`
	VmDisplayName string `json:"vm_display_name"`
}

// JobDetail KHÔNG có status và enabled - hai field đó chỉ có ở endpoint list.
type JobDetail struct {
	Id                    string           `json:"id"`
	Name                  string           `json:"name"`
	VpcId                 string           `json:"vpc_id"`
	Description           string           `json:"description"`
	ScheduleEnabled       bool             `json:"schedule_enabled"`
	BackupObject          []BackupObject   `json:"backup_object"`
	BackupSchedule        *SchedulePayload `json:"backup_schedule"`
	BackupRetention       RetentionPayload `json:"backup_retention"`
	NotificationMethodIds []string         `json:"notification_method_ids"`
	IsCapacityTierEnabled bool             `json:"is_capacity_tier_enabled"`
}

type JobDetailResponse struct {
	Data JobDetail `json:"data"`
}

type JobListItem struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	Enabled         bool   `json:"enabled"`
	ScheduleEnabled bool   `json:"schedule_enabled"`
	NextRun         string `json:"next_run"`
	LatestRun       string `json:"latest_run"`
}

type JobListResponse struct {
	Items      []JobListItem `json:"items"`
	Count      int           `json:"count"`
	TotalCount int           `json:"total_count"`
}

type InstanceItem struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type InstanceListResponse struct {
	Data  []InstanceItem `json:"data"`
	Total int            `json:"total"`
}
