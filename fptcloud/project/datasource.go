package fptcloud_project

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

	projectName, ok := d.GetOk("name")
	if !ok {
		return diag.FromErr(fmt.Errorf("please enter project name"))
	}

	findProject := FindProjectParam{Name: projectName.(string)}
	project, err := service.FindProject(ctx, tenant.Id, findProject)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(project.Id)
	err = d.Set("name", project.Name)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("id", project.Id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

