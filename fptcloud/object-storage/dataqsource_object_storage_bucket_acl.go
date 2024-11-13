package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceBucketAcl() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBucketAclRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VPC ID",
			},
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the bucket to config the ACL",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"canned_acl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Access Control List (ACL) status of the bucket which can be one of the following values: private, public-read, default is private",
			},
			"status": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "The status after configuring the bucket ACL",
			},
			"bucket_acl": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"owner": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"display_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"grants": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"grantee": {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"display_name": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"id": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"type": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"permission": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceBucketAclRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	bucketName := d.Get("bucket_name").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf("region %s is not enabled", regionName))
	}
	r := service.GetBucketAcl(vpcId, s3ServiceDetail.S3ServiceId, bucketName)
	if !r.Status {
		return diag.Errorf("failed to get bucket ACL for bucket %s", bucketName)
	}
	bucketAcl := []interface{}{
		map[string]interface{}{
			"owner": []interface{}{
				map[string]interface{}{
					"display_name": r.Owner.DisplayName,
					"id":           r.Owner.ID,
				},
			},
			"grants": func() []interface{} {
				grants := make([]interface{}, len(r.Grants))
				for i, grant := range r.Grants {
					grants[i] = map[string]interface{}{
						"grantee": []interface{}{
							map[string]interface{}{
								"display_name": grant.Grantee.DisplayName,
								"id":           grant.Grantee.ID,
								"type":         grant.Grantee.Type,
							},
						},
						"permission": grant.Permission,
					}
				}
				return grants
			}(),
		},
	}
	d.SetId(bucketName)
	if err := d.Set("bucket_acl", bucketAcl); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	if err := d.Set("canned_acl", r.CannedACL); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}
	if err := d.Set("status", r.Status); err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}

	return nil
}
