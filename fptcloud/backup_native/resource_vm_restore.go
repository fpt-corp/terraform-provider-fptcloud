package fptcloud_backup_native

import (
	"context"
	"fmt"
	"regexp"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	modeOverwrite = "overwrite"
	modeKeep      = "keep"
)

var (
	restorePointTimePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?$`)
	restoreNamePattern      = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	keepOnlyArguments       = []string{"instance_name", "flavor_id", "subnet_id", "security_group_ids", "floating_ip", "private_ip"}
)

func ResourceVmRestore() *schema.Resource {
	return &schema.Resource{
		Description: "Restores a VM from a Backup Native restore point. Applying the resource runs the restore once and waits for it to finish. " +
			"Destroying it only removes it from state; it never changes the VM. `overwrite` restores onto the original VM; `keep` creates a new VM and keeps the original. " +
			"The API does not return the new VM's id; find it by `instance_name` with the `fptcloud_instances` data source. " +
			"Run one restore per VM at a time; two concurrent restores of the same VM can be matched to the wrong history row.",
		CreateContext: createVmRestore,
		ReadContext:   readOneShot,
		DeleteContext: deleteOneShot,
		CustomizeDiff: customizeVmRestore,
		Schema:        resourceVmRestoreSchema(),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
	}
}

func resourceVmRestoreSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {Type: schema.TypeString, Required: true, ForceNew: true, Description: "ID of the VPC."},
		"vm_id":  {Type: schema.TypeString, Required: true, ForceNew: true, Description: "ID of the VM the restore point belongs to."},
		"restore_point_created_at": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			ValidateFunc: validation.StringMatch(restorePointTimePattern,
				"must be the created_at value of a restore point as returned by fptcloud_backup_native_vm_restore_points, without a Z suffix or offset, for example 2026-07-01T02:00:00"),
			Description: "`created_at` of the restore point, copied from `fptcloud_backup_native_vm_restore_points`. The backend picks the restore point closest to this time within 10 minutes.",
		},
		"mode": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.StringInSlice([]string{modeOverwrite, modeKeep}, false),
			Description:  "`overwrite` restores onto the original VM. `keep` creates a new VM and keeps the original.",
		},
		"power_on":     {Type: schema.TypeBool, Optional: true, ForceNew: true, Default: true, Description: "Whether the restored VM is powered on."},
		"new_password": {Type: schema.TypeString, Optional: true, ForceNew: true, Sensitive: true, Description: "New password for the restored VM."},
		"instance_name": {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			ValidateFunc: validation.All(
				validation.StringLenBetween(1, 50),
				validation.StringMatch(restoreNamePattern, "must contain only letters, digits and hyphens"),
			),
			Description: "`keep` only. Name of the new VM, at most 50 characters.",
		},
		"flavor_id":          {Type: schema.TypeString, Optional: true, ForceNew: true, Description: "`keep` only, required for `keep`. ID of the new VM's flavor."},
		"subnet_id":          {Type: schema.TypeString, Optional: true, ForceNew: true, Description: "`keep` only, required for `keep`. ID of the subnet (the `id` of `fptcloud_subnet`)."},
		"security_group_ids": {Type: schema.TypeList, Optional: true, ForceNew: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "`keep` only. IDs of security groups for the new VM."},
		"floating_ip": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "`keep` only. Omit for no floating IP, `new` to allocate one, or the ID of a floating IP that is not in use.",
		},
		"private_ip":  {Type: schema.TypeString, Optional: true, ForceNew: true, ValidateFunc: validation.IsIPv4Address, Description: "`keep` only. Private IP of the new VM."},
		"status":      {Type: schema.TypeString, Computed: true, Description: "Result of the restore, `Success` after a successful apply."},
		"started_at":  {Type: schema.TypeString, Computed: true, Description: "Start of the restore, RFC 3339 UTC."},
		"finished_at": {Type: schema.TypeString, Computed: true, Description: "End of the restore, RFC 3339 UTC."},
	}
}

// checkRestoreArguments enforces which arguments go with which mode. set
// lists the arguments the configuration sets.
func checkRestoreArguments(mode string, set map[string]bool) error {
	switch mode {
	case modeOverwrite:
		for _, key := range keepOnlyArguments {
			if set[key] {
				return fmt.Errorf("%s can only be set when mode is keep", key)
			}
		}
	case modeKeep:
		for _, key := range []string{"flavor_id", "subnet_id"} {
			if !set[key] {
				return fmt.Errorf("%s is required when mode is keep", key)
			}
		}
	}
	return nil
}

func customizeVmRestore(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if !d.NewValueKnown("mode") {
		return nil
	}
	set := map[string]bool{}
	for _, key := range keepOnlyArguments {
		if !d.NewValueKnown(key) {
			// An unknown value is set; the check runs again at apply.
			set[key] = true
			continue
		}
		if _, ok := d.GetOk(key); ok {
			set[key] = true
		}
	}
	return checkRestoreArguments(d.Get("mode").(string), set)
}

func stringList(raw interface{}) []string {
	list, _ := raw.([]interface{})
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func createVmRestore(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	svc := NewVmService(client)
	vpcId := d.Get("vpc_id").(string)
	vmId := d.Get("vm_id").(string)
	createdAt := d.Get("restore_point_created_at").(string)
	mode := d.Get("mode").(string)
	submitted := time.Now().UTC()

	historyType := "Restore"
	if mode != modeOverwrite {
		historyType = "Restore keep"
	}
	// Rows already in the history belong to earlier restores of this VM.
	seen, err := vmHistoryRowsSeen(svc, vpcId, vmId, historyType, submitted)
	if err != nil {
		return diag.FromErr(err)
	}

	if mode == modeOverwrite {
		err = svc.RestoreVm(vpcId, VmRestorePayload{
			VmId:        vmId,
			CreatedAt:   createdAt,
			IsOverwrite: true,
			NewPassword: optional(d.Get("new_password").(string)),
			PowerOn:     d.Get("power_on").(bool),
		})
	} else {
		var payload VmRestoreKeepPayload
		payload, err = buildRestoreKeepPayload(vpcId, keepArgs{
			VmId:             vmId,
			CreatedAt:        createdAt,
			InstanceName:     d.Get("instance_name").(string),
			NewPassword:      d.Get("new_password").(string),
			FlavorId:         d.Get("flavor_id").(string),
			SubnetId:         d.Get("subnet_id").(string),
			FloatingIp:       d.Get("floating_ip").(string),
			PrivateIp:        d.Get("private_ip").(string),
			SecurityGroupIds: stringList(d.Get("security_group_ids")),
			PowerOn:          d.Get("power_on").(bool),
		}, newKeepLookups(client))
		if err == nil {
			err = svc.RestoreVmKeep(vpcId, payload)
		}
	}
	if err != nil {
		return diag.FromErr(err)
	}

	row, err := waitVmRestore(ctx, svc, vpcId, vmId, historyType, submitted, seen, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%s/%s/%d", vmId, mode, submitted.Unix()))
	return setValues(d, map[string]interface{}{
		"status":      row.Status,
		"started_at":  formatAPITime(row.StartTime),
		"finished_at": formatAPITime(row.EndTime),
	})
}

// readOneShot keeps state as it is: a finished restore has nothing to refresh.
func readOneShot(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return nil
}

// deleteOneShot removes the resource from state and changes nothing in the cloud.
func deleteOneShot(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
