package fptcloud

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/fptcloud/flavor"
	"terraform-provider-fptcloud/fptcloud/floating-ip"
	"terraform-provider-fptcloud/fptcloud/floating-ip-association"
	"terraform-provider-fptcloud/fptcloud/image"
	"terraform-provider-fptcloud/fptcloud/instance"
	"terraform-provider-fptcloud/fptcloud/instance-group"
	"terraform-provider-fptcloud/fptcloud/instance-group-policy"
	"terraform-provider-fptcloud/fptcloud/security-group"
	"terraform-provider-fptcloud/fptcloud/security-group-rule"
	"terraform-provider-fptcloud/fptcloud/ssh"
	"terraform-provider-fptcloud/fptcloud/storage"
	"terraform-provider-fptcloud/fptcloud/storage-policy"
	"terraform-provider-fptcloud/fptcloud/subnet"
	"terraform-provider-fptcloud/fptcloud/vpc"
)

var (
	// ProviderVersion is the version of the provider to set in the User-Agent header
	ProviderVersion = "dev"

	// ProdAPI is the Base URL for Fptcloud Production API
	ProdAPI = common.DefaultApiUrl
)

// Provider fptcloud provider
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("FPTCLOUD_TOKEN", ""),
				Description: "This is the Fpt cloud API token. Alternatively, this can also be specified using `FPTCLOUD_TOKEN` environment variable.",
			},
			"tenant_name": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("FPTCLOUD_TENANT_NAME", ""),
				Description: "The tenant name to use",
			},
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("FPTCLOUD_REGION", ""),
				Description: "The region to use (VN/HAN | VN/SGN)",
			},
			"api_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("FPTCLOUD_API_URL", ProdAPI),
				Description: "The URL to use",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"fptcloud_storage_policy":        fptcloud_storage_policy.DataSourceStoragePolicy(),
			"fptcloud_storage":               fptcloud_storage.DataSourceStorage(),
			"fptcloud_ssh_key":               fptcloud_ssh.DataSourceSSHKey(),
			"fptcloud_vpc":                   fptcloud_vpc.NewDataSource(),
			"fptcloud_flavor":                fptcloud_flavor.DataSourceFlavor(),
			"fptcloud_image":                 fptcloud_image.DataSourceImage(),
			"fptcloud_security_group":        fptcloud_security_group.DataSourceSecurityGroup(),
			"fptcloud_instance":              fptcloud_instance.DataSourceInstance(),
			"fptcloud_instance_group_policy": fptcloud_instance_group_policy.DataSourceInstanceGroupPolicy(),
			"fptcloud_instance_group":        fptcloud_instance_group.DataSourceInstanceGroup(),
			"fptcloud_floating_ip":           fptcloud_floating_ip.DataSourceFloatingIp(),
			"fptcloud_subnet":                fptcloud_subnet.DataSourceSubnet(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"fptcloud_storage":                 fptcloud_storage.ResourceStorage(),
			"fptcloud_ssh_key":                 fptcloud_ssh.ResourceSSHKey(),
			"fptcloud_security_group":          fptcloud_security_group.ResourceSecurityGroup(),
			"fptcloud_security_group_rule":     fptcloud_security_group_rule.ResourceSecurityGroupRule(),
			"fptcloud_instance":                fptcloud_instance.ResourceInstance(),
			"fptcloud_instance_group":          fptcloud_instance_group.ResourceInstanceGroup(),
			"fptcloud_floating_ip":             fptcloud_floating_ip.ResourceFloatingIp(),
			"fptcloud_floating_ip_association": fptcloud_floating_ip_association.ResourceFloatingIpAssociation(),
			"fptcloud_subnet":                  fptcloud_subnet.ResourceSubnet(),
		},
		ConfigureContextFunc: providerConfigureContext,
	}
}

// Provider configuration
func providerConfigureContext(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	var regionValue, tokenValue, tenantNameValue, apiURL string
	var client *common.Client
	var err error

	if region, ok := d.GetOk("region"); ok {
		regionValue = region.(string)
	}

	if tenantName, ok := d.GetOk("tenant_name"); ok {
		tenantNameValue = tenantName.(string)
	}

	if token, ok := d.GetOk("token"); ok {
		tokenValue = token.(string)
	} else {
		return nil, diag.Errorf("[ERR] token not found")
	}

	if apiEndpoint, ok := d.GetOk("api_endpoint"); ok {
		apiURL = apiEndpoint.(string)
	} else {
		apiURL = ProdAPI
	}
	client, err = common.NewClientWithURL(tokenValue, apiURL, regionValue, tenantNameValue)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	userAgent := &common.Component{
		Name:    "terraform-provider-fptcloud",
		Version: ProviderVersion,
	}
	client.SetUserAgent(userAgent)

	log.Printf("[DEBUG] Fptcloud API URL: %s\n", apiURL)
	log.Printf("[DEBUG] Fptcloud tenant name: %s\n", tenantNameValue)
	return client, diags
}
