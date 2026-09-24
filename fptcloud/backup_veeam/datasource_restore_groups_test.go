package fptcloud_backup_veeam

import (
	"testing"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceRestoreGroupsShape(t *testing.T) {
	ds := DataSourceBackupVeeamRestoreGroups()
	assert.NotNil(t, ds.ReadContext)
	assert.Nil(t, ds.CreateContext)
	assert.Nil(t, ds.InternalValidate(nil, false))
}

// The body mirrors what the backend's restores/groups endpoint returns,
// including the "vcd" field the provider ignores and a legacy row whose vm_id
// is null.
func TestReadRestoreGroupsMapsTheServerRows(t *testing.T) {
	client, server, err := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/backup/restores": `{
			"items": [
				{
					"id": "some-restore-point",
					"vm_id": "vm-1",
					"vm_name": "db-renamed",
					"restore_vm_name": "db",
					"job_id": "job-1",
					"job_name": "job-db-daily",
					"restore_at": "2026-09-14T03:02:10",
					"restore_point_count": 3,
					"total_backup_size": 12.5,
					"is_deleted": false,
					"vmw_id": "vm-101"
				},
				{
					"id": "legacy-point",
					"vm_id": null,
					"vm_name": "old-vm",
					"restore_vm_name": "old-vm",
					"job_id": "job-2",
					"job_name": "job-old",
					"restore_at": "2026-08-01T01:00:00",
					"restore_point_count": 1,
					"total_backup_size": 1,
					"is_deleted": true
				}
			],
			"count": 2,
			"vcd": "vcd-01"
		}`,
	})
	assert.Nil(t, err)
	defer server.Close()

	d := schema.TestResourceDataRaw(t, dataSourceBackupVeeamRestoreGroupsSchema, map[string]interface{}{
		"vpc_id": "vpc-1",
	})
	diags := readBackupVeeamRestoreGroups(nil, d, client)
	assert.False(t, diags.HasError(), "%v", diags)

	assert.Equal(t, "vpc-1", d.Id())
	assert.Equal(t, 2, d.Get("groups.#"))
	assert.Equal(t, "vm-1", d.Get("groups.0.vm_id"))
	assert.Equal(t, "db-renamed", d.Get("groups.0.vm_name"))
	assert.Equal(t, "db", d.Get("groups.0.restore_vm_name"))
	assert.Equal(t, "job-1", d.Get("groups.0.backup_job_id"))
	assert.Equal(t, "job-db-daily", d.Get("groups.0.backup_job_name"))
	assert.Equal(t, "2026-09-14T03:02:10", d.Get("groups.0.restore_at"))
	assert.Equal(t, 3, d.Get("groups.0.restore_point_count"))
	assert.Equal(t, 12.5, d.Get("groups.0.total_backup_size"))
	assert.Equal(t, false, d.Get("groups.0.is_deleted"))

	assert.Equal(t, "", d.Get("groups.1.vm_id"))
	assert.Equal(t, true, d.Get("groups.1.is_deleted"))
}

// The server's id is an arbitrary restore point of the group, not the latest,
// so it must not surface where it could be mistaken for one.
func TestRestoreGroupsDoNotExposeTheServerId(t *testing.T) {
	groups := flattenRestoreGroups([]RestoreGroupItem{{Id: "some-restore-point", VmId: "vm-1"}})
	assert.NotContains(t, groups[0].(map[string]interface{}), "id")
}
