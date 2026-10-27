package fptcloud_database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	common "terraform-provider-fptcloud/commons"
)

var (
	_ datasource.DataSource              = &datasourceDatabaseResources{}
	_ datasource.DataSourceWithConfigure = &datasourceDatabaseResources{}
)

type datasourceDatabaseResources struct {
	api *databaseApiClient
}

func NewDataSourceDatabaseResources() datasource.DataSource {
	return &datasourceDatabaseResources{}
}

// usage is the compute usage of a group of nodes.
type usage struct {
	NodeCount *int64   `tfsdk:"node_count" json:"node_count"`
	Cpu       *int64   `tfsdk:"cpu" json:"cpu"`
	RamGb     *float64 `tfsdk:"ram_gb" json:"ram_gb"`
	DiskGb    *int64   `tfsdk:"disk_gb" json:"disk_gb"`
}

type backupUsage struct {
	CapacityGb *int64   `tfsdk:"capacity_gb" json:"capacity_gb"`
	UsedGb     *float64 `tfsdk:"used_gb" json:"used_gb"`
}

type pitrUsage struct {
	Enabled       *bool    `tfsdk:"enabled" json:"enabled"`
	ArchiveSizeGb *float64 `tfsdk:"archive_size_gb" json:"archive_size_gb"`
}

type proxyItem struct {
	LbClusterId   *string  `tfsdk:"lb_cluster_id" json:"lb_cluster_id"`
	LbClusterName *string  `tfsdk:"lb_cluster_name" json:"lb_cluster_name"`
	LbType        *string  `tfsdk:"lb_type" json:"lb_type"`
	Status        *string  `tfsdk:"status" json:"status"`
	NodeCount     *int64   `tfsdk:"node_count" json:"node_count"`
	Cpu           *int64   `tfsdk:"cpu" json:"cpu"`
	RamGb         *float64 `tfsdk:"ram_gb" json:"ram_gb"`
	DiskGb        *int64   `tfsdk:"disk_gb" json:"disk_gb"`
}

type proxyUsage struct {
	NodeCount *int64      `tfsdk:"node_count" json:"node_count"`
	Cpu       *int64      `tfsdk:"cpu" json:"cpu"`
	RamGb     *float64    `tfsdk:"ram_gb" json:"ram_gb"`
	DiskGb    *int64      `tfsdk:"disk_gb" json:"disk_gb"`
	Items     []proxyItem `tfsdk:"items" json:"items"`
}

type vpcCluster struct {
	ClusterId   *string    `tfsdk:"cluster_id" json:"cluster_id"`
	ClusterName *string    `tfsdk:"cluster_name" json:"cluster_name"`
	TypeDb      *string    `tfsdk:"type_db" json:"type_db"`
	Status      *string    `tfsdk:"status" json:"status"`
	Database    usage      `tfsdk:"database" json:"database"`
	Backup      backupUsage `tfsdk:"backup" json:"backup"`
	Pitr        pitrUsage  `tfsdk:"pitr" json:"pitr"`
	Proxy       proxyUsage `tfsdk:"proxy" json:"proxy"`
}

type vpcTotal struct {
	ClusterCount     *int64      `tfsdk:"cluster_count" json:"cluster_count"`
	PitrEnabledCount *int64      `tfsdk:"pitr_enabled_count" json:"pitr_enabled_count"`
	Database         usage       `tfsdk:"database" json:"database"`
	Backup           backupUsage `tfsdk:"backup" json:"backup"`
	Pitr             pitrUsage  `tfsdk:"pitr" json:"pitr"`
	Proxy            usage       `tfsdk:"proxy" json:"proxy"`
}

type vpcResources struct {
	Total    vpcTotal     `json:"total"`
	Clusters []vpcCluster `json:"clusters"`
}

type databaseResourcesModel struct {
	Id       types.String `tfsdk:"id"`
	VpcId    types.String `tfsdk:"vpc_id"`
	Total    *vpcTotal    `tfsdk:"total"`
	Clusters []vpcCluster `tfsdk:"clusters"`
}

// getVpcResources reads every database cluster and DB proxy of a VPC
// with their compute and backup usage.
func (m *databaseApiClient) getVpcResources(vpcId string) (*vpcResources, error) {
	path := common.ApiPath.DatabaseVpcResources(vpcId)

	body, err := m.getWithRetry(path)
	if err != nil {
		return nil, fmt.Errorf("failed calling path %s: %w", path, err)
	}

	if berr := checkBusinessError(body); berr != nil {
		return nil, berr
	}

	var resp struct {
		Data vpcResources `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf(
			"failed parsing %s: %w, body=%s",
			path,
			err,
			truncateBody(body),
		)
	}

	if resp.Data.Clusters == nil {
		resp.Data.Clusters = []vpcCluster{}
	}

	for i := range resp.Data.Clusters {
		if resp.Data.Clusters[i].Proxy.Items == nil {
			resp.Data.Clusters[i].Proxy.Items = []proxyItem{}
		}
	}

	return &resp.Data, nil
}

func (d *datasourceDatabaseResources) Metadata(
	ctx context.Context,
	request datasource.MetadataRequest,
	response *datasource.MetadataResponse,
) {
	response.TypeName = request.ProviderTypeName + "_database_resources"
}

func (d *datasourceDatabaseResources) Configure(
	ctx context.Context,
	request datasource.ConfigureRequest,
	response *datasource.ConfigureResponse,
) {
	d.api = configureDatabaseApi(request.ProviderData, &response.Diagnostics)
}

var (
	usageTypes = map[string]attr.Type{
		"node_count": types.Int64Type,
		"cpu":        types.Int64Type,
		"ram_gb":     types.Float64Type,
		"disk_gb":    types.Int64Type,
	}

	backupTypes = map[string]attr.Type{
		"capacity_gb": types.Int64Type,
		"used_gb":     types.Float64Type,
	}

	pitrTypes = map[string]attr.Type{
		"enabled":        types.BoolType,
		"archive_size_gb": types.Float64Type,
	}
)

func withTypes(
	base map[string]attr.Type,
	extra map[string]attr.Type,
) map[string]attr.Type {
	out := make(map[string]attr.Type, len(base)+len(extra))

	for k, v := range base {
		out[k] = v
	}

	for k, v := range extra {
		out[k] = v
	}

	return out
}

func vpcTotalTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cluster_count":      types.Int64Type,
		"pitr_enabled_count": types.Int64Type,
		"database":           types.ObjectType{AttrTypes: usageTypes},
		"backup":             types.ObjectType{AttrTypes: backupTypes},
		"pitr":               types.ObjectType{AttrTypes: pitrTypes},
		"proxy":              types.ObjectType{AttrTypes: usageTypes},
	}
}

func vpcClusterTypes() map[string]attr.Type {
	proxyItemTypes := withTypes(usageTypes, map[string]attr.Type{
		"lb_cluster_id":   types.StringType,
		"lb_cluster_name": types.StringType,
		"lb_type":         types.StringType,
		"status":          types.StringType,
	})

	return map[string]attr.Type{
		"cluster_id":   types.StringType,
		"cluster_name": types.StringType,
		"type_db":      types.StringType,
		"status":       types.StringType,
		"database":     types.ObjectType{AttrTypes: usageTypes},
		"backup":       types.ObjectType{AttrTypes: backupTypes},
		"pitr":         types.ObjectType{AttrTypes: pitrTypes},
		"proxy": types.ObjectType{
			AttrTypes: withTypes(usageTypes, map[string]attr.Type{
				"items": types.ListType{
					ElemType: types.ObjectType{
						AttrTypes: proxyItemTypes,
					},
				},
			}),
		},
	}
}

func (d *datasourceDatabaseResources) Schema(
	ctx context.Context,
	request datasource.SchemaRequest,
	response *datasource.SchemaResponse,
) {
	response.Schema = schema.Schema{
		Description: "Lists every database cluster and DB proxy of a VPC, with the nodes, CPU, RAM, disk and backup they use, and the totals of the VPC.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:   true,
				Description: "The Id of the VPC.",
			},
			"vpc_id": schema.StringAttribute{
				Required:   true,
				Description: "The Id of the VPC.",
			},
			"total": schema.ObjectAttribute{
				Computed: true,
				Description: "Totals over all the clusters of the VPC: `cluster_count`, `pitr_enabled_count`, and `database`, `backup`, `pitr` and `proxy` usage.",
				AttributeTypes: vpcTotalTypes(),
			},
			"clusters": schema.ListAttribute{
				Computed: true,
				Description: "The database clusters of the VPC. `database` and `proxy` hold `node_count`, `cpu`, `ram_gb` and `disk_gb`; `proxy.items` lists the DB proxies (`lb_cluster_id`, `lb_cluster_name`, `lb_type`, `status` and their own usage); `backup.used_gb` is null when the cluster has no backup data.",
				ElementType: types.ObjectType{
					AttrTypes: vpcClusterTypes(),
				},
			},
		},
	}
}

func (d *datasourceDatabaseResources) Read(
	ctx context.Context,
	request datasource.ReadRequest,
	response *datasource.ReadResponse,
) {
	var state databaseResourcesModel

	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	vpcId := strings.TrimSpace(state.VpcId.ValueString())
	if vpcId == "" {
		response.Diagnostics.AddError(
			"Missing vpc_id",
			"vpc_id must be a non empty string",
		)
		return
	}

	res, err := d.api.getVpcResources(vpcId)
	if err != nil {
		response.Diagnostics.AddError(
			"Error reading database resources",
			err.Error(),
		)
		return
	}

	state.Id = types.StringValue(vpcId)
	state.Total = &res.Total
	state.Clusters = res.Clusters

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}
