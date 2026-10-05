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

// --- Restore ---------------------------------------------------------------

// RestoreGroupItem is one row of the portal's Restore tab: an instance, the job
// that protects it, and how many restore points it has.
type RestoreGroupItem struct {
	Id                string  `json:"id"`
	VmId              string  `json:"vm_id"`
	VmName            string  `json:"vm_name"`
	RestoreVmName     string  `json:"restore_vm_name"`
	JobId             string  `json:"job_id"`
	JobName           string  `json:"job_name"`
	RestoreAt         string  `json:"restore_at"`
	RestorePointCount int     `json:"restore_point_count"`
	TotalBackupSize   float64 `json:"total_backup_size"`
	IsDeleted         bool    `json:"is_deleted"`
}

type RestoreGroupListResponse struct {
	Items []RestoreGroupItem `json:"items"`
	Count int                `json:"count"`
}

// RestorePointItem is one selectable row in the portal's "Restore Point" table:
// date, size and type.
type RestorePointItem struct {
	Id             string  `json:"id"`
	VpcId          string  `json:"vpc_id"`
	BackupId       string  `json:"backup_id"`
	BackupJobId    string  `json:"backup_job_id"`
	BackupObjectId string  `json:"backup_object_id"`
	VmId           string  `json:"vm_id"`
	VmName         string  `json:"vm_name"`
	VmDisplayName  string  `json:"vm_display_name"`
	PointType      string  `json:"point_type"`
	Algorithm      string  `json:"algorithm"`
	RestoreAt      string  `json:"restore_at"`
	Status         string  `json:"status"`
	BackupFileSize float64 `json:"backup_file_size"`
}

type RestorePointListResponse struct {
	Items []RestorePointItem `json:"items"`
	Count int                `json:"count"`
}

// RestorePayload is the body of POST .../restores/{id}/restore.
//
// keep_original_vm is deliberately absent. The model on the API side declares
// it, but the handler never passes it on - restore_with_pending() takes only
// restore_vm_point_id, power_on_after_restore and quick_rollback - so sending
// it would be accepted and silently dropped. Keeping the original instance is
// what the clone endpoint is for, which is out of scope here.
type RestorePayload struct {
	RestoreVmPointId    string `json:"restore_vm_point_id"`
	PowerOnAfterRestore bool   `json:"power_on_after_restore"`
	QuickRollback       bool   `json:"quick_rollback"`
}

// RestoreResponse is what the restore endpoint answers. It carries no status
// field: the outcome has to be polled from the restore point itself.
type RestoreResponse struct {
	Status       bool   `json:"status"`
	Message      string `json:"message"`
	ErrorType    string `json:"error_type"`
	ResourceId   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
}

// --- Restore keep (clone) --------------------------------------------------

// RestoreClonePayload is the body of POST .../restores/{id}/restore/clone -
// the operation the portal calls "Restore keep": the original instance stays
// untouched and the restore point is brought back as a NEW instance.
//
// KeepOriginalVm is always sent as true and is deliberately not exposed in the
// schema. The boundary turns it into two VBR flags at once - KeepOriginalVM
// and the inverse of QuickRollback - so sending false here does not mean
// "clone without keeping": it makes this endpoint overwrite the original
// instance. A field whose false value turns a safe operation into a
// destructive one does not belong in a provider schema.
type RestoreClonePayload struct {
	RestoreVmPointId    string `json:"restore_vm_point_id"`
	NewVmName           string `json:"new_vm_name"`
	PowerOnAfterRestore bool   `json:"power_on_after_restore"`
	KeepOriginalVm      bool   `json:"keep_original_vm"`
}

// --- Instant recovery ------------------------------------------------------

// InstantRecoveryDestination carries the name of the instance an instant
// recovery session mounts as. Only restored_vm_name is modelled: the backend
// drops destination_host and datastore when it hands the request to Celery, so
// the other fields would be accepted and silently ignored.
type InstantRecoveryDestination struct {
	RestoredVmName string `json:"restored_vm_name"`
}

// InstantRecoveryTypeCustomized is the only session type the provider starts:
// the backup is mounted as a new instance. The API also knows
// "OriginalLocation", but the portal hides the option that would send it, so
// no customer session has ever used it - see ResourceBackupVeeamInstantRecovery.
const InstantRecoveryTypeCustomized = "Customized"

// InstantRecoveryPayload is the body of the instant recovery endpoint.
//
// Type is not exposed in the schema: it is always Customized, and letting it
// be configured would allow the combination "Customized with no destination",
// which makes the backend raise.
type InstantRecoveryPayload struct {
	RestorePointId       string                      `json:"restore_point_id"`
	Type                 string                      `json:"type"`
	PowerUp              bool                        `json:"power_up"`
	NicsEnabled          bool                        `json:"nics_enabled"`
	VmTagsRestoreEnabled bool                        `json:"vm_tags_restore_enabled"`
	Destination          *InstantRecoveryDestination `json:"destination,omitempty"`
}

// InstantRecoveryResponse is what starting a session answers.
//
// HistoryId is the only handle the API hands back, and there is no endpoint
// that reads a history row by id - so it is kept as an attribute for matching
// against the portal's History tab, and is NOT usable for polling. The session
// itself has to be found in the mount list.
type InstantRecoveryResponse struct {
	Status       bool   `json:"status"`
	Message      string `json:"message"`
	ErrorType    string `json:"error_type"`
	HistoryId    string `json:"history_id"`
	ResourceId   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
}

// MountItem is one active instant recovery session, as the portal's Instant
// Recovery tab lists it.
//
// State comes straight from VBR and the portal has no enum for it, so no
// meaning is attached to any particular value here - see WaitForMountPresent.
// ReadyMigrate is NOT a property of the mount: the backend sets it from the
// mounted instance's own power state in the portal database.
type MountItem struct {
	VmId             string `json:"vm_id"`
	RecoveredVmName  string `json:"recovered_vm_name"`
	VmMountId        string `json:"vm_mount_id"`
	VmMountName      string `json:"vm_mount_name"`
	State            string `json:"state"`
	RestorePointTime string `json:"restore_point_time"`
	BackupId         string `json:"backup_id"`
	Mode             string `json:"mode"`
	ReadyMigrate     bool   `json:"ready_migrate"`
}

// MountListResponse is VPC-wide: it merges the sessions of every VBR the
// tenant has, and it only contains mounts whose restore point still has a row
// in the portal database.
type MountListResponse struct {
	Data []MountItem `json:"data"`
}

// without it the endpoint answers HTTP 400, the only non-200 error in the
// whole backup family. The provider reads the value out of the mount list so
// the user never has to supply it.
