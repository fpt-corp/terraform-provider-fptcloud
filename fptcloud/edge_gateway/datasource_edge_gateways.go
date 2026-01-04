package fptcloud_edge_gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	common "terraform-provider-fptcloud/commons"
)

var (
	_ datasource.DataSource              = &datasourceEdgeGateways{}
	_ datasource.DataSourceWithConfigure = &datasourceEdgeGateways{}
)

type datasourceEdgeGateways struct {
	client *common.Client
}

func NewDataSourceEdgeGateways() datasource.DataSource {
	return &datasourceEdgeGateways{}
}

func (d *datasourceEdgeGateways) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_edge_gateways"
}

func (d *datasourceEdgeGateways) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Retrieves a list of FPT Cloud edge gateways. If name is provided, returns only edge gateways matching that name.",
		Attributes: map[string]schema.Attribute{
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "VPC id to filter edge gateways",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the edge gateway to filter. If empty, returns all edge gateways.",
			},
			"edge_gateways": schema.ListAttribute{
				Computed:    true,
				Description: "List of edge gateways",
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":              types.StringType,
						"name":            types.StringType,
						"edge_gateway_id": types.StringType,
						"vpc_id":          types.StringType,
					},
				},
			},
		},
	}
}

func (d *datasourceEdgeGateways) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state edgeGatewaysModel
	diags := request.Config.Get(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	edgeGatewayList, err := d.fetchEdgeGateways(ctx, state.VpcId.ValueString())
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting edge gateway list", err.Error()))
		return
	}

	// Filter by name if provided
	var filteredList []EdgeGatewayData
	nameFilter := ""
	if !state.Name.IsNull() && !state.Name.IsUnknown() {
		nameFilter = state.Name.ValueString()
	}

	for _, eg := range *edgeGatewayList {
		if nameFilter == "" || eg.Name == nameFilter {
			filteredList = append(filteredList, eg)
		}
	}

	// Build the edge_gateways list
	edgeGatewaysList, listDiags := d.buildEdgeGatewaysList(filteredList)
	response.Diagnostics.Append(listDiags...)
	if response.Diagnostics.HasError() {
		return
	}

	state.EdgeGateways = edgeGatewaysList

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
}

func (d *datasourceEdgeGateways) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*common.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *datasourceEdgeGateways) fetchEdgeGateways(_ context.Context, vpcId string) (*[]EdgeGatewayData, error) {
	res, err := d.client.SendGetRequest(common.ApiPath.EdgeGatewayList(vpcId))
	if err != nil {
		return nil, err
	}

	var r EdgeGatewayResponse
	if err = json.Unmarshal(res, &r); err != nil {
		return nil, err
	}

	return &r.Data, nil
}

func (d *datasourceEdgeGateways) buildEdgeGatewaysList(items []EdgeGatewayData) (types.List, diag2.Diagnostics) {
	attrTypes := map[string]attr.Type{
		"id":              types.StringType,
		"name":            types.StringType,
		"edge_gateway_id": types.StringType,
		"vpc_id":          types.StringType,
	}

	if len(items) == 0 {
		return types.ListValueMust(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{}), nil
	}

	edgeGatewayObjects := make([]attr.Value, 0, len(items))
	for _, item := range items {
		obj, diags := types.ObjectValue(attrTypes, map[string]attr.Value{
			"id":              types.StringValue(item.Id),
			"name":            types.StringValue(item.Name),
			"edge_gateway_id": types.StringValue(item.EdgeGatewayId),
			"vpc_id":          types.StringValue(item.VpcId),
		})
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: attrTypes}), diags
		}
		edgeGatewayObjects = append(edgeGatewayObjects, obj)
	}

	return types.ListValue(types.ObjectType{AttrTypes: attrTypes}, edgeGatewayObjects)
}

type edgeGatewaysModel struct {
	VpcId        types.String `tfsdk:"vpc_id"`
	Name         types.String `tfsdk:"name"`
	EdgeGateways types.List   `tfsdk:"edge_gateways"`
}

