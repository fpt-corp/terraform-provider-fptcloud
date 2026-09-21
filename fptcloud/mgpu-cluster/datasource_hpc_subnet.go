package fptcloud_mgpu_cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &datasourceHpcSubnet{}
	_ datasource.DataSourceWithConfigure = &datasourceHpcSubnet{}
)

// datasourceHpcSubnet lists the OSP-specific HPC subnet catalog
// (GET .../vmware/vpc/{vpcId}/hpc/subnets), optionally narrowed by filter —
// same shape as fptcloud_subnet: no filter returns every subnet, a filter
// narrows the list. create-cluster on OSP needs vm_subnet (the CIDR) and
// osp_network_id, neither of which the regular fptcloud_subnet data source
// exposes.
type datasourceHpcSubnet struct {
	client            *commons.Client
	mgpuClusterClient *MgpuClusterApiClient
	tenancyClient     *fptcloud_dfke.TenancyApiClient
}

type hpcSubnetFilterModel struct {
	Key    types.String   `tfsdk:"key"`
	Values []types.String `tfsdk:"values"`
}

type hpcSubnetDataSourceModel struct {
	VpcId   types.String           `tfsdk:"vpc_id"`
	Filter  []hpcSubnetFilterModel `tfsdk:"filter"`
	Subnets []hpcSubnetModel       `tfsdk:"subnets"`
}

type hpcSubnetModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	SubnetCidr   types.String `tfsdk:"subnet_cidr"`
	Description  types.String `tfsdk:"description"`
	Gateway      types.String `tfsdk:"gateway"`
	Status       types.String `tfsdk:"status"`
	OspNetworkId types.String `tfsdk:"osp_network_id"`
	NetworkAclId types.String `tfsdk:"network_acl_id"`
}

func (d *datasourceHpcSubnet) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*commons.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *commons.Client, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}

	d.client = client
	d.mgpuClusterClient = newMgpuClusterApiClient(d.client)
	d.tenancyClient = fptcloud_dfke.NewTenancyApiClient(d.client)
}

func (d *datasourceHpcSubnet) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_hpc_subnet"
}

func (d *datasourceHpcSubnet) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "List the HPC (bare metal GPU) subnet catalog for a VPC on OSP, optionally narrowed by filter. " +
			"Each entry's subnet_cidr and osp_network_id are what fptcloud_managed_gpu_cluster needs to create a bare-metal cluster on that subnet.",
		Attributes: map[string]schema.Attribute{
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "VPC ID",
			},
			"subnets": schema.ListAttribute{
				Computed:    true,
				Description: "HPC subnets matching the filter (every subnet in the VPC when no filter is set)",
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":             types.StringType,
						"name":           types.StringType,
						"subnet_cidr":    types.StringType,
						"description":    types.StringType,
						"gateway":        types.StringType,
						"status":         types.StringType,
						"osp_network_id": types.StringType,
						"network_acl_id": types.StringType,
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.ListNestedBlock{
				Description: "Optional filters, matched against every subnet with AND semantics. Omit to list every subnet in the VPC.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:    true,
							Description: "Field to filter on: id or name",
						},
						"values": schema.ListAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: "Values to match — a subnet matches if its field equals any of these",
						},
					},
				},
			},
		},
	}
}

func (d *datasourceHpcSubnet) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state hpcSubnetDataSourceModel
	diags := request.Config.Get(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	vpcId := state.VpcId.ValueString()
	platform, err := d.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error getting VPC platform", err.Error()))
		return
	}

	path := commons.ApiPath.ManagedGpuClusterHpcSubnets(vpcId, 1, 256)
	body, err := d.mgpuClusterClient.sendGet(path, strings.ToUpper(platform))
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error fetching HPC subnets", err.Error()))
		return
	}

	var resp hpcSubnetListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error parsing HPC subnets response", err.Error()))
		return
	}

	matches := filterHpcSubnets(resp.Data, state.Filter)

	subnets := make([]hpcSubnetModel, 0, len(matches))
	for _, s := range matches {
		subnets = append(subnets, hpcSubnetModel{
			ID:           types.StringValue(s.ID),
			Name:         types.StringValue(s.Name),
			SubnetCidr:   types.StringValue(s.SubnetCidr),
			Description:  types.StringValue(s.Description),
			Gateway:      types.StringValue(s.Gateway),
			Status:       types.StringValue(s.Status),
			OspNetworkId: types.StringValue(s.OspNetworkId),
			NetworkAclId: types.StringValue(s.NetworkAclId),
		})
	}
	state.Subnets = subnets

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
}

// filterHpcSubnets keeps subnets matching every filter block (AND across
// blocks), where a block matches if the subnet's field equals any of its
// values (OR within a block) — the same semantics fptcloud_subnet's filter
// uses. No filters means no narrowing: every subnet is kept.
func filterHpcSubnets(subnets []hpcSubnet, filters []hpcSubnetFilterModel) []hpcSubnet {
	if len(filters) == 0 {
		return subnets
	}

	result := make([]hpcSubnet, 0, len(subnets))
	for _, s := range subnets {
		if matchesAllFilters(s, filters) {
			result = append(result, s)
		}
	}
	return result
}

func matchesAllFilters(s hpcSubnet, filters []hpcSubnetFilterModel) bool {
	for _, f := range filters {
		if !matchesFilter(s, f) {
			return false
		}
	}
	return true
}

func matchesFilter(s hpcSubnet, f hpcSubnetFilterModel) bool {
	var field string
	switch f.Key.ValueString() {
	case "id":
		field = s.ID
	case "name":
		field = s.Name
	default:
		return false
	}

	for _, v := range f.Values {
		if field == v.ValueString() {
			return true
		}
	}
	return false
}

func NewDataSourceHpcSubnet() datasource.DataSource {
	return &datasourceHpcSubnet{}
}
