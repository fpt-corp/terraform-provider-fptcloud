package fptcloud_backup_native

import "encoding/json"

// envelope is the {message, data, status} body both APIs use. Fields the
// backend leaves null are dropped by some routes and kept by others, so every
// field here tolerates absence.
type envelope struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Status  *bool           `json:"status"`
}

// ---- v1 OSP native backup ----

type VmJobCreatePayload struct {
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Notification        []string               `json:"notification"`
	AppliedInstances    []string               `json:"applied_instances"`
	NumberLatestBackup  int                    `json:"number_latest_backup"`
	BackupMode          string                 `json:"backup_mode"`
	QuickBackupInput    *VmQuickBackupInput    `json:"quick_backup_input,omitempty"`
	AdvancedBackupInput *VmAdvancedBackupInput `json:"advanced_backup_input,omitempty"`
	TimeZone            string                 `json:"time_zone"`
	TenantUserId        string                 `json:"tenant_user_id"`
}

type VmQuickBackupInput struct {
	QuickBackupFreq VmQuickBackupFreq `json:"quick_backup_freq"`
	QuickBackupTime string            `json:"quick_backup_time"`
}

type VmQuickBackupFreq struct {
	Full        string `json:"full"`
	Incremental string `json:"incremental"`
}

type VmAdvancedBackupInput struct {
	IncrementalBackupFreq string `json:"incremental_backup_freq"`
	IncrementalBackupTime string `json:"incremental_backup_time"`
	EnableFullBackup      bool   `json:"enable_full_backup"`
	FullyBackupFreq       string `json:"fully_backup_freq"`
	FullyBackupDate       *int   `json:"fully_backup_date"`
}

// VmJobUpdatePayload is a full replacement: the v1 worker overwrites every
// optional field it is not sent with NULL (UpdateBackupJobController.py:232-240),
// and a null recipients or instances list fails with HTTP 500.
type VmJobUpdatePayload struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Interval              int      `json:"interval"`
	NumberLatestBackup    int      `json:"number_latest_backup"`
	IncrementalBackupFreq string   `json:"incremental_backup_freq"`
	FullBackupFreq        string   `json:"full_backup_freq"`
	FullBackupDate        *int     `json:"full_backup_date"`
	StartTime             string   `json:"start_time"`
	Recipients            []string `json:"recipients"`
	Instances             []string `json:"instances"`
}

type VmJobDetail struct {
	Id                    string           `json:"id"`
	Name                  string           `json:"name"`
	Description           string           `json:"description"`
	IncrementalBackupFreq string           `json:"incremental_backup_freq"`
	NumberLatestBackup    int              `json:"number_latest_backup"`
	FullBackupFreq        string           `json:"full_backup_freq"`
	FullBackupDate        *int             `json:"full_backup_date"`
	Status                string           `json:"status"`
	StartTime             string           `json:"start_time"`
	NextRun               string           `json:"next_run"`
	CreatedAt             string           `json:"created_at"`
	IsEnabled             bool             `json:"is_enabled"`
	Instances             []VmJobInstance  `json:"instances"`
	Recipients            []VmJobRecipient `json:"recipients"`
}

type VmJobInstance struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	IsDeleted   bool   `json:"is_deleted"`
}

type VmJobRecipient struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type VmJobListItem struct {
	Id        string          `json:"id"`
	Name      string          `json:"name"`
	Status    string          `json:"status"`
	NextRun   string          `json:"next_run"`
	IsEnabled bool            `json:"is_enabled"`
	Instances []VmJobInstance `json:"instances"`
}

// TaskProgress is the v1 Celery task status. It is not wrapped in an envelope.
type TaskProgress struct {
	State      string `json:"state"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

type VmRestorePoint struct {
	VmId          string  `json:"vm_id"`
	Size          float64 `json:"size"`
	CreatedAt     string  `json:"created_at"`
	IsIncremental bool    `json:"is_incremental"`
	IsAvailable   bool    `json:"is_available"`
	BackupJobId   string  `json:"backup_job_id"`
	BackupJobName string  `json:"backup_job_name"`
}

type VmHistory struct {
	VmId      string `json:"vm_id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type VmRestorePayload struct {
	VmId        string  `json:"vm_id"`
	CreatedAt   string  `json:"created_at"`
	IsOverwrite bool    `json:"is_overwrite"`
	NewPassword *string `json:"new_password"`
	PowerOn     bool    `json:"power_on"`
}

// VmRestoreKeepPayload is the restore-new body. security_groups and
// floating_ip must never be null: the backend fails with HTTP 500 on null.
type VmRestoreKeepPayload struct {
	VmId           string            `json:"vm_id"`
	CreatedAt      string            `json:"created_at"`
	IsOverwrite    bool              `json:"is_overwrite"`
	InstanceName   *string           `json:"instance_name"`
	NewPassword    *string           `json:"new_password"`
	PowerOn        bool              `json:"power_on"`
	HwFlavor       RestoreFlavor     `json:"hw_flavor"`
	Subnet         IdName            `json:"subnet"`
	SecurityGroups []IdName          `json:"security_groups"`
	FloatingIp     RestoreFloatingIp `json:"floating_ip"`
	PrivateIp      *string           `json:"private_ip"`
}

type RestoreFlavor struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Cpu  int    `json:"cpu"`
	Ram  int    `json:"ram"`
}

type IdName struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type RestoreFloatingIp struct {
	Id string `json:"id"`
	Ip string `json:"ip"`
}

type FloatingIpItem struct {
	Id        string `json:"id"`
	IpAddress string `json:"ip_address"`
}

// ---- v2 Backup Volume ----

// VolumeJobPayload is the create and update body. Update must always carry
// backup_objects and backup_time_zone (spec, volume job lifecycle).
type VolumeJobPayload struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	Notifications  []string `json:"notifications"`
	BackupObjects  []string `json:"backup_objects"`
	Retention      int      `json:"retention"`
	BackupMode     int      `json:"backup_mode"`
	FullBackupDate *int     `json:"full_backup_date,omitempty"`
	BackupTime     string   `json:"backup_time"`
	BackupTimeZone string   `json:"backup_time_zone"`
}

type VolumeFailure struct {
	VolumeId string `json:"volume_id"`
	Name     string `json:"name"`
	Error    string `json:"error"`
}

// flexString accepts a JSON string or number. The v2 list sends job_type as
// a string ("0", "1"), coerced from an int constant (constant.py:232-234).
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexString(n.String())
	return nil
}

type VolumeJobListItem struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	CreatedAt string            `json:"created_at"`
	IsEnabled bool              `json:"is_enabled"`
	NextRun   string            `json:"next_run"`
	JobType   flexString        `json:"job_type"`
	Instances []VolumeJobTarget `json:"instances"`
}

type VolumeJobTarget struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	IsDeleted   bool   `json:"is_deleted"`
}

type VolumeJobDetail struct {
	Id             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Notifications  []IdName          `json:"notifications"`
	Retention      int               `json:"retention"`
	FullBackupDate *int              `json:"full_backup_date"`
	BackupMode     int               `json:"backup_mode"`
	BackupTime     string            `json:"backup_time"`
	Status         string            `json:"status"`
	CreatedAt      string            `json:"created_at"`
	IsEnabled      bool              `json:"is_enabled"`
	BackupTimeZone string            `json:"backup_time_zone"`
	Targets        []VolumeJobTarget `json:"targets"`
}

type VolumeRestorePoint struct {
	Id            string  `json:"id"`
	CreatedAt     string  `json:"created_at"`
	Status        string  `json:"status"`
	IsAvailable   bool    `json:"is_available"`
	IsIncremental bool    `json:"is_incremental"`
	Size          float64 `json:"size"`
}

type VolumeRestorePayload struct {
	BackupId string                `json:"backup_id"`
	Metadata VolumeRestoreMetadata `json:"metadata"`
}

type VolumeRestoreMetadata struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

// VolumeHistoryPage is the combined-history body. It has no envelope.
type VolumeHistoryPage struct {
	Items   []VolumeHistoryItem `json:"items"`
	HasMore bool                `json:"has_more"`
}

type VolumeHistoryItem struct {
	SourceType      string `json:"source_type"`
	ObjectId        string `json:"object_id"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	HistoryMetadata struct {
		ExtractInfo map[string]interface{} `json:"extract_info"`
	} `json:"history_metadata"`
}
