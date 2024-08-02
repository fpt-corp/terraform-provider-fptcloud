package fptcloud_vpc

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	common "terraform-provider-fptcloud/commons"
)

type DataSource struct{}

func NewDataSource() *schema.Resource {
	res := DataSource{}

	return &schema.Resource{
		ReadContext: res.Read,
		Schema:      dataSourceSchema,
	}
}

func (r DataSource) Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client)
	service := NewService(client)

	tenant, err := service.GetTenant(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	vpcName, ok := d.GetOk("name")
	if !ok {
		return diag.FromErr(fmt.Errorf("please enter vpc name"))
	}
	findVPC := FindVPCParam{Name: vpcName.(string)}
	vpc, err := service.FindVPC(ctx, tenant.Id, findVPC)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(vpc.Id)
	err = d.Set("name", vpc.Name)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("status", vpc.Status)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
