package fptcloud_snapshot

// Snapshot lifecycle states. The platform reports the infrastructure's own
// status uppercased, so a settled snapshot is ACTIVE on OpenStack (Glance) and
// AVAILABLE on VMware.
const (
	StatusCreating      = "CREATING"
	StatusQueued        = "QUEUED"
	StatusSaving        = "SAVING"
	StatusImporting     = "IMPORTING"
	StatusBackingUp     = "BACKING-UP"
	StatusRestoring     = "RESTORING"
	StatusActive        = "ACTIVE"
	StatusAvailable     = "AVAILABLE"
	StatusDeleting      = "DELETING"
	StatusPendingDelete = "PENDING_DELETE"
	StatusUnmanaging    = "UNMANAGING"
	StatusDeleted       = "DELETED"
	StatusError         = "ERROR"
	StatusErrorDeleting = "ERROR_DELETING"
	StatusKilled        = "KILLED"
	StatusDeactivated   = "DEACTIVATED"
)

// SettledStatuses are the states in which a snapshot is finished and usable.
var SettledStatuses = []string{StatusActive, StatusAvailable}

// CreatingStatuses are the states on the way to settled. Anything outside these
// and SettledStatuses aborts the wait instead of being polled to the timeout.
var CreatingStatuses = []string{
	StatusCreating,
	StatusQueued,
	StatusSaving,
	StatusImporting,
	StatusBackingUp,
	StatusRestoring,
}

// DeletingStatuses are the states a snapshot passes through on its way out.
var DeletingStatuses = []string{
	StatusActive,
	StatusAvailable,
	StatusDeleting,
	StatusPendingDelete,
	StatusUnmanaging,
	// VMware marks the row deleted rather than removing it.
	StatusDeleted,
}

var failedStatuses = map[string]struct{}{
	StatusError:         {},
	StatusErrorDeleting: {},
	StatusKilled:        {},
	StatusDeactivated:   {},
}

// IsFailed reports whether a status means the platform gave up on the snapshot.
func IsFailed(status string) bool {
	_, failed := failedStatuses[status]
	return failed
}

// MaxNameLength mirrors the platform's snapshot name validator.
const MaxNameLength = 30

// CreateSnapshotDTO is the create request body. SnapshotName is required on
// OpenStack and ignored on VMware; IncludeRam is the mirror image.
type CreateSnapshotDTO struct {
	InstanceId   string   `json:"instance_id"`
	SnapshotName string   `json:"snapshot_name,omitempty"`
	IncludeRam   bool     `json:"include_ram,omitempty"`
	TagIds       []string `json:"tag_ids,omitempty"`
}

// VolumeSnapshot is one underlying volume snapshot of an instance snapshot.
type VolumeSnapshot struct {
	SnapshotId        string `json:"snapshot_id"`
	VolumeId          string `json:"volume_id"`
	IsRoot            bool   `json:"is_root"`
	VolumeSize        int    `json:"volume_size"`
	StoragePolicyName string `json:"storage_policy_name"`
	Status            string `json:"status"`
}

// Snapshot is the snapshot resource as the API reports it. InfraSnapshotId is
// the Glance image id on OpenStack, empty until the snapshot settles; VMware
// records neither a name nor an infrastructure id.
type Snapshot struct {
	Id              string           `json:"id"`
	VpcId           string           `json:"vpc_id"`
	InstanceId      string           `json:"vm_id"`
	InstanceName    string           `json:"vm_name"`
	Name            string           `json:"name"`
	Status          string           `json:"status"`
	SnapshotType    string           `json:"snapshot_type"`
	SizeGb          float64          `json:"size_gb"`
	InfraSnapshotId string           `json:"infra_snapshot_id"`
	CreatedBy       string           `json:"created_by"`
	CreatedAt       string           `json:"created_at"`
	Volume          []VolumeSnapshot `json:"volume"`
	Tags            []Tag            `json:"tags"`
}

// Tag is one tag attached to a snapshot.
type Tag struct {
	Id    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// snapshotResponse is the envelope of the create and read endpoints.
type snapshotResponse struct {
	Status  bool     `json:"status"`
	Message string   `json:"message"`
	Data    Snapshot `json:"data"`
}

// apiError is the envelope the API returns for a refusal.
type apiError struct {
	Status    bool   `json:"status"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}
