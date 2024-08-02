package fptcloud_vpc

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	schemaId     = "id"
	schemaName   = "name"
	schemaStatus = "status"
)

var dataSourceSchema = map[string]*schema.Schema{
	schemaId: {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		ExactlyOneOf: []string{schemaId, schemaName, schemaStatus},
	},
	schemaName: {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		ExactlyOneOf: []string{schemaId, schemaName, schemaStatus},
		Description:  "The name of VPC",
	},
	schemaStatus: {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.NoZeroValues,
		ExactlyOneOf: []string{schemaId, schemaName, schemaStatus},
		Description:  "The status of VPC",
	},
}

var resourceSchema = map[string]*schema.Schema{
	schemaName: {
		Type:        schema.TypeString,
		Required:    true,
		Description: "name",
		ForceNew:    true,
	},
}
