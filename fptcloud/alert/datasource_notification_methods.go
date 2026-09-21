package fptcloud_alert

import (
	"context"
	"strings"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func DataSourceAlertNotificationMethods() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the notification methods configured for a VPC. Use it to fill `notification_method_ids` on a backup job.",
		ReadContext: readAlertNotificationMethods,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The ID of the VPC.",
			},
			"level": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "VPC",
				ValidateFunc: validation.StringInSlice([]string{"VPC", "PROJECT"}, false),
				Description:  "The scope of the notification methods: `VPC` or `PROJECT`.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Only return methods of this channel type, for example `EMAIL`. Webhook methods are never returned by the API.",
			},
			"methods": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":      {Type: schema.TypeString, Computed: true},
						"name":    {Type: schema.TypeString, Computed: true},
						"type":    {Type: schema.TypeString, Computed: true},
						"address": {Type: schema.TypeString, Computed: true},
						"level":   {Type: schema.TypeString, Computed: true},
					},
				},
				Description: "The notification methods found. Match on `address`, not `name`: the API groups results by address, so `name` is not stable when two methods share an address.",
			},
		},
	}
}

func readAlertNotificationMethods(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewAlertService(client)
	vpcId := d.Get("vpc_id").(string)

	response, err := service.ListNotificationMethods(vpcId, d.Get("level").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	wantedType := strings.TrimSpace(d.Get("type").(string))
	methods := make([]interface{}, 0, len(response.Data))
	for _, item := range response.Data {
		if wantedType != "" && !strings.EqualFold(item.Type, wantedType) {
			continue
		}
		methods = append(methods, map[string]interface{}{
			"id":      item.Id,
			"name":    item.Name,
			"type":    item.Type,
			"address": item.Address,
			"level":   item.Level,
		})
	}

	if err := d.Set("methods", methods); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(vpcId)
	return nil
}
