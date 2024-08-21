package fptcloud_image

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/data-list"
)

// DataSourceImage function returns a schema.Resource that represents a Storage.
// This can be used to query and retrieve details about a specific Image in the infrastructure.
func DataSourceImage() *schema.Resource {
	dataListConfig := &data_list.ResourceConfig{
		Description:         "Retrieves information about the image that fpt cloud supports, with the ability to filter the results.",
		RecordSchema:        imageSchema(),
		ResultAttributeName: "images",
		FlattenRecord:       flattenImage,
		GetRecords:          getImages,
		ExtraQuerySchema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The vpc id of the image",
			},
		},
	}

	return data_list.NewResource(dataListConfig)

}
func imageSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The id of the image",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the image",
		},
		"catalog": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The catalog of the image",
		},
		"is_gpu": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "The image is gpu or not",
		},
	}
}

func flattenImage(image, _ interface{}, _ map[string]interface{}) (map[string]interface{}, error) {

	s := image.(Image)

	flattened := map[string]interface{}{}
	flattened["name"] = s.Name
	flattened["id"] = s.ID
	flattened["catalog"] = s.Catalog
	flattened["is_gpu"] = s.IsGpu

	return flattened, nil
}

func getImages(m interface{}, extra map[string]interface{}) ([]interface{}, error) {
	apiClient := m.(*common.Client)
	imageService := NewImageService(apiClient)

	vpcId, ok := extra["vpc_id"].(string)
	if !ok {
		return nil, fmt.Errorf("[ERR] vpc id is required")
	}

	images, err := imageService.ListImage(vpcId)
	if err != nil {
		return nil, fmt.Errorf("[ERR] Failed to retrieve images: %s", err)
	}

	var templates []interface{}
	for _, image := range *images {
		templates = append(templates, image)
	}

	return templates, nil
}
