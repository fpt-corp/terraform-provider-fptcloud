package fptcloud_backup_veeam

import (
	"context"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBackupVeeamInstances() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the instances that can be assigned to a Backup Veeam job.",
		ReadContext: readBackupVeeamInstances,
		Schema:      dataSourceBackupVeeamInstancesSchema,
	}
}

func readBackupVeeamInstances(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)
	vpcId := d.Get("vpc_id").(string)

	response, err := service.ListInstances(
		vpcId,
		d.Get("not_backup").(bool),
		d.Get("job_id").(string),
		d.Get("status").(string),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	instances := make([]interface{}, 0, len(response.Data))
	for _, item := range response.Data {
		instances = append(instances, map[string]interface{}{
			"id":     item.Id,
			"name":   item.Name,
			"status": item.Status,
		})
	}

	if err := d.Set("instances", instances); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(vpcId)
	return nil
}
