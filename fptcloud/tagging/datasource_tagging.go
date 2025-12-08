package fptcloud_tagging

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// DataSourceTagging returns the schema for the tagging data source
func DataSourceTagging() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTaggingRead,
		Schema: map[string]*schema.Schema{
			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter tags by key",
			},
			"value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter tags by value",
			},
			"scope_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter tags by scope type",
			},
			"tags": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"color": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"scope_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"resource_scopes": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceTaggingRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewTaggingService(client)

	// Get filters if provided
	key := ""
	if v, ok := d.GetOk("key"); ok {
		key = v.(string)
	}

	value := ""
	if v, ok := d.GetOk("value"); ok {
		value = v.(string)
	}

	// Get tags using service
	tagList, err := service.List(ctx, key, value)
	if err != nil {
		return diag.FromErr(err)
	}

	// Validate response
	if tagList == nil {
		return diag.Errorf("Received nil response from tag list")
	}

	// Set unique ID for datasource
	d.SetId(fmt.Sprintf("tags-%d", time.Now().Unix()))

	// Set tags in schema
	if err := d.Set("tags", flattenTags(tagList.Data)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// TagListResponse represents the API response structure
type TagListResponse struct {
	Status  bool     `json:"status"`
	Message string   `json:"message"`
	Data    *ListTag `json:"data"`
}
type ListTag struct {
	Data  []Tag `json:"data"`
	Total int16 `json:"total"`
}

// Tag represents a single tag in the response
type Tag struct {
	ID             string   `json:"id"`
	Key            *string  `json:"key"`
	Value          *string  `json:"value"`
	Color          *string  `json:"color"`
	ScopeType      string   `json:"scope_type"`
	ResourceScopes []string `json:"resource_scopes"`
}

type TagResponse struct {
	TagID string `json:"tag_id"`
}

// TagGetResponse represents the API response structure for GetTag
type TagGetResponse struct {
	Status  bool      `json:"status"`
	Message string    `json:"message"`
	Data    Tag `json:"data"`
}

// flattenTags converts the API response into a format suitable for the schema
func flattenTags(tags []Tag) []interface{} {
	var result []interface{}
	for _, tag := range tags {
		value := ""
		if tag.Value != nil {
			value = *tag.Value
		}

		key := ""
		if tag.Key != nil {
			key = *tag.Key
		}

		color := ""
		if tag.Color != nil {
			color = *tag.Color
		}

		t := map[string]interface{}{
			"id":              tag.ID,
			"key":             key,
			"value":           value,
			"color":           color,
			"scope_type":      tag.ScopeType,
			"resource_scopes": tag.ResourceScopes,
		}
		result = append(result, t)
	}
	return result
}
