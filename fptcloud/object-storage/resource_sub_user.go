package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSubUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSubUserCreate,
		ReadContext:   resourceSubUserRead,
		DeleteContext: resourceSubUserDelete,
		Schema: map[string]*schema.Schema{
			"role": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"user_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"active": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"access_keys": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceSubUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	subUserId := d.Get("user_id").(string)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)

	req := SubUser{
		Role:   d.Get("role").(string),
		UserId: subUserId,
	}

	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	err := objectStorageService.CreateSubUser(req, vpcId, s3ServiceDetail.S3ServiceId)
	if !err.Status {
		return diag.FromErr(fmt.Errorf("error creating sub-user: %s", err.Message))
	}

	// Set the resource ID after successful creation
	d.SetId(subUserId)
	if err := d.Set("user_id", subUserId); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}

	return nil
}

func resourceSubUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	subUserId := d.Id()
	subUser := objectStorageService.DetailSubUser(vpcId, s3ServiceDetail.S3ServiceId, subUserId)
	if subUser == nil {
		d.SetId("")
		return diag.Errorf("sub-user with ID %s not found", subUserId)
	}

	if subUser.UserID == "" {
		d.SetId("")
		return diag.Errorf("sub-user with ID %s not found", subUserId)
	}

	if err := d.Set("user_id", subUser.UserID); err != nil {
		return diag.FromErr(err)
	}
	if subUser.Arn != "" {
		if err := d.Set("arn", subUser.Arn); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("role", subUser.Role); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("active", subUser.Active); err != nil {
		return diag.FromErr(err)
	}
	if subUser.CreatedAt != "" {
		if err := d.Set("created_at", subUser.CreatedAt); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("access_keys", subUser.AccessKeys); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSubUserDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	client := m.(*common.Client)
	objectStorageService := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	s3ServiceDetail := getServiceEnableRegion(objectStorageService, vpcId, d.Get("region_name").(string))
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, d.Get("region_name").(string)))
	}
	err := objectStorageService.DeleteSubUser(vpcId, s3ServiceDetail.S3ServiceId, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
