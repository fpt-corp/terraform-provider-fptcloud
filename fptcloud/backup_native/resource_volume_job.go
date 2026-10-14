package fptcloud_backup_native

import (
	"context"
	"fmt"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceVolumeJob() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a volume backup job of the Backup Native service. Only FKE persistent volumes (names starting with `pv-fke`) can be backed up. " +
			"Destroying the resource deletes the job; its restore points are kept and still count against the backup quota.",
		CreateContext: createVolumeJob,
		ReadContext:   readVolumeJob,
		UpdateContext: updateVolumeJob,
		DeleteContext: deleteVolumeJob,
		CustomizeDiff: customdiff.All(customizeSchedule, customizeVolumeDescription),
		Schema:        resourceVolumeJobSchema(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},
		Importer: &schema.ResourceImporter{StateContext: importJob("backup_native_volume_job")},
	}
}

func resourceVolumeJobSchema() map[string]*schema.Schema {
	description := jobDescriptionSchema()
	description.Description = "Description, at most 100 characters. Once set it cannot be removed: the API ignores an empty description."
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "ID of the VPC.",
		},
		"name":        jobNameSchema("volume"),
		"description": description,
		"volume_ids": {
			Type:        schema.TypeSet,
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "IDs of the FKE persistent volumes to back up. A volume can belong to only one backup job.",
		},
		"notification_method_ids": {
			Type:        schema.TypeSet,
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "IDs of alert notification methods that receive the job's notifications. See the `fptcloud_alert_notification_methods` data source.",
		},
		"retention": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntBetween(1, 1000),
			Description:  "Number of restore points kept per volume, 1-1000.",
		},
		"schedule": scheduleSchema(),
		"schedule_enabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Whether the schedule runs.",
		},
		"status":     {Type: schema.TypeString, Computed: true, Description: "Status of the job."},
		"created_at": {Type: schema.TypeString, Computed: true, Description: "Creation time, RFC 3339 UTC."},
	}
}

// checkDescriptionChange rejects removing a description: v2 update ignores an
// empty value (update_backup_job_controller.py:515-521).
func checkDescriptionChange(oldValue, newValue string) error {
	if oldValue != "" && newValue == "" {
		return fmt.Errorf("description cannot be removed from a volume backup job once set, because the API ignores an empty description; set a different description instead")
	}
	return nil
}

func customizeVolumeDescription(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if d.Id() == "" || !d.HasChange("description") || !d.NewValueKnown("description") {
		return nil
	}
	oldRaw, newRaw := d.GetChange("description")
	return checkDescriptionChange(oldRaw.(string), newRaw.(string))
}

// expandVolumeJob builds the full create and update body. Update always
// carries backup_objects (omitting it is HTTP 500) and backup_time_zone
// (omitting it switches the job to UTC).
func expandVolumeJob(d *schema.ResourceData) VolumeJobPayload {
	p := VolumeJobPayload{
		Name:          d.Get("name").(string),
		Description:   d.Get("description").(string),
		Notifications: setToStrings(d.Get("notification_method_ids")),
		BackupObjects: setToStrings(d.Get("volume_ids")),
		Retention:     d.Get("retention").(int),
	}
	applyVolumeSchedule(&p, expandSchedule(d.Get("schedule")))
	return p
}

func findVolumeJobByName(svc VolumeService, vpcId, name string) (*VolumeJobListItem, error) {
	jobs, err := svc.ListVolumeJobs(vpcId)
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

func applyVolumeJobUpdate(ctx context.Context, svc VolumeService, vpcId, jobId string, p VolumeJobPayload, timeout time.Duration) error {
	if err := waitVolumeJobSettled(ctx, svc, vpcId, jobId, timeout); err != nil {
		return err
	}
	failures, err := svc.UpdateVolumeJob(vpcId, jobId, p)
	if err != nil {
		return err
	}
	if len(failures) > 0 {
		return fmt.Errorf("the volume backup job %s was updated, but these volumes could not be added: %s", jobId, formatVolumeFailures(failures))
	}
	return nil
}

// setVolumeJobEnabled waits for a running operation to finish first, because
// v2 accepts actions only from a settled job.
func setVolumeJobEnabled(ctx context.Context, svc VolumeService, vpcId, jobId string, enabled bool, timeout time.Duration) error {
	if err := waitVolumeJobSettled(ctx, svc, vpcId, jobId, timeout); err != nil {
		return err
	}
	return svc.SetVolumeJobEnabled(vpcId, jobId, enabled)
}

// withoutFailedVolumes drops the volumes a create could not add, so the
// update that follows does not send them again.
func withoutFailedVolumes(volumeIds []string, failures []VolumeFailure) []string {
	failed := map[string]bool{}
	for _, f := range failures {
		failed[f.VolumeId] = true
	}
	kept := make([]string, 0, len(volumeIds))
	for _, id := range volumeIds {
		if !failed[id] {
			kept = append(kept, id)
		}
	}
	return kept
}

func createVolumeJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	name := d.Get("name").(string)
	timeout := d.Timeout(schema.TimeoutCreate)

	existing, err := findVolumeJobByName(svc, vpcId, name)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing != nil {
		return diag.Errorf("a volume backup job named %q already exists in VPC %s; import it with the id vpc/%s/backup_native_volume_job/%s or choose another name",
			name, vpcId, vpcId, existing.Id)
	}

	payload := expandVolumeJob(d)
	failures, err := svc.CreateVolumeJob(vpcId, payload)
	if err != nil {
		return diag.FromErr(err)
	}

	// The create response never carries the job id.
	job, err := findVolumeJobByName(svc, vpcId, name)
	if err != nil {
		return diag.FromErr(err)
	}
	if job == nil {
		if len(failures) > 0 {
			return diag.Errorf("creating the volume backup job %q partly failed and the job does not appear in the VPC's job list: %s", name, formatVolumeFailures(failures))
		}
		return diag.Errorf("the volume backup job %q was created but does not appear in the VPC's job list", name)
	}
	d.SetId(job.Id)

	// A partial create is a warning, not an error: an error with the id set
	// taints the resource, and the next apply would replace the job instead
	// of adding the missing volumes.
	var diags diag.Diagnostics
	if len(failures) > 0 {
		payload.BackupObjects = withoutFailedVolumes(payload.BackupObjects, failures)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("the volume backup job %s was created, but these volumes could not be added: %s", job.Id, formatVolumeFailures(failures)),
			Detail:   "The job is saved in state without them; the next plan shows them as a change to volume_ids.",
		})
	}

	// v2 create does not store description; update does.
	if payload.Description != "" {
		if err := applyVolumeJobUpdate(ctx, svc, vpcId, job.Id, payload, timeout); err != nil {
			return append(diags, diag.FromErr(err)...)
		}
	}
	if !d.Get("schedule_enabled").(bool) {
		if err := setVolumeJobEnabled(ctx, svc, vpcId, job.Id, false, timeout); err != nil {
			return append(diags, diag.FromErr(err)...)
		}
	}
	return append(diags, readVolumeJob(ctx, d, m)...)
}

func readVolumeJob(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	job, err := svc.GetVolumeJob(d.Get("vpc_id").(string), d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if job == nil {
		d.SetId("")
		return nil
	}

	volumeIds := make([]string, 0, len(job.Targets))
	for _, target := range job.Targets {
		volumeIds = append(volumeIds, target.Id)
	}
	notificationIds := make([]string, 0, len(job.Notifications))
	for _, notification := range job.Notifications {
		notificationIds = append(notificationIds, notification.Id)
	}

	return setValues(d, map[string]interface{}{
		"name":                    job.Name,
		"description":             job.Description,
		"volume_ids":              volumeIds,
		"notification_method_ids": notificationIds,
		"retention":               job.Retention,
		"schedule":                flattenSchedule(volumeScheduleFromDetail(job)),
		"schedule_enabled":        job.IsEnabled,
		"status":                  job.Status,
		"created_at":              formatAPITime(job.CreatedAt),
	})
}

func updateVolumeJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	timeout := d.Timeout(schema.TimeoutUpdate)

	if d.HasChangesExcept("schedule_enabled") {
		if err := applyVolumeJobUpdate(ctx, svc, vpcId, d.Id(), expandVolumeJob(d), timeout); err != nil {
			return diag.FromErr(err)
		}
	}
	// Compared with the job, not the plan, as in the VM job.
	job, err := svc.GetVolumeJob(vpcId, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if job == nil {
		return diag.Errorf("volume backup job %s no longer exists", d.Id())
	}
	if want := d.Get("schedule_enabled").(bool); job.IsEnabled != want {
		if err := setVolumeJobEnabled(ctx, svc, vpcId, d.Id(), want, timeout); err != nil {
			return diag.FromErr(err)
		}
	}
	return readVolumeJob(ctx, d, m)
}

func deleteVolumeJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	// The API rejects a delete while a backup runs, so wait for it first.
	job, err := svc.GetVolumeJob(vpcId, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if job == nil {
		return nil
	}
	if err := waitVolumeJobSettled(ctx, svc, vpcId, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		// An earlier delete that timed out may finish while this one waits.
		if job, getErr := svc.GetVolumeJob(vpcId, d.Id()); getErr == nil && job == nil {
			return nil
		}
		return diag.FromErr(err)
	}
	if err := svc.DeleteVolumeJob(vpcId, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	if err := waitVolumeJobGone(ctx, svc, vpcId, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
