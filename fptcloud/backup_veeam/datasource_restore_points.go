package fptcloud_backup_veeam

import (
	"context"
	"sort"
	"strings"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBackupVeeamRestorePoints() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the restore points of one protected instance - the same table the portal shows when you pick " +
			"\"Restore\" on an instance. Use it to choose the `restore_point_id` for a `fptcloud_backup_veeam_restore`.",
		ReadContext: readBackupVeeamRestorePoints,
		Schema:      dataSourceBackupVeeamRestorePointsSchema,
	}
}

func readBackupVeeamRestorePoints(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)

	vpcId := d.Get("vpc_id").(string)
	jobId := d.Get("backup_job_id").(string)
	vmId := d.Get("vm_id").(string)

	response, err := service.ListRestorePoints(vpcId, jobId, vmId)
	if err != nil {
		return diag.FromErr(err)
	}

	wantedType := strings.TrimSpace(d.Get("point_type").(string))
	items := make([]RestorePointItem, 0, len(response.Items))
	for _, item := range response.Items {
		if wantedType != "" && !strings.EqualFold(item.PointType, wantedType) {
			continue
		}
		items = append(items, item)
	}

	sortRestorePointsNewestFirst(items)

	points := make([]interface{}, 0, len(items))
	for _, item := range items {
		points = append(points, map[string]interface{}{
			"id":               item.Id,
			"restore_at":       item.RestoreAt,
			"point_type":       item.PointType,
			"backup_file_size": item.BackupFileSize,
			"status":           item.Status,
			"vm_display_name":  item.VmDisplayName,
		})
	}

	if err := d.Set("points", points); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(jobId + "/" + vmId)
	return nil
}

// sortRestorePointsNewestFirst puts the most recent restore point at index 0.
//
// The API returns them in an order it does not promise, and "restore the latest
// backup" is the common case; leaving the order to chance would make points[0]
// silently mean a different point from one plan to the next. restore_at is an
// ISO-8601 timestamp, so string order is chronological order.
func sortRestorePointsNewestFirst(items []RestorePointItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].RestoreAt > items[j].RestoreAt
	})
}
