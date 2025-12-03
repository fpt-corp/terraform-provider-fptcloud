package fptcloud_instance

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	common "terraform-provider-fptcloud/commons"
	"time"
)

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
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
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

	foundInstance, err := instanceService.Find(findInstanceModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve instance: %s", err)
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

	return nil
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
func resourceInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	instanceService := NewInstanceService(apiClient)

	vpcId := d.Get("vpc_id").(string)
	hasChangedName := d.HasChange("name")
	hasChangeFlavor := d.HasChange("flavor_name")
	hasChangeStatus := d.HasChange("status")
	hasChangeTags := d.HasChange("tag_ids")

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

	if hasChangeFlavor {
		newFlavorName := d.Get("flavor_name").(string)
		flavor, flavorErr := instanceService.GetFlavorByName(vpcId, newFlavorName)
		if flavorErr != nil {
			return diag.Errorf("[ERR] Flavor not found %s", flavorErr)
		}
		_, err := instanceService.Resize(vpcId, d.Id(), flavor.ID)
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

	return resourceInstanceRead(ctx, d, m)
}
