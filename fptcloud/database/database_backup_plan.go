package fptcloud_database

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// backupJobSpec is one backup job as configured or as found on the portal.
type backupJobSpec struct {
	JobId     int64 // 0 when the job does not exist yet
	Name      string
	Retention int64
	Hours     []int64
	Minute    int64
	Days      []int64
}

// backupConfig is the whole backup setup of a cluster.
type backupConfig struct {
	CapacityGb int64
	Full       *backupJobSpec
	Diff       *backupJobSpec
}

type backupStepKind string

const (
	stepDeleteDiff backupStepKind = "delete diff job"
	stepDeleteFull backupStepKind = "delete full job"
	stepUpdateFull backupStepKind = "update full job"
	stepUpdateDiff backupStepKind = "update diff job"
	stepCreateFull backupStepKind = "create full job"
	stepCreateDiff backupStepKind = "create diff job"
	stepCapacity   backupStepKind = "increase capacity"
)

// backupStep is one call to the portal. Job is the target job (with the id of an existing job);
// Capacity is set for stepCapacity.
type backupStep struct {
	Kind     backupStepKind
	Job      backupJobSpec
	Capacity int64
}

func sameInts(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]int64(nil), a...)
	y := append([]int64(nil), b...)
	sort.Slice(x, func(i, j int) bool { return x[i] < x[j] })
	sort.Slice(y, func(i, j int) bool { return y[i] < y[j] })
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func (a backupJobSpec) sameSchedule(b backupJobSpec) bool {
	return a.Minute == b.Minute && sameInts(a.Hours, b.Hours) && sameInts(a.Days, b.Days)
}

// planBackupChanges returns the calls that bring the current setup to the desired one, in the fixed order
// delete diff, delete full, update, create full, create diff, increase capacity.
// The retention of the diff job is set by the system from the full job, so it is not compared.
func planBackupChanges(current, desired backupConfig) ([]backupStep, error) {
	if desired.Diff != nil && desired.Full == nil {
		return nil, errors.New("diff_job needs full_job")
	}
	if current.Full != nil && desired.Full != nil && current.Full.Name != desired.Full.Name {
		return nil, fmt.Errorf("the name of the full job cannot be changed after creation; set full_job.name to %q", current.Full.Name)
	}
	if current.Diff != nil && desired.Diff != nil && current.Diff.Name != desired.Diff.Name {
		return nil, fmt.Errorf("the name of the diff job cannot be changed after creation; set diff_job.name to %q", current.Diff.Name)
	}
	if desired.CapacityGb < current.CapacityGb {
		return nil, fmt.Errorf("capacity_gb cannot be decreased (now %d GB); set it to %d or more", current.CapacityGb, current.CapacityGb)
	}

	var steps []backupStep

	if current.Diff != nil && desired.Diff == nil {
		steps = append(steps, backupStep{Kind: stepDeleteDiff, Job: *current.Diff})
	}
	if current.Full != nil && desired.Full == nil {
		steps = append(steps, backupStep{Kind: stepDeleteFull, Job: *current.Full})
	}

	if current.Full != nil && desired.Full != nil &&
		(!current.Full.sameSchedule(*desired.Full) || current.Full.Retention != desired.Full.Retention) {
		target := *desired.Full
		target.JobId = current.Full.JobId
		steps = append(steps, backupStep{Kind: stepUpdateFull, Job: target})
	}
	if current.Diff != nil && desired.Diff != nil && !current.Diff.sameSchedule(*desired.Diff) {
		target := *desired.Diff
		target.JobId = current.Diff.JobId
		target.Retention = desired.Full.Retention
		steps = append(steps, backupStep{Kind: stepUpdateDiff, Job: target})
	}

	if current.Full == nil && desired.Full != nil {
		steps = append(steps, backupStep{Kind: stepCreateFull, Job: *desired.Full})
	}
	if current.Diff == nil && desired.Diff != nil {
		target := *desired.Diff
		target.Retention = desired.Full.Retention
		steps = append(steps, backupStep{Kind: stepCreateDiff, Job: target})
	}

	if desired.CapacityGb > current.CapacityGb {
		steps = append(steps, backupStep{Kind: stepCapacity, Capacity: desired.CapacityGb})
	}
	return steps, nil
}

var (
	backupJobNameRe       = regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)
	backupAllowedMinutes  = []int64{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 59}
	backupAllowedRetainOk = func() map[int64]bool {
		m := map[int64]bool{90: true, 180: true, 365: true}
		for i := int64(1); i <= 31; i++ {
			m[i] = true
		}
		return m
	}()
)

const backupMinCapacityGb = 10

// engines that have a full backup job only, and engines Terraform cannot back up (they use other endpoints).
var (
	backupDiffEngines   = map[string]bool{"psql_patroni": true, "postgres": true, "mysql_innodb": true, "mariadb": true, "sqlserver": true, "timescaledb": true}
	backupBlockedEngine = map[string]bool{"mongodb_enterprise": true, "mongodb_sharding": true}
)

// checkBackupEngine rejects the engines that are not supported, or that cannot have a diff job.
func checkBackupEngine(typeDb string, withDiff bool) error {
	engine := strings.ToLower(strings.TrimSpace(typeDb))
	if backupBlockedEngine[engine] {
		return fmt.Errorf("backup of %s is not supported by Terraform yet: the portal uses other endpoints for it", typeDb)
	}
	if withDiff && !backupDiffEngines[engine] {
		return fmt.Errorf("diff_job is not available for %s; only full_job can be used", typeDb)
	}
	return nil
}

// validateBackupJob returns the problems of one job. label is "full_job" or "diff_job".
func validateBackupJob(label string, j backupJobSpec, checkRetention bool) []string {
	var problems []string
	if n := len(j.Name); n < 6 || n > 25 {
		problems = append(problems, fmt.Sprintf("%s.name must have 6 to 25 characters, got %d", label, n))
	}
	if !backupJobNameRe.MatchString(j.Name) {
		problems = append(problems, fmt.Sprintf("%s.name may contain only letters, digits, spaces, \"_\" and \"-\"", label))
	}
	if checkRetention && !backupAllowedRetainOk[j.Retention] {
		problems = append(problems, fmt.Sprintf("%s.retention must be 1 to 31, 90, 180 or 365, got %d", label, j.Retention))
	}
	if len(j.Hours) == 0 {
		problems = append(problems, label+".hours must not be empty")
	}
	for _, h := range j.Hours {
		if h < 0 || h > 23 {
			problems = append(problems, fmt.Sprintf("%s.hours must be between 0 and 23, got %d", label, h))
		}
	}
	if len(j.Days) == 0 {
		problems = append(problems, label+".days_of_week must not be empty")
	}
	for _, d := range j.Days {
		if d < 1 || d > 7 {
			problems = append(problems, fmt.Sprintf("%s.days_of_week must be between 1 (Monday) and 7 (Sunday), got %d", label, d))
		}
	}
	okMinute := false
	for _, m := range backupAllowedMinutes {
		okMinute = okMinute || m == j.Minute
	}
	if !okMinute {
		problems = append(problems, fmt.Sprintf("%s.minute must be one of 0, 5, 10, ..., 55, 59, got %d", label, j.Minute))
	}
	return problems
}

// validateBackupConfig checks everything that does not need the API.
func validateBackupConfig(cfg backupConfig) []string {
	var problems []string
	if cfg.CapacityGb < backupMinCapacityGb {
		problems = append(problems, fmt.Sprintf("capacity_gb must be at least %d, got %d", backupMinCapacityGb, cfg.CapacityGb))
	}
	if cfg.Diff != nil && cfg.Full == nil {
		problems = append(problems, "diff_job needs full_job")
	}
	if cfg.Full != nil {
		problems = append(problems, validateBackupJob("full_job", *cfg.Full, true)...)
	}
	if cfg.Diff != nil {
		problems = append(problems, validateBackupJob("diff_job", *cfg.Diff, false)...)
	}
	if cfg.Full != nil && cfg.Diff != nil {
		if cfg.Full.Name == cfg.Diff.Name {
			problems = append(problems, "full_job.name and diff_job.name must be different")
		}
		for _, h := range cfg.Full.Hours {
			for _, d := range cfg.Diff.Hours {
				if h == d {
					problems = append(problems, fmt.Sprintf("hour %d is used by both full_job and diff_job; the two jobs cannot run in the same hour", h))
				}
			}
		}
	}
	return problems
}
