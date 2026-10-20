package fptcloud_backup_native

import (
	"context"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceVolumeRestorePoints() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the restore points of a volume, newest first. Restore points of a deleted volume backup job are listed too.",
		ReadContext: readVolumeRestorePointsDataSource,
		Schema: map[string]*schema.Schema{
			"vpc_id":    {Type: schema.TypeString, Required: true, Description: "ID of the VPC."},
			"volume_id": {Type: schema.TypeString, Required: true, Description: "ID of the volume."},
			"restore_points": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Restore points, newest first.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"id":             {Type: schema.TypeString, Computed: true, Description: "ID of the restore point. Pass it to `restore_point_id` of `fptcloud_backup_native_volume_restore`."},
					"created_at":     {Type: schema.TypeString, Computed: true, Description: "Creation time, RFC 3339 UTC."},
					"status":         {Type: schema.TypeString, Computed: true, Description: "Status of the restore point."},
					"is_available":   {Type: schema.TypeBool, Computed: true, Description: "Whether the restore point can be restored."},
					"is_incremental": {Type: schema.TypeBool, Computed: true, Description: "Whether this is an incremental backup."},
					"size":           {Type: schema.TypeFloat, Computed: true, Description: "Size in GB."},
				}},
			},
		},
	}
}

func readVolumeRestorePointsDataSource(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcId := d.Get("vpc_id").(string)
	volumeId := d.Get("volume_id").(string)
	points, err := NewVolumeService(m.(*common.Client)).ListVolumeRestorePoints(vpcId, volumeId)
	if err != nil {
		return diag.FromErr(err)
	}
	items := make([]interface{}, 0, len(points))
	for _, i := range newestFirst(func(i int) string { return points[i].CreatedAt }, len(points)) {
		p := points[i]
		items = append(items, map[string]interface{}{
			"id":             p.Id,
			"created_at":     formatAPITime(p.CreatedAt),
			"status":         p.Status,
			"is_available":   p.IsAvailable,
			"is_incremental": p.IsIncremental,
			"size":           p.Size,
		})
	}
	d.SetId(vpcId + "/" + volumeId)
	return setValues(d, map[string]interface{}{"restore_points": items})
}
