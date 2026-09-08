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
		// Backup job la resource async, tao/xoa co the mat vai phut. Block
		// timeouts cho khach dat rieng tung job thay vi dung timeout chung
		// cua provider.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 4 || parts[0] != "vpc" || parts[2] != "backup_veeam_job" {
					return nil, fmt.Errorf("import id sai định dạng, cần vpc/<vpc_id>/backup_veeam_job/<job_id>")
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
		// Luôn true: portal cũng hardcode như vậy khi tạo, và endpoint update
		// ghi đè giá trị này bằng giá trị trong DB nên gửi gì cũng vô hại.
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
			Days:    allDays, // luôn gửi đủ 7 ngày, giống portal
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
			Months:           allMonths, // luôn gửi đủ 12 tháng
		}
		if block := nested("monthly"); block != nil {
			if v, ok := block["run_at"].(string); ok && v != "" {
				monthly.RunAt = v
			}
			if v, ok := block["day_number_in_month"].(string); ok && v != "" {
				monthly.DayNumberInMonth = v
			}
			// XOR: chỉ gửi field ứng với day_number_in_month.
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

	setters := map[string]interface{}{
		"name":                     detail.Name,
		"description":              detail.Description,
		"schedule_enabled":         detail.ScheduleEnabled,
		"is_capacity_tier_enabled": detail.IsCapacityTierEnabled,
		"vm_ids":                   vmIds,
		"vm_display_names":         displayNames,
		"notification_method_ids":  detail.NotificationMethodIds,
		"retention": []interface{}{map[string]interface{}{
			"cycles":     detail.BackupRetention.Cycles,
			"limit_type": detail.BackupRetention.LimitType,
		}},
	}
	for key, value := range setters {
		if err := d.Set(key, value); err != nil {
			return fmt.Errorf("không ghi được %s vào state: %v", key, err)
		}
	}

	if schedule := flattenSchedule(detail.BackupSchedule); schedule != nil {
		if err := d.Set("schedule", schedule); err != nil {
			return fmt.Errorf("không ghi được schedule vào state: %v", err)
		}
	}
	return nil
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
				"type":   daily.Type,
				"run_at": daily.RunAt,
			}}
		}
	case "monthly":
		if m := payload.MonthlySchedule; m != nil {
			monthly := map[string]interface{}{
				"run_at":              m.RunAt,
				"day_number_in_month": m.DayNumberInMonth,
			}
			// Chỉ map field đang có nghĩa. Backend đọc ra day_of_month = 1 khi
			// job không dùng onDay; map nó vào state sẽ tạo diff giả.
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
				// Bitmap hỏng (toàn 0): không đoán giá trị, để nguyên 0/0 cho
				// plan hiện diff so với config.
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

	response, err := service.CreateJob(vpcId, expandJobPayload(d))
	if err != nil {
		return diag.FromErr(err)
	}

	jobId := response.BackupJobId
	if jobId == "" {
		jobId = response.ResourceId
	}
	if jobId == "" {
		return diag.Errorf("API báo tạo backup job thành công nhưng không trả về id")
	}
	// SetId TRƯỚC khi poll: nếu poll timeout thì state vẫn giữ id để
	// terraform destroy dọn được, không để job mồ côi.
	d.SetId(jobId)

	name := strings.TrimSpace(d.Get("name").(string))
	if _, err := WaitForJobSettled(ctx, service, vpcId, jobId, name, d.Timeout(schema.TimeoutCreate)); err != nil {
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
		// Job bị xoá ngoài Terraform - đây là drift, không phải lỗi.
		d.SetId("")
		return nil
	}

	if err := flattenJobDetail(d, detail); err != nil {
		return diag.FromErr(err)
	}

	// status và enabled CHỈ có ở endpoint list, detail không trả về.
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

	if _, err := service.UpdateJob(vpcId, d.Id(), expandJobPayload(d)); err != nil {
		return diag.FromErr(err)
	}

	name := strings.TrimSpace(d.Get("name").(string))
	if _, err := WaitForJobSettled(ctx, service, vpcId, d.Id(), name, d.Timeout(schema.TimeoutUpdate)); err != nil {
		return diag.FromErr(err)
	}

	return readBackupVeeamJob(ctx, d, m)
}

func deleteBackupVeeamJob(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)
	vpcId := d.Get("vpc_id").(string)

	// Đọc trước: job đã bị xoá thì coi như xong. Gọi thẳng DELETE lên một job
	// không tồn tại sẽ trả HTTP 500 vì backend không null-check.
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
