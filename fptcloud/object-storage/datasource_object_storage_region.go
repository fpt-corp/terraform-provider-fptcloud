package fptcloud_object_storage

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceS3ServiceEnableResponse() *schema.Resource {
	return &schema.Resource{
		ReadContext: resourceS3ServiceEnableResponseRead,
		Schema: map[string]*schema.Schema{
			"s3_enable_services": {
				Type: schema.TypeList,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"s3_service_name": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of the S3 service also known as region_name, could be used to create/delete another resources",
					},
					"s3_service_id": {
						Type:     schema.TypeString,
						Required: true,
					},
					"s3_platform": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The platform of the S3 service",
					},
				}},
				Computed: true,
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the VPC",
			},
		},
	}
}

func resourceS3ServiceEnableResponseRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewObjectStorageService(client)
	vpcId := d.Get("vpc_id").(string)
	listRegion := service.CheckServiceEnable(vpcId)
	if listRegion.Total == 0 {
		return diag.FromErr(fmt.Errorf("no region is enabled"))
	}
	var formattedData []interface{}

	for _, item := range listRegion.Data {
		formattedData = append(formattedData, map[string]interface{}{
			"s3_service_name": item.S3ServiceName,
			"s3_service_id":   item.S3ServiceID,
			"s3_platform":     item.S3Platform,
		})
	}
	if listRegion.Data == nil {
		return diag.FromErr(fmt.Errorf("failed to get response from API"))
	}
	if listRegion.Total == 0 {
		return diag.FromErr(fmt.Errorf("no region is enabled"))
	}
	if err := d.Set("s3_enable_services", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting data: %v", err))
	}
	d.SetId(vpcId)
	return nil
}
