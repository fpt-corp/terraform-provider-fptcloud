package fptcloud_backup_native

import (
	"context"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceVolumeBackup() *schema.Resource {
	return &schema.Resource{
		Description: "Runs a Backup Native volume backup job once, outside its schedule, the same operation as \"Start backup\" in the portal. " +
			"Applying the resource waits for a running backup of the job to finish, starts the run and waits for it to end; " +
			"destroying the resource only removes it from state. Change a value in `triggers` to run the backup again. " +
			"A failed run fails the apply and leaves the resource tainted, so the next apply runs it again. " +
			"A job with one volume may be retried by the backend after a failed run. " +
			"When the job stays in `pending`, the apply runs out its timeout; only a later run or the backend clears that state.",
		CreateContext: createVolumeBackup,
		ReadContext:   readOneShot,
		DeleteContext: deleteOneShot,
		Schema:        manualBackupSchema("fptcloud_backup_native_volume_job"),
		// No UpdateContext and no Importer, for the reasons given on
		// ResourceVmBackup.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
	}
}

func createVolumeBackup(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	jobId := d.Get("job_id").(string)
	return createManualBackup(ctx, d, manualBackupRun{
		settle: func(ctx context.Context, timeout time.Duration) error {
			return waitVolumeJobSettled(ctx, svc, vpcId, jobId, settleToRun, timeout)
		},
		start: func() error { return svc.StartVolumeJobBackup(vpcId, jobId) },
		wait: func(ctx context.Context, timeout time.Duration) (string, error) {
			return waitVolumeJobRun(ctx, svc, vpcId, jobId, timeout)
		},
	})
}
