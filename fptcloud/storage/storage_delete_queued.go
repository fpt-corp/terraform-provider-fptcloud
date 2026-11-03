package fptcloud_storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// queuedDeleteTimeout is how long a delete waits for the storage to go
const queuedDeleteTimeout = 15 * time.Minute

// queuedDeletePollInterval, queuedRedeleteAfter and queuedDeleteConfirmFor are vars only so tests can run faster.
var (
	queuedDeletePollInterval = 5 * time.Second
	// The queued delete waits up to 2 minutes for the volume to go, then syncs
	// the storages; a storage still ENABLED well after that means the task
	// failed without a sync, so the delete is queued again. A volume already
	// gone is only synced by the new task.
	queuedRedeleteAfter = 3 * time.Minute
	// A storage sync marks every storage of the VPC DISABLED, then sets the ones
	// it finds back to ENABLED: a storage that still exists reads DISABLED while
	// another resource syncs. DISABLED only counts once it has lasted this long.
	queuedDeleteConfirmFor = 30 * time.Second
)

const (
	storageEnabled  = "ENABLED"
	storageDisabled = "DISABLED"
)

// deleteStorageQueued deletes a storage through the console's queued delete, so
// that disks deleted together do not wait for each other inside one HTTP
// request. The storage is detached first, the queued delete does not do it
// and a volume in use cannot be deleted. It returns once the storage is gone:
// not found, or DISABLED (no longer in the infrastructure) for queuedDeleteConfirmFor.
func deleteStorageQueued(ctx context.Context, service StorageService, vpcId string, storageId string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var deleteQueuedAt, disabledSince time.Time
	deletesQueued := 0
	var lastErr error
	lastStatus := "unknown"

	for {
		storage, err := service.Get(vpcId, storageId)
		if err == nil && storage.Status != storageDisabled {
			disabledSince = time.Time{}
		}
		switch {
		case err != nil && isStorageNotFound(err):
			return nil
		case err != nil:
			lastErr = fmt.Errorf("reading the storage: %w", err)
			log.Printf("[WARN] Reading storage %s while deleting it failed, retrying: %s", storageId, err)
		case storage.Status == storageDisabled:
			lastStatus = storageDisabled
			if disabledSince.IsZero() {
				disabledSince = time.Now()
			}
			if time.Since(disabledSince) >= queuedDeleteConfirmFor {
				return nil
			}
		case storage.InstanceId != "":
			lastStatus = fmt.Sprintf("%s, attached to instance %s", storage.Status, storage.InstanceId)
			if _, err := service.UpdateAttachedInstance(vpcId, storageId, nil); err != nil {
				lastErr = fmt.Errorf("detaching the storage from instance %s: %w", storage.InstanceId, err)
				log.Printf("[WARN] Detaching storage %s before deleting it failed, retrying: %s", storageId, err)
			}
			deleteQueuedAt = time.Time{}
		case storage.Status != storageEnabled:
			// another operation is running on the storage, the delete would be refused or lost
			lastStatus = storage.Status
		case deleteQueuedAt.IsZero() || time.Since(deleteQueuedAt) >= queuedRedeleteAfter:
			lastStatus = storage.Status
			if err := service.DeleteStorageAsync(vpcId, storageId); err != nil {
				if errors.Is(err, ErrStorageDeleteRefused) {
					return err
				}
				lastErr = err
				log.Printf("[WARN] Queueing the delete of storage %s failed, retrying: %s", storageId, err)
			} else {
				deleteQueuedAt = time.Now()
				deletesQueued++
				lastErr = nil
			}
		default:
			lastStatus = storage.Status
		}

		if time.Now().Add(queuedDeletePollInterval).After(deadline) {
			message := fmt.Sprintf("storage %s was not deleted within %s (last status: %s, delete queued %d time(s))",
				storageId, timeout, lastStatus, deletesQueued)
			if lastErr != nil {
				message += fmt.Sprintf(", last error: %s", lastErr)
			}
			return errors.New(message + "; the storage stays in the state, check it in the console and apply again")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(queuedDeletePollInterval):
		}
	}
}
