package fptcloud_backup_veeam

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var jobNamePattern = regexp.MustCompile(`^[\w\-\s.]+$`)

// The portal only ever sends these two. The backend also accepts
// SelectedDays, but nothing has ever sent it, so it is not exposed rather than
// committing to behaviour nobody has exercised.
var scheduleDailyTypes = []string{"Everyday", "weekDays"}

var monthlyDayNumbers = []string{"first", "second", "third", "fourth", "last", "onDay"}

// LOWERCASE - unlike the day names elsewhere, which are capitalised. This is
// how the API actually behaves; the wrong casing is rejected.
var monthlyDaysOfWeek = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

// The backend accepts 0 as well, but the portal never sends it; stay with the
// set the portal uses.
var periodFullPeriods = []int{1, 2, 3, 4, 6, 8, 12, 24}

var runAtPattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d:[0-5]\d$`)

var resourceBackupVeeamJobSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC that owns this backup job.",
	},
	"name": {
		Type:     schema.TypeString,
		Required: true,
		// The server calls .strip() on the job name. Without normalising it here,
		// state and API disagree forever and every plan proposes a change.
		StateFunc: func(v interface{}) string {
			return strings.TrimSpace(v.(string))
		},
		ValidateFunc: func(v interface{}, k string) ([]string, []error) {
			name := strings.TrimSpace(v.(string))
			if name == "" {
				return nil, []error{fmt.Errorf("%s must not be empty", k)}
			}
			if len(name) > 50 {
				return nil, []error{fmt.Errorf("%s must not be longer than 50 characters", k)}
			}
			if !jobNamePattern.MatchString(name) {
				return nil, []error{fmt.Errorf("%s may only contain letters, digits, underscores, hyphens, spaces and dots", k)}
			}
			return nil, nil
		},
		Description: "The name of the backup job. Must be unique within the VPC, at most 50 characters.",
	},
	"description": {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringLenBetween(0, 100),
		Description:  "The description of the backup job, at most 100 characters.",
	},
	"vm_ids": {
		Type:        schema.TypeSet,
		Required:    true,
		MinItems:    1,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "The IDs of the instances protected by this backup job. Changing this list updates the job in place and does not delete existing restore points.",
	},
	"schedule_enabled": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Whether the backup job has a schedule.",
	},
	"is_capacity_tier_enabled": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		ForceNew:    true,
		Description: "Whether capacity tier is enabled. The API refuses to change this on an existing job, so changing it destroys and recreates the job, losing its restore points.",
	},
	"retention": {
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"cycles": {
					Type:         schema.TypeInt,
					Required:     true,
					ValidateFunc: validation.IntAtLeast(1),
					Description:  "How many restore points or days to keep.",
				},
				"limit_type": {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringInSlice([]string{"Days", "Cycles"}, false),
					Description:  "The unit of `cycles`: `Days` or `Cycles` (shown as \"Restore points\" in the portal).",
				},
			},
		},
		Description: "The retention policy of the backup job.",
	},
	"schedule": {
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringInSlice([]string{"daily", "monthly", "period"}, false),
					Description:  "Which schedule block applies: `daily`, `monthly` or `period`.",
				},
				"daily": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type": {
								Type:         schema.TypeString,
								Optional:     true,
								Default:      "Everyday",
								ValidateFunc: validation.StringInSlice(scheduleDailyTypes, false),
								Description:  "`Everyday` for all seven days, `weekDays` for working days only.",
							},
							"run_at": {
								Type:         schema.TypeString,
								Optional:     true,
								Default:      "22:00:00",
								ValidateFunc: validation.StringMatch(runAtPattern, "must be in HH:MM:SS format"),
								Description:  "The time of day the job runs, in `HH:MM:SS`.",
							},
						},
					},
					Description: "Runs once a day. Use it when `type` is `daily`.",
				},
				"monthly": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"run_at": {
								Type:         schema.TypeString,
								Optional:     true,
								Default:      "22:00:00",
								ValidateFunc: validation.StringMatch(runAtPattern, "must be in HH:MM:SS format"),
								Description:  "The time of day the job runs, in `HH:MM:SS`.",
							},
							"day_number_in_month": {
								Type:         schema.TypeString,
								Optional:     true,
								Default:      "fourth",
								ValidateFunc: validation.StringInSlice(monthlyDayNumbers, false),
								Description:  "Pick the day by week position (`first`, `second`, `third`, `fourth`, `last`) or by calendar day (`onDay`).",
							},
							"day_of_week": {
								Type:         schema.TypeString,
								Optional:     true,
								ValidateFunc: validation.StringInSlice(monthlyDaysOfWeek, false),
								Description:  "Lowercase day name. Required when `day_number_in_month` is not `onDay`, and must be omitted when it is.",
							},
							"day_of_month": {
								Type:         schema.TypeInt,
								Optional:     true,
								ValidateFunc: validation.IntBetween(1, 32),
								Description:  "Calendar day 1-31, or 32 meaning the last day of the month. Required when `day_number_in_month` is `onDay`, and must be omitted otherwise.",
							},
						},
					},
					Description: "Runs once a month. Use it when `type` is `monthly`.",
				},
				"period": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"full_period": {
								Type:         schema.TypeInt,
								Optional:     true,
								Default:      12,
								ValidateFunc: validation.IntInSlice(periodFullPeriods),
								Description:  "How many hours between runs.",
							},
							"start_hour": {
								Type:         schema.TypeInt,
								Optional:     true,
								Default:      0,
								ValidateFunc: validation.IntBetween(0, 23),
								Description:  "First hour of the day the job may run.",
							},
							"end_hour": {
								Type:         schema.TypeInt,
								Optional:     true,
								Default:      23,
								ValidateFunc: validation.IntBetween(0, 23),
								Description:  "Last hour of the day the job may run. Must be greater than or equal to `start_hour`: the window cannot wrap past midnight.",
							},
						},
					},
					Description: "Runs every few hours within a daily window. Use it when `type` is `period`.",
				},
			},
		},
		Description: "The schedule of the backup job.",
	},
	"notification_method_ids": {
		Type:     schema.TypeSet,
		Optional: true,
		// Computed as well as Optional so that an empty set and null are not
		// treated as different values. SDKv2 stores an empty set as null after
		// create, while Read writes an empty set, and every later plan then
		// reports drift on a job nobody touched. The cost is the usual
		// Optional+Computed one: deleting the attribute from the configuration
		// keeps the current methods, so clear them with an explicit `= []`.
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Description: "IDs of notification methods. Always source these from the `fptcloud_alert_notification_methods` data source: " +
			"the API silently ignores IDs it does not recognise, so a hand-typed ID leaves the job with no notifications and no error. " +
			"Set it to `[]` to remove every notification method; removing the argument keeps the current ones.",
	},
	"enabled": {
		Type:        schema.TypeBool,
		Computed:    true,
		Description: "Whether the schedule is currently running. Read-only: pause or resume a job from the portal.",
	},
	"status": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The job status as of the last synchronisation, not a real-time value from Veeam.",
	},
	"vm_display_names": {
		Type:        schema.TypeSet,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "Display names of the protected instances. For reading only - instance names can change.",
	},
}

var dataSourceBackupVeeamJobsSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC to list backup jobs from.",
	},
	"name": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Only return jobs whose name matches this value.",
	},
	"status": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Only return jobs with this status.",
	},
	"jobs": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id":               {Type: schema.TypeString, Computed: true},
				"name":             {Type: schema.TypeString, Computed: true},
				"description":      {Type: schema.TypeString, Computed: true},
				"status":           {Type: schema.TypeString, Computed: true},
				"enabled":          {Type: schema.TypeBool, Computed: true},
				"schedule_enabled": {Type: schema.TypeBool, Computed: true},
				"next_run":         {Type: schema.TypeString, Computed: true},
				"latest_run":       {Type: schema.TypeString, Computed: true},
			},
		},
		Description: "The backup jobs found in the VPC.",
	},
}

var dataSourceBackupVeeamInstancesSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC to list instances from.",
	},
	// Named after the API's own vocabulary for this state - VmBackupState
	// calls it PROTECTED / UNPROTECTED - rather than after the query parameter
	// it sets, which is `not_backup`. That parameter is a switch ("apply the
	// not-backed-up filter"), so reading it as a description of the result gets
	// it backwards: not_backup=false applies no filter and returns every
	// instance, protected ones included.
	//
	// Defaults to true because that is what this data source is for: the
	// instances you can put into a job. Returning protected instances by
	// default produces a configuration that plans cleanly and then fails on
	// apply with duplicateVm.
	"unprotected_only": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  true,
		Description: "Only return instances that do not belong to a backup job yet - the ones assignable to a job. " +
			"Set it to `false` to list every instance in the VPC, protected or not.",
	},
	"job_id": {
		Type:     schema.TypeString,
		Optional: true,
		Description: "Keep the instances belonging to this job in the result instead of filtering them out as already protected. " +
			"Use it when editing an existing job. It has no effect unless `unprotected_only` is `true`.",
	},
	"status": {
		Type:     schema.TypeString,
		Optional: true,
		Description: "Comma-separated instance statuses to include, for example `POWERED_ON,POWERED_OFF`. " +
			"The server filters by status ONLY when this is set: left empty, the result can include instances " +
			"that are initialising, unresolved or not yet deployed, none of which can be added to a job.",
	},
	"instances": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id":     {Type: schema.TypeString, Computed: true},
				"name":   {Type: schema.TypeString, Computed: true},
				"status": {Type: schema.TypeString, Computed: true},
			},
		},
		Description: "The instances that can be assigned to a backup job. Instances that are initialising, unresolved, not deployed, or mounted for instant recovery are excluded by the server.",
	},
}

// validateJobDiff only pulls the schedule block out of the diff and hands it
// to the pure function below. Splitting it this way keeps the validation
// logic testable without building a *schema.ResourceDiff through SDK
// internals.
func validateJobDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	// vm_display_names is derived from vm_ids by the server. Without marking it
	// unknown, a plan that changes vm_ids still prints the previous names, which
	// reads as though the instances were not changing at all.
	if diff.Id() != "" && diff.HasChange("vm_ids") {
		if err := diff.SetNewComputed("vm_display_names"); err != nil {
			return err
		}
	}

	scheduleRaw, ok := diff.GetOk("schedule")
	if !ok {
		return nil
	}
	scheduleList, ok := scheduleRaw.([]interface{})
	if !ok || len(scheduleList) == 0 || scheduleList[0] == nil {
		return nil
	}
	scheduleMap, ok := scheduleList[0].(map[string]interface{})
	if !ok {
		return nil
	}
	return validateScheduleConfig(scheduleMap)
}

// validateScheduleConfig blocks three configurations the API accepts but acts
// on incorrectly. It takes a plain map so it can be tested directly.
func validateScheduleConfig(scheduleMap map[string]interface{}) error {
	scheduleType, _ := scheduleMap["type"].(string)

	hasBlock := func(key string) bool {
		list, ok := scheduleMap[key].([]interface{})
		return ok && len(list) > 0 && list[0] != nil
	}
	blockOf := func(key string) map[string]interface{} {
		list, ok := scheduleMap[key].([]interface{})
		if !ok || len(list) == 0 || list[0] == nil {
			return nil
		}
		m, _ := list[0].(map[string]interface{})
		return m
	}

	// 1. Only the block matching type may be declared.
	for _, key := range []string{"daily", "monthly", "period"} {
		if key != scheduleType && hasBlock(key) {
			return fmt.Errorf(
				"schedule.type is %q but a %q block is declared; only the block matching type may be used",
				scheduleType, key)
		}
	}

	// 2. A period window must not wrap past midnight - the bitmap would be all
	//    zeroes and the job would never run, and the API says nothing.
	if scheduleType == "period" {
		if period := blockOf("period"); period != nil {
			start, _ := period["start_hour"].(int)
			end, _ := period["end_hour"].(int)
			if start > end {
				return fmt.Errorf(
					"schedule.period.start_hour (%d) must be less than or equal to end_hour (%d); "+
						"the window cannot wrap past midnight. To run through the night set start_hour = 0 "+
						"and end_hour = 23, and use full_period to control how often it runs", start, end)
			}
		}
	}

	// 3. day_of_week and day_of_month are mutually exclusive, decided by
	//    day_number_in_month.
	if scheduleType == "monthly" {
		if monthly := blockOf("monthly"); monthly != nil {
			dayNumber, _ := monthly["day_number_in_month"].(string)
			if dayNumber == "" {
				dayNumber = "fourth"
			}
			dayOfWeek, _ := monthly["day_of_week"].(string)
			dayOfMonth, _ := monthly["day_of_month"].(int)

			if dayNumber == "onDay" {
				if dayOfMonth == 0 {
					return fmt.Errorf("schedule.monthly.day_number_in_month = \"onDay\" requires day_of_month (1-31, or 32 meaning the last day of the month)")
				}
				if dayOfWeek != "" {
					return fmt.Errorf("schedule.monthly.day_number_in_month = \"onDay\" must not be combined with day_of_week")
				}
			} else {
				if dayOfMonth != 0 {
					return fmt.Errorf("schedule.monthly.day_of_month only applies when day_number_in_month = \"onDay\"; it is currently %q", dayNumber)
				}
				if dayOfWeek == "" {
					return fmt.Errorf("schedule.monthly.day_number_in_month = %q requires day_of_week", dayNumber)
				}
			}
		}
	}

	return nil
}

// dataSourceBackupVeeamJobSchema mirrors the resource field names so
// flattenJobDetail can be reused verbatim. It is written out rather than
// derived from the resource schema because computed fields must carry no
// ValidateFunc and no Default, which the resource schema does.
//
// `status` and `enabled` are deliberately absent: the detail endpoint does not
// return them. Read those from fptcloud_backup_veeam_jobs.
var dataSourceBackupVeeamJobSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC that owns the job.",
	},
	"job_id": {
		Type:         schema.TypeString,
		Optional:     true,
		Computed:     true,
		ExactlyOneOf: []string{"job_id", "name"},
		Description:  "The ID of the job to read. Set either this or `name`.",
	},
	"name": {
		Type:         schema.TypeString,
		Optional:     true,
		Computed:     true,
		ExactlyOneOf: []string{"job_id", "name"},
		Description: "The name of the job to read, matched exactly. Set either this or `job_id`. " +
			"Looking a job up by name costs one extra request, because only the list endpoint can search by name.",
	},
	"description": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"schedule_enabled": {
		Type:        schema.TypeBool,
		Computed:    true,
		Description: "Whether the job has a schedule.",
	},
	"is_capacity_tier_enabled": {
		Type:     schema.TypeBool,
		Computed: true,
	},
	"vm_ids": {
		Type:        schema.TypeSet,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "IDs of the instances this job protects.",
	},
	"vm_display_names": {
		Type:        schema.TypeSet,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "Display names of the protected instances. Instance names can change, so do not match on them.",
	},
	"notification_method_ids": {
		Type:     schema.TypeSet,
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},
	"retention": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"cycles":     {Type: schema.TypeInt, Computed: true},
				"limit_type": {Type: schema.TypeString, Computed: true},
			},
		},
	},
	"schedule": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {Type: schema.TypeString, Computed: true},
				"daily": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type":   {Type: schema.TypeString, Computed: true},
							"run_at": {Type: schema.TypeString, Computed: true},
						},
					},
				},
				"monthly": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"run_at":              {Type: schema.TypeString, Computed: true},
							"day_number_in_month": {Type: schema.TypeString, Computed: true},
							"day_of_week":         {Type: schema.TypeString, Computed: true},
							"day_of_month":        {Type: schema.TypeInt, Computed: true},
						},
					},
				},
				"period": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"full_period": {Type: schema.TypeInt, Computed: true},
							"start_hour":  {Type: schema.TypeInt, Computed: true},
							"end_hour":    {Type: schema.TypeInt, Computed: true},
						},
					},
				},
			},
		},
	},
}

// resourceBackupVeeamRestoreSchema describes a RESTORE, which is an action
// rather than a piece of infrastructure.
//
// Terraform has no first-class way to say "run this once". The convention for
// an action - the one aws_lambda_invocation and terraform_data use - is a
// resource whose arguments are all ForceNew: applying it performs the action,
// changing any argument performs it again, and destroying it only forgets. That
// is what this is. `terraform destroy` CANNOT undo a restore.
var resourceBackupVeeamRestoreSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC that owns the backup job.",
	},
	"backup_job_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description: "The ID of the backup job the restore point belongs to. Needed because the API has no endpoint that reads a " +
			"restore point by ID on its own - a point can only be read from the list belonging to one job and one instance, " +
			"which is how the provider checks the ID before it fires the restore.",
	},
	"vm_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the instance to restore. It must be the instance the restore point was taken from.",
	},
	"restore_point_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description: "The ID of the restore point to restore from. Source it from the `fptcloud_backup_veeam_restore_points` " +
			"data source. Changing it runs the restore again.",
	},
	"quick_rollback": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  false,
		ForceNew: true,
		Description: "Restore only the blocks that changed since the restore point was taken, which is faster. " +
			"Shown as \"Quick rollback\" in the portal.",
	},
	"power_on_after_restore": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		ForceNew:    true,
		Description: "Power the instance on once the restore finishes.",
	},
	"triggers": {
		Type:     schema.TypeMap,
		Optional: true,
		ForceNew: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Description: "Arbitrary values that force the restore to run again when they change. Use it to repeat a restore " +
			"from the SAME restore point, which no other argument can express because every other argument would still be equal.",
	},

	"restored_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "When the restore point being restored from was taken.",
	},
	"vm_display_name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Display name of the restored instance, read back from the API.",
	},
	"point_type": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "`full` or `incremental` - the kind of restore point that was used.",
	},
}

var dataSourceBackupVeeamRestorePointsSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC.",
	},
	"backup_job_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the backup job that produced the restore points.",
	},
	"vm_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the protected instance.",
	},
	"point_type": {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringInSlice([]string{"full", "incremental"}, false),
		Description:  "Only return restore points of this kind: `full` or `incremental`.",
	},
	"points": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id":               {Type: schema.TypeString, Computed: true},
				"restore_at":       {Type: schema.TypeString, Computed: true},
				"point_type":       {Type: schema.TypeString, Computed: true},
				"backup_file_size": {Type: schema.TypeFloat, Computed: true},
				"status":           {Type: schema.TypeString, Computed: true},
				"vm_display_name":  {Type: schema.TypeString, Computed: true},
			},
		},
		Description: "The restore points, newest first - the same order and the same columns the portal's restore dialog shows.",
	},
}

var dataSourceBackupVeeamRestoreGroupsSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC.",
	},
	"groups": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"vm_id": {
					Type:     schema.TypeString,
					Computed: true,
					Description: "The ID of the protected instance. Empty for legacy restore points the server could not map " +
						"to an instance; `fptcloud_backup_veeam_restore_points` cannot list those.",
				},
				"vm_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The instance's current name.",
				},
				"restore_vm_name": {
					Type:     schema.TypeString,
					Computed: true,
					Description: "The instance's name when it was backed up. It differs from `vm_name` once the instance " +
						"has been renamed.",
				},
				"is_deleted": {
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Whether the instance has been deleted. Its restore points are still listed.",
				},
				"backup_job_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the backup job that produced the restore points.",
				},
				"backup_job_name": {Type: schema.TypeString, Computed: true},
				"restore_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "When the most recent restore point of this group was taken.",
				},
				"restore_point_count": {Type: schema.TypeInt, Computed: true},
				"total_backup_size":   {Type: schema.TypeFloat, Computed: true},
			},
		},
		Description: "One entry per instance and backup job that has restore points, most recent first - the same table " +
			"the portal's Restore tab shows. Feed `vm_id` and `backup_job_id` into `fptcloud_backup_veeam_restore_points`. `vm_id` is empty for legacy restore points the server could not map to an instance; those cannot be listed.",
	},
}

// resourceBackupVeeamRestoreCloneSchema describes a "Restore keep": the same
// action shape as a plain restore, but the restore point comes back as a NEW
// instance and the original is left running.
//
// keep_original_vm is absent on purpose - see RestoreClonePayload.
var resourceBackupVeeamRestoreCloneSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC that owns the backup job.",
	},
	"backup_job_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description: "The ID of the backup job the restore point belongs to. Needed because the API has no endpoint that reads a " +
			"restore point by ID on its own - a point can only be read from the list belonging to one job and one instance, " +
			"which is how the provider checks the ID before it fires the restore.",
	},
	"vm_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the instance the restore point was taken from. This instance is NOT modified.",
	},
	"restore_point_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description: "The ID of the restore point to restore from. Source it from the `fptcloud_backup_veeam_restore_points` " +
			"data source. Changing it runs the restore again.",
	},
	"new_instance_name": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		StateFunc:    func(value interface{}) string { return strings.TrimSpace(value.(string)) },
		Description: "Name for the new instance the restore point is brought back as. It must not be in use by another " +
			"instance in the VPC. The server trims leading and trailing whitespace.",
	},
	"power_on_after_restore": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		ForceNew:    true,
		Description: "Power the new instance on once the restore finishes.",
	},
	"triggers": {
		Type:     schema.TypeMap,
		Optional: true,
		ForceNew: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Description: "Arbitrary values that force the restore to run again when they change. Use it to repeat a restore " +
			"from the SAME restore point, which no other argument can express because every other argument would still be equal.",
	},

	"new_instance_id": {
		Type:     schema.TypeString,
		Computed: true,
		Description: "ID of the instance that was created, looked up by name once the restore finished. Empty when the " +
			"lookup did not find it - the restore itself still succeeded, and the instance is visible in the portal.",
	},
	"restored_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "When the restore point being restored from was taken.",
	},
	"vm_display_name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Display name of the instance the restore point was taken from, read back from the API.",
	},
	"point_type": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "`full` or `incremental` - the kind of restore point that was used.",
	},
}

var resourceBackupVeeamInstantRecoverySchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC that owns the backup job.",
	},
	"backup_job_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description: "The ID of the backup job the restore point belongs to. The provider reads the restore point from the " +
			"job's list to learn which mounted session belongs to this resource.",
	},
	"vm_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the instance the restore point was taken from.",
	},
	"restore_point_id": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the restore point to mount. Source it from the `fptcloud_backup_veeam_restore_points` data source.",
	},
	"power_up": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		ForceNew:    true,
		Description: "Power the mounted instance on once the session starts.",
	},
	"nics_enabled": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  false,
		ForceNew: true,
		Description: "Connect the mounted instance's network interfaces. Leaving this off is the safer default: a mounted " +
			"instance with its network connected can collide with the original instance, which is still running.",
	},
	"vm_tags_restore_enabled": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		ForceNew:    true,
		Description: "Restore the instance's tags along with it.",
	},
	"mount_id": {
		Type:     schema.TypeString,
		Computed: true,
		Description: "ID of the session in the portal's Instant Recovery tab, once it appears there. Empty while the " +
			"platform is still mounting, and empty again once somebody migrates or stops the session.",
	},
	"state": {
		Type:     schema.TypeString,
		Computed: true,
		Description: "The session state as Veeam Backup & Replication reports it. Passed through unchanged; the provider " +
			"attaches no meaning to any particular value.",
	},
	"ready_migrate": {
		Type:     schema.TypeBool,
		Computed: true,
		Description: "Whether the platform considers the session ready to be migrated - to be kept as a normal instance. " +
			"Migrating is done in the portal; this attribute only reports the flag.",
	},
	"mount_name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Name of the mounted instance as it appears in the portal's Instant Recovery tab.",
	},
	"mounted_instance_id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "ID of the mounted instance, once the portal has a record of it.",
	},
	"mode": {
		Type:     schema.TypeString,
		Computed: true,
		Description: "Which kind of session the platform reports this as. Always `Customized`, the only kind the provider " +
			"starts.",
	},
	"restore_point_time": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "When the restore point being mounted was taken.",
	},
	"backup_id": {
		Type:     schema.TypeString,
		Computed: true,
		Description: "ID of the backup the session was mounted from, as the session itself reports it. Note that this is NOT the " +
			"`backup_id` of the restore point the session was started from - the two differ.",
	},
	"history_id": {
		Type:     schema.TypeString,
		Computed: true,
		Description: "ID of the history entry the API created for this session. Useful for matching the session against the " +
			"portal's History tab; there is no endpoint that reads it back.",
	},
	"new_instance_name": {
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.NoZeroValues,
		StateFunc:    func(value interface{}) string { return strings.TrimSpace(value.(string)) },
		Description: "Name for the instance the session mounts as. It must not be in use by another instance in the VPC. " +
			"The provider also uses this name to recognise its own session in the VPC-wide session list.",
	},
}

var dataSourceBackupVeeamInstantRecoverySessionsSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The ID of the VPC.",
	},
	"sessions": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"vm_mount_id":        {Type: schema.TypeString, Computed: true},
				"vm_mount_name":      {Type: schema.TypeString, Computed: true},
				"recovered_vm_name":  {Type: schema.TypeString, Computed: true},
				"vm_id":              {Type: schema.TypeString, Computed: true},
				"state":              {Type: schema.TypeString, Computed: true},
				"mode":               {Type: schema.TypeString, Computed: true},
				"restore_point_time": {Type: schema.TypeString, Computed: true},
				"backup_id":          {Type: schema.TypeString, Computed: true},
				"ready_migrate":      {Type: schema.TypeBool, Computed: true},
			},
		},
		Description: "Every instant recovery session open in the VPC, including sessions started from the portal.",
	},
}
