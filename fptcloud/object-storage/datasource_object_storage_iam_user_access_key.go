package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The listing never carries a secret - the API returns one only at create - so
// this data source can report which keys exist and when, but never their value.
func DataSourceIamUserAccessKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIamUserAccessKeyRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VPC ID",
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"user_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the IAM user whose access keys to list",
			},
			"list_access_key": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The IAM user's access keys. Secrets are never included",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_key_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"created_at": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"created_by": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"total": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "How many access keys the IAM user holds",
			},
		},
	}
}

func dataSourceIamUserAccessKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)

	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	userName := d.Get("user_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}

	list, err := service.ListIamUserAccessKeys(vpcId, s3ServiceDetail.S3ServiceId, userName)
	if err != nil {
		return diag.FromErr(err)
	}

	// A user with no keys is a normal state - it simply cannot sign a request
	// yet - so an empty list is reported rather than treated as an error.
	formattedData := make([]interface{}, 0, len(list.AccessKeys))
	for _, accessKey := range list.AccessKeys {
		formattedData = append(formattedData, map[string]interface{}{
			"access_key_id": accessKey.AccessKeyID,
			"created_at":    accessKey.CreatedAt,
			"created_by":    accessKey.CreatedBy,
		})
	}
	if err := d.Set("list_access_key", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting list_access_key: %s", err))
	}
	if err := d.Set("total", list.Total); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(userName)

	return nil
}
