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
	Key      string   `json:"key"`
	Value    string   `json:"value,omitempty"`
	Color    string   `json:"color,omitempty"`
	VPCNames []string `json:"vpc_names"`
}

// UpdateTagInput represents input parameters for updating a tag
type UpdateTagInput struct {
	Key      string   `json:"key"`
	Value    string   `json:"value,omitempty"`
	Color    string   `json:"color,omitempty"`
	VPCNames []string `json:"vpc_names"`
}

// TagResponse represents the API response for tag operations
type TagResponse struct {
	TagID string `json:"tag_id"`
}

// TagDetail represents the detailed tag information
type TagDetail struct {
	ID        string   `json:"id"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Color     []string `json:"color"`
	ScopeType string   `json:"scope_type"`
	VPCNames  []string `json:"vpc_names"`
}

func ResourceTagging() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTaggingCreate,
		ReadContext:   resourceTaggingRead,
		UpdateContext: resourceTaggingUpdate,
		DeleteContext: resourceTaggingDelete,

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
			"vpc_names": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "List of VPC names to associate with this tag.",
			},
			"scope_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The scope type of the tag (e.g., VPC, ORG).",
			},
		},
	}
}
func resourceTaggingCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewTaggingService(client)

	var vpcNames []string
	if v, ok := d.GetOk("vpc_names"); ok {
		vpcSet := v.(*schema.Set)
		vpcNames = make([]string, vpcSet.Len())
		for i, vpc := range vpcSet.List() {
			vpcNames[i] = vpc.(string)
		}
	}

	input := &CreateTagInput{
		Key:      d.Get("key").(string),
		VPCNames: vpcNames,
	}

	if v, ok := d.GetOk("value"); ok {
		input.Value = v.(string)
	}
	if v, ok := d.GetOk("color"); ok {
		input.Color = v.(string)
	}

	response, err := service.Create(ctx, input)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(response.TagID)
	log.Printf("[INFO] Created tag with ID: %s", response.TagID)

	return resourceTaggingRead(ctx, d, m)
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
	if len(tagDetail.Color) > 0 {
		if err := d.Set("color", tagDetail.Color[0]); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("vpc_names", tagDetail.VPCNames); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("scope_type", tagDetail.ScopeType); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceTaggingUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewTaggingService(client)

	var vpcNames []string
	if v, ok := d.GetOk("vpc_names"); ok {
		vpcSet := v.(*schema.Set)
		vpcNames = make([]string, vpcSet.Len())
		for i, vpc := range vpcSet.List() {
			vpcNames[i] = vpc.(string)
		}
	}

	input := &UpdateTagInput{
		Key:      d.Get("key").(string),
		VPCNames: vpcNames,
	}

	if v, ok := d.GetOk("value"); ok {
		input.Value = v.(string)
	}
	if v, ok := d.GetOk("color"); ok {
		input.Color = v.(string)
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
