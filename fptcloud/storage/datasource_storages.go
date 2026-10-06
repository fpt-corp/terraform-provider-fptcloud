package fptcloud_storage

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	common "terraform-provider-fptcloud/commons"
)

// DataSourceStorages function returns a schema.Resource that represents a list of storages.
// Its ReadContext fetches every storage in a VPC, walking the portal's paginated list endpoint
// instead of calling Find once per storage.
func DataSourceStorages() *schema.Resource {
	return &schema.Resource{
		Description: strings.Join([]string{
			"Use this data source to retrieve all storages in a VPC, including root, local and external disks.",
			"Results are retrieved page by page, up to `page_size` storages per API call, so listing a large VPC never requires one request per storage. The attributes of each item match those of the `fptcloud_storage` data source.",
		}, "\n\n"),
		ReadContext: dataSourceStoragesRead,
		Schema:      dataSourceStoragesSchema,
	}
}

func dataSourceStoragesRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	storageService := NewStorageService(apiClient)

	listModel := StorageListDTO{
		VpcId:    d.Get("vpc_id").(string),
		PageSize: d.Get("page_size").(int),
	}

	storages, err := storageService.ListAll(listModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve storages: %s", err)
	}

	formatted := make([]interface{}, 0, len(storages))
	for _, st := range storages {
		// The singular endpoint reports the display name when one is set.
		name := st.Name
		if st.DisplayName != nil && *st.DisplayName != "" {
			name = *st.DisplayName
		}

		tagIds := make([]interface{}, 0, len(st.Tags))
		for _, tag := range st.Tags {
			tagIds = append(tagIds, tag.ID)
		}

		encrypted := false
		if st.Encrypted != nil {
			encrypted = bool(*st.Encrypted)
		}

		formatted = append(formatted, map[string]interface{}{
			"id":                st.ID,
			"vpc_id":            st.VpcId,
			"name":              name,
			"type":              st.StorageType,
			"size_gb":           sizeMbToGb(st.SizeMb),
			"storage_policy":    stringFromPtr(st.StoragePolicyName),
			"storage_policy_id": stringFromPtr(st.StoragePolicyId),
			"instance_id":       stringFromPtr(st.InstanceId),
			"tag_ids":           tagIds,
			"created_at":        st.CreatedAt,
			"status":            st.Status,
			"instance_name":     stringFromPtr(st.InstanceName),
			"description":       stringFromPtr(st.Description),
			"disk_id":           stringFromPtr(st.DiskId),
			"encrypted":         encrypted,
			"zone_id":           stringFromPtr(st.ZoneId),
		})
	}

	if err := d.Set("storages", formatted); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(listModel.VpcId)
	return nil
}

// sizeMbToGb converts the list endpoint's MB size to GB, rounding up the same
// way the singular endpoint computes size_gb.
func sizeMbToGb(sizeMb int) int {
	return (sizeMb + 1023) / 1024
}

func stringFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
