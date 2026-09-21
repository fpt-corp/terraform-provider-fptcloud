package fptcloud_backup_veeam

import (
	"context"
	"fmt"
	"strings"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBackupVeeamJob() *schema.Resource {
	return &schema.Resource{
		Description: "Reads one Backup Veeam job, including its schedule, retention, protected instances " +
			"and notification methods. The `fptcloud_backup_veeam_jobs` data source returns none of those: " +
			"it lists jobs with their status only.",
		ReadContext: readBackupVeeamJobDataSource,
		Schema:      dataSourceBackupVeeamJobSchema,
	}
}

func readBackupVeeamJobDataSource(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)
	vpcId := d.Get("vpc_id").(string)

	jobId := strings.TrimSpace(d.Get("job_id").(string))
	if jobId == "" {
		resolved, err := resolveJobIdByName(service, vpcId, strings.TrimSpace(d.Get("name").(string)))
		if err != nil {
			return diag.FromErr(err)
		}
		jobId = resolved
	}

	detail, err := service.GetJobDetail(vpcId, jobId)
	if err != nil {
		return diag.FromErr(err)
	}
	if detail == nil {
		return diag.Errorf("backup job %s was not found in VPC %s", jobId, vpcId)
	}

	// The same flatten the resource uses, so the two can never describe the
	// same job differently - including the daily type canonicalisation.
	if err := flattenJobDetail(d, detail); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(detail.Id)
	return nil
}

// resolveJobIdByName turns a job name into its id through the list endpoint,
// which is the only endpoint that can search by name - detail takes an id.
//
// The name filter is a partial match on the server side, so the result is
// narrowed again here: without that, "db" would match "db-daily" and the data
// source would silently return whichever came first.
func resolveJobIdByName(service BackupVeeamService, vpcId string, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("either job_id or name must be set")
	}

	list, err := service.ListJobs(vpcId, name, "")
	if err != nil {
		return "", err
	}

	matches := make([]JobListItem, 0, 1)
	for _, item := range list.Items {
		if item.Name == name {
			matches = append(matches, item)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0].Id, nil
	case 0:
		return "", fmt.Errorf("no backup job named %q exists in VPC %s", name, vpcId)
	default:
		ids := make([]string, 0, len(matches))
		for _, item := range matches {
			ids = append(ids, item.Id)
		}
		// Job names are unique per VPC, so this should be unreachable. Report it
		// rather than picking one, because picking one silently would make the
		// caller act on a job they did not choose.
		return "", fmt.Errorf("%d backup jobs are named %q in VPC %s (%s); use job_id instead",
			len(matches), name, vpcId, strings.Join(ids, ", "))
	}
}
