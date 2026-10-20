package fptcloud_backup_native

import (
	"context"
	"sort"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// newestFirst returns the indexes 0..n-1 ordered by created_at, newest first.
// Values that do not parse sort last.
func newestFirst(createdAt func(i int) string, n int) []int {
	order := make([]int, n)
	parsed := make([]int64, n)
	for i := 0; i < n; i++ {
		order[i] = i
		if t, err := parseAPITime(createdAt(i)); err == nil {
			parsed[i] = t.UnixNano()
		} else {
			parsed[i] = -1 << 62
		}
	}
	sort.SliceStable(order, func(a, b int) bool { return parsed[order[a]] > parsed[order[b]] })
	return order
}

func DataSourceVmRestorePoints() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the restore points of a VM, newest first.",
		ReadContext: readVmRestorePointsDataSource,
		Schema: map[string]*schema.Schema{
			"vpc_id": {Type: schema.TypeString, Required: true, Description: "ID of the VPC."},
			"vm_id":  {Type: schema.TypeString, Required: true, Description: "ID of the VM."},
			"restore_points": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Restore points, newest first.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"created_at": {Type: schema.TypeString, Computed: true,
						Description: "Creation time in the API's form, naive UTC without a suffix. Pass it unchanged to `restore_point_created_at` of `fptcloud_backup_native_vm_restore`."},
					"size":            {Type: schema.TypeFloat, Computed: true, Description: "Size in GB."},
					"is_incremental":  {Type: schema.TypeBool, Computed: true, Description: "Whether this is an incremental backup."},
					"is_available":    {Type: schema.TypeBool, Computed: true, Description: "Whether the restore point can be restored."},
					"backup_job_id":   {Type: schema.TypeString, Computed: true, Description: "ID of the job that created it."},
					"backup_job_name": {Type: schema.TypeString, Computed: true, Description: "Name of the job that created it."},
				}},
			},
		},
	}
}

func readVmRestorePointsDataSource(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcId := d.Get("vpc_id").(string)
	vmId := d.Get("vm_id").(string)
	points, err := NewVmService(m.(*common.Client)).ListVmRestorePoints(vpcId, vmId)
	if err != nil {
		return diag.FromErr(err)
	}
	items := make([]interface{}, 0, len(points))
	for _, i := range newestFirst(func(i int) string { return points[i].CreatedAt }, len(points)) {
		p := points[i]
		items = append(items, map[string]interface{}{
			"created_at":      p.CreatedAt,
			"size":            p.Size,
			"is_incremental":  p.IsIncremental,
			"is_available":    p.IsAvailable,
			"backup_job_id":   p.BackupJobId,
			"backup_job_name": p.BackupJobName,
		})
	}
	d.SetId(vpcId + "/" + vmId)
	return setValues(d, map[string]interface{}{"restore_points": items})
}
