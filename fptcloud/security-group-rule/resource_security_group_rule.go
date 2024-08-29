package fptcloud_security_group_rule

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	common "terraform-provider-fptcloud/commons"
	"time"
)

// ResourceSecurityGroupRule function returns a schema.Resource that represents a security group rule.
// This can be used to create, read, and delete operations for a security group rule in the infrastructure.
func ResourceSecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a security group rule resource to manager rule in security group.",
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vpc id of the security group rule",
				ForceNew:    true,
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the security group rule",
				ForceNew:    true,
			},
			"direction": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The direction of the rule can be `INGRESS` or `EGRESS`.",
				ValidateFunc: validation.StringInSlice([]string{
					"INGRESS", "EGRESS",
				}, false),
				ForceNew: true,
			},
			"action": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The action of the rule can be allow or deny. When we set the `action = 'ALLOW'`, this is going to add a rule to allow traffic. Similarly, setting `action = 'DENY'` will deny the traffic.",
				ValidateFunc: validation.StringInSlice([]string{
					"ALLOW", "DENY",
				}, false),
				ForceNew: true,
			},
			"protocol": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The protocol of the security group rule include value `TCP`, `UDP`, `ICMP` or `ALL`",
				ValidateFunc: validation.StringInSlice([]string{
					"TCP", "UDP", "ICMP", "ALL",
				}, false),
				ForceNew: true,
			},
			"port_range": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The port or port range to open, if the protocol is `ALL` this field is required `ALL`",
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
			},
			"sources": {
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The sources of the rule, can be a CIDR notation or a IP address, pass `ALL` if you want to open for all IP",
				ForceNew:    true,
			},
			"ip_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ip type of the security group rule",
				ForceNew:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the security group rule",
				ForceNew:    true,
			},
			"security_group_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The security group id of the security group rule",
				ForceNew:    true,
			},
		},
		CreateContext: resourceSecurityGroupRuleCreate,
		ReadContext:   resourceSecurityGroupRuleRead,
		DeleteContext: resourceSecurityGroupRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

// function to create a new security group rule
func resourceSecurityGroupRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupRuleService := NewSecurityGroupRuleService(apiClient)

	createdModel := CreateSecurityGroupRuleDto{}
	vpcId, okVpcId := d.GetOk("vpc_id")

	if !okVpcId {
		return diag.Errorf("[ERR] VPC id is required")
	}

	if direction, ok := d.GetOk("direction"); ok {
		createdModel.Direction = direction.(string)
	}

	if action, ok := d.GetOk("action"); ok {
		createdModel.Action = action.(string)
	}

	if protocol, ok := d.GetOk("protocol"); ok {
		createdModel.Protocol = protocol.(string)
	}

	if portRange, ok := d.GetOk("port_range"); ok {
		createdModel.PortRange = portRange.(string)
	}

	if sources, ok := d.GetOk("sources"); ok {
		sourceList := sources.([]interface{})
		createdModel.Sources = make([]string, len(sourceList))
		for i, v := range sourceList {
			createdModel.Sources[i] = v.(string)
		}
	}

	if description, ok := d.GetOk("description"); ok {
		descriptionValue := description.(string)
		createdModel.Description = &descriptionValue
	}

	if securityGroupId, ok := d.GetOk("security_group_id"); ok {
		createdModel.SecurityGroupId = securityGroupId.(string)
	}

	securityGroupRuleId, err := securityGroupRuleService.Create(vpcId.(string), createdModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to create security group rule: %s", err)
	}

	d.SetId(securityGroupRuleId)
	setError := d.Set("vpc_id", vpcId.(string))
	if setError != nil {
		return diag.Errorf("[ERR] Failed to create security group rule")
	}

	// Waiting for status active
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"PENDING", "UPDATING"},
		Target:  []string{"REALIZED", "ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			resp, err := securityGroupRuleService.Find(vpcId.(string), securityGroupRuleId)
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
		return diag.Errorf("[Error] Waiting for security group rule (%s) to be created: %s", d.Id(), err)
	}

	return resourceSecurityGroupRuleRead(ctx, d, m)
}

// function to read a security group rule
func resourceSecurityGroupRuleRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupRuleService := NewSecurityGroupRuleService(apiClient)

	vpcId, okVpc := d.GetOk("vpc_id")
	if !okVpc {
		return diag.Errorf("[ERR] VPC id is required")
	}

	securityGroupRuleId, okId := d.GetOk("id")
	if !okId {
		return diag.Errorf("[ERR] Security group id is required")
	}

	foundSecurityGroupRule, err := securityGroupRuleService.Find(vpcId.(string), securityGroupRuleId.(string))
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve security group rule: %s", err)
	}

	// Set other attributes
	var setError error
	d.SetId(foundSecurityGroupRule.ID)
	setError = d.Set("vpc_id", foundSecurityGroupRule.VpcId)
	setError = d.Set("direction", foundSecurityGroupRule.Direction)
	setError = d.Set("action", foundSecurityGroupRule.Action)
	setError = d.Set("protocol", foundSecurityGroupRule.Protocol)
	setError = d.Set("port_range", foundSecurityGroupRule.PortRange)
	setError = d.Set("sources", foundSecurityGroupRule.Sources)
	setError = d.Set("ip_type", foundSecurityGroupRule.IpType)
	setError = d.Set("description", foundSecurityGroupRule.Description)
	setError = d.Set("security_group_id", foundSecurityGroupRule.SecurityGroupId)

	if setError != nil {
		return diag.Errorf("[ERR]Security group rule could not be found")
	}

	return nil
}

// function to delete a security group rule
func resourceSecurityGroupRuleDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	securityGroupRuleService := NewSecurityGroupRuleService(apiClient)

	log.Printf("[INFO] Deleting the security group rule %s", d.Id())

	vpcId, okVpcId := d.GetOk("vpc_id")

	if !okVpcId {
		return diag.Errorf("[ERR] Vpc id is required")
	}

	_, err := securityGroupRuleService.Delete(vpcId.(string), d.Id())
	if err != nil {
		return diag.Errorf("[ERR] An error occurred while trying to delete the security group rule %s", err)
	}

	deleteStateConf := &retry.StateChangeConf{
		Pending: []string{"DELETING"},
		Target:  []string{"SUCCESS"},
		Refresh: func() (interface{}, string, error) {
			resp, err := securityGroupRuleService.Find(vpcId.(string), d.Id())
			if err != nil {
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
		return diag.Errorf("[Error] Waiting for security group rule (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}
