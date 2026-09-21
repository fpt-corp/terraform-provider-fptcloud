package fptcloud_backup_veeam

import (
	"context"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceBackupVeeamRestore() *schema.Resource {
	return &schema.Resource{
		Description: "Restores an instance from a backup restore point, the same operation as \"Restore Instance\" in the portal. " +
			"This is an ACTION, not a piece of infrastructure: applying it overwrites the running instance, and " +
			"`terraform destroy` does NOT undo it - it only stops Terraform tracking the restore.",
		CreateContext: createBackupVeeamRestore,
		ReadContext:   readBackupVeeamRestore,
		DeleteContext: deleteBackupVeeamRestore,
		Schema:        resourceBackupVeeamRestoreSchema,
		// No UpdateContext: every argument is ForceNew, so any change runs the
		// restore again rather than editing something that already happened.
		// No Timeouts block: nothing here waits. Apply ends when the platform
		// has accepted the request, so the only budget that matters is the
		// provider's HTTP timeout.
		//
		// No Importer: there is nothing to adopt. A restore that happened in the
		// portal left no object behind for Terraform to manage.
	}
}

func createBackupVeeamRestore(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)

	vpcId := d.Get("vpc_id").(string)
	jobId := d.Get("backup_job_id").(string)
	vmId := d.Get("vm_id").(string)
	pointId := d.Get("restore_point_id").(string)

	// Check the restore point exists before firing the restore. The API answers
	// a missing point with {"status": false, "message": "Not Found"}, which does
	// not say WHICH of the three ids was wrong.
	point, err := service.GetRestorePoint(vpcId, jobId, vmId, pointId)
	if err != nil {
		return diag.FromErr(err)
	}
	if point == nil {
		return diag.Errorf(
			"restore point %s does not belong to instance %s in backup job %s. "+
				"List the available points with the fptcloud_backup_veeam_restore_points data source",
			pointId, vmId, jobId)
	}
	// No "is a restore already running" check is made here, on purpose. Both
	// signals that could carry it get stuck: the restore point's status is set
	// to PENDING on request and never set back, and a history row can sit in
	// WORKING indefinitely if the Veeam session sync loses track of it. A guard
	// built on either would refuse every later restore of that instance -
	// blocking the user out of their own data. Overlapping restores are the
	// backend's business: the standard restore workflow serialises per VPC.

	payload := RestorePayload{
		RestoreVmPointId:    pointId,
		PowerOnAfterRestore: d.Get("power_on_after_restore").(bool),
		QuickRollback:       d.Get("quick_rollback").(bool),
	}

	if _, err := service.Restore(vpcId, payload); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(pointId)

	// Apply ends here, with the request accepted by the platform. It does NOT
	// wait for the restore to finish, and it does not report the outcome
	// either: no signal the API offers for that is dependable. The restore
	// point's status is set to PENDING and never set back, and the only place
	// the outcome is recorded - the History tab - is a list with no id per
	// entry, so an entry can only be matched by instance name, type and
	// timestamp. The provider does not guess: progress and outcome belong in
	// the portal's History tab.
	return readBackupVeeamRestore(ctx, d, m)
}

// readBackupVeeamRestore reports the restore point this restore ran from.
//
// It deliberately does NOT clear the id when the point is gone. For an ordinary
// resource a missing object means drift and Terraform should recreate it; here
// recreating would restore the instance a second time, unasked. A restore point
// that has since been deleted does not make the past restore untrue.
func readBackupVeeamRestore(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)

	point, err := service.GetRestorePoint(
		d.Get("vpc_id").(string),
		d.Get("backup_job_id").(string),
		d.Get("vm_id").(string),
		d.Id(),
	)
	if err != nil {
		return diag.FromErr(err)
	}
	if point == nil {
		return nil
	}

	setters := map[string]interface{}{
		"restored_at":     point.RestoreAt,
		"vm_display_name": point.VmDisplayName,
		"point_type":      point.PointType,
	}
	for key, value := range setters {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("could not write %s to state: %v", key, err)
		}
	}

	return nil
}

// deleteBackupVeeamRestore forgets the restore. It calls nothing: a restore
// cannot be rolled back, and the restore point itself belongs to the backup
// job, not to this resource.
func deleteBackupVeeamRestore(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "A restore cannot be undone",
		Detail: "Terraform has stopped tracking this restore, but the instance keeps the data it was restored to. " +
			"To go back to an earlier state, restore from a different restore point.",
	}}
}
