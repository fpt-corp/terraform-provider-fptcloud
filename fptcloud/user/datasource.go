package fptcloud_user

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"
	fptcloud_vpc "terraform-provider-fptcloud/fptcloud/vpc"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type DataSource struct{}

func NewDataSource() *schema.Resource {
	ds := DataSource{}
	return &schema.Resource{
		ReadContext: ds.Read,
		Schema:      dataSourceSchema,
	}
}

func (r DataSource) Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client)
	service := NewService(client)

	tenant, err := fptcloud_vpc.NewService(client).GetTenant(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	tenantId := tenant.Id

	rawEmails := d.Get(schemaEmails).([]interface{})
	if len(rawEmails) == 0 {
		return diag.Errorf("emails must not be empty")
	}

	// Get all users from org (GET request)
	users, err := service.ListUsers(ctx, tenantId)
	if err != nil {
		return diag.FromErr(err)
	}

	// Build map for quick lookup (case-insensitive)
	byEmail := make(map[string]string)
	for _, u := range users {
		email := u.Email
		if email != "" {
			byEmail[email] = u.Id
		}
	}

	// Resolve emails to IDs in the same order as input
	resolved := make([]string, len(rawEmails))
	for i, v := range rawEmails {
		email := v.(string)
		id, ok := byEmail[email]
		if !ok {
			return diag.Errorf("user with email '%s' not found in org %s", email, tenantId)
		}
		resolved[i] = id
	}

	if err := d.Set(schemaIds, resolved); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("users-%s", tenantId))
	return nil
}
