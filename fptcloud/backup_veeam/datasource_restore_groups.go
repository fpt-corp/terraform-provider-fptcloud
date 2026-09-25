package fptcloud_backup_veeam

import (
	"context"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBackupVeeamRestoreGroups() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the instances that have restore points in a VPC, one entry per instance and backup job - the " +
			"same table the portal's Restore tab shows. Use it to find the `vm_id` and `backup_job_id` that " +
			"`fptcloud_backup_veeam_restore_points` needs, without reading the backup jobs first.",
		ReadContext: readBackupVeeamRestoreGroups,
		Schema:      dataSourceBackupVeeamRestoreGroupsSchema,
	}
}

func readBackupVeeamRestoreGroups(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)

	vpcId := d.Get("vpc_id").(string)

	response, err := service.ListRestoreGroups(vpcId)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("groups", flattenRestoreGroups(response.Items)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(vpcId)
	return nil
}

// flattenRestoreGroups leaves out the item's id on purpose: the server fills it
// with an arbitrary restore point of the group, not the latest one, so exposing
// it would invite restoring the wrong point.
func flattenRestoreGroups(items []RestoreGroupItem) []interface{} {
	groups := make([]interface{}, 0, len(items))
	for _, item := range items {
		groups = append(groups, map[string]interface{}{
			"vm_id":               item.VmId,
			"vm_name":             item.VmName,
			"restore_vm_name":     item.RestoreVmName,
			"is_deleted":          item.IsDeleted,
			"backup_job_id":       item.JobId,
			"backup_job_name":     item.JobName,
			"restore_at":          item.RestoreAt,
			"restore_point_count": item.RestorePointCount,
			"total_backup_size":   item.TotalBackupSize,
		})
	}
	return groups
}
