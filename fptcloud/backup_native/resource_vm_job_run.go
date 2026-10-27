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

func ResourceVmJobRun() *schema.Resource {
	return &schema.Resource{
		Description: "Runs a Backup Native VM backup job once, outside its schedule, the same operation as \"Start backup\" in the portal. " +
			"Applying the resource checks the job's status, starts the run and waits for it to end. " +
			"A job that is still running a backup, a restore or a change fails the apply at once, without starting a run. " +
			"Destroying the resource only removes it from state. Change a value in `triggers` to run the backup again. " +
			jobRunOutcomeDescription +
			"A job in state `Failed` refuses a new run until another run starts. " +
			"When the job stays in `Pending`, the apply runs out its timeout; only a later run or the backend clears that state.",
		CreateContext: createVmJobRun,
		ReadContext:   readVmJobRun,
		DeleteContext: deleteOneShot,
		Schema:        jobRunSchema("fptcloud_backup_native_vm_job"),
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

// jobRunOutcomeDescription is the part of both job run descriptions that says
// what a run that does not succeed leaves in state.
const jobRunOutcomeDescription = "Once the run has started, the resource is saved whatever the outcome, and the next apply does not run it again. " +
	"A failed run, a timeout or an interrupted apply (Ctrl+C) ends the apply with a warning, not an error, and stores the outcome in `status` and `message`. " +
	"The backup goes on without Terraform after a timeout or an interrupt; each later plan or apply reads the job's status " +
	"and stores the outcome once the run ends. That status may come from a later scheduled run of the same job. " +
	"Only interrupt an apply once: a second Ctrl+C stops Terraform before it saves the run, and the next apply starts another. "

// jobRunSchema is the schema of both job run resources.
// jobResource names the job resource whose id job_id takes.
func jobRunSchema(jobResource string) map[string]*schema.Schema {
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
		"triggers": triggersSchema("run the backup again"),
		"status": {
			Type:     schema.TypeString,
			Computed: true,
			Description: "The job's status when the run ended: `Success` or `success` after a successful run, " +
				"the API's failure status (`Failed` or `error` for a VM job, `error` for a volume job) after a failed one. " +
				runStatusesDescription,
		},
		"message": runMessageSchema(),
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

// jobRunSteps holds the three API steps of a job run. check marks
// a failed API read with readFailed, so that it is retried.
type jobRunSteps struct {
	check func() error
	start func() error
	wait  func(ctx context.Context, timeout time.Duration) (string, error)
}

// createJobRun checks that the job can start a run now, starts it and
// waits for it to end. It does not wait for a busy job: only the run it
// started is waited on. The id is set once the run has started, and from then
// on every outcome is saved, so the next apply does not run it again.
func createJobRun(ctx context.Context, d *schema.ResourceData, run jobRunSteps) diag.Diagnostics {
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
		return saveRunOutcome(ctx, d, err)
	}
	return setValues(d, map[string]interface{}{
		"status":      status,
		"message":     "",
		"finished_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// jobRunResult returns the values to store for a run the provider stopped
// watching, given the job's status now, or nil while the run goes on. found
// is false when the job is gone.
func jobRunResult(kind, jobId, status string, found bool, failed map[string]bool) map[string]interface{} {
	var err error
	switch s := strings.ToLower(status); {
	case !found:
		err = jobDeletedDuringRunError(kind, jobId)
	case failed[s]:
		err = jobRunFailedError(kind, jobId, status)
	case s != jobStatusSuccess:
		return nil
	}
	values := map[string]interface{}{
		"status":      status,
		"message":     "",
		"finished_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err != nil {
		values["status"], values["message"] = runOutcome(context.Background(), err)
	}
	return values
}

func readVmJobRun(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVmService(m.(*common.Client))
	jobId := d.Get("job_id").(string)
	return refreshRun(d, func() (map[string]interface{}, error) {
		job, err := svc.GetVmJob(d.Get("vpc_id").(string), jobId)
		if err != nil {
			return nil, err
		}
		if job == nil {
			return jobRunResult("VM", jobId, "", false, vmJobRunFailed), nil
		}
		return jobRunResult("VM", jobId, job.Status, true, vmJobRunFailed), nil
	})
}

func createVmJobRun(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	svc := NewVmService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	jobId := d.Get("job_id").(string)
	started := false
	defer func() {
		if started {
			syncVmJobs(svc, vpcId, &diags)
		}
	}()
	return createJobRun(ctx, d, jobRunSteps{
		check: func() error {
			job, err := svc.GetVmJob(vpcId, jobId)
			if err != nil {
				return readFailed(err)
			}
			if job == nil {
				return fmt.Errorf("VM backup job %s no longer exists", jobId)
			}
			status := strings.ToLower(job.Status)
			if status == jobStatusFailed {
				return vmJobFailedError(jobId)
			}
			if !jobStableStatuses[status] {
				return jobBusyError("VM", "start a backup", jobId, job.Status)
			}
			return nil
		},
		start: func() error {
			if err := svc.StartVmJobBackup(vpcId, jobId); err != nil {
				return err
			}
			started = true
			return nil
		},
		wait: func(ctx context.Context, timeout time.Duration) (string, error) {
			return waitVmJobRun(ctx, svc, vpcId, jobId, timeout)
		},
	})
}
