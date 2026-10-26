package fptcloud_backup_native

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// purgeStatusSuccess is the status a successful purge stores, in the case
// the v1 history uses.
const purgeStatusSuccess = "Success"

func ResourceVmRestorePointsPurge() *schema.Resource {
	return &schema.Resource{
		Description: "Deletes every Backup Native restore point of a VM. " +
			"It deletes the restore points taken under every job, past and current, and frees their backup quota; the VM's next backup is a full backup. " +
			"Applying the resource waits while a backup or restore of the VM runs, starts the deletion and waits for it to end. " +
			"Destroying the resource only removes it from state; deleted restore points cannot be recovered. " +
			"Change a value in `triggers` to delete the restore points again. " +
			"Run one purge per VM at a time; two concurrent purges of the same VM can be matched to the wrong history row.",
		CreateContext: createVmRestorePointsPurge,
		ReadContext:   readOneShot,
		DeleteContext: deleteOneShot,
		Schema:        resourceVmRestorePointsPurgeSchema(),
		// No UpdateContext and no Importer, as in the job run resources.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
	}
}

func resourceVmRestorePointsPurgeSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "ID of the VPC.",
		},
		"vm_id": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "ID of the VM whose restore points are deleted. The VM does not need to be in a backup job.",
		},
		"triggers": triggersSchema("delete the restore points again"),
		"restore_point_count": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Number of restore points the VM had when the purge started. `0` means there was nothing to delete and no request was sent.",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Result of the purge, `Success` after a successful apply.",
		},
		"started_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "When the provider started the purge, RFC 3339 UTC.",
		},
		"finished_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "When the provider saw the purge end, RFC 3339 UTC.",
		},
	}
}

func listVmRestorePoints(ctx context.Context, svc VmService, vpcId, vmId string) ([]VmRestorePoint, error) {
	var points []VmRestorePoint
	err := retryRead(ctx, func() error {
		var err error
		if points, err = svc.ListVmRestorePoints(vpcId, vmId); err != nil {
			return readFailed(err)
		}
		return nil
	})
	return points, err
}

func createVmRestorePointsPurge(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	svc := NewVmService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	vmId := d.Get("vm_id").(string)
	deadline := time.Now().Add(d.Timeout(schema.TimeoutCreate))
	started := time.Now().UTC()

	points, err := listVmRestorePoints(ctx, svc, vpcId, vmId)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(points) > 0 {
		// Synced even when the purge fails: it may have deleted some already.
		defer syncVmJobs(svc, vpcId, &diags)
		if err := purgeVmRestorePoints(ctx, svc, vpcId, vmId, deadline); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(fmt.Sprintf("%s/%d", vmId, started.Unix()))
	diags = setValues(d, map[string]interface{}{
		"restore_point_count": len(points),
		"status":              purgeStatusSuccess,
		"started_at":          started.Format(time.RFC3339),
		"finished_at":         time.Now().UTC().Format(time.RFC3339),
	})
	if diags.HasError() || len(points) == 0 {
		return diags
	}

	// The worker skips restore points an earlier purge is still deleting, and
	// a backup can finish while the purge runs.
	left, err := listVmRestorePoints(ctx, svc, vpcId, vmId)
	if err != nil {
		return append(diags, diag.Diagnostic{Severity: diag.Warning, Summary: fmt.Sprintf("could not check whether restore points of VM %s remain: %v", vmId, err)})
	}
	if len(left) > 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("%d restore points of VM %s remain after the purge", len(left), vmId),
			Detail: "They are restore points an earlier purge is still deleting, or ones a backup created while this purge ran. " +
				"Change a value in triggers to delete them.",
		})
	}
	return diags
}

// purgeVmRestorePoints submits the delete-all request, retrying while the API
// refuses it as busy, and waits until the purge ends.
func purgeVmRestorePoints(ctx context.Context, svc VmService, vpcId, vmId string, deadline time.Time) error {
	submitted := time.Now().UTC()
	// Rows already in the history belong to earlier purges of this VM.
	seen, err := vmHistoryRowsSeen(svc, vpcId, vmId, historyTypePurge, submitted)
	if err != nil {
		return err
	}

	var refusal error
	submit := newStateConf([]string{"busy"}, []string{"accepted"}, func() (interface{}, string, error) {
		err := svc.DeleteAllVmRestorePoints(vpcId, vmId)
		if errors.As(err, &restorePointsBusy{}) {
			refusal = err
			return struct{}{}, "busy", nil
		}
		if err != nil {
			return nil, "", err
		}
		return struct{}{}, "accepted", nil
	}, time.Until(deadline))
	// The first request goes out at once; Delay only spaces the retries.
	submit.Delay = 0
	if _, err := submit.WaitForStateContext(ctx); err != nil {
		if refusal != nil {
			return fmt.Errorf("waiting for the API to accept deleting the restore points of VM %s: %w; last refusal: %v", vmId, err, refusal)
		}
		return err
	}

	wait := newStateConf([]string{"deleting"}, []string{"done"}, func() (interface{}, string, error) {
		rows, err := svc.ListVmHistories(vpcId, historyTypePurge, submitted.Add(-historyClockSkew))
		if err != nil {
			return nil, "", readFailed(err)
		}
		if row := findVmHistoryRow(rows, vmId, historyTypePurge, submitted, seen); row != nil {
			switch strings.ToLower(row.Status) {
			case historyStatusSuccess:
				return row, "done", nil
			case historyStatusFailed:
				return nil, "", fmt.Errorf("deleting the restore points of VM %s failed: %s. Some restore points may already be deleted; apply again to delete the rest",
					vmId, row.Reason)
			}
		}
		// No row appears when the worker finds no restore point on the
		// platform, so an empty list also ends the wait.
		points, err := svc.ListVmRestorePoints(vpcId, vmId)
		if err != nil {
			return nil, "", readFailed(err)
		}
		if len(points) == 0 {
			return struct{}{}, "done", nil
		}
		return struct{}{}, "deleting", nil
	}, time.Until(deadline))
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for the restore points of VM %s to be deleted: %w", vmId, err)
	}
	return nil
}
