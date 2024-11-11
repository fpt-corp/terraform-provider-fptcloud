package fptcloud

import (
	"context"
	"log"
	common "terraform-provider-fptcloud/commons"
	fptcloud_flavor "terraform-provider-fptcloud/fptcloud/flavor"
	fptcloud_floating_ip "terraform-provider-fptcloud/fptcloud/floating-ip"
	fptcloud_floating_ip_association "terraform-provider-fptcloud/fptcloud/floating-ip-association"
	fptcloud_image "terraform-provider-fptcloud/fptcloud/image"
	fptcloud_instance "terraform-provider-fptcloud/fptcloud/instance"
	fptcloud_instance_group "terraform-provider-fptcloud/fptcloud/instance-group"
	fptcloud_instance_group_policy "terraform-provider-fptcloud/fptcloud/instance-group-policy"

	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"
	fptcloud_security_group "terraform-provider-fptcloud/fptcloud/security-group"
	fptcloud_security_group_rule "terraform-provider-fptcloud/fptcloud/security-group-rule"
	fptcloud_ssh "terraform-provider-fptcloud/fptcloud/ssh"
	fptcloud_storage "terraform-provider-fptcloud/fptcloud/storage"
	fptcloud_storage_policy "terraform-provider-fptcloud/fptcloud/storage-policy"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"
	fptcloud_vpc "terraform-provider-fptcloud/fptcloud/vpc"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			"fptcloud_storage_policy":                   fptcloud_storage_policy.DataSourceStoragePolicy(),
			"fptcloud_storage":                          fptcloud_storage.DataSourceStorage(),
			"fptcloud_ssh_key":                          fptcloud_ssh.DataSourceSSHKey(),
			"fptcloud_vpc":                              fptcloud_vpc.NewDataSource(),
			"fptcloud_flavor":                           fptcloud_flavor.DataSourceFlavor(),
			"fptcloud_image":                            fptcloud_image.DataSourceImage(),
			"fptcloud_security_group":                   fptcloud_security_group.DataSourceSecurityGroup(),
			"fptcloud_instance":                         fptcloud_instance.DataSourceInstance(),
			"fptcloud_instance_group_policy":            fptcloud_instance_group_policy.DataSourceInstanceGroupPolicy(),
			"fptcloud_instance_group":                   fptcloud_instance_group.DataSourceInstanceGroup(),
			"fptcloud_floating_ip":                      fptcloud_floating_ip.DataSourceFloatingIp(),
			"fptcloud_subnet":                           fptcloud_subnet.DataSourceSubnet(),
			"fptcloud_object_storage_access_key":        fptcloud_object_storage.DataSourceAccessKey(),
			"fptcloud_object_storage_sub_user":          fptcloud_object_storage.DataSourceSubUser(),
			"fptcloud_object_storage_bucket":            fptcloud_object_storage.DataSourceBucket(),
			"fptcloud_object_storage_bucket_policy":     fptcloud_object_storage.DataSourceBucketPolicy(),
			"fptcloud_object_storage_bucket_cors":       fptcloud_object_storage.DataSourceBucketCors(),
			"fptcloud_object_storage_bucket_versioning": fptcloud_object_storage.DataSourceBucketVersioning(),
			"fptcloud_object_storage_lifecycle":         fptcloud_object_storage.DataSourceBucketLifecycle(),
			"fptcloud_object_storage_static_website":    fptcloud_object_storage.DataSourceBucketStaticWebsite(),
			"fptcloud_object_storage_sub_user_detail":   fptcloud_object_storage.DataSourceSubUserDetail(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"fptcloud_storage":                              fptcloud_storage.ResourceStorage(),
			"fptcloud_ssh_key":                              fptcloud_ssh.ResourceSSHKey(),
			"fptcloud_security_group":                       fptcloud_security_group.ResourceSecurityGroup(),
			"fptcloud_security_group_rule":                  fptcloud_security_group_rule.ResourceSecurityGroupRule(),
			"fptcloud_instance":                             fptcloud_instance.ResourceInstance(),
			"fptcloud_instance_group":                       fptcloud_instance_group.ResourceInstanceGroup(),
			"fptcloud_floating_ip":                          fptcloud_floating_ip.ResourceFloatingIp(),
			"fptcloud_floating_ip_association":              fptcloud_floating_ip_association.ResourceFloatingIpAssociation(),
			"fptcloud_subnet":                               fptcloud_subnet.ResourceSubnet(),
			"fptcloud_object_storage_bucket":                fptcloud_object_storage.ResourceBucket(),
			"fptcloud_object_storage_sub_user":              fptcloud_object_storage.ResourceSubUser(),
			"fptcloud_object_storage_access_key":            fptcloud_object_storage.ResourceAccessKey(),
			"fptcloud_object_storage_bucket_cors":           fptcloud_object_storage.ResourceBucketCors(),
			"fptcloud_object_storage_bucket_policy":         fptcloud_object_storage.ResourceBucketPolicy(),
			"fptcloud_object_storage_bucket_versioning":     fptcloud_object_storage.ResourceBucketVersioning(),
			"fptcloud_object_storage_bucket_static_website": fptcloud_object_storage.ResourceBucketStaticWebsite(),
			"fptcloud_object_storage_bucket_acl":            fptcloud_object_storage.ResourceBucketAcl(),
			"fptcloud_object_storage_sub_user_key":          fptcloud_object_storage.ResourceSubUserKeys(),
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
