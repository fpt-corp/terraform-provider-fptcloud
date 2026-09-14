package fptcloud_backup_veeam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The SDK cancels the context at the CRUD timeout, so StateChangeConf must be
// given something strictly smaller - otherwise the two race and the user can
// end up with "context deadline exceeded" instead of a message naming the job.
func TestPollTimeoutStaysBelowCrudTimeout(t *testing.T) {
	for _, crud := range []time.Duration{30 * time.Minute, 45 * time.Minute, 2 * time.Minute} {
		got := pollTimeout(crud)
		assert.Less(t, got, crud, "crud timeout %s", crud)
		assert.Greater(t, got, time.Duration(0), "crud timeout %s", crud)
	}
}

func TestPollTimeoutTrimsFixedMargin(t *testing.T) {
	assert.Equal(t, 29*time.Minute+30*time.Second, pollTimeout(30*time.Minute))
}

// A timeout too small to trim is halved, so the margin can never produce a
// zero or negative budget - StateChangeConf treats 0 as "no timeout" and would
// poll until the context died.
func TestPollTimeoutHalvesShortTimeouts(t *testing.T) {
	assert.Equal(t, 15*time.Second, pollTimeout(30*time.Second))
	assert.Equal(t, 30*time.Second, pollTimeout(time.Minute))
}

type stubDetailService struct {
	BackupVeeamService
	calls   int
	answers [][]string
}

func (s *stubDetailService) GetJobDetail(_ string, jobId string) (*JobDetail, error) {
	idx := s.calls
	if idx >= len(s.answers) {
		idx = len(s.answers) - 1
	}
	s.calls++

	objects := make([]BackupObject, 0, len(s.answers[idx]))
	for _, id := range s.answers[idx] {
		objects = append(objects, BackupObject{VmId: id})
	}
	return &JobDetail{Id: jobId, BackupObject: objects}, nil
}

// The API reports the old instance list for a while after an update settles, so
// the wait has to keep polling until it matches rather than accept the first
// answer.
func TestWaitForVmIdsSettledPollsUntilTheListMatches(t *testing.T) {
	svc := &stubDetailService{answers: [][]string{
		{"vm-1", "vm-2"}, // stale: the removed instance is still listed
		{"vm-1", "vm-2"}, // still stale
		{"vm-1"},         // settled
	}}

	err := WaitForVmIdsSettled(context.Background(), svc, "vpc-1", "job-1", []string{"vm-1"}, 2*time.Minute)
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, svc.calls, 3, "must have polled past the stale answers")
}

func TestWaitForVmIdsSettledAcceptsAMatchImmediately(t *testing.T) {
	svc := &stubDetailService{answers: [][]string{{"vm-2", "vm-1"}}}

	// Order is irrelevant: vm_ids is a set.
	err := WaitForVmIdsSettled(context.Background(), svc, "vpc-1", "job-1", []string{"vm-1", "vm-2"}, time.Minute)
	assert.Nil(t, err)
}

func TestSameVmIdsComparesAsASet(t *testing.T) {
	objects := []BackupObject{{VmId: "a"}, {VmId: "b"}}
	assert.True(t, sameVmIds(objects, map[string]bool{"b": true, "a": true}))
	assert.False(t, sameVmIds(objects, map[string]bool{"a": true}))
	assert.False(t, sameVmIds(objects, map[string]bool{"a": true, "b": true, "c": true}))
	assert.False(t, sameVmIds(objects, map[string]bool{"a": true, "c": true}))
}
