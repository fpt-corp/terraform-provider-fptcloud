package fptcloud_backup_native

// Job statuses of the v1 and v2 APIs. The APIs differ in case ("Failed",
// "success"), so a status is lowercased before it is compared with these.
const (
	jobStatusInitiated = "initiated"
	jobStatusAvailable = "available"
	jobStatusPending   = "pending"
	jobStatusUpdating  = "updating"
	jobStatusSuccess   = "success"
	jobStatusError     = "error"
	// jobStatusFailed is v1 only: a VM failed in the job's last run.
	jobStatusFailed = "failed"

	// The statuses a job reads when its delete failed.
	vmJobStatusDeleteFailed     = "error_deleting"
	volumeJobStatusDeleteFailed = "deleting error"
)

// jobStableStatuses are the statuses in which a job accepts an update, a
// schedule switch and a manual backup: v1 update, switch_schedule and
// start_backup (SwitchScheduleController.py:50), v2 update, actions and
// executions. Anything else, including a percentage while a backup runs,
// means busy, except a v1 "failed": see settlePurpose.
var jobStableStatuses = map[string]bool{
	jobStatusInitiated: true,
	jobStatusAvailable: true,
	jobStatusSuccess:   true,
	jobStatusError:     true,
}

// States of a v1 Celery task, uppercased before they are compared.
const (
	taskStateSuccess = "SUCCESS"
	taskStateFailure = "FAILURE"
	taskStateRevoked = "REVOKED"
)

// Statuses of a history row, lowercased before they are compared.
const (
	historyStatusSuccess = "success"
	historyStatusFailed  = "failed"
)

// History types of the actions the provider waits on. History rows carry no
// id, so the type is part of how a row is matched to an action.
const (
	historyTypeRestore = "Restore"
	// historyTypeRestoreKeep is a restore into a new VM (v1) or a new volume
	// (v2); both APIs write the same type.
	historyTypeRestoreKeep = "Restore keep"
	// historyTypePurge is the type of a delete-all request
	// (HistoryActionType.BACKUP_RESTORE_POINTS_DELETION).
	historyTypePurge = "Delete all backup restore point"
)

// historySourceVolume is the source_type of a volume's rows in the v2
// combined history.
const historySourceVolume = "volume"
