package fptcloud_backup_veeam

import (
	"context"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBackupVeeamJobs() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the backup jobs of the Backup Veeam service in a VPC.",
		ReadContext: readBackupVeeamJobs,
		Schema:      dataSourceBackupVeeamJobsSchema,
	}
}

func readBackupVeeamJobs(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)
	vpcId := d.Get("vpc_id").(string)

	response, err := service.ListJobs(vpcId, d.Get("name").(string), d.Get("status").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	jobs := make([]interface{}, 0, len(response.Items))
	for _, item := range response.Items {
		jobs = append(jobs, map[string]interface{}{
			"id":               item.Id,
			"name":             item.Name,
			"description":      item.Description,
			"status":           item.Status,
			"enabled":          item.Enabled,
			"schedule_enabled": item.ScheduleEnabled,
			"next_run":         item.NextRun,
			"latest_run":       item.LatestRun,
		})
	}

	if err := d.Set("jobs", jobs); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(vpcId)
	return nil
}
