package fptcloud_backup_veeam

import (
	"context"
	"strings"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Instant recovery is modelled as an ACTION, the same shape as the two restore
// resources: applying it asks the platform to mount the backup, and that is
// where Terraform's involvement ends.
//
// It deliberately does NOT manage the session that the mount creates. No
// waiting for the mount to appear, no migrate, and no unmount on destroy -
// those live in the portal. The reason is the backend, not taste: the mount
// list is the only record of a session, it carries no restore point id, and
// every "is it done yet" signal available (the restore point status, the
// history row, ready_migrate) can stick indefinitely. A resource that promised
// a lifecycle on top of that would fail applies and strand state for reasons
// the user cannot act on.
//
// The cost is real and is documented loudly: a session left open blocks the
// backup job of the instance it was mounted from. Use the
// fptcloud_backup_veeam_instant_recovery_sessions data source to find sessions
// nobody has dealt with.
//
// Only the "Customized" mode is offered - mount the backup as a NEW instance -
// because that is the only one the product actually ships. The API also has an
// "OriginalLocation" mode on restores/instant-recovery/{point}, but the portal
// hides the radio button that would select it (display: none in
// CreateInstantRecoveryForm.jsx), so every session a customer has ever started
// is a Customized one. Mounting a backup on top of a running instance's own
// identity is not a path worth being the first to exercise.
func ResourceBackupVeeamInstantRecovery() *schema.Resource {
	return &schema.Resource{
		Description: "Asks the platform to start an Instant Recovery session: the backup is mounted as a NEW instance " +
			"that runs directly from it, leaving the original instance alone. This is an ACTION, not a piece of " +
			"infrastructure. Terraform does not manage the session it creates: `terraform destroy` only stops tracking " +
			"the request, and an open session BLOCKS the backup job of the instance it was mounted from until somebody " +
			"migrates or stops it in the portal.",
		CreateContext: createInstantRecovery,
		ReadContext:   readInstantRecovery,
		DeleteContext: deleteInstantRecovery,
		Schema:        resourceBackupVeeamInstantRecoverySchema,
		// No UpdateContext and no Importer: every argument is ForceNew, so any
		// change asks for a new session, and a request that already went out
		// leaves nothing to adopt.
	}
}

func createInstantRecovery(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewBackupVeeamService(client)

	vpcId := d.Get("vpc_id").(string)
	jobId := d.Get("backup_job_id").(string)
	vmId := d.Get("vm_id").(string)
	pointId := d.Get("restore_point_id").(string)
	mountName := strings.TrimSpace(d.Get("new_instance_name").(string))

	// Check the restore point before firing: the API answers a missing point
	// with {"status": false, "message": "Not Found"}, which does not say which
	// of the three ids was wrong.
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

	taken, err := instanceNameTaken(service, vpcId, mountName)
	if err != nil {
		return diag.FromErr(err)
	}
	if taken {
		return diag.Errorf(
			"an instance named %q already exists in VPC %s; pick another new_instance_name", mountName, vpcId)
	}

	payload := InstantRecoveryPayload{
		RestorePointId:       pointId,
		Type:                 InstantRecoveryTypeCustomized,
		PowerUp:              d.Get("power_up").(bool),
		NicsEnabled:          d.Get("nics_enabled").(bool),
		VmTagsRestoreEnabled: d.Get("vm_tags_restore_enabled").(bool),
		Destination:          &InstantRecoveryDestination{RestoredVmName: mountName},
	}

	response, err := service.StartInstantRecovery(vpcId, pointId, payload)
	if err != nil {
		return diag.FromErr(err)
	}

	// The name is part of the id so that two sessions started from the same
	// restore point do not collide.
	d.SetId(pointId + "/" + mountName)
	if err := d.Set("history_id", response.HistoryId); err != nil {
		return diag.Errorf("could not write history_id to state: %v", err)
	}

	return readInstantRecoveryWith(service, d)
}

func readInstantRecovery(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return readInstantRecoveryWith(NewBackupVeeamService(m.(*common.Client)), d)
}

// readInstantRecoveryWith reports the session the request created, if it can
// still be found.
//
// Not finding it is NOT drift: the session may not have mounted yet, or it may
// have been migrated or stopped in the portal - all of which are outcomes of an
// action that already happened. Clearing the id for any of them would make
// Terraform mount the backup a second time, unasked.
func readInstantRecoveryWith(service BackupVeeamService, d *schema.ResourceData) diag.Diagnostics {
	vpcId := d.Get("vpc_id").(string)
	mountName := strings.TrimSpace(d.Get("new_instance_name").(string))
	if mountName == "" {
		// Nothing left to recognise the session by. Keep what state already has.
		return nil
	}

	mount, err := service.FindMountForRestorePoint(vpcId, mountName, "", "")
	if err != nil {
		// Several sessions that cannot be told apart is a reason to report
		// nothing, not to fail a refresh of an action that already ran.
		return diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  "Could not identify the instant recovery session",
			Detail:   err.Error(),
		}}
	}
	if mount == nil {
		return nil
	}

	setters := map[string]interface{}{
		"mount_id":            mount.VmMountId,
		"mount_name":          mount.VmMountName,
		"mounted_instance_id": mount.VmId,
		"state":               mount.State,
		"ready_migrate":       mount.ReadyMigrate,
		"mode":                mount.Mode,
		"restore_point_time":  mount.RestorePointTime,
		"backup_id":           mount.BackupId,
	}
	for key, value := range setters {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("could not write %s to state: %v", key, err)
		}
	}
	return nil
}

// deleteInstantRecovery forgets the request. It calls nothing: the session is
// not Terraform's to end, and unmounting one silently on destroy would throw
// away whatever was written to the mounted instance.
func deleteInstantRecovery(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	mountName := d.Get("mount_name").(string)
	if mountName == "" {
		mountName = strings.TrimSpace(d.Get("new_instance_name").(string))
	}
	subject := "The instant recovery session"
	if mountName != "" {
		subject = "The instant recovery session \"" + mountName + "\""
	}

	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "The instant recovery session was left open",
		Detail: subject + " is not stopped by destroying this resource. While it stays open, the backup job of the " +
			"instance it was mounted from cannot run. Migrate it or stop it in the portal's Instant Recovery tab; the " +
			"fptcloud_backup_veeam_instant_recovery_sessions data source lists the sessions still open in the VPC.",
	}}
}
