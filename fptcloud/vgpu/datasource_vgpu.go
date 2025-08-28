package fptcloud_vgpu

import (
	"fmt"
	common "terraform-provider-fptcloud/commons"
	data_list "terraform-provider-fptcloud/commons/data-list"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceVGpu function returns a schema.Resource that represents a vGPU.
// This can be used to query and retrieve details about a specific vGPU in the infrastructure.
func DataSourceVGpu() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description:         "Retrieves information about the vGPU that fpt cloud supports, with the ability to filter the results.",
		RecordSchema:        vgpuSchema(),
		ResultAttributeName: "vgpus",
		FlattenRecord:       flattenVGpu,
		GetRecords:          getVGpus,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the vGPU",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func vgpuSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the vGPU",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the vGPU",
		},
		"display_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The display name of the vGPU",
		},
		"created_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The creation date of the vGPU",
		},
		"memory": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The memory size (GB) of the vGPU",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The status of the vGPU",
		},
		"is_dedicated": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the vGPU is dedicated",
		},
		"service_type_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Optional:    true,
			Description: "The service type ID of the vGPU",
		},
		"platform": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The platform of the vGPU",
		},
		"parent_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The parent ID of the vGPU",
		},
		"enable_nvme": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether NVMe is enabled for the vGPU",
		},
	}
}

func flattenVGpu(vgpu, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := vgpu.(VGpu)

	flattened := map[string]interface{}{}
	flattened["id"] = s.ID
	flattened["name"] = s.Name
	flattened["display_name"] = s.DisplayName
	flattened["created_at"] = s.CreatedAt
	flattened["memory"] = s.Memory
	flattened["status"] = s.Status
	flattened["is_dedicated"] = s.IsDedicated
	if s.ServiceTypeID != nil {
		flattened["service_type_id"] = *s.ServiceTypeID
	}
	flattened["platform"] = s.Platform
	flattened["parent_id"] = s.ParentID
	flattened["enable_nvme"] = s.EnableNvme

	return flattened, nil
}

func getVGpus(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	vgpuService := NewVGpuService(apiClient)

	vpcId, ok := extra["vpc_id"].(string)
	if !ok {
		return nil, fmt.Errorf("[ERR] vpc id is required")
	}

	vgpus, err := vgpuService.ListVGpu(vpcId)
	if err != nil {
		return nil, fmt.Errorf("[ERR] Failed to retrieve vGPUs: %s", err)
	}

	var templates []interface{}
	for _, vgpu := range *vgpus {
		templates = append(templates, vgpu)
	}

	return templates, nil
}
