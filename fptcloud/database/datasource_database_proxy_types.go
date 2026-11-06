package fptcloud_database

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &datasourceDatabaseProxyTypes{}
	_ datasource.DataSourceWithConfigure = &datasourceDatabaseProxyTypes{}
)

type datasourceDatabaseProxyTypes struct {
	api *databaseApiClient
}

func NewDataSourceDatabaseProxyTypes() datasource.DataSource {
	return &datasourceDatabaseProxyTypes{}
}

type databaseProxyTypesDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	ClusterId types.String `tfsdk:"cluster_id"`
	Types     types.List   `tfsdk:"types"`
}

func (d *datasourceDatabaseProxyTypes) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_proxy_types"
}

func (d *datasourceDatabaseProxyTypes) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	d.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (d *datasourceDatabaseProxyTypes) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Lists the proxy types (`lb_type`) a FPT Cloud database cluster accepts, for example `maxscale` for MariaDB and `mysqlrouter` for MySQL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Id of the database cluster.",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster.",
			},
			"types": schema.ListAttribute{
				Computed:    true,
				Description: "The `lb_type` values the cluster accepts.",
				ElementType: types.StringType,
			},
		},
	}
}

func (d *datasourceDatabaseProxyTypes) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state databaseProxyTypesDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	clusterId := strings.TrimSpace(state.ClusterId.ValueString())
	if clusterId == "" {
		response.Diagnostics.AddError("Missing cluster_id", "cluster_id must be a non empty string")
		return
	}

	proxyTypes, err := d.api.listProxyTypes(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading proxy types", err.Error())
		return
	}

	elems := make([]attr.Value, 0, len(proxyTypes))
	for _, t := range proxyTypes {
		elems = append(elems, types.StringValue(t))
	}
	state.Id = types.StringValue(clusterId)
	state.Types = types.ListValueMust(types.StringType, elems)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}
