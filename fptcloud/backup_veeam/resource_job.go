package fptcloud_backup_veeam

import (
	"context"
	"fmt"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceBackupVeeamJob() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a backup job of the Backup Veeam service.",
		CreateContext: createBackupVeeamJob,
		ReadContext:   readBackupVeeamJob,
		UpdateContext: updateBackupVeeamJob,
		DeleteContext: deleteBackupVeeamJob,
		CustomizeDiff: validateJobDiff,
		Schema:        resourceBackupVeeamJobSchema,
		// A backup job is asynchronous: creating or deleting one can take several
		// minutes. The timeouts block lets a user set a limit per job instead of
		// relying on the provider-wide timeout.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 4 || parts[0] != "vpc" || parts[2] != "backup_veeam_job" {
					return nil, fmt.Errorf("malformed import id, expected vpc/<vpc_id>/backup_veeam_job/<job_id>")
				}
				if err := d.Set("vpc_id", parts[1]); err != nil {
					return nil, err
				}
				d.SetId(parts[3])
				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func expandJobPayload(d *schema.ResourceData) CreateJobPayload {
	vmIds := make([]string, 0)
	for _, v := range d.Get("vm_ids").(*schema.Set).List() {
		vmIds = append(vmIds, v.(string))
	}

	notificationIds := make([]string, 0)
	if raw, ok := d.GetOk("notification_method_ids"); ok {
		for _, v := range raw.(*schema.Set).List() {
			notificationIds = append(notificationIds, v.(string))
		}
	}

	payload := CreateJobPayload{
		Name:        strings.TrimSpace(d.Get("name").(string)),
		Description: d.Get("description").(string),
		// Always true: the portal hardcodes it on create too, and the update
		// endpoint overwrites this with the value from the database, so whatever
		// is sent here is harmless.
		Enabled:               true,
		ScheduleEnabled:       d.Get("schedule_enabled").(bool),
		VmIds:                 vmIds,
		NotificationMethodIds: notificationIds,
		IsCapacityTierEnabled: d.Get("is_capacity_tier_enabled").(bool),
	}

	if retentionList, ok := d.GetOk("retention"); ok {
		items := retentionList.([]interface{})
		if len(items) > 0 && items[0] != nil {
			retention := items[0].(map[string]interface{})
			payload.Retention = RetentionPayload{
				Cycles:    retention["cycles"].(int),
				LimitType: retention["limit_type"].(string),
			}
		}
	}

	payload.Schedule = expandSchedule(d)
	return payload
}

func expandSchedule(d *schema.ResourceData) *SchedulePayload {
	raw, ok := d.GetOk("schedule")
	if !ok {
		return nil
	}
	items, ok := raw.([]interface{})
	if !ok || len(items) == 0 || items[0] == nil {
		return nil
	}
	scheduleMap, ok := items[0].(map[string]interface{})
	if !ok {
		return nil
	}
	scheduleType, _ := scheduleMap["type"].(string)

	nested := func(key string) map[string]interface{} {
		list, ok := scheduleMap[key].([]interface{})
		if !ok || len(list) == 0 || list[0] == nil {
			return nil
		}
		m, _ := list[0].(map[string]interface{})
		return m
	}

	payload := &SchedulePayload{ScheduleType: scheduleType}

	switch scheduleType {
	case "daily":
		daily := &DailySchedulePayload{
			Enabled: true,
			Type:    "Everyday",
			RunAt:   "22:00:00",
			Days:    allDays, // always send all seven days, like the portal
		}
		if block := nested("daily"); block != nil {
			if v, ok := block["type"].(string); ok && v != "" {
				daily.Type = v
			}
			if v, ok := block["run_at"].(string); ok && v != "" {
				daily.RunAt = v
			}
		}
		payload.DailySchedule = daily

	case "monthly":
		monthly := &MonthlySchedulePayload{
			Enabled:          true,
			RunAt:            "22:00:00",
			DayNumberInMonth: "fourth",
			Months:           allMonths, // always send all twelve months
		}
		if block := nested("monthly"); block != nil {
			if v, ok := block["run_at"].(string); ok && v != "" {
				monthly.RunAt = v
			}
			if v, ok := block["day_number_in_month"].(string); ok && v != "" {
				monthly.DayNumberInMonth = v
			}
			// XOR: only send the field that matches day_number_in_month.
			if monthly.DayNumberInMonth == "onDay" {
				if v, ok := block["day_of_month"].(int); ok {
					monthly.DayOfMonth = v
				}
			} else if v, ok := block["day_of_week"].(string); ok && v != "" {
				monthly.DayOfWeek = v
			}
		}
		payload.MonthlySchedule = monthly

	case "period":
		fullPeriod := 12
		startHour := 0
		endHour := 23
		if block := nested("period"); block != nil {
			if v, ok := block["full_period"].(int); ok && v != 0 {
				fullPeriod = v
			}
			if v, ok := block["start_hour"].(int); ok {
				startHour = v
			}
			if v, ok := block["end_hour"].(int); ok {
				endHour = v
			}
		}
		payload.PeriodSchedule = &PeriodSchedulePayload{
			Enabled:    true,
			Type:       "Hours",
			FullPeriod: fullPeriod,
			Schedules:  BuildPeriodBitmap(startHour, endHour),
		}
	}

	return payload
}

func flattenJobDetail(d *schema.ResourceData, detail *JobDetail) error {
	vmIds := make([]string, 0, len(detail.BackupObject))
	displayNames := make([]string, 0, len(detail.BackupObject))
	for _, obj := range detail.BackupObject {
		vmIds = append(vmIds, obj.VmId)
		displayNames = append(displayNames, obj.VmDisplayName)
	}

	// A nil slice and an empty slice are the same thing in Go but not in
	// Terraform: nil lands in state as null, while a configuration that
	// computes an empty list - a `for` expression matching nothing, or a
	// literal [] - is a known empty set. Every refresh then reports drift on a
	// job that has not changed. The API omits notification_method_ids for a job
	// with no notifications, so normalise it here.
	notificationIds := detail.NotificationMethodIds
	if notificationIds == nil {
		notificationIds = []string{}
	}

	setters := map[string]interface{}{
		"name":                     detail.Name,
		"description":              detail.Description,
		"schedule_enabled":         detail.ScheduleEnabled,
		"is_capacity_tier_enabled": detail.IsCapacityTierEnabled,
		"vm_ids":                   vmIds,
		"vm_display_names":         displayNames,
		"notification_method_ids":  notificationIds,
		"retention": []interface{}{map[string]interface{}{
			"cycles":     detail.BackupRetention.Cycles,
			"limit_type": detail.BackupRetention.LimitType,
		}},
	}
	for key, value := range setters {
		if err := d.Set(key, value); err != nil {
			return fmt.Errorf("could not write %s to state: %v", key, err)
		}
	}

	if schedule := flattenSchedule(detail.BackupSchedule); schedule != nil {
		if err := d.Set("schedule", schedule); err != nil {
			return fmt.Errorf("could not write schedule to state: %v", err)
		}
	}
	return nil
}

// canonicalDailyType maps the daily schedule type the API reads back onto the
// spelling the schema accepts.
//
// The create endpoint takes `weekDays`, but the detail endpoint passes Veeam's
// own value straight through, which is `WeekDays`. The backend lowercases the
// monthly fields on the way out (JobDetailController.__get_monthly_schedule)
// and simply forgot to do the same for the daily type. Writing the raw value
// into state gives every subsequent plan a diff that never converges: apply it
// and the API answers `WeekDays` again.
func canonicalDailyType(value string) string {
	for _, candidate := range scheduleDailyTypes {
		if strings.EqualFold(candidate, value) {
			return candidate
		}
	}
	// An unrecognised value is left alone rather than guessed at, so the plan
	// shows the mismatch instead of the provider hiding it.
	return value
}

func flattenSchedule(payload *SchedulePayload) []interface{} {
	if payload == nil {
		return nil
	}

	scheduleMap := map[string]interface{}{"type": payload.ScheduleType}

	switch payload.ScheduleType {
	case "daily":
		if daily := payload.DailySchedule; daily != nil {
			scheduleMap["daily"] = []interface{}{map[string]interface{}{
				"type":   canonicalDailyType(daily.Type),
				"run_at": daily.RunAt,
			}}
		}
	case "monthly":
		if m := payload.MonthlySchedule; m != nil {
			monthly := map[string]interface{}{
				"run_at":              m.RunAt,
				"day_number_in_month": m.DayNumberInMonth,
			}
			// Only map the field that currently carries meaning. The backend reads
			// day_of_month back as 1 when the job does not use onDay; writing that
			// into state would produce a phantom diff.
			if m.DayNumberInMonth == "onDay" {
				monthly["day_of_month"] = m.DayOfMonth
			} else {
				monthly["day_of_week"] = m.DayOfWeek
			}
			scheduleMap["monthly"] = []interface{}{monthly}
		}
	case "period":
		if p := payload.PeriodSchedule; p != nil {
			startHour, endHour, ok := ParsePeriodBitmap(p.Schedules)
			if !ok {
				// Broken bitmap (all zeroes): do not guess, leave it at 0/0 so the
				// plan shows a diff against the configuration.
				startHour, endHour = 0, 0
			}
			scheduleMap["period"] = []interface{}{map[string]interface{}{
				"full_period": p.FullPeriod,
				"start_hour":  startHour,
				"end_hour":    endHour,
			}}
		}
	}

	return []interface{}{scheduleMap}
}

func createBackupVeeamJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)
	vpcId := d.Get("vpc_id").(string)

	payload := expandJobPayload(d)

	response, err := service.CreateJob(vpcId, payload)
	if err != nil {
		return diag.FromErr(err)
	}

	jobId := response.BackupJobId
	if jobId == "" {
		jobId = response.ResourceId
	}
	if jobId == "" {
		return diag.Errorf("the API reported the backup job was created but returned no id")
	}
	// SetId BEFORE polling: if the poll times out, state still holds the id so
	// terraform destroy can clean it up rather than leaving an orphaned job.
	d.SetId(jobId)

	name := strings.TrimSpace(d.Get("name").(string))
	if _, err := WaitForJobSettled(ctx, service, vpcId, jobId, name, d.Timeout(schema.TimeoutCreate)); err != nil {
		return diag.FromErr(err)
	}
	if err := WaitForVmIdsSettled(ctx, service, vpcId, jobId, payload.VmIds, d.Timeout(schema.TimeoutCreate)); err != nil {
		return diag.FromErr(err)
	}

	return readBackupVeeamJob(ctx, d, m)
}

func readBackupVeeamJob(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)
	vpcId := d.Get("vpc_id").(string)

	detail, err := service.GetJobDetail(vpcId, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if detail == nil {
		// The job was deleted outside Terraform - that is drift, not an error.
		d.SetId("")
		return nil
	}

	if err := flattenJobDetail(d, detail); err != nil {
		return diag.FromErr(err)
	}

	// status and enabled ONLY exist on the list endpoint; detail returns neither.
	item, err := service.FindJobInList(vpcId, d.Id(), detail.Name)
	if err != nil {
		return diag.FromErr(err)
	}
	if item == nil {
		d.SetId("")
		return nil
	}
	if err := d.Set("status", item.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("enabled", item.Enabled); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func updateBackupVeeamJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)
	vpcId := d.Get("vpc_id").(string)

	payload := expandJobPayload(d)

	if _, err := service.UpdateJob(vpcId, d.Id(), payload); err != nil {
		return diag.FromErr(err)
	}

	name := strings.TrimSpace(d.Get("name").(string))
	if _, err := WaitForJobSettled(ctx, service, vpcId, d.Id(), name, d.Timeout(schema.TimeoutUpdate)); err != nil {
		return diag.FromErr(err)
	}

	// Status settles before the instance list does, so reading now would store
	// the previous list and leave the next plan showing a diff.
	if err := WaitForVmIdsSettled(ctx, service, vpcId, d.Id(), payload.VmIds, d.Timeout(schema.TimeoutUpdate)); err != nil {
		return diag.FromErr(err)
	}

	return readBackupVeeamJob(ctx, d, m)
}

func deleteBackupVeeamJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)
	vpcId := d.Get("vpc_id").(string)

	// Read first: if the job is already gone the work is done. Calling DELETE
	// straight on a job that does not exist answers HTTP 500, because the
	// backend does not null-check.
	detail, err := service.GetJobDetail(vpcId, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if detail == nil {
		d.SetId("")
		return nil
	}

	if err := service.DeleteJob(vpcId, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	if err := WaitForJobGone(ctx, service, vpcId, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
