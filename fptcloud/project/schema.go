package fptcloud_project

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	schemaId   = "id"
	schemaName = "name"
)

var dataSourceSchema = map[string]*schema.Schema{
	schemaName: {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The name of the project",
	},
	schemaId: {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The ID of the project",
	},
}

