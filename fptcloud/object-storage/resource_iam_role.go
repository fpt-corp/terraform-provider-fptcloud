package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// An IAM role is an identity that trusted IAM users may assume. What the role
// may then do is its inline policy, a separate resource; who may assume it is
// trusted_users, held here.
func ResourceIamRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIamRoleCreate,
		ReadContext:   resourceIamRoleRead,
		UpdateContext: resourceIamRoleUpdate,
		DeleteContext: resourceIamRoleDelete,
		Schema: map[string]*schema.Schema{
			"role_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the IAM role. The API has no rename, so changing it replaces the role",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC ID",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"trusted_users": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Description: "Names of the IAM users allowed to assume this role. A set, because the trust policy " +
					"holds principals in its own order. An update replaces the whole list, and a name the account " +
					"cannot resolve is dropped by the API rather than rejected - such a name will show as " +
					"permanent drift until it is removed from the configuration or the user is created",
			},
			"role_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM role ID assigned by the storage platform",
			},
			"arn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM role ARN",
			},
			"path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IAM role path",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the IAM role was created",
			},
		},
	}
}

func trustedUsersFromSchema(d *schema.ResourceData) []string {
	raw := d.Get("trusted_users").(*schema.Set).List()
	trustedUsers := make([]string, 0, len(raw))
	for _, name := range raw {
		trustedUsers = append(trustedUsers, name.(string))
	}
	return trustedUsers
}

func resourceIamRoleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	roleName := d.Get("role_name").(string)
	trustedUsers := trustedUsersFromSchema(d)

	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	resp := service.CreateIamRole(vpcId, s3ServiceDetail.S3ServiceId, roleName, trustedUsers)
	if !resp.Status {
		switch reconcileIamRole(service, vpcId, s3ServiceDetail.S3ServiceId, roleName, trustedUsers) {
		case createAdopted:
			// The role exists with the requested trust policy: an earlier attempt
			// committed and only its response was lost.
		case createConflict:
			return diag.Errorf("IAM role %s already exists trusting a different set of users: %s", roleName, resp.Message)
		default:
			return diag.FromErr(fmt.Errorf("error creating IAM role: %s", resp.Message))
		}
	}

	d.SetId(roleName)
	return resourceIamRoleRead(ctx, d, m)
}

func resourceIamRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	role := service.GetIamRole(vpcId, s3ServiceDetail.S3ServiceId, d.Id())
	if role == nil || role.RoleName == "" {
		d.SetId("")
		return nil
	}

	fields := map[string]interface{}{
		"role_name":     role.RoleName,
		"role_id":       role.RoleID,
		"arn":           role.Arn,
		"path":          role.Path,
		"created_at":    role.CreatedAt,
		"trusted_users": role.TrustedUsers,
	}
	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceIamRoleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	if !d.HasChange("trusted_users") {
		return resourceIamRoleRead(ctx, d, m)
	}

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	// One full-replace call covers both attaching and detaching: the trust
	// policy is rewritten from the list as a whole.
	resp := service.UpdateIamRoleTrustedUsers(vpcId, s3ServiceDetail.S3ServiceId, d.Id(), trustedUsersFromSchema(d))
	if !resp.Status {
		return diag.FromErr(fmt.Errorf("error updating trusted users of IAM role %s: %s", d.Id(), resp.Message))
	}

	return resourceIamRoleRead(ctx, d, m)
}

func resourceIamRoleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	// The API removes the role's policies before it deletes the role itself.
	if err := service.DeleteIamRole(vpcId, s3ServiceDetail.S3ServiceId, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
