package fptcloud_user

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

const (
	schemaId     = "id"
	schemaEmails = "emails"
	schemaIds    = "ids"
)

var dataSourceSchema = map[string]*schema.Schema{
	schemaEmails: {
		Type:        schema.TypeList,
		Required:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "List of user emails to resolve to IDs",
	},
	schemaIds: {
		Type:        schema.TypeList,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "Resolved user IDs in the same order as input emails",
	},
	schemaId: {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Static ID for the data source",
	},
}
