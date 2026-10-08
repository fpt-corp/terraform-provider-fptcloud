package fptcloud_instance

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	common "terraform-provider-fptcloud/commons"
)

// DataSourceInstances function returns a schema.Resource that represents a list of instances.
// Its ReadContext fetches every instance in a VPC, walking the portal's paginated list endpoint
// instead of calling Find once per instance.
func DataSourceInstances() *schema.Resource {
	return &schema.Resource{
		Description: strings.Join([]string{
			"Use this data source to retrieve all instances in a VPC.",
			"Results are retrieved page by page, up to `page_size` instances per API call, so listing a large VPC never requires one request per instance. The attributes of each item match those of the `fptcloud_instance` data source wherever the list API provides the same information.",
			"~> **Note:** `subnet_id`, `storage_size_gb`, `storage_policy` and `security_group_ids` are not included in the list results. Use the `fptcloud_instance` data source to read them for a specific instance.",
		}, "\n\n"),
		ReadContext: dataSourceInstancesRead,
		Schema:      dataSourceInstancesSchema,
	}
}

func dataSourceInstancesRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	instanceService := NewInstanceService(apiClient)

	listModel := InstanceListDTO{
		VpcId:    d.Get("vpc_id").(string),
		PageSize: d.Get("page_size").(int),
	}

	instances, err := instanceService.ListAll(listModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve instances: %s", err)
	}

	formatted := make([]interface{}, 0, len(instances))
	for _, inst := range instances {
		flavorName := ""
		if inst.Flavor != nil {
			flavorName = inst.Flavor.Name
		}

		tagIds := make([]interface{}, 0, len(inst.VmTags))
		for _, tag := range inst.VmTags {
			tagIds = append(tagIds, tag.ID)
		}

		formatted = append(formatted, map[string]interface{}{
			"id":                stringValue(inst.ID),
			"vpc_id":            stringValue(inst.VpcId),
			"name":              inst.Name,
			"guest_os":          stringValue(inst.GuestOs),
			"host_name":         stringValue(inst.HostName),
			"status":            inst.Status,
			"private_ip":        stringValue(inst.IpAddress),
			"public_ip":         stringValue(inst.IpPublic),
			"memory_mb":         inst.MemoryMb,
			"cpu_number":        inst.CpuNumber,
			"flavor_id":         stringValue(inst.FlavorId),
			"flavor_name":       flavorName,
			"instance_group_id": stringValue(inst.VmGroupId),
			"tag_ids":           tagIds,
			"gpu_name":          stringValue(inst.GpuName),
			"is_nvme":           inst.IsNvme,
			"created_at":        inst.CreatedAt,
			"ipv6_address":      stringValue(inst.Ipv6Address),
			"subnet_name":       stringValue(inst.NetworkName),
			"updated_at":        inst.UpdatedAt,
		})
	}

	if err := d.Set("instances", formatted); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(listModel.VpcId)
	return nil
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
