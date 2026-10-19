package fptcloud_backup_native

import (
	"context"
	"fmt"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceVmBackup() *schema.Resource {
	return &schema.Resource{
		Description: "Runs a Backup Native VM backup job once, outside its schedule, the same operation as \"Start backup\" in the portal. " +
			"Applying the resource checks the job's status, starts the run and waits for it to end. " +
			"A job that is still running a backup, a restore or a change fails the apply at once, without starting a run. " +
			"Destroying the resource only removes it from state. Change a value in `triggers` to run the backup again. " +
			"A failed run fails the apply and leaves the resource tainted, so the next apply runs it again. " +
			"A job in state `Failed` refuses a new run until another run starts. " +
			"When the job stays in `Pending`, the apply runs out its timeout; only a later run or the backend clears that state.",
		CreateContext: createVmBackup,
		ReadContext:   readOneShot,
		DeleteContext: deleteOneShot,
		Schema:        manualBackupSchema("fptcloud_backup_native_vm_job"),
		// No UpdateContext: every argument is ForceNew, so any change runs the
		// backup again rather than editing a run that already happened.
		//
		// No Importer: there is nothing to adopt. A run started in the portal
		// left no object behind for Terraform to manage.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
	}
}

// manualBackupSchema is the schema of both manual backup resources.
// jobResource names the job resource whose id job_id takes.
func manualBackupSchema(jobResource string) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "ID of the VPC that owns the backup job.",
		},
		"job_id": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  fmt.Sprintf("ID of the backup job to run, the `id` of a `%s`.", jobResource),
		},
		"triggers": {
			Type:     schema.TypeMap,
			Optional: true,
			ForceNew: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Description: "Arbitrary values that run the backup again when they change. Use it to run the same job again, " +
				"which no other argument can express because every other argument stays equal.",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The job's status when the run ended, `Success` or `success` after a successful apply.",
		},
		"started_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "When the provider started the run, RFC 3339 UTC.",
		},
		"finished_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "When the provider saw the run end, RFC 3339 UTC.",
		},
	}
}

// manualBackupRun holds the three API steps of a manual backup. check marks
// a failed API read with readFailed, so that it is retried.
type manualBackupRun struct {
	check func() error
	start func() error
	wait  func(ctx context.Context, timeout time.Duration) (string, error)
}

// createManualBackup checks that the job can start a run now, starts it and
// waits for it to end. It does not wait for a busy job: only the run it
// started is waited on. The id is set once the run has started, so a run that
// fails or times out leaves the resource tainted and the next apply runs it
// again.
func createManualBackup(ctx context.Context, d *schema.ResourceData, run manualBackupRun) diag.Diagnostics {
	begin := time.Now()
	timeout := d.Timeout(schema.TimeoutCreate)
	if err := retryRead(ctx, run.check); err != nil {
		return diag.FromErr(err)
	}
	started := time.Now().UTC()
	if err := run.start(); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%s/%d", d.Get("job_id").(string), started.Unix()))
	if diags := setValues(d, map[string]interface{}{"started_at": started.Format(time.RFC3339)}); diags.HasError() {
		return diags
	}
	status, err := run.wait(ctx, timeout-time.Since(begin))
	if err != nil {
		return diag.FromErr(err)
	}
	return setValues(d, map[string]interface{}{
		"status":      status,
		"finished_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func createVmBackup(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVmService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	jobId := d.Get("job_id").(string)
	return createManualBackup(ctx, d, manualBackupRun{
		check: func() error {
			job, err := svc.GetVmJob(vpcId, jobId)
			if err != nil {
				return readFailed(err)
			}
			if job == nil {
				return fmt.Errorf("VM backup job %s no longer exists", jobId)
			}
			status := strings.ToLower(job.Status)
			if status == "failed" {
				return vmJobFailedError(jobId)
			}
			if !vmJobStableStatuses[status] {
				return jobBusyError("VM", jobId, job.Status)
			}
			return nil
		},
		start: func() error { return svc.StartVmJobBackup(vpcId, jobId) },
		wait: func(ctx context.Context, timeout time.Duration) (string, error) {
			return waitVmJobRun(ctx, svc, vpcId, jobId, timeout)
		},
	})
}
