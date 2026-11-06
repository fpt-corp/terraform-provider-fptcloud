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
	_ datasource.DataSource              = &datasourceDatabaseProxies{}
	_ datasource.DataSourceWithConfigure = &datasourceDatabaseProxies{}
)

type datasourceDatabaseProxies struct {
	api *databaseApiClient
}

func NewDataSourceDatabaseProxies() datasource.DataSource {
	return &datasourceDatabaseProxies{}
}

type databaseProxiesDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	ClusterId types.String `tfsdk:"cluster_id"`
	Proxies   types.List   `tfsdk:"proxies"`
}

func databaseProxyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                types.StringType,
		"name":              types.StringType,
		"lb_type":           types.StringType,
		"high_availability": types.BoolType,
		"node_cpu":          types.Int64Type,
		"node_ram":          types.Int64Type,
		"storage_profile":   types.StringType,
		"status":            types.StringType,
		"node_count":        types.Int64Type,
		"vm_sync":           types.BoolType,
		"created_at":        types.StringType,
	}
}

// databaseProxiesList converts the rows of load-balancing/list for the proxies attribute.
func databaseProxiesList(items []proxyListItem) types.List {
	elems := make([]attr.Value, 0, len(items))
	for _, p := range items {
		elems = append(elems, types.ObjectValueMust(databaseProxyAttrTypes(), map[string]attr.Value{
			"id":                types.StringValue(p.LbClusterId),
			"name":              types.StringValue(string(p.LbClusterName)),
			"lb_type":           types.StringValue(string(p.LbType)),
			"high_availability": types.BoolValue(proxyIsHA(p.IsLbCluster)),
			"node_cpu":          types.Int64Value(int64(p.NodeCpu)),
			"node_ram":          types.Int64Value(int64(p.NodeRam)),
			"storage_profile":   types.StringValue(string(p.StorageProfile)),
			"status":            types.StringValue(string(p.Status)),
			"node_count":        types.Int64Value(int64(p.MasterCount) + int64(p.SlaveCount)),
			"vm_sync":           types.BoolValue(p.VmSync),
			"created_at":        types.StringValue(string(p.CreatedAt)),
		}))
	}
	return types.ListValueMust(types.ObjectType{AttrTypes: databaseProxyAttrTypes()}, elems)
}

func (d *datasourceDatabaseProxies) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_proxies"
}

func (d *datasourceDatabaseProxies) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	d.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

func (d *datasourceDatabaseProxies) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Lists the DB proxies of a FPT Cloud database cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Id of the database cluster.",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster.",
			},
			"proxies": schema.ListAttribute{
				Computed:    true,
				Description: "DB proxies of the cluster. `id` is the Id to use with `fptcloud_database_proxy`; `node_count` is the number of nodes the proxy is made of (2 with high availability).",
				ElementType: types.ObjectType{AttrTypes: databaseProxyAttrTypes()},
			},
		},
	}
}

func (d *datasourceDatabaseProxies) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state databaseProxiesDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	clusterId := strings.TrimSpace(state.ClusterId.ValueString())
	if clusterId == "" {
		response.Diagnostics.AddError("Missing cluster_id", "cluster_id must be a non empty string")
		return
	}

	items, err := d.api.listProxies(clusterId)
	if err != nil {
		response.Diagnostics.AddError("Error reading proxies", err.Error())
		return
	}

	state.Id = types.StringValue(clusterId)
	state.Proxies = databaseProxiesList(items)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}
