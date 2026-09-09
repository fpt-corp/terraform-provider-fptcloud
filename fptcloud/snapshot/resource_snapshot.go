package fptcloud_snapshot

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	common "terraform-provider-fptcloud/commons"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var nameFormat = regexp.MustCompile(`^[a-zA-Z0-9_\-. ]+$`)

func ResourceSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "Snapshots every volume attached to an instance at a point in time. The snapshot is " +
			"crash-consistent: the disks are captured as they are, without quiescing the guest filesystem. " +
			"A snapshot cannot be renamed, so changing any argument replaces it.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "Id of the vpc the instance belongs to",
			},
			"instance_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "Id of the instance to snapshot",
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateSnapshotName,
				Description: "Name for the snapshot, unique within the vpc. Up to 30 characters of letters, digits, " +
					"underscores, dashes, dots and spaces. Left unset, the platform names it",
			},
			"include_ram": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "Capture the guest's memory alongside its disks, where the platform supports it",
			},
			"tag_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tag ids to apply to the snapshot, applied when it is created",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current status of the snapshot, as the platform reports it",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the snapshot was taken",
			},
			"created_by": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User that created the snapshot",
			},
			"size_gb": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Total size of the snapshot, in GB",
			},
			"snapshot_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Type of the snapshot, as the platform reports it",
			},
			"infra_snapshot_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Id of the snapshot on the underlying infrastructure",
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Tags currently attached to the snapshot",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":    {Type: schema.TypeString, Computed: true},
						"key":   {Type: schema.TypeString, Computed: true},
						"value": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
		CreateContext: resourceSnapshotCreate,
		ReadContext:   resourceSnapshotRead,
		UpdateContext: nil,
		DeleteContext: resourceSnapshotDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 4 || parts[0] != "vpc" || parts[2] != "snapshot" {
					return nil, fmt.Errorf("invalid import id format, expected vpc/<vpc_id>/snapshot/<snapshot_id>")
				}

				if err := d.Set("vpc_id", parts[1]); err != nil {
					return nil, fmt.Errorf("error setting vpc id: %s", err)
				}
				d.SetId(parts[3])

				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func validateSnapshotName(v interface{}, key string) (warns []string, errs []error) {
	value, ok := v.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected %s to be a string", key)}
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, []error{fmt.Errorf("%s cannot be empty", key)}
	}
	if len(trimmed) > MaxNameLength {
		return nil, []error{fmt.Errorf("%s must be at most %d characters, got %d", key, MaxNameLength, len(trimmed))}
	}
	if !nameFormat.MatchString(trimmed) {
		return nil, []error{fmt.Errorf("%s can only contain letters, digits, underscores, dashes, dots and spaces, got %q", key, value)}
	}

	return nil, nil
}

func resourceSnapshotCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewSnapshotService(apiClient)

	vpcId := d.Get("vpc_id").(string)
	instanceId := d.Get("instance_id").(string)
	name := strings.TrimSpace(d.Get("name").(string))

	createModel := CreateSnapshotDTO{
		InstanceId:   instanceId,
		SnapshotName: name,
		IncludeRam:   d.Get("include_ram").(bool),
	}
	if tags, ok := d.GetOk("tag_ids"); ok {
		createModel.TagIds = expandTagIDs(tags.(*schema.Set))
	}

	created, err := service.CreateSnapshot(vpcId, createModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to create snapshot of instance %s: %s", instanceId, err)
	}

	// Recorded before the wait so a snapshot that does not settle in time still
	// ends up in state instead of being created again on the next apply.
	d.SetId(created.Id)

	createStateConf := &retry.StateChangeConf{
		Pending: CreatingStatuses,
		Target:  SettledStatuses,
		Refresh: func() (interface{}, string, error) {
			snapshot, err := service.GetSnapshot(vpcId, created.Id)
			if err != nil {
				return nil, "", err
			}
			if snapshot == nil {
				return nil, "", nil
			}
			if IsFailed(snapshot.Status) {
				return nil, "", fmt.Errorf("the platform reported status %s", snapshot.Status)
			}
			return snapshot, snapshot.Status, nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}

	if _, err := createStateConf.WaitForStateContext(ctx); err != nil {
		return diag.Errorf(
			"[ERR] Waiting for snapshot %s of instance %s to settle: %s. "+
				"It has been recorded so it can be refreshed or destroyed",
			created.Id, instanceId, err,
		)
	}

	return resourceSnapshotRead(ctx, d, m)
}

func resourceSnapshotRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewSnapshotService(apiClient)

	vpcId, ok := d.GetOk("vpc_id")
	if !ok || vpcId.(string) == "" {
		return diag.Errorf("[ERR] vpc_id is missing from state. Import the snapshot with 'vpc/<vpc_id>/snapshot/<snapshot_id>'")
	}

	log.Printf("[INFO] Retrieving the instance snapshot %s", d.Id())

	snapshot, err := service.GetSnapshot(vpcId.(string), d.Id())
	if err != nil {
		return diag.Errorf("[ERR] Failed retrieving the instance snapshot %s: %s", d.Id(), err)
	}

	if snapshot == nil {
		// Deleted outside Terraform; drop it so the next plan recreates it.
		log.Printf("[WARN] Instance snapshot %s no longer exists, removing it from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("vpc_id", snapshot.VpcId); err != nil {
		return diag.FromErr(err)
	}
	if snapshot.InstanceId != "" {
		if err := d.Set("instance_id", snapshot.InstanceId); err != nil {
			return diag.FromErr(err)
		}
	}
	if current, ok := d.GetOk("name"); !ok || current.(string) == "" {
		if err := d.Set("name", snapshot.Name); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("status", snapshot.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("created_at", snapshot.CreatedAt); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("created_by", snapshot.CreatedBy); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("size_gb", snapshot.SizeGb); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("snapshot_type", snapshot.SnapshotType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("infra_snapshot_id", snapshot.InfraSnapshotId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tags", flattenTags(snapshot.Tags)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSnapshotDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	service := NewSnapshotService(apiClient)

	vpcId, ok := d.GetOk("vpc_id")
	if !ok || vpcId.(string) == "" {
		return diag.Errorf("[ERR] vpc_id is missing from state, cannot delete snapshot %s", d.Id())
	}

	log.Printf("[INFO] Deleting the instance snapshot %s", d.Id())

	if err := service.DeleteSnapshot(vpcId.(string), d.Id()); err != nil {
		return diag.Errorf("[ERR] Failed to delete the instance snapshot %s: %s", d.Id(), err)
	}

	deleteStateConf := &retry.StateChangeConf{
		Pending: DeletingStatuses,
		Target:  []string{},
		Refresh: func() (interface{}, string, error) {
			current, err := service.GetSnapshot(vpcId.(string), d.Id())
			if err != nil {
				return nil, "", err
			}
			if current == nil {
				return nil, "", nil
			}
			if IsFailed(current.Status) {
				return nil, "", fmt.Errorf("the platform reported status %s", current.Status)
			}
			return current, current.Status, nil
		},
		Timeout:    time.Duration(apiClient.Timeout) * time.Minute,
		Delay:      3 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	if _, err := deleteStateConf.WaitForStateContext(ctx); err != nil {
		return diag.Errorf("[Error] Waiting for instance snapshot (%s) to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func flattenTags(tags []Tag) []interface{} {
	flattened := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		flattened = append(flattened, map[string]interface{}{
			"id":    tag.Id,
			"key":   tag.Key,
			"value": tag.Value,
		})
	}
	return flattened
}

func expandTagIDs(tagSet *schema.Set) []string {
	tagIds := make([]string, 0, tagSet.Len())
	for _, tag := range tagSet.List() {
		tagIds = append(tagIds, tag.(string))
	}
	return tagIds
}
