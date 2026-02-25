package fptcloud_tagging

import (
	"context"
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	scopeTypeProject = "PROJECT"
	scopeTypeVPC     = "VPC"
)

// DataSourceTaggingResource returns the schema for the tagging resource (projects/vpcs) data source
func DataSourceTaggingResource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTaggingResourceRead,
		Schema: map[string]*schema.Schema{
			"scope_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Filter by scope: PROJECT or VPC.",
				ValidateFunc: validation.StringInSlice([]string{scopeTypeProject, scopeTypeVPC}, false),
			},
			"resource_names": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Optional set of substrings to filter items by name. Partial match, case-insensitive.",
			},
			"items": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of projects or VPCs after filtering by scope_type and resource_names. Empty if no data.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":   {Type: schema.TypeString, Computed: true},
						"name": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceTaggingResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewTaggingService(client)

	scopeType := strings.TrimSpace(strings.ToUpper(d.Get("scope_type").(string)))

	resp, err := service.ListProjectVpc(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("tagging-resource-%d", time.Now().Unix()))

	var items []interface{}

	if resp.Data == nil {
		_ = d.Set("items", items)
		return nil
	}

	var nameFilters []string
	if v, ok := d.GetOk("resource_names"); ok {
		for _, name := range v.(*schema.Set).List() {
			nameFilters = append(nameFilters, strings.ToLower(strings.TrimSpace(name.(string))))
		}
	}
	hasNameFilter := len(nameFilters) > 0

	// matchesNameFilter returns true if itemName contains any filter (partial match, case-insensitive)
	matchesNameFilter := func(itemName string) bool {
		lower := strings.ToLower(itemName)
		for _, f := range nameFilters {
			if strings.Contains(lower, f) {
				return true
			}
		}
		return false
	}

	if scopeType == scopeTypeProject {
		list := resp.Data.Projects
		if list == nil {
			list = []TagProjectVpcProject{}
		}
		for _, p := range list {
			if hasNameFilter && !matchesNameFilter(p.Name) {
				continue
			}
			items = append(items, map[string]interface{}{
				"id":   p.ID,
				"name": p.Name,
			})
		}
	} else {
		list := resp.Data.Vpcs
		if list == nil {
			list = []TagProjectVpcVpc{}
		}
		for _, v := range list {
			if hasNameFilter && !matchesNameFilter(v.Name) {
				continue
			}
			items = append(items, map[string]interface{}{
				"id":   v.ID,
				"name": v.Name,
			})
		}
	}

	if items == nil {
		items = []interface{}{}
	}
	if err := d.Set("items", items); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
