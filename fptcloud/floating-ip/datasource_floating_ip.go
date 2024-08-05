package fptcloud_floating_ip

import (
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"
	data_list "terraform-provider-fptcloud/commons/data-list"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceFloatingIp function returns a schema.Resource that represents a floating ip.
// This can be used to query and retrieve details about a specific floating ip in the infrastructure using its id or name.
func DataSourceFloatingIp() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description: strings.Join([]string{
			"Get information on a floating ip for use in other resources. This data source provides all of the floating ip properties as configured on your account.",
			"An error will be raised if the provided floating ip name does not exist in your account.",
		}, "\n\n"),
		RecordSchema:        floatingIpSchema(),
		ResultAttributeName: "floating_ops",
		FlattenRecord:       flattenFloatingIp,
		GetRecords:          getFloatingIps,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the floating ip",
			},
		},
	}

	return data_list.NewResource(dataListConfig)
}

func floatingIpSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vpc_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The vpc id of the floating ip",
		},
		"id": {
			Type:     schema.TypeString,
			Computed: true,
			//ValidateFunc: validation.NoZeroValues,
			//ExactlyOneOf: []string{"id", "ip_address"},
			Description: "The id of the floating ip",
		},
		"ip_address": {
			Type:     schema.TypeString,
			Computed: true,
			//ValidateFunc: validation.NoZeroValues,
			//ExactlyOneOf: []string{"id", "ip_address"},
			Description: "The ip address of the floating ip",
		},
		"nat_type": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The nat type of the floating ip",
		},
		//"instance": {
		//	Type:        schema.TypeList,
		//	Computed:    true,
		//	Description: "The instance of the floating ip",
		//},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The status of the floating ip",
		},
		"created_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The created at of the floating ip",
		},
	}
}

func flattenFloatingIp(floatingIp, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	s := floatingIp.(FloatingIp)

	flattened := map[string]interface{}{}
	flattened["id"] = s.ID
	flattened["ip_address"] = s.IpAddress
	flattened["nat_type"] = s.NatType
	flattened["status"] = s.Status
	flattened["created_at"] = s.CreatedAt

	return flattened, nil
}

func getFloatingIps(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	service := NewFloatingIpService(apiClient)

	vpcId, okVpcId := extra["vpc_id"].(string)
	if !okVpcId {
		return nil, fmt.Errorf("[ERR] Vpc id is required")
	}

	result, err := service.ListFloatingIp(vpcId)
	if err != nil || len(*result) == 0 {
		return nil, fmt.Errorf("[ERR] Failed to retrieve floating ip: %s", err)
	}

	var templates []interface{}
	for _, item := range *result {
		templates = append(templates, item)
	}

	return templates, nil
}
