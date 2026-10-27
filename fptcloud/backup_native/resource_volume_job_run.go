package fptcloud_backup_native

import (
	"context"
	"fmt"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceVolumeJobRun() *schema.Resource {
	return &schema.Resource{
		Description: "Runs a Backup Native volume backup job once, outside its schedule, the same operation as \"Start backup\" in the portal. " +
			"Applying the resource checks the job's status, starts the run and waits for it to end. " +
			"A job that is still running a backup, a restore or a change fails the apply at once, without starting a run. " +
			"Destroying the resource only removes it from state. Change a value in `triggers` to run the backup again. " +
			"A failed run fails the apply and leaves the resource tainted, so the next apply runs it again. " +
			"A job with one volume may be retried by the backend after a failed run. " +
			"When the job stays in `pending`, the apply runs out its timeout; only a later run or the backend clears that state.",
		CreateContext: createVolumeJobRun,
		ReadContext:   readOneShot,
		DeleteContext: deleteOneShot,
		Schema:        jobRunSchema("fptcloud_backup_native_volume_job"),
		// No UpdateContext and no Importer, for the reasons given on
		// ResourceVmJobRun.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
	}
}

func createVolumeJobRun(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	jobId := d.Get("job_id").(string)
	return createJobRun(ctx, d, jobRunSteps{
		check: func() error {
			job, err := svc.GetVolumeJob(vpcId, jobId)
			if err != nil {
				return readFailed(err)
			}
			if job == nil {
				return fmt.Errorf("volume backup job %s no longer exists", jobId)
			}
			if !jobStableStatuses[strings.ToLower(job.Status)] {
				return jobBusyError("volume", "start a backup", jobId, job.Status)
			}
			return nil
		},
		start: func() error { return svc.StartVolumeJobBackup(vpcId, jobId) },
		wait: func(ctx context.Context, timeout time.Duration) (string, error) {
			return waitVolumeJobRun(ctx, svc, vpcId, jobId, timeout)
		},
	})
}
