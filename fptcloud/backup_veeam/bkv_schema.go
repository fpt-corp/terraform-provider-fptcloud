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
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "IDs of notification methods. Always source these from the `fptcloud_alert_notification_methods` data source: the API silently ignores IDs it does not recognise, so a hand-typed ID leaves the job with no notifications and no error.",
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
	"not_backup": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "When true, only return instances that do not belong to an active backup job yet.",
	},
	"job_id": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Keep the instances that belong to this job in the result. Use it when editing an existing job.",
	},
	"status": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Only return instances with this status.",
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
