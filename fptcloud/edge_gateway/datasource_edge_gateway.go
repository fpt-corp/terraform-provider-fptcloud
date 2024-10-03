package fptcloud_edge_gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	common "terraform-provider-fptcloud/commons"
)

var (
	_ datasource.DataSource              = &datasourceEdgeGateway{}
	_ datasource.DataSourceWithConfigure = &datasourceEdgeGateway{}
)

type datasourceEdgeGateway struct {
	client *common.Client
}

func NewDataSourceEdgeGateway() datasource.DataSource {
	return &datasourceEdgeGateway{}
}

func (d *datasourceEdgeGateway) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_edge_gateway"
}

func (d *datasourceEdgeGateway) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Retrieves information about FPT Cloud edge gateway",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the edge_gateway",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the compute edge_gateway",
			},
			"edge_gateway_id": schema.StringAttribute{
				Computed:    true,
				Description: "Edge gateway id",
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "VPC id",
			},
		},
	}
}

func (d *datasourceEdgeGateway) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state edge_gateway
	diags := request.Config.Get(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	edgeGatewayList, err := d.internalRead(ctx, &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting edge_gateway list", err.Error()))
		return
	}

	var foundEdgeGateway edgeGatewayData
	for _, edgeGateway := range *edgeGatewayList {
		if edgeGateway.Name == state.Name.ValueString() {
			foundEdgeGateway = edgeGateway
			break
		}
	}

	if foundEdgeGateway.Id == "" {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(
			"No such edge_gateway",
			fmt.Sprintf("No matching edge_gateway with name %s was found", state.Name.ValueString()),
		))
		return
	}

	state.Id = types.StringValue(foundEdgeGateway.Id)
	state.EdgeGatewayId = types.StringValue(foundEdgeGateway.EdgeGatewayId)
	state.VpcId = types.StringValue(foundEdgeGateway.VpcId)
	state.Name = types.StringValue(foundEdgeGateway.Name)

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (d *datasourceEdgeGateway) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*common.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *internal.ClientV1, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *datasourceEdgeGateway) internalRead(_ context.Context, state *edge_gateway) (*[]edgeGatewayData, error) {
	vpcId := state.VpcId.ValueString()

	res, err := d.client.SendGetRequest(common.ApiPath.EdgeGatewayList(vpcId))

	if err != nil {
		return nil, err
	}

	var r edgeGatewayResponse
	if err = json.Unmarshal(res, &r); err != nil {
		return nil, err
	}

	return &r.Data, nil
}

type edge_gateway struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	EdgeGatewayId types.String `tfsdk:"edge_gateway_id"`
	VpcId         types.String `tfsdk:"vpc_id"`
}

type edgeGatewayData struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	EdgeGatewayId string `json:"edge_gateway_id"`
	VpcId         string `json:"vpc_id"`
}

type edgeGatewayResponse struct {
	Data []edgeGatewayData `json:"data"`
}
