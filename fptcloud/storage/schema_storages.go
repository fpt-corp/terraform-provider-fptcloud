package fptcloud_storage

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	common "terraform-provider-fptcloud/commons"
)

// dataSourceStoragesSchema describes the schema of the fptcloud_storages data source.
// It requires a vpc_id and optionally a page_size (max 25); every storage in the VPC is
// returned in the `storages` list. Item attributes keep the names used by the
// fptcloud_storage data source wherever the list endpoint returns the same value.
var dataSourceStoragesSchema = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the VPC whose storages are retrieved.",
	},
	"page_size": {
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      common.ListPageSize,
		ValidateFunc: validation.IntBetween(1, common.ListPageSize),
		Description:  "The number of storages requested per API call. Must be between `1` and `25`; defaults to `25`. The provider requests successive pages until every storage has been retrieved.",
	},
	"storages": {
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The storages in the VPC.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the storage.",
				},
				"vpc_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the VPC the storage belongs to.",
				},
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the storage. May be empty for a root or local disk created together with its instance.",
				},
				"type": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The type of the storage: `ROOT`, `LOCAL` or `EXTERNAL`.",
				},
				"size_gb": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The size of the storage, in GB.",
				},
				"storage_policy": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the storage policy.",
				},
				"storage_policy_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the storage policy.",
				},
				"instance_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the instance the storage is attached to. Empty when the storage is not attached.",
				},
				"tag_ids": {
					Type:        schema.TypeList,
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "The IDs of the tags attached to the storage.",
				},
				"created_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The time the storage was created.",
				},
				"status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The current status of the storage.",
				},
				"instance_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the instance the storage is attached to. Empty when the storage is not attached.",
				},
				"description": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The description of the storage.",
				},
				"disk_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the disk backing the storage on the underlying platform.",
				},
				"encrypted": {
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Whether the storage is encrypted.",
				},
				"zone_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the availability zone of the storage, when the VPC uses zones.",
				},
			},
		},
	},
}
