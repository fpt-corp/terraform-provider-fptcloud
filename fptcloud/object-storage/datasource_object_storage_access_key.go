package fptcloud_object_storage

import (
	"context"
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
				Description: "The region name to create the access key",
			},
			"access_keys": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
	_, err := service.ListAccessKeys(vpcId, s3ServiceDetail.S3ServiceId)
	if err != nil {
		return diag.FromErr(err)
	}

	// if len(accessKeys.Credentials) > 0 {
	// 	d.SetId(fmt.Sprintf("access_keys_%d", len(accessKeys)))
	// 	if err := d.Set("access_keys", flattenAccessKeys(accessKeys)); err != nil {
	// 		return diag.FromErr(err)
	// 	}
	// }

	return nil
}

// func flattenAccessKeys(accessKeys AccessKey) []interface{} {
// 	var result []interface{}
// 	for _, ak := range accessKeys.Credentials {
// 		for _, cred := range ak.Credentials {
// 			credMap := map[string]interface{}{
// 				"id":          cred.ID,
// 				"credentials": flattenCredentials(cred.Credentials),
// 			}
// 			result = append(result, credMap)
// 		}
// 	}
// 	return result
// }

func flattenCredentials(credentials []struct {
	AccessKey   string      `json:"accessKey"`
	Active      bool        `json:"active"`
	CreatedDate interface{} `json:"createdDate"`
}) []interface{} {
	var result []interface{}
	for _, cred := range credentials {
		credMap := map[string]interface{}{
			"access_key": cred.AccessKey,
			"active":     cred.Active,
		}
		result = append(result, credMap)
	}
	return result
}
