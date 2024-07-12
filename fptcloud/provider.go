package fptcloud

import (
	"fmt"
	"log"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"terraform-provider-fptcloud/fptcloud/ssh"
	"terraform-provider-fptcloud/fptcloud/storage"
)

var (
	// ProviderVersion is the version of the provider to set in the User-Agent header
	ProviderVersion = "dev"

	// ProdAPI is the Base URL for Fptcloud Production API
	ProdAPI = "https://console-api.fptcloud.com/api"
)

// Provider fptcloud provider
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("FPTCLOUD_TOKEN", ""),
				Description: "This is the Fpt cloud API token. Alternatively, this can also be specified using `FPTCLOUD_TOKEN` environment variable.",
			},
			"tenant_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("FPTCLOUD_TENANT_NAME", ""),
				Description: "The tenant name to use",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("FPTCLOUD_REGION", ""),
				Description: "The region to use",
			},
			"api_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("FPTCLOUD_API_URL", ProdAPI),
				Description: "The URL to use",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"fptcloud_volume":  storage.DataSourceStorage(),
			"fptcloud_ssh_key": ssh.DataSourceSSHKey(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"fptcloud_volume": storage.ResourceStorage(),
		},
		ConfigureFunc: providerConfigure,
	}
}

// Provider configuration
func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var regionValue, tokenValue, apiURL string
	var client *common.Client
	var err error

	if region, ok := d.GetOk("region"); ok {
		regionValue = region.(string)
	}

	if token, ok := d.GetOk("token"); ok {
		tokenValue = token.(string)
	} else {
		return nil, fmt.Errorf("[ERR] token not found")
	}

	if apiEndpoint, ok := d.GetOk("api_endpoint"); ok {
		apiURL = apiEndpoint.(string)
	} else {
		apiURL = ProdAPI
	}
	client, err = common.NewClientWithURL(tokenValue, apiURL, regionValue)
	if err != nil {
		return nil, err
	}

	userAgent := &common.Component{
		Name:    "terraform-provider-fptcloud",
		Version: ProviderVersion,
	}
	client.SetUserAgent(userAgent)

	log.Printf("[DEBUG] Fptcloud API URL: %s\n", apiURL)
	return client, nil
}
