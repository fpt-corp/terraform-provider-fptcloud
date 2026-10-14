package fptcloud_backup_native

import (
	"context"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"
	fptcloud_vpc "terraform-provider-fptcloud/fptcloud/vpc"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceVmJob() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a VM backup job of the Backup Native service. Destroying the resource deletes the job through the user API. " +
			"The API currently also deletes the restore points of the job's instances when the job is deleted.",
		CreateContext: createVmJob,
		ReadContext:   readVmJob,
		UpdateContext: updateVmJob,
		DeleteContext: deleteVmJob,
		CustomizeDiff: customizeSchedule,
		Schema:        resourceVmJobSchema(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},
		Importer: &schema.ResourceImporter{StateContext: importJob("backup_native_vm_job")},
	}
}

func resourceVmJobSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "ID of the VPC.",
		},
		"name":        jobNameSchema("VM"),
		"description": jobDescriptionSchema(),
		"instance_ids": {
			Type:        schema.TypeSet,
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "IDs of the instances to back up. An instance can belong to only one backup job.",
		},
		"notification_method_ids": {
			Type:        schema.TypeSet,
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "IDs of EMAIL alert notification methods that receive the job's notifications. See the `fptcloud_alert_notification_methods` data source.",
		},
		"retention": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntBetween(1, 1000),
			Description:  "Number of restore points kept per instance, 1-1000.",
		},
		"schedule": scheduleSchema(),
		"schedule_enabled": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
			Description: "Whether the schedule runs. When the backup quota runs out the backend sets this to false on every job in the VPC; " +
				"the next plan then shows it changing back to true, and the apply fails until the quota is raised.",
		},
		"status":     {Type: schema.TypeString, Computed: true, Description: "Status of the job."},
		"next_run":   {Type: schema.TypeString, Computed: true, Description: "Next scheduled run, RFC 3339 UTC."},
		"created_at": {Type: schema.TypeString, Computed: true, Description: "Creation time, RFC 3339 UTC."},
	}
}

func expandVmJobCreate(d *schema.ResourceData, tenantId string) VmJobCreatePayload {
	p := VmJobCreatePayload{
		Name:               d.Get("name").(string),
		Description:        d.Get("description").(string),
		Notification:       setToStrings(d.Get("notification_method_ids")),
		AppliedInstances:   setToStrings(d.Get("instance_ids")),
		NumberLatestBackup: d.Get("retention").(int),
		TenantUserId:       tenantId,
	}
	applyVmCreateSchedule(&p, expandSchedule(d.Get("schedule")))
	return p
}

// expandVmJobUpdate builds the full update body. interval must be 1: without
// it the backend builds an rrule with INTERVAL=None (awx_handler.py:433).
func expandVmJobUpdate(d *schema.ResourceData, now time.Time) (VmJobUpdatePayload, error) {
	p := VmJobUpdatePayload{
		Name:               d.Get("name").(string),
		Description:        d.Get("description").(string),
		Interval:           1,
		NumberLatestBackup: d.Get("retention").(int),
		Recipients:         setToStrings(d.Get("notification_method_ids")),
		Instances:          setToStrings(d.Get("instance_ids")),
	}
	err := applyVmUpdateSchedule(&p, expandSchedule(d.Get("schedule")), now)
	return p, err
}

// sameVmSchedule reports whether two schedules run at the same UTC time with
// the same mode and day. v1 stores only UTC, so a time_zone change that keeps
// the UTC time needs no API call.
func sameVmSchedule(a, b Schedule, now time.Time) bool {
	if a.FullBackupFrequency != b.FullBackupFrequency || a.FullBackupDay != b.FullBackupDay {
		return false
	}
	ua, errA := scheduleTimeToUTC(a.Time, a.TimeZone, now)
	ub, errB := scheduleTimeToUTC(b.Time, b.TimeZone, now)
	if errA != nil || errB != nil {
		return false
	}
	return ua.Format("15:04") == ub.Format("15:04")
}

func vmJobNeedsUpdate(d *schema.ResourceData) bool {
	if !d.HasChangesExcept("schedule_enabled") {
		return false
	}
	if d.HasChangesExcept("schedule_enabled", "schedule") {
		return true
	}
	oldRaw, newRaw := d.GetChange("schedule")
	return !sameVmSchedule(expandSchedule(oldRaw), expandSchedule(newRaw), time.Now())
}

func findVmJobByName(svc VmService, vpcId, name string) (*VmJobListItem, error) {
	jobs, err := svc.ListVmJobs(vpcId)
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		if jobs[i].Name == name {
			return &jobs[i], nil
		}
	}
	return nil, nil
}

func switchVmJob(ctx context.Context, svc VmService, vpcId, jobId string, enable bool, timeout time.Duration) error {
	if _, err := waitVmJobSettled(ctx, svc, vpcId, jobId, timeout); err != nil {
		return err
	}
	taskId, err := svc.SwitchVmJobSchedule(vpcId, jobId, enable)
	if err != nil {
		return err
	}
	action := "disabling the VM backup job schedule"
	if enable {
		action = "enabling the VM backup job schedule"
	}
	_, err = waitForTask(ctx, svc, vpcId, taskId, action, timeout)
	return err
}

func createVmJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	svc := NewVmService(client)
	vpcId := d.Get("vpc_id").(string)
	name := d.Get("name").(string)
	timeout := d.Timeout(schema.TimeoutCreate)

	existing, err := findVmJobByName(svc, vpcId, name)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing != nil {
		return diag.Errorf("a VM backup job named %q already exists in VPC %s; import it with the id vpc/%s/backup_native_vm_job/%s or choose another name",
			name, vpcId, vpcId, existing.Id)
	}

	// Without tenant_user_id the backend reports the service as disabled.
	tenant, err := fptcloud_vpc.NewService(client).GetTenant(ctx)
	if err != nil {
		return diag.Errorf("looking up the tenant id of %q: %v", client.TenantName, err)
	}

	taskId, err := svc.CreateVmJob(vpcId, expandVmJobCreate(d, tenant.Id))
	if err != nil {
		return diag.FromErr(err)
	}
	message, err := waitForTask(ctx, svc, vpcId, taskId, "creating the VM backup job", timeout)
	if err != nil {
		return diag.FromErr(err)
	}

	// The create response carries a task id, never the job id.
	job, err := findVmJobByName(svc, vpcId, name)
	if err != nil {
		return diag.FromErr(err)
	}
	if job == nil {
		return diag.Errorf("the VM backup job %q was created but does not appear in the VPC's job list", name)
	}
	d.SetId(job.Id)

	var diags diag.Diagnostics
	if strings.Contains(strings.ToLower(message), "quota") {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "The VM backup job was created with its schedule disabled",
			Detail:   message + ". The backup quota is exceeded; the next plan shows schedule_enabled changing back to true.",
		})
	}
	if !d.Get("schedule_enabled").(bool) && job.IsEnabled {
		if err := switchVmJob(ctx, svc, vpcId, job.Id, false, timeout); err != nil {
			return append(diags, diag.FromErr(err)...)
		}
	}
	return append(diags, readVmJob(ctx, d, m)...)
}

func readVmJob(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVmService(m.(*common.Client))
	job, err := svc.GetVmJob(d.Get("vpc_id").(string), d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if job == nil {
		d.SetId("")
		return nil
	}

	sched, err := vmScheduleFromDetail(job, expandSchedule(d.Get("schedule")).TimeZone)
	if err != nil {
		return diag.FromErr(err)
	}
	instanceIds := make([]string, 0, len(job.Instances))
	for _, instance := range job.Instances {
		instanceIds = append(instanceIds, instance.Id)
	}
	recipientIds := make([]string, 0, len(job.Recipients))
	for _, recipient := range job.Recipients {
		recipientIds = append(recipientIds, recipient.Id)
	}

	return setValues(d, map[string]interface{}{
		"name":                    job.Name,
		"description":             job.Description,
		"instance_ids":            instanceIds,
		"notification_method_ids": recipientIds,
		"retention":               job.NumberLatestBackup,
		"schedule":                flattenSchedule(sched),
		"schedule_enabled":        job.IsEnabled,
		"status":                  job.Status,
		"next_run":                formatAPITime(job.NextRun),
		"created_at":              formatAPITime(job.CreatedAt),
	})
}

func updateVmJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVmService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	timeout := d.Timeout(schema.TimeoutUpdate)

	if vmJobNeedsUpdate(d) {
		if _, err := waitVmJobSettled(ctx, svc, vpcId, d.Id(), timeout); err != nil {
			return diag.FromErr(err)
		}
		payload, err := expandVmJobUpdate(d, time.Now())
		if err != nil {
			return diag.FromErr(err)
		}
		taskId, err := svc.UpdateVmJob(vpcId, d.Id(), payload)
		if err != nil {
			return diag.FromErr(err)
		}
		if _, err := waitForTask(ctx, svc, vpcId, taskId, "updating the VM backup job", timeout); err != nil {
			return diag.FromErr(err)
		}
	}

	// Compared with the job, not the plan: the backend changes is_enabled on
	// its own when the quota runs out, and switch_schedule rejects a switch
	// to the state the job is already in.
	job, err := svc.GetVmJob(vpcId, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if job == nil {
		return diag.Errorf("VM backup job %s no longer exists", d.Id())
	}
	if want := d.Get("schedule_enabled").(bool); job.IsEnabled != want {
		if err := switchVmJob(ctx, svc, vpcId, d.Id(), want, timeout); err != nil {
			return diag.FromErr(err)
		}
	}
	return readVmJob(ctx, d, m)
}

func deleteVmJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVmService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	// The API rejects a delete while a backup runs, so wait for it first.
	job, err := svc.GetVmJob(vpcId, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if job == nil {
		return nil
	}
	if _, err := waitVmJobSettled(ctx, svc, vpcId, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		// An earlier delete that timed out may finish while this one waits.
		if job, getErr := svc.GetVmJob(vpcId, d.Id()); getErr == nil && job == nil {
			return nil
		}
		return diag.FromErr(err)
	}
	if err := svc.DeleteVmJob(vpcId, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	if err := waitVmJobGone(ctx, svc, vpcId, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
