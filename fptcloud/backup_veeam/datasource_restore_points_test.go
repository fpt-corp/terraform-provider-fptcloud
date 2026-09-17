package fptcloud_backup_veeam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataSourceRestorePointsShape(t *testing.T) {
	ds := DataSourceBackupVeeamRestorePoints()
	assert.NotNil(t, ds.ReadContext)
	assert.Nil(t, ds.CreateContext)
	assert.Nil(t, ds.InternalValidate(nil, false))
}

// points[0] has to be the newest restore point: "restore the latest backup" is
// the common case, and the API does not promise an order.
func TestRestorePointsAreSortedNewestFirst(t *testing.T) {
	items := []RestorePointItem{
		{Id: "old", RestoreAt: "2026-09-12T03:01:55"},
		{Id: "new", RestoreAt: "2026-09-14T03:02:10"},
		{Id: "mid", RestoreAt: "2026-09-13T03:02:09"},
	}

	sortRestorePointsNewestFirst(items)

	assert.Equal(t, "new", items[0].Id)
	assert.Equal(t, "mid", items[1].Id)
	assert.Equal(t, "old", items[2].Id)
}
