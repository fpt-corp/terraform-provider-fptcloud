package fptcloud_backup_native

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

// Every one-shot resource runs its action again only through triggers, so a
// changed value must replace the resource.
func TestOneShotResourcesHaveTriggers(t *testing.T) {
	resources := map[string]*schema.Resource{
		"vm_job_run":              ResourceVmJobRun(),
		"volume_job_run":          ResourceVolumeJobRun(),
		"vm_restore":              ResourceVmRestore(),
		"volume_restore":          ResourceVolumeRestore(),
		"vm_restore_points_purge": ResourceVmRestorePointsPurge(),
	}
	for name, r := range resources {
		triggers, ok := r.Schema["triggers"]
		if !assert.True(t, ok, name) {
			continue
		}
		assert.Equal(t, schema.TypeMap, triggers.Type, name)
		assert.True(t, triggers.Optional, name)
		assert.True(t, triggers.ForceNew, name)
		assert.Contains(t, triggers.Description, "YYYY-MM-DD HH:mm:ss", name)
		assert.NoError(t, r.InternalValidate(nil, true), name)
	}
}
