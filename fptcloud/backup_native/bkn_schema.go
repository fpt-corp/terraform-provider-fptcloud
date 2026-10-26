package fptcloud_backup_native

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var jobNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)

func jobNameSchema(kind string) *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
		ValidateFunc: validation.All(
			validation.StringLenBetween(2, 26),
			validation.StringMatch(jobNamePattern, "must start with a lowercase letter and contain only lowercase letters, digits and hyphens"),
		),
		Description: fmt.Sprintf("Name of the %s backup job, unique in the VPC. 2-26 characters: lowercase letters, digits and hyphens, starting with a letter.", kind),
	}
}

func jobDescriptionSchema() *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringLenBetween(0, 100),
		Description:  "Description, at most 100 characters.",
	}
}

// triggersSchema is the triggers argument of the one-shot resources. action
// says what a changed value does, for example "run the backup again".
func triggersSchema(action string) *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		ForceNew: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Description: fmt.Sprintf("Values that %s when they change, "+
			"which no other argument can express because every other argument stays equal. "+
			"Any string works; only a change matters. "+
			"We recommend the time you ask for the action, in the form `YYYY-MM-DD HH:mm:ss`, for example `2026-10-26 09:00:00`, "+
			"so the state records when each one was requested. "+
			"Avoid `timestamp()`: it changes on every plan and repeats the action on every apply.", action),
	}
}

// importJob parses vpc/<vpc_id>/<kind>/<job_id>.
func importJob(kind string) schema.StateContextFunc {
	return func(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
		parts := strings.Split(d.Id(), "/")
		if len(parts) != 4 || parts[0] != "vpc" || parts[2] != kind || parts[1] == "" || parts[3] == "" {
			return nil, fmt.Errorf("malformed import id %q, expected vpc/<vpc_id>/%s/<job_id>", d.Id(), kind)
		}
		if err := d.Set("vpc_id", parts[1]); err != nil {
			return nil, err
		}
		d.SetId(parts[3])
		return []*schema.ResourceData{d}, nil
	}
}

func setToStrings(raw interface{}) []string {
	set, ok := raw.(*schema.Set)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, set.Len())
	for _, v := range set.List() {
		out = append(out, v.(string))
	}
	sort.Strings(out)
	return out
}

func setValues(d *schema.ResourceData, values map[string]interface{}) diag.Diagnostics {
	for key, value := range values {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("setting %s: %v", key, err)
		}
	}
	return nil
}
