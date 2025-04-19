package fptcloud

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	common "terraform-provider-fptcloud/commons"
	fptcloud_database "terraform-provider-fptcloud/fptcloud/database"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_edge_gateway "terraform-provider-fptcloud/fptcloud/edge_gateway"
	fptcloud_mfke "terraform-provider-fptcloud/fptcloud/mfke"
)

var (
	_ provider.Provider = &xplatProvider{}
)

type xplatProviderModel struct {
	Region      types.String `tfsdk:"region"`
	Token       types.String `tfsdk:"token"`
	TenantName  types.String `tfsdk:"tenant_name"`
	ApiEndpoint types.String `tfsdk:"api_endpoint"`
	Timeout     types.Int64  `tfsdk:"timeout"`
}

type xplatProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

func NewXplatProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &xplatProvider{
			version: version,
		}
	}
}

func (x *xplatProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "fptcloud"
	response.Version = x.version
}

func (x *xplatProvider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "",
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Description: "The region to use (VN/HAN | VN/SGN | JP/JCSI2)",
				Optional:    true,
			},

			"token": schema.StringAttribute{
				Description: "This is the Fpt cloud API token. Alternatively, this can also be specified using `FPTCLOUD_TOKEN` environment variable.",
				Optional:    true,
			},

			"tenant_name": schema.StringAttribute{
				Description: "The tenant name to use",
				Optional:    true,
			},

			"api_endpoint": schema.StringAttribute{
				Description: "The URL to use",
				Optional:    true,
			},

			"timeout": schema.Int64Attribute{
				Description: "Timeout in minutes (optional)",
				Optional:    true,
			},
		},
	}
}

func (x *xplatProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring FPTCloud client")
	var config xplatProviderModel

	diags := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	token := os.Getenv("FPTCLOUD_TOKEN")
	region := os.Getenv("FPTCLOUD_REGION")
	tenantName := os.Getenv("FPTCLOUD_TENANT_NAME")
	apiEndpoint := os.Getenv("FPTCLOUD_API_URL")
	var timeout int = 5

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	if !config.Region.IsNull() {
		region = config.Region.ValueString()
	}

	if !config.TenantName.IsNull() {
		tenantName = config.TenantName.ValueString()
	}

	if !config.ApiEndpoint.IsNull() {
		apiEndpoint = config.ApiEndpoint.ValueString()
	}

	if !config.Timeout.IsNull() {
		timeout = int(config.Timeout.ValueInt64())
	}

	if apiEndpoint == "" {
		apiEndpoint = ProdAPI
	}

	if token == "" {
		response.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing token",
			"Token must be specified to authenticate to provision resources",
		)
	}

	if response.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "token", token)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "token")
	tflog.Debug(ctx, "Creating FPTCloud client")

	client, err := common.NewClientWithURL(token, apiEndpoint, region, tenantName, timeout)

	if err != nil {
		response.Diagnostics.AddError("Error creating client", err.Error())
		return
	}

	userAgent := &common.Component{
		Name:    "terraform-provider-fptcloud",
		Version: ProviderVersion,
	}
	client.SetUserAgent(userAgent)

	response.DataSourceData = client
	response.ResourceData = client

	tflog.Info(ctx, "Configured FPTCloud client", map[string]any{
		"success":      true,
		"api_endpoint": apiEndpoint,
		"tenant_name":  tenantName,
	})
}

func (x *xplatProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		fptcloud_dfke.NewDataSourceDedicatedKubernetesEngine,
		fptcloud_mfke.NewDataSourceManagedKubernetesEngine,
		fptcloud_edge_gateway.NewDataSourceEdgeGateway,
	}
}

func (x *xplatProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		fptcloud_dfke.NewResourceDedicatedKubernetesEngine,
		fptcloud_dfke.NewResourceDedicatedKubernetesEngineState,
		fptcloud_mfke.NewResourceManagedKubernetesEngine,
		fptcloud_mfke.NewResourceDedicatedKubernetesEngineState,
		fptcloud_database.NewResourceDatabase,
		fptcloud_database.NewResourceDatabaseStatus,
	}
}
