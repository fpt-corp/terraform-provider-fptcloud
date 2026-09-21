package fptcloud_backup_veeam

import (
	"context"
	"strings"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceBackupVeeamRestoreClone() *schema.Resource {
	return &schema.Resource{
		Description: "Restores an instance from a backup restore point into a NEW instance, leaving the original one " +
			"running - the same operation as \"Restore keep\" in the portal. This is an ACTION, not a piece of infrastructure: " +
			"`terraform destroy` does NOT remove the instance it created, it only stops Terraform tracking the restore.",
		CreateContext: createBackupVeeamRestoreClone,
		ReadContext:   readBackupVeeamRestoreClone,
		DeleteContext: deleteBackupVeeamRestoreClone,
		Schema:        resourceBackupVeeamRestoreCloneSchema,
		// No UpdateContext and no Importer, for the same reasons as
		// fptcloud_backup_veeam_restore: every argument is ForceNew, and a
		// restore that already happened leaves nothing to adopt.
		// No Timeouts block: nothing here waits - see resource_restore.go.
	}
}

func createBackupVeeamRestoreClone(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)

	vpcId := d.Get("vpc_id").(string)
	jobId := d.Get("backup_job_id").(string)
	vmId := d.Get("vm_id").(string)
	pointId := d.Get("restore_point_id").(string)
	newName := strings.TrimSpace(d.Get("new_instance_name").(string))

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
	// No "already running" refusal - see the note in resource_restore.go: every
	// signal available for it gets stuck, and a stuck signal would lock the
	// user out of restoring that instance ever again.

	// The API checks the name too, but only after it has already talked to
	// Veeam. Checking here turns a late failure into a plan-time-ish one.
	taken, err := instanceNameTaken(service, vpcId, newName)
	if err != nil {
		return diag.FromErr(err)
	}
	if taken {
		return diag.Errorf(
			"an instance named %q already exists in VPC %s; pick another new_instance_name", newName, vpcId)
	}

	payload := RestoreClonePayload{
		RestoreVmPointId:    pointId,
		NewVmName:           newName,
		PowerOnAfterRestore: d.Get("power_on_after_restore").(bool),
		// Always true: this endpoint is only safe with the original kept. See
		// RestoreClonePayload.
		KeepOriginalVm: true,
	}

	if _, err := service.RestoreClone(vpcId, payload); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(pointId + "/" + newName)

	// Apply ends here, with the request accepted. It does not wait for the
	// restore to finish - see the note in resource_restore.go - which also
	// means the new instance does not exist yet, so new_instance_id stays
	// empty until a later refresh finds it by name.
	return readBackupVeeamRestoreClone(ctx, d, m)
}

// readBackupVeeamRestoreClone reports the restore point this restore ran from.
//
// Like the plain restore, it never clears the id when the point is gone:
// recreating would run the restore a second time, and a deleted restore point
// does not make the past restore untrue.
func readBackupVeeamRestoreClone(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)

	point, err := service.GetRestorePoint(
		d.Get("vpc_id").(string),
		d.Get("backup_job_id").(string),
		d.Get("vm_id").(string),
		d.Get("restore_point_id").(string),
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

	vpcId := d.Get("vpc_id").(string)

	// The response of the restore carries the id of the ORIGINAL instance, not
	// the new one, so the new instance can only be found by name - and only
	// once the platform has actually created it. Until then this stays empty,
	// which is not an error.
	newId, err := findInstanceIdByName(service, vpcId, d.Get("new_instance_name").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	if newId != "" {
		if err := d.Set("new_instance_id", newId); err != nil {
			return diag.Errorf("could not write new_instance_id to state: %v", err)
		}
	}
	return nil
}

// deleteBackupVeeamRestoreClone forgets the restore. It deliberately does not
// delete the instance that was created: Terraform did not create it as a
// managed instance, and deleting an instance nobody asked to delete is far
// worse than leaving one behind.
func deleteBackupVeeamRestoreClone(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	name := d.Get("new_instance_name").(string)
	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "The restored instance was kept",
		Detail: "Terraform has stopped tracking this restore. The instance it created (\"" + name + "\") still exists " +
			"and still costs resources. Delete it in the portal, or manage it with a fptcloud_instance resource.",
	}}
}

// instanceNameTaken reports whether an instance of this name already exists in
// the VPC. The comparison is case-insensitive on purpose: two instances whose
// names differ only in case are the kind of thing that gets noticed much later.
func instanceNameTaken(service BackupVeeamService, vpcId string, name string) (bool, error) {
	id, err := findInstanceIdByName(service, vpcId, name)
	return id != "", err
}

func findInstanceIdByName(service BackupVeeamService, vpcId string, name string) (string, error) {
	// unprotected_only is off: the instance being looked for is brand new and
	// unprotected, but so is every other name that could collide with it.
	list, err := service.ListInstances(vpcId, false, "", "")
	if err != nil {
		return "", err
	}
	wanted := strings.TrimSpace(name)
	for _, item := range list.Data {
		if strings.EqualFold(strings.TrimSpace(item.Name), wanted) {
			return item.Id, nil
		}
	}
	return "", nil
}
