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
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter tags by name",
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
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"scope_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vpc_names": {
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

	// Get name filter if provided
	name := ""
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	}

	// Get tags using service
	tagList, err := service.List(ctx, name)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set unique ID for datasource
	d.SetId(fmt.Sprintf("tags-%d", time.Now().Unix()))

	// Set tags in schema
	if err := d.Set("tags", flattenTags(tagList.Data.Tags)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// TagListResponse represents the API response structure
type TagListResponse struct {
	Status  bool     `json:"status"`
	Message string   `json:"message"`
	Data    TagsData `json:"data"`
}

type TagsData struct {
	Total int   `json:"total"`
	Tags  []Tag `json:"tags"`
}

// Tag represents a single tag in the response
type Tag struct {
	ID        string   `json:"id"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Color     []string `json:"color"`
	ScopeType string   `json:"scope_type"`
	VPCNames  []string `json:"vpc_names"`
}

// flattenTags converts the API response into a format suitable for the schema
func flattenTags(tags []Tag) []interface{} {
	var result []interface{}
	for _, tag := range tags {
		t := map[string]interface{}{
			"id":         tag.ID,
			"key":        tag.Key,
			"value":      tag.Value,
			"color":      tag.Color,
			"scope_type": tag.ScopeType,
			"vpc_names":  tag.VPCNames,
		}
		result = append(result, t)
	}
	return result
}
