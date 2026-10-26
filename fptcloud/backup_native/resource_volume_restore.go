package fptcloud_backup_native

import (
	"context"
	"fmt"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceVolumeRestore() *schema.Resource {
	return &schema.Resource{
		Description: "Restores a Backup Native volume restore point into a new volume. The API has no in-place restore for volumes. " +
			"Applying the resource runs the restore once and waits for it to finish; destroying it only removes it from state. " +
			"Change a value in `triggers` to run the restore again. " +
			"The API does not return the new volume's id; find it by `name` with the `fptcloud_storages` data source. " +
			"Run one restore per volume at a time; two concurrent restores of the same volume can be matched to the wrong history row.",
		CreateContext: createVolumeRestore,
		ReadContext:   readOneShot,
		DeleteContext: deleteOneShot,
		Schema:        resourceVolumeRestoreSchema(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
	}
}

func resourceVolumeRestoreSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id":           {Type: schema.TypeString, Required: true, ForceNew: true, Description: "ID of the VPC."},
		"volume_id":        {Type: schema.TypeString, Required: true, ForceNew: true, Description: "ID of the volume the restore point belongs to."},
		"restore_point_id": {Type: schema.TypeString, Required: true, ForceNew: true, Description: "ID of the restore point, from `fptcloud_backup_native_volume_restore_points`."},
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			ValidateFunc: validation.All(
				validation.StringLenBetween(1, 50),
				validation.StringMatch(restoreNamePattern, "must contain only letters, digits and hyphens"),
			),
			Description: "Name of the new volume, at most 50 characters: letters, digits and hyphens.",
		},
		"size_gb": {
			Type:         schema.TypeInt,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.IntAtLeast(1),
			Description:  "Size of the new volume in GB. The API requires at least the size of the source volume.",
		},
		"triggers":    triggersSchema("run the restore again"),
		"status":      {Type: schema.TypeString, Computed: true, Description: "Result of the restore, `success` after a successful apply."},
		"started_at":  {Type: schema.TypeString, Computed: true, Description: "Start of the restore, RFC 3339 UTC."},
		"finished_at": {Type: schema.TypeString, Computed: true, Description: "End of the restore, RFC 3339 UTC."},
	}
}

func createVolumeRestore(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	volumeId := d.Get("volume_id").(string)
	restorePointId := d.Get("restore_point_id").(string)
	submitted := time.Now().UTC()

	// Rows already in the history belong to earlier restores of this volume.
	seen, err := volumeRestoreRowsSeen(svc, vpcId, volumeId, submitted)
	if err != nil {
		return diag.FromErr(err)
	}
	err = svc.RestoreVolume(vpcId, volumeId, VolumeRestorePayload{
		BackupId: restorePointId,
		Metadata: VolumeRestoreMetadata{Name: d.Get("name").(string), Size: d.Get("size_gb").(int)},
	})
	if err != nil {
		return diag.FromErr(err)
	}

	row, err := waitVolumeRestore(ctx, svc, vpcId, volumeId, submitted, seen, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%s/%s/%d", volumeId, restorePointId, submitted.Unix()))
	return setValues(d, map[string]interface{}{
		"status":      row.Status,
		"started_at":  formatAPITime(row.StartTime),
		"finished_at": formatAPITime(row.EndTime),
	})
}
