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

func ResourceVolumeRestorePointsPurge() *schema.Resource {
	return &schema.Resource{
		Description: "Deletes every Backup Native restore point of a volume. " +
			"It deletes the restore points taken under every job, past and current, and frees their backup quota; the volume's next backup is a full backup. " +
			"Applying the resource waits while a backup or restore of the volume runs, starts the deletion and waits for it to end. " +
			"Destroying the resource only removes it from state; deleted restore points cannot be recovered. " +
			"Change a value in `triggers` to delete the restore points again. " +
			"Run one purge per volume at a time; two concurrent purges of the same volume can be matched to the wrong history row. " +
			"Once the API has accepted the purge, the resource is saved whatever the outcome, and the next apply does not purge again. " +
			"A failed purge, a timeout or an interrupted apply (Ctrl+C) ends the apply with a warning, not an error, and stores the outcome in `status` and `message`. " +
			"The purge goes on without Terraform after a timeout or an interrupt; each later plan or apply checks the purge history and the restore points, " +
			"and stores the outcome once the purge ends. " +
			"Only interrupt an apply once: a second Ctrl+C stops Terraform before it saves the purge, and the next apply purges again.",
		CreateContext: createVolumeRestorePointsPurge,
		ReadContext:   readVolumeRestorePointsPurge,
		DeleteContext: deleteOneShot,
		Schema:        resourceVolumeRestorePointsPurgeSchema(),
		// No UpdateContext and no Importer, as in the job run resources.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
	}
}

func resourceVolumeRestorePointsPurgeSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "ID of the VPC.",
		},
		"volume_id": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "ID of the volume whose restore points are deleted. The volume does not need to be in a backup job.",
		},
		"triggers": triggersSchema("delete the restore points again"),
		"restore_point_count": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Number of restore points the volume had when the purge started. `0` means there was nothing to delete and no request was sent.",
		},
		"status": {
			Type:     schema.TypeString,
			Computed: true,
			Description: "Result of the purge: `success`, or the status of its history row when the purge failed, `failed`. " +
				runStatusesDescription,
		},
		"message": runMessageSchema(),
		"started_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "When the API accepted the purge, RFC 3339 UTC.",
		},
		"finished_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "When the provider saw the purge succeed, RFC 3339 UTC. Empty while the purge runs and after a failure.",
		},
	}
}

func listVolumeRestorePoints(ctx context.Context, svc VolumeService, vpcId, volumeId string) ([]VolumeRestorePoint, error) {
	var points []VolumeRestorePoint
	err := retryRead(ctx, func() error {
		var err error
		if points, err = svc.ListVolumeRestorePoints(vpcId, volumeId); err != nil {
			return readFailed(err)
		}
		return nil
	})
	return points, err
}

func createVolumeRestorePointsPurge(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	volumeId := d.Get("volume_id").(string)
	deadline := time.Now().Add(d.Timeout(schema.TimeoutCreate))

	started := time.Now().UTC()
	points, err := listVolumeRestorePoints(ctx, svc, vpcId, volumeId)
	if err != nil {
		return diag.FromErr(err)
	}
	var seen map[string]bool
	if len(points) > 0 {
		if started, seen, err = submitVolumeRestorePointsPurge(ctx, svc, vpcId, volumeId, deadline); err != nil {
			return diag.FromErr(err)
		}
	}
	// Once the API has accepted the purge, it is saved whatever happens next,
	// so that the next apply does not purge again.
	d.SetId(fmt.Sprintf("%s/%d", volumeId, started.Unix()))
	if diags := setValues(d, map[string]interface{}{"restore_point_count": len(points), "started_at": started.Format(time.RFC3339)}); diags.HasError() {
		return diags
	}
	if len(points) > 0 {
		if err := waitVolumeRestorePointsPurge(ctx, svc, vpcId, volumeId, started, seen, deadline); err != nil {
			return saveRunOutcome(ctx, d, err)
		}
	}

	diags := setValues(d, map[string]interface{}{
		"status":      historyStatusSuccess,
		"message":     "",
		"finished_at": time.Now().UTC().Format(time.RFC3339),
	})
	if diags.HasError() || len(points) == 0 {
		return diags
	}

	// The worker skips restore points an earlier purge is still deleting, and
	// a backup can finish while the purge runs.
	left, err := listVolumeRestorePoints(ctx, svc, vpcId, volumeId)
	if err != nil {
		return append(diags, diag.Diagnostic{Severity: diag.Warning, Summary: fmt.Sprintf("could not check whether restore points of volume %s remain: %v", volumeId, err)})
	}
	if len(left) > 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("%d restore points of volume %s remain after the purge", len(left), volumeId),
			Detail: "They are restore points an earlier purge is still deleting, or ones a backup created while this purge ran. " +
				"Change a value in triggers to delete them.",
		})
	}
	return diags
}

// readVolumeRestorePointsPurge checks once on a purge the provider stopped
// watching, on the first purge row of the volume after the id's time.
func readVolumeRestorePointsPurge(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	svc := NewVolumeService(m.(*common.Client))
	vpcId := d.Get("vpc_id").(string)
	volumeId := d.Get("volume_id").(string)
	return refreshRun(d, func() (map[string]interface{}, error) {
		started, err := idTime(d.Id())
		if err != nil {
			return nil, err
		}
		var row *VolumeHistoryItem
		// Rows come newest first, so the last one visited is the first purge
		// after the submit.
		err = scanVolumeHistoryRows(svc, vpcId, volumeId, historyTypePurge, started, func(item *VolumeHistoryItem) bool {
			row = item
			return false
		})
		if err != nil {
			return nil, err
		}
		status, err := volumeRestorePointsPurgeEnded(svc, vpcId, volumeId, row)
		var failed runFailedError
		switch {
		case errors.As(err, &failed):
			return map[string]interface{}{"status": failed.status, "message": failed.Error()}, nil
		case err != nil:
			return nil, err
		case status == "":
			return nil, nil
		}
		return map[string]interface{}{"status": status, "message": "", "finished_at": time.Now().UTC().Format(time.RFC3339)}, nil
	})
}

// submitVolumeRestorePointsPurge sends the delete-all request, retrying while
// the API refuses it as busy. It returns when the accepted request went out,
// and the purge rows already in the history then, which belong to earlier
// purges.
func submitVolumeRestorePointsPurge(ctx context.Context, svc VolumeService, vpcId, volumeId string, deadline time.Time) (time.Time, map[string]bool, error) {
	var refusal error
	var sent time.Time
	var seen map[string]bool
	submit := newStateConf([]string{"busy"}, []string{"accepted"}, func() (interface{}, string, error) {
		sent = time.Now().UTC()
		var err error
		if seen, err = volumeHistoryRowsSeen(svc, vpcId, volumeId, historyTypePurge, sent); err != nil {
			return nil, "", readFailed(err)
		}
		err = svc.DeleteAllVolumeRestorePoints(vpcId, volumeId)
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
	if _, err := waitForState(ctx, submit); err != nil {
		if refusal != nil {
			return sent, nil, fmt.Errorf("waiting for the API to accept deleting the restore points of volume %s: %w; last refusal: %v", volumeId, err, refusal)
		}
		return sent, nil, err
	}
	return sent, seen, nil
}

// volumeRestorePointsPurgeEnded returns the status of a purge that has ended,
// or "" while it runs. row is the purge's history row, or nil. A failed row
// returns a runFailed error.
func volumeRestorePointsPurgeEnded(svc VolumeService, vpcId, volumeId string, row *VolumeHistoryItem) (string, error) {
	if row != nil {
		switch strings.ToLower(row.Status) {
		case historyStatusSuccess:
			return row.Status, nil
		case historyStatusFailed:
			reason := "the API gave no reason"
			if r, ok := row.HistoryMetadata.ExtractInfo["status"]; ok {
				reason = fmt.Sprint(r)
			}
			return "", runFailed(row.Status, fmt.Errorf("deleting the restore points of volume %s failed: %s. "+
				"Some restore points may already be deleted; change a value in triggers to delete the rest", volumeId, reason))
		}
	}
	// No row appears when the worker finds no restore point on the platform,
	// so an empty list also ends the purge.
	points, err := svc.ListVolumeRestorePoints(vpcId, volumeId)
	if err != nil {
		return "", readFailed(err)
	}
	if len(points) == 0 {
		return historyStatusSuccess, nil
	}
	return "", nil
}

// waitVolumeRestorePointsPurge waits until the purge sent at since ends.
func waitVolumeRestorePointsPurge(ctx context.Context, svc VolumeService, vpcId, volumeId string, since time.Time, seen map[string]bool, deadline time.Time) error {
	wait := newStateConf([]string{"deleting"}, []string{"done"}, func() (interface{}, string, error) {
		row, err := findVolumeHistoryRow(svc, vpcId, volumeId, historyTypePurge, since, seen)
		if err != nil {
			return nil, "", readFailed(err)
		}
		status, err := volumeRestorePointsPurgeEnded(svc, vpcId, volumeId, row)
		if err != nil || status == "" {
			return struct{}{}, "deleting", err
		}
		return struct{}{}, "done", nil
	}, time.Until(deadline))
	if _, err := waitForState(ctx, wait); err != nil {
		return fmt.Errorf("waiting for the restore points of volume %s to be deleted: %w", volumeId, err)
	}
	return nil
}
