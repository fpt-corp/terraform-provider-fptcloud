package fptcloud_tagging

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	common "terraform-provider-fptcloud/commons"
)

// CreateTagInput represents input parameters for creating a tag
type CreateTagInput struct {
	Key         string   `json:"key"`
	Value       string   `json:"value,omitempty"`
	Color       string   `json:"color,omitempty"`
	ScopeType   string   `json:"scope_type"`
	ResourceIds []string `json:"resource_ids"`
}

// UpdateTagInput represents input parameters for updating a tag
type UpdateTagInput struct {
	Key         string   `json:"key"`
	Value       string   `json:"value,omitempty"`
	Color       string   `json:"color,omitempty"`
	ScopeType   string   `json:"scope_type,omitempty"`
	ResourceIds []string `json:"resource_ids,omitempty"`
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
			"resource_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of resource ids to associate with this tag.",
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
	if err := d.Set("resource_ids", tagDetail.ResourceIds); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceTaggingCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewTaggingService(client)

	var resourceScopes []string
	if v, ok := d.GetOk("resource_ids"); ok {
		resourceScopesSet := v.(*schema.Set)
		resourceScopes = make([]string, resourceScopesSet.Len())
		for i, resourceScope := range resourceScopesSet.List() {
			resourceScopes[i] = resourceScope.(string)
		}
	}

	input := &CreateTagInput{
		Key:         d.Get("key").(string),
		ResourceIds: resourceScopes,
	}

	if v, ok := d.GetOk("value"); ok {
		input.Value = v.(string)
	}
	if v, ok := d.GetOk("color"); ok {
		input.Color = v.(string)
	}
	if v, ok := d.GetOk("scope_type"); ok {
		input.ScopeType = v.(string)
	}
	response, err := service.Create(ctx, input)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(response.TagID)
	log.Printf("[INFO] Created tag with ID: %s", response.TagID)

	return resourceTaggingRead(ctx, d, m)
}

func resourceTaggingUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewTaggingService(client)

	var resourceIds []string
	if v, ok := d.GetOk("resource_ids"); ok {
		resourceIdsSet := v.(*schema.Set)
		resourceIds = make([]string, resourceIdsSet.Len())
		for i, resourceScope := range resourceIdsSet.List() {
			resourceIds[i] = resourceScope.(string)
		}
	}

	input := &UpdateTagInput{
		Key:         d.Get("key").(string),
		ResourceIds: resourceIds,
	}

	if v, ok := d.GetOk("value"); ok {
		input.Value = v.(string)
	}
	if v, ok := d.GetOk("color"); ok {
		input.Color = v.(string)
	}

	if v, ok := d.GetOk("scope_type"); ok {
		input.ScopeType = v.(string)
	}

	_, err := service.Update(ctx, d.Id(), input)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updated tag with ID: %s", d.Id())

	return resourceTaggingRead(ctx, d, m)
}

func resourceTaggingDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewTaggingService(client)

	var diags diag.Diagnostics

	_, err := service.Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted tag with ID: %s", d.Id())
	d.SetId("")

	return diags
}
