package fptcloud_subnet

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-fptcloud/commons"
)

var (
	_ datasource.DataSource              = &datasourceSubnet{}
	_ datasource.DataSourceWithConfigure = &datasourceSubnet{}
)

type datasourceSubnet struct {
	client *commons.Client
}

func NewDataSourceSubnet() datasource.DataSource {
	return &datasourceSubnet{}
}

func (d *datasourceSubnet) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_subnet_v1"
}

func (d *datasourceSubnet) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Subnets",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the subnet",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the subnet",
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the VPC containing the subnet",
			},
		},
	}
}

func (d *datasourceSubnet) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state subnet
	diags := request.Config.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	subnetList, err := d.internalRead(state.VpcId.ValueString())
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting subnet list", err.Error()))
		return
	}

	var sub subnetData

	for _, sn := range *subnetList {
		if sn.Name == state.Name.ValueString() {
			if sub.ID != "" {
				response.Diagnostics.Append(diag2.NewErrorDiagnostic(
					"Duplicate subnet name",
					"Subnet list contains two networks with identical name. For safety reasons this is an error. Report this to support.",
				))
				return
			} else {
				sub = sn
			}
		}
	}

	if sub.ID == "" {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(
			"No such subnet",
			fmt.Sprintf("No subnet with name \"%s\" was found", state.Name),
		))
	}

	state.ID = types.StringValue(sub.ID)
	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (d *datasourceSubnet) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*commons.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *commons.Client, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *datasourceSubnet) internalRead(vpcId string) (*[]subnetData, error) {
	url := commons.ApiPath.Subnet(vpcId)
	res, err := d.client.SendGetRequest(url)

	if err != nil {
		return nil, err
	}

	var r subnetResponse
	if err = json.Unmarshal(res, &r); err != nil {
		return nil, err
	}

	return &r.Data, nil
}

type subnet struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	VpcId types.String `tfsdk:"vpc_id"`
}

type subnetData struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	DefaultGateway     string      `json:"defaultGateway"`
	SubnetPrefixLength int         `json:"subnetPrefixLength"`
	NetworkID          interface{} `json:"network_id"`
	NetworkType        string      `json:"networkType"`
}

type subnetResponse struct {
	Data []subnetData `json:"data"`
}
