package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// data_source_object_storage_access_key.go
func DataSourceAccessKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAccessKeyRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"region_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region name that's are the same with the region name in the S3 service. Currently, we have: HCM-01, HCM-02, HN-01, HN-02",
			},
			"credentials": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAccessKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	regionName := d.Get("region_name").(string)
	s3ServiceDetail := getServiceEnableRegion(service, vpcId, regionName)
	if s3ServiceDetail.S3ServiceId == "" {
		return diag.FromErr(fmt.Errorf(regionError, regionName))
	}
	keys, err := service.ListAccessKeys(vpcId, s3ServiceDetail.S3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}

	var formattedData []interface{}
	for _, key := range keys.Credentials {
		for _, cred := range key.Credentials {
			formattedData = append(formattedData, map[string]interface{}{
				"access_key": cred.AccessKey,
				"active":     cred.Active,
			})
		}
	}
	if err := d.Set("credentials", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting data: %v", err))
	}
	d.SetId(vpcId)

	return nil
}
