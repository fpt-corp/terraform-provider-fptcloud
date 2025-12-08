package fptcloud_tagging

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	common "terraform-provider-fptcloud/commons"
)

// CreateTagInput represents input parameters for creating a tag
type CreateTagInput struct {
	Key            string   `json:"key"`
	Value          string   `json:"value,omitempty"`
	Color          string   `json:"color,omitempty"`
	ScopeType      string   `json:"scope_type"`
	ResourceScopes []string `json:"resource_scopes"`
}

// UpdateTagInput represents input parameters for updating a tag
type UpdateTagInput struct {
	Key            string   `json:"key"`
	Value          string   `json:"value,omitempty"`
	Color          string   `json:"color,omitempty"`
	ScopeType      string   `json:"scope_type,omitempty"`
	ResourceScopes []string `json:"resource_scopes,omitempty"`
}


func ResourceTagging() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTaggingCreate,
		ReadContext:   resourceTaggingRead,
		UpdateContext: resourceTaggingUpdate,
		DeleteContext: resourceTaggingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The key of the tag. This field is required.",
			},
			"value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The value of the tag. This field is optional.",
			},
			"color": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The color of the tag. This field is optional. Valid values are: red, green, blue, etc.",
			},
			"scope_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The scope type of the tag (e.g., VPC, PROJECT, ORG).",
			},
			"resource_scopes": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of resource scopes to associate with this tag.",
			},
		},
	}
}
func resourceTaggingRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewTaggingService(client)

	var diags diag.Diagnostics

	tagDetail, err := service.Get(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("key", tagDetail.Key); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("value", tagDetail.Value); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("color", tagDetail.Color); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("scope_type", tagDetail.ScopeType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_scopes", tagDetail.ResourceScopes); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceTaggingCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Create operation is not yet implemented for tagging resource")
}

func resourceTaggingUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Update operation is not yet implemented for tagging resource")
}

func resourceTaggingDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Delete operation is not yet implemented for tagging resource")
}
