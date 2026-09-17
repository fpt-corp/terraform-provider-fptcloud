package fptcloud_backup_veeam

import (
	"context"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBackupVeeamInstantRecoverySessions() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the Instant Recovery sessions open in a VPC - the same table the portal's Instant Recovery tab " +
			"shows. Use it to find sessions nobody is tracking: an open session stops the backup job of the instance it was " +
			"mounted from until it is migrated or stopped.",
		ReadContext: readBackupVeeamInstantRecoverySessions,
		Schema:      dataSourceBackupVeeamInstantRecoverySessionsSchema,
	}
}

func readBackupVeeamInstantRecoverySessions(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)

	vpcId := d.Get("vpc_id").(string)

	response, err := service.ListMounts(vpcId)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("sessions", flattenMounts(response.Data)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(vpcId)
	return nil
}

func flattenMounts(items []MountItem) []interface{} {
	sessions := make([]interface{}, 0, len(items))
	for _, item := range items {
		sessions = append(sessions, map[string]interface{}{
			"vm_mount_id":        item.VmMountId,
			"vm_mount_name":      item.VmMountName,
			"recovered_vm_name":  item.RecoveredVmName,
			"vm_id":              item.VmId,
			"state":              item.State,
			"mode":               item.Mode,
			"restore_point_time": item.RestorePointTime,
			"backup_id":          item.BackupId,
			"ready_migrate":      item.ReadyMigrate,
		})
	}
	return sessions
}
