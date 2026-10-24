package fptcloud_database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullJob(id int64) *backupJobSpec {
	return &backupJobSpec{JobId: id, Name: "daily-full", Retention: 7, Hours: []int64{1}, Minute: 30, Days: []int64{1, 2, 3, 4, 5, 6, 7}}
}

func diffJob(id int64) *backupJobSpec {
	return &backupJobSpec{JobId: id, Name: "hourly-diff", Retention: 7, Hours: []int64{5, 9}, Minute: 0, Days: []int64{1, 2, 3, 4, 5}}
}

func kinds(steps []backupStep) []backupStepKind {
	if len(steps) == 0 {
		return nil
	}
	out := make([]backupStepKind, len(steps))
	for i, s := range steps {
		out[i] = s.Kind
	}
	return out
}

func TestPlanBackupChanges(t *testing.T) {
	changedFull := func(j *backupJobSpec) *backupJobSpec { c := *j; c.Hours = []int64{2}; return &c }
	longerFull := func(j *backupJobSpec) *backupJobSpec { c := *j; c.Retention = 14; return &c }
	changedDiff := func(j *backupJobSpec) *backupJobSpec { c := *j; c.Minute = 15; return &c }

	tests := []struct {
		name            string
		current, wanted backupConfig
		want            []backupStepKind
	}{
		{"nothing to do", backupConfig{10, fullJob(1), diffJob(2)}, backupConfig{10, fullJob(0), diffJob(0)}, nil},
		{"hour order does not matter", backupConfig{10, diffJob(2), nil}, backupConfig{10, diffJob(0), nil}, nil},
		{"remove both jobs: diff first", backupConfig{10, fullJob(1), diffJob(2)}, backupConfig{10, nil, nil}, []backupStepKind{stepDeleteDiff, stepDeleteFull}},
		{"remove the diff job only", backupConfig{10, fullJob(1), diffJob(2)}, backupConfig{10, fullJob(0), nil}, []backupStepKind{stepDeleteDiff}},
		{"change both schedules", backupConfig{10, fullJob(1), diffJob(2)}, backupConfig{10, changedFull(fullJob(0)), changedDiff(diffJob(0))}, []backupStepKind{stepUpdateFull, stepUpdateDiff}},
		{"longer retention updates the full job only", backupConfig{10, fullJob(1), diffJob(2)}, backupConfig{10, longerFull(fullJob(0)), diffJob(0)}, []backupStepKind{stepUpdateFull}},
		{"add both jobs: full then diff", backupConfig{10, nil, nil}, backupConfig{10, fullJob(0), diffJob(0)}, []backupStepKind{stepCreateFull, stepCreateDiff}},
		{"add the diff job", backupConfig{10, fullJob(1), nil}, backupConfig{10, fullJob(0), diffJob(0)}, []backupStepKind{stepCreateDiff}},
		{"capacity goes last", backupConfig{10, fullJob(1), nil}, backupConfig{20, changedFull(fullJob(0)), diffJob(0)}, []backupStepKind{stepUpdateFull, stepCreateDiff, stepCapacity}},
		{"deletes come before updates and creates", backupConfig{10, fullJob(1), diffJob(2)}, backupConfig{10, changedFull(fullJob(0)), nil}, []backupStepKind{stepDeleteDiff, stepUpdateFull}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := planBackupChanges(tt.current, tt.wanted)
			require.NoError(t, err)
			assert.Equal(t, tt.want, kinds(steps))
		})
	}
}

func TestPlanBackupChangesTargets(t *testing.T) {
	steps, err := planBackupChanges(backupConfig{10, fullJob(1), diffJob(2)},
		backupConfig{15, &backupJobSpec{Name: "daily-full", Retention: 14, Hours: []int64{1}, Minute: 30, Days: []int64{1}}, &backupJobSpec{Name: "hourly-diff", Hours: []int64{6}, Minute: 0, Days: []int64{1}}})
	require.NoError(t, err)
	require.Len(t, steps, 3)
	assert.Equal(t, int64(1), steps[0].Job.JobId, "the update targets the existing job")
	assert.Equal(t, int64(2), steps[1].Job.JobId)
	assert.Equal(t, int64(14), steps[1].Job.Retention, "the diff job takes the retention of the full job")
	assert.Equal(t, int64(15), steps[2].Capacity)
}

func TestPlanBackupChangesRejects(t *testing.T) {
	_, err := planBackupChanges(backupConfig{10, fullJob(1), nil}, backupConfig{10, nil, diffJob(0)})
	assert.ErrorContains(t, err, "diff_job needs full_job")

	renamed := fullJob(0)
	renamed.Name = "other-name"
	_, err = planBackupChanges(backupConfig{10, fullJob(1), nil}, backupConfig{10, renamed, nil})
	assert.ErrorContains(t, err, `set full_job.name to "daily-full"`)

	renamedDiff := diffJob(0)
	renamedDiff.Name = "other-name"
	_, err = planBackupChanges(backupConfig{10, fullJob(1), diffJob(2)}, backupConfig{10, fullJob(0), renamedDiff})
	assert.ErrorContains(t, err, `set diff_job.name to "hourly-diff"`)

	_, err = planBackupChanges(backupConfig{20, nil, nil}, backupConfig{10, nil, nil})
	assert.ErrorContains(t, err, "cannot be decreased")
}

func TestValidateBackupConfig(t *testing.T) {
	assert.Empty(t, validateBackupConfig(backupConfig{10, fullJob(0), diffJob(0)}))
	assert.Empty(t, validateBackupConfig(backupConfig{200, nil, nil}))

	bad := func(cfg backupConfig, want string) {
		t.Helper()
		problems := validateBackupConfig(cfg)
		require.NotEmpty(t, problems)
		assert.Contains(t, strings_join(problems), want)
	}
	bad(backupConfig{9, nil, nil}, "capacity_gb must be at least 10")
	bad(backupConfig{10, nil, diffJob(0)}, "diff_job needs full_job")

	f := fullJob(0)
	f.Name = "short"
	bad(backupConfig{10, f, nil}, "6 to 25 characters")
	f.Name = "this-name-is-way-too-long-for-it"
	bad(backupConfig{10, f, nil}, "6 to 25 characters")
	f.Name = "bad/name!"
	bad(backupConfig{10, f, nil}, "letters, digits")
	f = fullJob(0)
	f.Retention = 40
	bad(backupConfig{10, f, nil}, "retention must be 1 to 31")
	f = fullJob(0)
	f.Hours = []int64{24}
	bad(backupConfig{10, f, nil}, "between 0 and 23")
	f = fullJob(0)
	f.Days = []int64{0, 8}
	bad(backupConfig{10, f, nil}, "between 1 (Monday) and 7")
	f = fullJob(0)
	f.Hours = nil
	bad(backupConfig{10, f, nil}, "hours must not be empty")
	f = fullJob(0)
	f.Minute = 7
	bad(backupConfig{10, f, nil}, "minute must be one of")

	d := diffJob(0)
	d.Hours = []int64{1, 5}
	bad(backupConfig{10, fullJob(0), d}, "hour 1 is used by both")
	d = diffJob(0)
	d.Name = "daily-full"
	bad(backupConfig{10, fullJob(0), d}, "must be different")

	// A space, "_" and "-" are allowed in a name.
	f = fullJob(0)
	f.Name = "my daily_full-1"
	assert.Empty(t, validateBackupConfig(backupConfig{10, f, nil}))
}

func strings_join(s []string) string {
	out := ""
	for _, p := range s {
		out += p + "\n"
	}
	return out
}

func TestCheckBackupEngine(t *testing.T) {
	assert.NoError(t, checkBackupEngine("psql_patroni", true))
	assert.NoError(t, checkBackupEngine("mysql_innodb", true))
	assert.NoError(t, checkBackupEngine("redis_replication", false))
	assert.NoError(t, checkBackupEngine("mongodb_replicaset", false))

	for _, engine := range []string{"mongodb_replicaset", "redis_replication", "redis_sharding", "kafka", "clickhouse", "opensearch", "rabbitmq", "cassandra", "milvus"} {
		assert.ErrorContains(t, checkBackupEngine(engine, true), "diff_job is not available", engine)
	}
	for _, engine := range []string{"mongodb_enterprise", "mongodb_sharding"} {
		assert.ErrorContains(t, checkBackupEngine(engine, false), "not supported by Terraform yet", engine)
	}
}

func TestIntLists(t *testing.T) {
	assert.Equal(t, []int64{1, 5, 9}, parseIntList("9, 1,5"))
	assert.Equal(t, []int64{}, parseIntList(""))
	assert.Equal(t, "1,5,9", joinIntList([]int64{9, 1, 5}))
	assert.True(t, sameInts([]int64{1, 2}, []int64{2, 1}))
	assert.False(t, sameInts([]int64{1, 2}, []int64{1}))
}
