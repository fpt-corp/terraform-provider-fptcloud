package fptcloud_backup_veeam

// JobMutationResponse is the shape create, update and delete all return.
//
// WARNING: the API answers HTTP 200 EVEN ON FAILURE. A failure shows up as
// Status=false or a non-empty ErrorType. The business-error branch (such as
// duplicateVm) does NOT set "status" at all, so Go's false zero value happens
// to handle it correctly.
type JobMutationResponse struct {
	Status       bool   `json:"status"`
	Message      string `json:"message"`
	ErrorType    string `json:"error_type"`
	ResourceId   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	// BackupJobId in the create response carries the PORTAL-side UUID, not the
	// Veeam id. It shares its name with the backup_job_id column (the Veeam id)
	// but means something different.
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

// CreateJobPayload is used for BOTH create and update: the API reuses
// InputCreateJobModel and is not a PATCH, so always send the full payload.
type CreateJobPayload struct {
	Name                  string           `json:"name"`
	Description           string           `json:"description"`
	Enabled               bool             `json:"enabled"`
	ScheduleEnabled       bool             `json:"schedule_enabled"`
	Retention             RetentionPayload `json:"retaintion"` // misspelled ON THE API SIDE, kept as is
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

// JobDetail has NO status and NO enabled - both only exist on the list endpoint.
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
