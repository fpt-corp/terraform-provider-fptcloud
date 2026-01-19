package fptcloud_vpc

import (
	"context"
	"log"
	"strings"
	"time"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Resource struct{}

func NewResource() *schema.Resource {
	res := Resource{}

	return &schema.Resource{
		Description:   "Provides a FPT Cloud VPC resource. This can be used to create and manage VPCs in the infrastructure.",
		CreateContext: res.Create,
		ReadContext:   res.Read,
		DeleteContext: res.Delete,
		Schema:        resourceSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func (r Resource) Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*common.Client)
	service := NewService(apiClient)

	tenant, err := service.GetTenant(ctx)
	if err != nil {
		return diag.Errorf("[ERR] Failed to get tenant from provider configuration: %s", err)
	}

	orgId := tenant.Id
	name := d.Get(schemaName).(string)

	createModel := CreateVPCDTO{
		Name: name,
	}

	if hypervisor, ok := d.GetOk(schemaHypervisor); ok {
		createModel.Hypervisor = hypervisor.(string)
	}

	if owners, ok := d.GetOk(schemaOwners); ok {
		ownersList := owners.([]interface{})
		createModel.Owners = make([]string, len(ownersList))
		for i, owner := range ownersList {
			ownerStr := owner.(string)
			// Check if owner is an email (contains @) or UUID
			if strings.Contains(ownerStr, "@") {
				// Convert email to user ID
				userId, err := service.GetUserByEmail(ctx, orgId, ownerStr)
				if err != nil {
					return diag.Errorf("[ERR] Failed to get user ID for email '%s': %s", ownerStr, err)
				}
				createModel.Owners[i] = userId
			} else {
				// Assume it's already a user ID (UUID)
				createModel.Owners[i] = ownerStr
			}
		}
	}

	if projectIaasId, ok := d.GetOk(schemaProjectIaasId); ok {
		createModel.ProjectIaasId = projectIaasId.(string)
	}

	if subnetName, ok := d.GetOk(schemaSubnetName); ok {
		createModel.SubnetName = subnetName.(string)
	}

	if networkType, ok := d.GetOk(schemaNetworkType); ok {
		createModel.NetworkType = networkType.(string)
	}

	if cidr, ok := d.GetOk(schemaCIDR); ok {
		createModel.CIDR = cidr.(string)
	}

	if gatewayIp, ok := d.GetOk(schemaGatewayIp); ok {
		createModel.GatewayIp = gatewayIp.(string)
	}

	if staticIpPoolFrom, ok := d.GetOk(schemaStaticIpPoolFrom); ok {
		createModel.StaticIpPoolFrom = staticIpPoolFrom.(string)
	}

	if staticIpPoolTo, ok := d.GetOk(schemaStaticIpPoolTo); ok {
		createModel.StaticIpPoolTo = staticIpPoolTo.(string)
	}

	if tagIds, ok := d.GetOk(schemaTagIds); ok {
		createModel.TagIds = expandTagIDs(tagIds.(*schema.Set))
	}

	log.Printf("[INFO] Creating VPC: %s in org/tenant: %s", name, orgId)

	result, err := service.CreateVPC(ctx, orgId, createModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to create VPC: %s", err)
	}

	if result == nil || result.Id == "" {
		return diag.Errorf("[ERR] VPC creation returned empty ID")
	}

	d.SetId(result.Id)

	// Waiting for VPC status to be ENABLED
	log.Printf("[INFO] Waiting for VPC %s to reach ENABLED status", result.Id)
	createStateConf := &retry.StateChangeConf{
		Pending: []string{"PENDING", "CREATING", "UPDATING"},
		Target:  []string{"ENABLED"},
		Refresh: func() (interface{}, string, error) {
			findModel := FindVPCParam{
				Name: name,
			}
			vpc, err := service.FindVPC(ctx, orgId, findModel)
			if err != nil {
				return nil, "", err
			}
			status := vpc.Status
			if status == "" {
				status = "PENDING"
			}
			return vpc, status, nil
		},
		Timeout:        time.Duration(apiClient.Timeout) * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     10 * time.Second,
		NotFoundChecks: 20,
	}
	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("[ERR] Error waiting for VPC (%s) to become ENABLED: %s", result.Id, err)
	}

	log.Printf("[INFO] VPC %s is now ENABLED", result.Id)

	return r.Read(ctx, d, meta)
}

func expandTagIDs(tagSet *schema.Set) []string {
	tagIds := make([]string, 0, tagSet.Len())
	for _, tag := range tagSet.List() {
		tagIds = append(tagIds, tag.(string))
	}
	return tagIds
}

func (r Resource) Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*common.Client)
	service := NewService(apiClient)

	log.Printf("[INFO] Retrieving VPC: %s", d.Id())

	// Resolve tenant/org again from provider configuration to read VPC
	tenant, err := service.GetTenant(ctx)
	if err != nil {
		return diag.Errorf("[ERR] Failed to get tenant from provider configuration: %s", err)
	}

	orgId := tenant.Id

	// Get VPC name from state (API only supports finding by name, not ID)
	vpcName, ok := d.GetOk(schemaName)
	if !ok {
		return diag.Errorf("[ERR] VPC name not found in state")
	}

	findModel := FindVPCParam{
		Name: vpcName.(string),
	}

	vpc, err := service.FindVPC(ctx, orgId, findModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve VPC: %s", err)
	}

	if vpc == nil {
		log.Printf("[WARN] VPC %s not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.SetId(vpc.Id)

	if err := d.Set(schemaOrgId, orgId); err != nil {
		return diag.Errorf("[ERR] Failed to set 'org_id': %s", err)
	}

	if err := d.Set(schemaName, vpc.Name); err != nil {
		return diag.Errorf("[ERR] Failed to set 'name': %s", err)
	}

	if err := d.Set(schemaStatus, vpc.Status); err != nil {
		return diag.Errorf("[ERR] Failed to set 'status': %s", err)
	}

	return nil
}

func (r Resource) Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Backend API for VPC delete is not implemented yet.
	// Return a clear error so users know deletion is not supported.
	return diag.Errorf("[ERR] Delete VPC is not supported yet")
}
