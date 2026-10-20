package fptcloud_backup_native

import (
	"context"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceVmJobs() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the VM backup jobs of a VPC.",
		ReadContext: readVmJobsDataSource,
		Schema: map[string]*schema.Schema{
			"vpc_id": {Type: schema.TypeString, Required: true, Description: "ID of the VPC."},
			"name":   {Type: schema.TypeString, Optional: true, Description: "Return only the job with this exact name."},
			"jobs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The VM backup jobs.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"id":               {Type: schema.TypeString, Computed: true, Description: "ID of the job."},
					"name":             {Type: schema.TypeString, Computed: true, Description: "Name of the job."},
					"status":           {Type: schema.TypeString, Computed: true, Description: "Status of the job. A number is the progress percentage of a running backup."},
					"schedule_enabled": {Type: schema.TypeBool, Computed: true, Description: "Whether the schedule runs."},
					"next_run":         {Type: schema.TypeString, Computed: true, Description: "Next scheduled run, RFC 3339 UTC."},
					"instance_ids":     {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "IDs of the instances in the job."},
				}},
			},
		},
	}
}

func readVmJobsDataSource(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcId := d.Get("vpc_id").(string)
	name := d.Get("name").(string)
	jobs, err := NewVmService(m.(*common.Client)).ListVmJobs(vpcId)
	if err != nil {
		return diag.FromErr(err)
	}
	items := make([]interface{}, 0, len(jobs))
	for _, job := range jobs {
		if name != "" && job.Name != name {
			continue
		}
		instanceIds := make([]string, 0, len(job.Instances))
		for _, instance := range job.Instances {
			instanceIds = append(instanceIds, instance.Id)
		}
		items = append(items, map[string]interface{}{
			"id":               job.Id,
			"name":             job.Name,
			"status":           job.Status,
			"schedule_enabled": job.IsEnabled,
			"next_run":         formatAPITime(job.NextRun),
			"instance_ids":     instanceIds,
		})
	}
	d.SetId(vpcId)
	return setValues(d, map[string]interface{}{"jobs": items})
}
