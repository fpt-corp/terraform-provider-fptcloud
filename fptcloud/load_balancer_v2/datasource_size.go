package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceSizes() *schema.Resource {
	return &schema.Resource{
		ReadContext: listSizes,
		Schema:      dataSourceSizes,
	}
}

func listSizes(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	response, err := service.ListSizes(vpcId)
	if err != nil {
		return diag.FromErr(err)
	}
	sizes := response.Sizes
	var formattedData []interface{}
	for _, size := range sizes {
		formattedData = append(formattedData, map[string]interface{}{
			"id":                     size.Id,
			"name":                   size.Name,
			"vip_amount":             size.VipAmount,
			"active_connection":      size.ActiveConnection,
			"application_throughput": size.ApplicationThroughput,
		})
	}
	if err := d.Set("sizes", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting size list: %v", err))
	}
	d.SetId(vpcId)

	return nil
}
