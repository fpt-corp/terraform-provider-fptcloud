package fptcloud_storage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "terraform-provider-fptcloud/commons"
)

// deleteStep is what one Get of the fake answers.
type deleteStep struct {
	storage *Storage
	err     error
}

// fakeDeleteService scripts Get and records the detach and delete calls.
type fakeDeleteService struct {
	StorageService
	gets      []deleteStep
	getCalls  int
	detachErr []error
	deleteErr []error
	calls     []string
}

func (f *fakeDeleteService) Get(_ string, _ string) (*Storage, error) {
	step := f.gets[len(f.gets)-1]
	if f.getCalls < len(f.gets) {
		step = f.gets[f.getCalls]
	}
	f.getCalls++
	f.calls = append(f.calls, "get")
	return step.storage, step.err
}

func (f *fakeDeleteService) UpdateAttachedInstance(_ string, _ string, instanceId *string) (*common.SimpleResponse, error) {
	if instanceId != nil {
		return nil, fmt.Errorf("unexpected attach to %s", *instanceId)
	}
	f.calls = append(f.calls, "detach")
	if len(f.detachErr) > 0 {
		err := f.detachErr[0]
		f.detachErr = f.detachErr[1:]
		return nil, err
	}
	return &common.SimpleResponse{}, nil
}

func (f *fakeDeleteService) DeleteStorageAsync(_ string, _ string) error {
	f.calls = append(f.calls, "delete")
	if len(f.deleteErr) > 0 {
		err := f.deleteErr[0]
		f.deleteErr = f.deleteErr[1:]
		return err
	}
	return nil
}

// fastQueuedDelete polls every millisecond and takes the first DISABLED as final.
func fastQueuedDelete(t *testing.T, redeleteAfter time.Duration) {
	t.Helper()
	interval, redelete, confirm := queuedDeletePollInterval, queuedRedeleteAfter, queuedDeleteConfirmFor
	queuedDeletePollInterval, queuedRedeleteAfter, queuedDeleteConfirmFor = time.Millisecond, redeleteAfter, 0
	t.Cleanup(func() {
		queuedDeletePollInterval, queuedRedeleteAfter, queuedDeleteConfirmFor = interval, redelete, confirm
	})
}

func enabled(instanceId string) deleteStep {
	return deleteStep{storage: &Storage{ID: "disk-1", Status: "ENABLED", InstanceId: instanceId}}
}

func disabled() deleteStep {
	return deleteStep{storage: &Storage{ID: "disk-1", Status: storageDisabled}}
}

func TestDeleteStorageQueued_DetachesThenQueuesTheDeleteAndWaitsForDisabled(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	service := &fakeDeleteService{gets: []deleteStep{enabled("vm-1"), enabled(""), enabled(""), disabled()}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

	require.NoError(t, err)
	assert.Equal(t, []string{"get", "detach", "get", "delete", "get", "get"}, service.calls)
}

func TestDeleteStorageQueued_DetachedDiskIsDeletedWithoutDetach(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	service := &fakeDeleteService{gets: []deleteStep{enabled(""), disabled()}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

	require.NoError(t, err)
	assert.Equal(t, []string{"get", "delete", "get"}, service.calls)
}

func TestDeleteStorageQueued_AlreadyGoneSendsNothing(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	notFound := common.HTTPError{Code: 500, Reason: `{"message":"{'error_code': '1501002', 'message': 'Storage not found'}"}`}

	for name, step := range map[string]deleteStep{
		"disabled":  disabled(),
		"not found": {err: notFound},
	} {
		t.Run(name, func(t *testing.T) {
			service := &fakeDeleteService{gets: []deleteStep{step}}

			err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

			require.NoError(t, err)
			assert.Equal(t, []string{"get"}, service.calls)
		})
	}
}

func TestDeleteStorageQueued_RefusedDeleteFailsAtOnce(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	refused := fmt.Errorf("%w (HTTP 400): Unable to delete storage with associated snapshots", ErrStorageDeleteRefused)
	service := &fakeDeleteService{gets: []deleteStep{enabled("")}, deleteErr: []error{refused}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrStorageDeleteRefused))
	assert.Contains(t, err.Error(), "associated snapshots")
	assert.Equal(t, []string{"get", "delete"}, service.calls)
}

func TestDeleteStorageQueued_RetriesPassingErrors(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	service := &fakeDeleteService{
		gets: []deleteStep{
			{err: errors.New("gateway timeout")},
			enabled("vm-1"), // the first detach fails
			enabled("vm-1"),
			enabled(""), // the first delete fails
			enabled(""),
			disabled(),
		},
		detachErr: []error{errors.New("System error, please try again !")},
		deleteErr: []error{errors.New("delete storage request failed (HTTP 502): bad gateway")},
	}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

	require.NoError(t, err)
	assert.Equal(t, []string{"get", "get", "detach", "get", "detach", "get", "delete", "get", "delete", "get"}, service.calls)
}

func TestDeleteStorageQueued_QueuesTheDeleteAgainWhenTheStorageStaysEnabled(t *testing.T) {
	fastQueuedDelete(t, 0)
	service := &fakeDeleteService{gets: []deleteStep{enabled(""), enabled(""), disabled()}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

	require.NoError(t, err)
	assert.Equal(t, []string{"get", "delete", "get", "delete", "get"}, service.calls)
}

func TestDeleteStorageQueued_DoesNotQueueTheDeleteAgainBeforeTheDelay(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	service := &fakeDeleteService{gets: []deleteStep{enabled(""), enabled(""), enabled(""), disabled()}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

	require.NoError(t, err)
	assert.Equal(t, []string{"get", "delete", "get", "get", "get"}, service.calls)
}

func TestDeleteStorageQueued_TimesOutWithTheLastStatusAndError(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	service := &fakeDeleteService{gets: []deleteStep{enabled("vm-1")}, detachErr: []error{
		errors.New("detach failed"), errors.New("detach failed"), errors.New("detach failed"),
	}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", 5*time.Millisecond)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "was not deleted within")
	assert.Contains(t, err.Error(), "attached to instance vm-1")
	assert.Contains(t, err.Error(), "delete queued 0 time(s)")
	assert.Contains(t, err.Error(), "detach failed")
	assert.Contains(t, err.Error(), "stays in the state")
}

func TestDeleteStorageQueued_TimesOutWhenTheStorageNeverGoes(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	service := &fakeDeleteService{gets: []deleteStep{enabled("")}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", 20*time.Millisecond)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "last status: ENABLED, delete queued 1 time(s)")
}

func TestDeleteStorageQueued_DisabledDuringASyncIsNotTakenAsDeleted(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	queuedDeleteConfirmFor = 20 * time.Millisecond
	// a sync of another storage shows this one DISABLED for a moment, then ENABLED again
	service := &fakeDeleteService{gets: []deleteStep{disabled(), enabled(""), disabled()}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

	require.NoError(t, err)
	assert.Equal(t, []string{"get", "get", "delete"}, service.calls[:3])
	assert.Greater(t, len(service.calls), 4, "DISABLED is read again until it has lasted queuedDeleteConfirmFor")
}

func TestDeleteStorageQueued_WaitsWhileAnotherOperationRuns(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	busy := deleteStep{storage: &Storage{ID: "disk-1", Status: "UPDATING"}}
	service := &fakeDeleteService{gets: []deleteStep{busy, busy, enabled(""), disabled()}}

	err := deleteStorageQueued(context.Background(), service, "vpc-1", "disk-1", time.Minute)

	require.NoError(t, err)
	assert.Equal(t, []string{"get", "get", "get", "delete", "get"}, service.calls)
}

func TestDeleteStorageQueued_StopsWhenTheContextIsCancelled(t *testing.T) {
	fastQueuedDelete(t, time.Hour)
	queuedDeletePollInterval = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &fakeDeleteService{gets: []deleteStep{enabled("")}}

	err := deleteStorageQueued(ctx, service, "vpc-1", "disk-1", 2*time.Hour)

	assert.ErrorIs(t, err, context.Canceled)
}
