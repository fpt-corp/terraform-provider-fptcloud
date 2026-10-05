package fptcloud_instance

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"strings"
	common "terraform-provider-fptcloud/commons"
	"time"
)

// configuredGpuName returns gpu_name only when it's actually present in the
// .tf config, not when Terraform is just carrying forward the Computed value
// from a previous apply (gpu_name is Optional+Computed so d.GetOk alone can't
// tell the two apart) — sending a stale GPU name would make the server reject
// a resize to a CPU flavor with "gpu_name is only applicable to GPU flavors".
func configuredGpuName(d *schema.ResourceData) string {
	raw := d.GetRawConfig()
	if raw.IsNull() {
		return ""
	}
	gpuNameVal := raw.GetAttr("gpu_name")
	if gpuNameVal.IsNull() {
		return ""
	}
	return gpuNameVal.AsString()
}

// ResourceInstance function returns a schema.Resource that represents an instance.
// This can be used to create, read, update and delete operations for an instance in the infrastructure.
func ResourceInstance() *schema.Resource {
	return &schema.Resource{
		Description:   "Provides a instance resource. This can be used to create, modify, and delete instances.",
		Schema:        resourceInstanceSchema,
		CreateContext: resourceInstanceCreate,
		UpdateContext: resourceInstanceUpdate,
		ReadContext:   resourceInstanceRead,
		DeleteContext: resourceInstanceDelete,
		CustomizeDiff: customdiff.All(
			resourceInstanceCustomizeDiff,
			forceNewOnceKnown("image_name"),
			forceNewOnceKnown("ssh_key"),
			forceNewOnceKnown("password"),
		),
		Importer: &schema.ResourceImporter{
			StateContext: resourceInstanceImport,
		},
	}
}

const instanceNotFoundCode = "1503002"

var transitionalStatuses = map[string]bool{
	"CREATING": true, "DELETING": true, "RESIZING": true, "VERIFY_RESIZE": true, "POWERING_ON": true,
	"POWERING_OFF": true, "REBOOT": true, "REBOOTING": true, "UPDATING_GPU": true,
}

func resourceInstanceImport(_ context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if common.IsVpcImportName(d.Id(), "instance") {
		return importInstanceByName(d, m)
	}

	vpcId, instanceId, err := common.ParseVpcImportId(d.Id(), "instance")
	if err != nil {
		return nil, err
	}

	instance, err := NewInstanceService(m.(*common.Client)).Get(vpcId, instanceId)
	if err != nil {
		if common.IsApiErrorCode(err, instanceNotFoundCode) {
			return nil, fmt.Errorf("unable to import fptcloud_instance: instance %s was not found in VPC %s; check the VPC id, and that the provider region and tenant_name are the ones the instance lives in", instanceId, vpcId)
		}
		return nil, fmt.Errorf("unable to import fptcloud_instance %s from VPC %s: %s", instanceId, vpcId, common.DescribeApiError(err))
	}
	if transitionalStatuses[instance.Status] {
		return nil, fmt.Errorf("unable to import fptcloud_instance: instance %s is %s, retry once the operation in progress is over", instanceId, instance.Status)
	}
	if isNvme(instance) {
		return nil, nvmeImportUnsupported(instanceId)
	}

	if err := d.Set("vpc_id", vpcId); err != nil {
		return nil, err
	}
	d.SetId(instanceId)
	return []*schema.ResourceData{d}, nil
}

func importInstanceByName(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	vpcId, name, err := common.ParseVpcImportName(d.Id(), "instance")
	if err != nil {
		return nil, err
	}

	instance, err := NewInstanceService(m.(*common.Client)).GetByName(vpcId, name)
	if err != nil {
		if common.IsApiErrorCode(err, instanceNotFoundCode) {
			return nil, fmt.Errorf("unable to import fptcloud_instance: no instance named %q was found in VPC %s; check the name, the VPC id, and that the provider region and tenant_name are the ones the instance lives in", name, vpcId)
		}
		return nil, fmt.Errorf("unable to import fptcloud_instance %q from VPC %s: %s", name, vpcId, common.DescribeApiError(err))
	}
	if transitionalStatuses[instance.Status] {
		return nil, fmt.Errorf("unable to import fptcloud_instance: instance %q is %s, retry once the operation in progress is over", name, instance.Status)
	}
	if isNvme(instance) {
		return nil, nvmeImportUnsupported(instance.ID)
	}

	if err := d.Set("vpc_id", vpcId); err != nil {
		return nil, err
	}
	d.SetId(strings.ToLower(instance.ID))
	return []*schema.ResourceData{d}, nil
}

func forceNewOnceKnown(key string) schema.CustomizeDiffFunc {
	return customdiff.ForceNewIfChange(key, func(_ context.Context, before, after, _ interface{}) bool {
		return before.(string) != "" && before.(string) != after.(string)
	})
}

// function to create a new instance
func resourceInstanceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	instanceService := NewInstanceService(apiClient)

	createdModel := CreateInstanceDTO{}
	vpcId, okVpcId := d.GetOk("vpc_id")

	if name, ok := d.GetOk("name"); ok {
		createdModel.Name = name.(string)
	}

	if privateIp, ok := d.GetOk("private_ip"); ok {
		PrivateIpValue := privateIp.(string)
		createdModel.PrivateIp = &PrivateIpValue
	}

	if publicIp, ok := d.GetOk("public_ip"); ok {
		publicIpValue := publicIp.(string)
		createdModel.PublicIp = &publicIpValue
	}

	if flavorName, ok := d.GetOk("flavor_name"); ok {
		createdModel.FlavorName = flavorName.(string)
	}

	if imageName, ok := d.GetOk("image_name"); ok {
		createdModel.ImageName = imageName.(string)
	}

	if subnetId, ok := d.GetOk("subnet_id"); ok {
		createdModel.SubnetId = subnetId.(string)
	}

	if storageSizeGb, ok := d.GetOk("storage_size_gb"); ok {
		createdModel.StorageSizeGb = storageSizeGb.(int)
	}

	if storagePolicyId, ok := d.GetOk("storage_policy_id"); ok {
		createdModel.StoragePolicyId = storagePolicyId.(string)
	}

	if securityGroupIds, ok := d.GetOk("security_group_ids"); ok {
		securityGroupIdsSet := securityGroupIds.(*schema.Set)
		securityGroupIdsList := make([]string, 0, len(securityGroupIdsSet.List()))
		for _, v := range securityGroupIdsSet.List() {
			securityGroupIdsList = append(securityGroupIdsList, v.(string))
		}
		createdModel.SecurityGroupIds = securityGroupIdsList
	}

	if gpuPlan, ok := d.GetOk("gpu_plan"); ok {
		billingType := mapGpuPlanToBillingType(gpuPlan.(string))
		createdModel.BillingType = &billingType
	}

	if gpuName, ok := d.GetOk("gpu_name"); ok {
		gpuNameValue := gpuName.(string)
		createdModel.GpuName = &gpuNameValue
	}

	if tags, ok := d.GetOk("tag_ids"); ok {
		tagsSet := tags.(*schema.Set)
		tagIds := make([]string, 0, tagsSet.Len())
		for _, tag := range tagsSet.List() {
			tagIds = append(tagIds, tag.(string))
		}
		createdModel.TagIds = tagIds
	}

	if instanceGroupId, ok := d.GetOk("instance_group_id"); ok {
		instanceGroupIdValue := instanceGroupId.(string)
		createdModel.InstanceGroupId = &instanceGroupIdValue
	}

	if sshKey, ok := d.GetOk("ssh_key"); ok {
		sshKeyValue := sshKey.(string)
		createdModel.SshKey = &sshKeyValue
	}

	if password, ok := d.GetOk("password"); ok {
		passwordValue := password.(string)
		createdModel.Password = &passwordValue
	}

	if okVpcId {
		createdModel.VpcId = vpcId.(string)
	}

	if createdModel.SubnetId == "" {
		return diag.Errorf("[ERR] Subnet id is required")
	}

	instanceId, err := instanceService.Create(createdModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to create instance: %s", err)
	}

	d.SetId(instanceId)
	setError := d.Set("vpc_id", vpcId.(string))
	if setError != nil {
		return diag.Errorf("[ERR] Failed to create instance")
	}

	// Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"CREATING"},
		Target:  []string{"POWERED_ON", "POWERED_OFF"},
		Refresh: func() (interface{}, string, error) {
			findModel := FindInstanceDTO{
				ID:    instanceId,
				VpcId: vpcId.(string),
			}
			resp, err := instanceService.Find(findModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}
			return resp, resp.Status, nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for instance (%s) to be created: %s", d.Id(), err)
	}

	return resourceInstanceRead(ctx, d, m)
}

// function to read an instance
func resourceInstanceRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	instanceService := NewInstanceService(apiClient)

	findInstanceModel := FindInstanceDTO{}

	if id, ok := d.GetOk("id"); ok {
		findInstanceModel.ID = id.(string)
	}

	if name, ok := d.GetOk("name"); ok {
		findInstanceModel.Name = name.(string)
	}

	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findInstanceModel.VpcId = vpcId.(string)
	}

	imported := d.Get("name").(string) == ""

	foundInstance, err := instanceService.Find(findInstanceModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve instance: %s", err)
	}
	if imported {
		return readImportedInstance(d, instanceService, foundInstance)
	}

	// Set other attributes
	d.SetId(foundInstance.ID)

	if err := d.Set("vpc_id", foundInstance.VpcId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", foundInstance.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", foundInstance.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("private_ip", foundInstance.PrivateIp); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("public_ip", foundInstance.PublicIp); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("flavor_name", foundInstance.FlavorName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("subnet_id", foundInstance.SubnetId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("security_group_ids", foundInstance.SecurityGroupIds); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("instance_group_id", foundInstance.InstanceGroupId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("created_at", foundInstance.CreatedAt); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tag_ids", foundInstance.TagIds); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("gpu_name", foundInstance.GpuName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("vm_type", deriveVmType(foundInstance.GpuName)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("gpu_plan", mapBillingTypeToGpuPlan(foundInstance.BillingType)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("is_nvme", foundInstance.IsNvme); err != nil {
		return diag.FromErr(err)
	}

	// find_instance only sees a root disk persisted as ROOT, which never happens on OSP, so read the disk listings instead
	rootStorage, err := instanceService.FindRootStorage(foundInstance.VpcId, foundInstance.ID)
	if err != nil {
		log.Printf("[WARN] Could not retrieve the root disk of instance %s: %s", foundInstance.ID, err)
		return nil
	}

	if rootStorage.SizeMb > 0 {
		if err := d.Set("storage_size_gb", rootStorage.SizeMb/1024); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func readImportedInstance(d *schema.ResourceData, instanceService InstanceService, foundInstance *InstanceModel) diag.Diagnostics {
	if foundInstance.ID != d.Id() {
		return diag.Errorf("unable to import fptcloud_instance: asked for instance %s, the API returned %s; nothing was recorded", d.Id(), foundInstance.ID)
	}
	if isNvme(foundInstance) {
		return diag.FromErr(nvmeImportUnsupported(foundInstance.ID))
	}
	// subnet_id forces a new instance: an empty one in the state would plan a replacement
	if foundInstance.SubnetId == "" {
		return diag.Errorf("unable to import fptcloud_instance: the API could not resolve the subnet of instance %s; nothing was recorded, retry later", foundInstance.ID)
	}

	rootStorage, rootErr := instanceService.FindRootStorage(foundInstance.VpcId, foundInstance.ID)
	if rootErr != nil {
		log.Printf("[WARN] Could not retrieve the root disk of instance %s: %s", foundInstance.ID, rootErr)
	}
	sizeGb := remoteRootStorageSizeGb(foundInstance, rootStorage)
	policyId := remoteRootStoragePolicyId(instanceService, foundInstance, rootStorage)
	if sizeGb == 0 || policyId == "" {
		return rootStorageUnreadable(foundInstance.ID, rootErr)
	}

	attributes := map[string]interface{}{
		"vpc_id":             foundInstance.VpcId,
		"name":               foundInstance.Name,
		"status":             foundInstance.Status,
		"private_ip":         foundInstance.PrivateIp,
		"public_ip":          foundInstance.PublicIp,
		"flavor_name":        foundInstance.FlavorName,
		"subnet_id":          foundInstance.SubnetId,
		"security_group_ids": foundInstance.SecurityGroupIds,
		"instance_group_id":  foundInstance.InstanceGroupId,
		"created_at":         foundInstance.CreatedAt,
		"tag_ids":            foundInstance.TagIds,
		"gpu_name":           foundInstance.GpuName,
		"vm_type":            deriveVmType(foundInstance.GpuName),
		"gpu_plan":           mapBillingTypeToGpuPlan(foundInstance.BillingType),
		"is_nvme":            foundInstance.IsNvme,
		"storage_size_gb":    sizeGb,
		"storage_policy_id":  policyId,
	}
	if foundInstance.ImageName != nil && *foundInstance.ImageName != "" {
		attributes["image_name"] = *foundInstance.ImageName
	}
	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

// isNvme tells a physical NVMe disk, which has no ROOT row and no storage policy
func isNvme(instance *InstanceModel) bool {
	return instance.IsNvme || instance.StoragePolicyId == NvmeStoragePolicy
}

func nvmeImportUnsupported(instanceId string) error {
	return fmt.Errorf("unable to import fptcloud_instance: instance %s uses a physical NVMe disk, which is on no storage policy, "+
		"and the storage_policy_id it was created with is not recorded by the API; importing NVMe instances is not supported yet", instanceId)
}

func rootStorageUnreadable(instanceId string, cause error) diag.Diagnostics {
	reason := "the API did not report it"
	if cause != nil {
		reason = cause.Error()
	}
	return diag.Errorf("could not read the root disk of instance %s (%s): its size and storage policy cannot be told, "+
		"so nothing was recorded; retry once the disk listings of the instance answer", instanceId, reason)
}

func remoteRootStorageSizeGb(instance *InstanceModel, rootStorage *RootStorageModel) int {
	switch {
	case rootStorage != nil && rootStorage.SizeMb > 0:
		return rootStorage.SizeMb / 1024
	case instance.StorageId != nil:
		return instance.StorageSizeGb
	}
	return 0
}

// remoteRootStoragePolicyId get the policy the root disk is on, "" when it cannot be told
func remoteRootStoragePolicyId(instanceService InstanceService, instance *InstanceModel, rootStorage *RootStorageModel) string {
	if isNvme(instance) {
		return ""
	}
	if rootStorage != nil && rootStorage.StoragePolicyId != "" {
		return rootStorage.StoragePolicyId
	}
	if instance.StorageId != nil && instance.StoragePolicyId != "" {
		return instance.StoragePolicyId
	}

	profile := ""
	if rootStorage != nil {
		profile = rootStorage.StoragePolicyName
	}
	if profile == "" {
		found, err := instanceService.FindRootStorageProfile(instance.VpcId, instance.ID)
		if err != nil {
			log.Printf("[WARN] Could not retrieve the storage profile of the root disk of instance %s: %s", instance.ID, err)
			return ""
		}
		profile = found
	}
	if profile == "" {
		return ""
	}
	storagePolicy, err := instanceService.FindStoragePolicyByProfile(instance.VpcId, profile)
	if err != nil {
		log.Printf("[WARN] Could not resolve the storage policy of instance %s: %s", instance.ID, err)
		return ""
	}
	return storagePolicy.ID
}

// function to delete an instance
func resourceInstanceDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	instanceService := NewInstanceService(apiClient)

	log.Printf("[INFO] Deleting the instance %s", d.Id())

	vpcId, okVpcId := d.GetOk("vpc_id")

	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	_, err := instanceService.Delete(vpcId.(string), d.Id())
	if err != nil {
		return diag.Errorf("[ERR] An error occurred while trying to delete the instance %s", err)
	}

	deleteStateConf := &retry.StateChangeConf{
		Pending: []string{"DELETING"},
		Target:  []string{"SUCCESS"},
		Refresh: func() (interface{}, string, error) {
			findInstanceModel := FindInstanceDTO{
				ID:    d.Id(),
				VpcId: vpcId.(string),
			}
			resp, err := instanceService.Find(findInstanceModel)
			if err != nil {
				// If the security group is not found, consider it deleted
				return 1, "SUCCESS", nil
			}

			return resp, resp.Status, nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err = deleteStateConf.WaitForStateContext(context.Background())
	if err != nil {
		return diag.Errorf("[Error] Waiting for instance (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

// function to update an instance
// resourceInstanceUpdate refreshes state from the server even when the update
// itself errors out partway through, so whatever changes did land before the
// failing step (e.g. a flavor resize that succeeded before a later billing
// plan change failed) are still reflected instead of leaving stale state.
func resourceInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	updateDiags := doResourceInstanceUpdate(ctx, d, m)
	readDiags := resourceInstanceRead(ctx, d, m)
	if updateDiags.HasError() {
		return append(updateDiags, readDiags...)
	}
	return readDiags
}

func doResourceInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	instanceService := NewInstanceService(apiClient)

	vpcId := d.Get("vpc_id").(string)
	hasChangedName := d.HasChange("name")
	hasChangeFlavor := d.HasChange("flavor_name")
	hasChangeStatus := d.HasChange("status")
	hasChangeTags := d.HasChange("tag_ids")
	hasChangeGpuPlan := d.HasChange("gpu_plan")
	hasChangeGpuName := d.HasChange("gpu_name")

	if hasChangedName {
		newName := d.Get("name").(string)
		_, err := instanceService.Rename(vpcId, d.Id(), newName)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while rename instance %s", err)
		}
	}

	if hasChangeStatus {
		status := d.Get("status").(string)
		_, err := instanceService.ChangeStatus(vpcId, d.Id(), status)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while change status instance %s", err)
		}
	}

	if hasChangeGpuName && !hasChangeFlavor {
		if gpuName, ok := d.GetOk("gpu_name"); ok {
			_, err := instanceService.GetFlavorByName(vpcId, d.Get("flavor_name").(string), gpuName.(string))
			if err != nil {
				return diag.Errorf("[ERR] An error occurred while verifying gpu name %s", err)
			}
		}
	}

	if hasChangeFlavor {
		newFlavorName := d.Get("flavor_name").(string)
		gpuName := configuredGpuName(d)
		flavor, flavorErr := instanceService.GetFlavorByName(vpcId, newFlavorName, gpuName)
		if flavorErr != nil {
			return diag.Errorf("[ERR] Flavor not found %s", flavorErr)
		}

		billingType := ""
		if gpuPlan, ok := d.GetOk("gpu_plan"); ok {
			billingType = mapGpuPlanToBillingType(gpuPlan.(string))
		}
		_, err := instanceService.Resize(vpcId, d.Id(), flavor.ID, billingType)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while resize instance %s", err)
		}

		updateStateConf := &retry.StateChangeConf{
			Pending: []string{"VERIFY_RESIZE"},
			Target:  []string{"POWERED_ON", "POWERED_OFF"},
			Refresh: func() (interface{}, string, error) {
				findModel := FindInstanceDTO{
					ID:    d.Id(),
					VpcId: vpcId,
				}
				resp, err := instanceService.Find(findModel)
				if err != nil {
					return 0, "", common.DecodeError(err)
				}
				return resp, resp.Status, nil
			},
			Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
			Delay:          3 * time.Second,
			MinTimeout:     3 * time.Second,
			NotFoundChecks: 120,
		}
		_, err = updateStateConf.WaitForStateContext(ctx)
		if err != nil {
			return diag.Errorf("[Error] Waiting for instance (%s) to be resize: %s", d.Id(), err)
		}
	}

	hasChangeStoragePolicy := d.HasChange("storage_policy_id")

	if d.HasChange("storage_size_gb") || hasChangeStoragePolicy {
		if diags := resizeInstanceRootDisk(ctx, d, apiClient, instanceService, vpcId, hasChangeStoragePolicy); diags != nil {
			return diags
		}
	}

	if hasChangeTags {
		tagsSet := d.Get("tag_ids").(*schema.Set)
		tagIds := make([]string, 0, tagsSet.Len())
		for _, tag := range tagsSet.List() {
			tagIds = append(tagIds, tag.(string))
		}

		_, err := instanceService.UpdateTags(vpcId, d.Id(), tagIds)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while updating instance tags %s", err)
		}
	}

	// When flavor_name also changed, gpu_plan was already sent along with the
	// resize request above; the server applies it as part of that same
	// operation, so a separate call here would just race it.
	if hasChangeGpuPlan && !hasChangeFlavor {
		gpuPlan := d.Get("gpu_plan").(string)
		_, err := instanceService.ChangeBillingType(vpcId, d.Id(), mapGpuPlanToBillingType(gpuPlan))
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while changing billing plan of instance %s: %s", d.Id(), err)
		}
	}

	return nil
}

// resourceInstanceCustomizeDiff rejects a shrink at plan time, the resize API only grows the root disk
func resourceInstanceCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if d.Id() == "" || !d.HasChange("storage_size_gb") {
		return nil
	}

	oldSize, newSize := d.GetChange("storage_size_gb")
	if newSize.(int) < oldSize.(int) {
		return fmt.Errorf(
			"[ERR] The root storage size of an instance can only be increased, got %d GB while the instance already has %d GB",
			newSize.(int), oldSize.(int),
		)
	}

	return nil
}

// resizeInstanceRootDisk resize and/or change the storage policy of the root disk in place
func resizeInstanceRootDisk(
	ctx context.Context,
	d *schema.ResourceData,
	apiClient *common.Client,
	instanceService InstanceService,
	vpcId string,
	hasChangeStoragePolicy bool,
) diag.Diagnostics {
	rootStorage, err := instanceService.FindRootStorage(vpcId, d.Id())
	if err != nil {
		return diag.Errorf("[ERR] An error occurred while retrieving the root disk of instance %s: %s", d.Id(), err)
	}

	storagePolicyId := d.Get("storage_policy_id").(string)
	storagePolicy, err := instanceService.FindStoragePolicy(vpcId, storagePolicyId)
	if err != nil {
		return diag.Errorf("[ERR] An error occurred while retrieving the storage policy %s: %s", storagePolicyId, err)
	}

	sizeGb := d.Get("storage_size_gb").(int)
	// the resize API expects the infrastructure id of the policy, not the id the provider exposes
	resizeModel := ResizeRootDiskDTO{
		DiskId:           rootStorage.DiskId,
		IncreaseInSizeMb: sizeGb * 1024,
		StoragePolicyId:  &storagePolicy.InfraId,
	}

	if _, err := instanceService.ResizeRootDisk(vpcId, d.Id(), resizeModel); err != nil {
		return diag.Errorf("[ERR] An error occurred while resizing the root disk of instance %s: %s", d.Id(), err)
	}

	// the API only queues the resize, poll the infrastructure until the root disk reports the new size
	resizeStateConf := &retry.StateChangeConf{
		Pending: []string{"RESIZING"},
		Target:  []string{"RESIZED"},
		Refresh: func() (interface{}, string, error) {
			resp, err := instanceService.FindRootStorage(vpcId, d.Id())
			if err != nil {
				return 0, "", err
			}

			if resp.SizeMb >= sizeGb*1024 && (!hasChangeStoragePolicy || isStoragePolicy(resp, storagePolicy)) {
				return resp, "RESIZED", nil
			}

			return resp, "RESIZING", nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	if _, err := resizeStateConf.WaitForStateContext(ctx); err != nil {
		return diag.Errorf("[Error] Waiting for the root disk of instance (%s) to be resized: %s", d.Id(), err)
	}

	return nil
}

// isStoragePolicy tells whether a root disk already sits on a storage policy, by id when the portal listed it or by name otherwise
func isStoragePolicy(rootStorage *RootStorageModel, storagePolicy *StoragePolicyDTO) bool {
	if rootStorage.StoragePolicyId != "" {
		return rootStorage.StoragePolicyId == storagePolicy.ID
	}

	// the infrastructure suffixes the policy name with the zone it belongs to
	return rootStorage.StoragePolicyName == storagePolicy.Name ||
		strings.HasPrefix(rootStorage.StoragePolicyName, storagePolicy.Name+"_")
}
