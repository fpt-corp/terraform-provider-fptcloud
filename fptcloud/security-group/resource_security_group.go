package fptcloud_security_group

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	common "terraform-provider-fptcloud/commons"
	"time"
)

// ResourceSecurityGroup function returns a schema.Resource that represents a security group.
// This can be used to create, read, update, and delete operations for a security group in the infrastructure.
func ResourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Description:   "Provides a Fpt cloud security group which can be attached to an instance in order to firewall.",
		Schema:        resourceSecurityGroup,
		CreateContext: resourceSecurityGroupCreate,
		ReadContext:   resourceSecurityGroupRead,
		UpdateContext: resourceSecurityGroupUpdate,
		DeleteContext: resourceSecurityGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

// function to create the new security group
func resourceSecurityGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupService := NewSecurityGroupService(apiClient)

	createdModel := CreatedSecurityGroupDTO{}
	vpcId, okVpcId := d.GetOk("vpc_id")

	if name, ok := d.GetOk("name"); ok {
		createdModel.Name = name.(string)
	}

	if subnetId, ok := d.GetOk("subnet_id"); ok {
		createdModel.SubnetId = subnetId.(string)
	}

	if securityGroupType, ok := d.GetOk("type"); ok {
		createdModel.Type = securityGroupType.(string)
	}

	if applyTo, ok := d.GetOk("apply_to"); ok {
		applyToList := applyTo.([]interface{})
		createdModel.ApplyTo = make([]string, len(applyToList))
		for i, v := range applyToList {
			createdModel.ApplyTo[i] = v.(string)
		}
	}

	if okVpcId {
		createdModel.VpcId = vpcId.(string)
	}

	if createdModel.SubnetId == "" {
		return diag.Errorf("[ERR] Subnet id is required")
	}

	securityGroupId, err := securityGroupService.Create(createdModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to create security group: %s", err)
	}

	d.SetId(securityGroupId)
	setError := d.Set("vpc_id", vpcId.(string))
	if setError != nil {
		return diag.Errorf("[ERR] Failed to create security group")
	}

	// Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"PENDING", "UPDATING"},
		Target:  []string{"REALIZED", "ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			findStorageModel := FindSecurityGroupDTO{
				ID:    securityGroupId,
				VpcId: vpcId.(string),
			}
			resp, err := securityGroupService.Find(findStorageModel)
			if err != nil {
				return 0, "", common.DecodeError(err)
			}
			return resp, resp.Status, nil
		},
		Timeout:        5 * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[Error] Waiting for security group (%s) to be created: %s", d.Id(), err)
	}

	return resourceSecurityGroupRead(ctx, d, m)
}

// function to read the security group
func resourceSecurityGroupRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupService := NewSecurityGroupService(apiClient)

	findSecurityGroupModel := FindSecurityGroupDTO{}

	findSecurityGroupModel.ID = d.Id()

	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findSecurityGroupModel.VpcId = vpcId.(string)
	}

	foundSecurityGroup, err := securityGroupService.Find(findSecurityGroupModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve security group: %s", err)
	}

	// Set other attributes
	var setError error
	d.SetId(foundSecurityGroup.ID)
	setError = d.Set("vpc_id", foundSecurityGroup.VpcId)
	setError = d.Set("name", foundSecurityGroup.Name)
	setError = d.Set("edge_gateway_id", foundSecurityGroup.EdgeGatewayId)
	setError = d.Set("type", foundSecurityGroup.Type)
	setError = d.Set("apply_to", foundSecurityGroup.ApplyTo)
	setError = d.Set("created_at", foundSecurityGroup.CreatedAt)

	if setError != nil {
		return diag.Errorf("[ERR]Security group could not be found")
	}

	return nil
}

// function to update the security group
func resourceSecurityGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupService := NewSecurityGroupService(apiClient)

	if d.HasChange("type") {
		return diag.Errorf("[ERR] Security group type can not be changed")
	}
	if d.HasChange("subnet_id") {
		return diag.Errorf("[ERR] Security group subnet can not be changed")
	}

	vpcId := d.Get("vpc_id").(string)
	hasChangedName := d.HasChange("name")
	hasChangeApplyTo := d.HasChange("apply_to")

	if hasChangedName {
		newName := d.Get("name").(string)
		_, err := securityGroupService.Rename(vpcId, d.Id(), newName)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while rename security group %s", err)
		}
	}

	if hasChangeApplyTo {
		applyToValue, ok := d.Get("apply_to").([]interface{})
		if !ok {
			applyToValue = []interface{}{}
		}
		applyTo := make([]string, len(applyToValue))
		for i, v := range applyToValue {
			applyTo[i] = v.(string)
		}

		_, err := securityGroupService.UpdateApplyTo(vpcId, d.Id(), applyTo)
		if err != nil {
			return diag.Errorf("[ERR] An error occurred while change apply to from security group %s: %s", d.Id(), err)
		}

		updateStateConf := &retry.StateChangeConf{
			Pending: []string{"PENDING", "UPDATING"},
			Target:  []string{"REALIZED", "ACTIVE"},
			Refresh: func() (interface{}, string, error) {
				findStorageModel := FindSecurityGroupDTO{
					ID:    d.Id(),
					VpcId: vpcId,
				}
				resp, err := securityGroupService.Find(findStorageModel)
				if err != nil {
					return 0, "", common.DecodeError(err)
				}
				return resp, resp.Status, nil
			},
			Timeout:        5 * time.Minute,
			Delay:          3 * time.Second,
			MinTimeout:     3 * time.Second,
			NotFoundChecks: 120,
		}
		_, err = updateStateConf.WaitForStateContext(ctx)
		if err != nil {
			return diag.Errorf("[Error] Waiting for security group (%s) to be updated: %s", d.Id(), err)
		}
	}
	return resourceSecurityGroupRead(ctx, d, m)
}

// function to delete the security group
func resourceSecurityGroupDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupService := NewSecurityGroupService(apiClient)

	log.Printf("[INFO] Deleting the security group %s", d.Id())

	vpcId, okVpcId := d.GetOk("vpc_id")

	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	_, err := securityGroupService.Delete(vpcId.(string), d.Id())

	deleteStateConf := &retry.StateChangeConf{
		Pending: []string{"DELETING"},
		Target:  []string{"SUCCESS"},
		Refresh: func() (interface{}, string, error) {
			findStorageModel := FindSecurityGroupDTO{
				ID:    d.Id(),
				VpcId: vpcId.(string),
			}
			resp, err := securityGroupService.Find(findStorageModel)
			if err != nil {
				// If the security group is not found, consider it deleted
				return 1, "SUCCESS", nil
			}

			return resp, resp.Status, nil
		},
		Timeout:        5 * time.Minute,
		Delay:          3 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 120,
	}
	_, err = deleteStateConf.WaitForStateContext(context.Background())
	if err != nil {
		return diag.Errorf("[Error] Waiting for security group (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}