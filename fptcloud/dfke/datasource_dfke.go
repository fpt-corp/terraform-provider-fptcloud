package fptcloud_dfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-fptcloud/commons"
)

var (
	_ datasource.DataSource              = &datasourceDedicatedKubernetesEngine{}
	_ datasource.DataSourceWithConfigure = &datasourceDedicatedKubernetesEngine{}
)

type datasourceDedicatedKubernetesEngine struct {
	client           *commons.Client
	dfkeClient       *dfkeApiClient
	tenancyApiClient *TenancyApiClient
}

func NewDataSourceDedicatedKubernetesEngine() datasource.DataSource {
	return &datasourceDedicatedKubernetesEngine{}
}

func (d *datasourceDedicatedKubernetesEngine) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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
	d.dfkeClient = newDfkeApiClient(client)

	t := NewTenancyApiClient(client)
	d.tenancyApiClient = t
}

func (d *datasourceDedicatedKubernetesEngine) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dedicated_kubernetes_engine_v1"
}

func (d *datasourceDedicatedKubernetesEngine) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Retrieves information about dedicated FKE clusters",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the cluster",
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "VPC ID",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "Cluster ID, as shown on the dashboard, usually has a length of 8 characters",
			},
			"cluster_name": schema.StringAttribute{
				Computed:    true,
				Description: "Cluster name",
			},
			"k8s_version": schema.StringAttribute{
				Computed:    true,
				Description: "Kubernetes version",
			},
			"master_type": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the flavor of master node",
			},
			"master_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of master node",
			},
			"master_disk_size": schema.Int64Attribute{
				Computed:    true,
				Description: "Master node disk capacity in GB",
			},
			"worker_type": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the flavor of worker node",
			},
			"worker_disk_size": schema.Int64Attribute{
				Computed:    true,
				Description: "Worker node disk capacity in GB",
			},
			"network_id": schema.StringAttribute{
				Computed:    true,
				Description: "Network UUID",
			},
			"lb_size": schema.StringAttribute{
				Computed:    true,
				Description: "Load balancer size",
			},
			"pod_network": schema.StringAttribute{
				Computed:    true,
				Description: "Pod network in CIDR notation",
			},
			"service_network": schema.StringAttribute{
				Computed:    true,
				Description: "Service network in CIDR notation",
			},
			"network_node_prefix": schema.Int64Attribute{
				Computed:    true,
				Description: "Network node prefix",
			},
			"max_pod_per_node": schema.Int64Attribute{
				Computed:    true,
				Description: "Max pods per node",
			},
			"nfs_status": schema.StringAttribute{
				Computed:    true,
				Description: "NFS status",
			},
			"nfs_disk_size": schema.Int64Attribute{
				Computed:    true,
				Description: "NFS disk size",
			},
			"storage_policy": schema.StringAttribute{
				Computed:    true,
				Description: "Storage policy",
			},
			"edge_id": schema.StringAttribute{
				Computed:    true,
				Description: "Edge ID",
			},
			"scale_min": schema.Int64Attribute{
				Computed:    true,
				Description: "Minimum number of nodes for autoscaling",
			},
			"scale_max": schema.Int64Attribute{
				Computed:    true,
				Description: "Maximum number of nodes for autoscaling",
			},
			"node_dns": schema.StringAttribute{
				Computed:    true,
				Description: "DNS server of nodes",
			},
			"ip_public_firewall": schema.StringAttribute{
				Computed:    true,
				Description: "IP public firewall",
			},
			"ip_private_firewall": schema.StringAttribute{
				Computed:    true,
				Description: "IP private firewall",
			},
			"region_id": schema.StringAttribute{
				Computed:    true,
				Description: "Region ID",
			},
		},
	}
}

func (d *datasourceDedicatedKubernetesEngine) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state dedicatedKubernetesEngine
	diags := request.Config.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	clusterId := state.ClusterId.ValueString()
	uuid, err := d.findClusterUUID(ctx, state.vpcId(), clusterId)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error resolving cluster UUID", err.Error()))
		return
	}

	_, err = d.internalRead(ctx, uuid, &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API", err.Error()))
		return
	}

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (d *datasourceDedicatedKubernetesEngine) internalRead(ctx context.Context, clusterId string, state *dedicatedKubernetesEngine) (*dedicatedKubernetesEngineReadResponse, error) {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+clusterId+", VPC ID "+vpcId)

	a, err := d.client.SendGetRequest(commons.ApiPath.DedicatedFKEGet(vpcId, clusterId))

	if err != nil {
		return nil, err
	}

	var readResponse dedicatedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &readResponse)
	if err != nil {
		tflog.Info(ctx, "Error unmarshalling cluster info for cluster "+clusterId)
		return nil, err
	}
	data := readResponse.Cluster

	var awx dedicatedKubernetesEngineParams
	err = json.Unmarshal([]byte(data.AwxParams), &awx)

	if err != nil {
		tflog.Info(ctx, "Error unmarshalling AWX params for cluster "+clusterId)
		tflog.Info(ctx, "AwxParams is "+data.AwxParams)
		return nil, err
	}

	// resolve edge ID
	edgeId, err := d.dfkeClient.FindEdgeByEdgeGatewayId(ctx, vpcId, data.EdgeID)
	if err != nil {
		return nil, err
	}

	state.ClusterId = types.StringValue(data.ClusterID)
	state.ClusterName = types.StringValue(data.Name)
	state.Version = types.StringValue(awx.K8SVersion)
	state.MasterType = types.StringValue(awx.MasterType)
	state.MasterCount = types.Int64Value(int64(awx.MasterCount))
	state.MasterDiskSize = types.Int64Value(int64(awx.MasterDiskSize))
	state.WorkerType = types.StringValue(awx.WorkerType)
	state.WorkerDiskSize = types.Int64Value(int64(awx.WorkerDiskSize))
	state.NetworkID = types.StringValue(data.NetworkID)
	state.LbSize = types.StringValue(awx.LbSize)
	state.PodNetwork = types.StringValue(awx.PodNetwork + "/" + awx.PodPrefix)
	state.ServiceNetwork = types.StringValue(awx.ServiceNetwork + "/" + awx.ServicePrefix)
	state.NetworkNodePrefix = types.Int64Value(int64(awx.NetworkNodePrefix))
	state.MaxPodPerNode = types.Int64Value(int64(awx.K8SMaxPod))
	state.NfsStatus = types.StringValue(awx.NfsStatus)
	state.NfsDiskSize = types.Int64Value(int64(awx.NfsDiskSize))
	state.StoragePolicy = types.StringValue(awx.StorageProfile)
	state.EdgeID = types.StringValue(edgeId)
	state.ScaleMin = types.Int64Value(int64(awx.ScaleMinSize))
	state.ScaleMax = types.Int64Value(int64(awx.ScaleMaxSize))
	state.NodeDNS = types.StringValue(awx.NodeDNS)
	state.IPPublicFirewall = types.StringValue(awx.IPPublicFirewall)
	state.IPPrivateFirewall = types.StringValue(awx.IPPrivateFirewall)
	state.VpcId = types.StringValue(data.VpcID)
	//state.CustomScript = awx.CustomScript
	//state.EnableCustomScript = awx.EnableCustomScript
	region, err := getRegionFromVpcId(d.tenancyApiClient, ctx, vpcId)
	if err != nil {
		return nil, err
	}
	state.RegionId = types.StringValue(region)

	return &readResponse, nil
}

func (d *datasourceDedicatedKubernetesEngine) findClusterUUID(_ context.Context, vpcId string, clusterId string) (string, error) {
	total := 1
	found := 0

	index := 1
	for found < total {
		path := commons.ApiPath.DedicatedFKEList(vpcId, index, 25)
		data, err := d.client.SendGetRequest(path)
		if err != nil {
			return "", err
		}

		var list dedicatedKubernetesEngineList
		err = json.Unmarshal(data, &list)
		if err != nil {
			return "", err
		}

		if list.Total == 0 {
			return "", errors.New("no cluster with such ID found")
		}

		if len(list.Data) == 0 {
			return "", errors.New("no cluster with such ID found")
		}

		total = list.Total
		index += 1
		for _, entry := range list.Data {
			if entry.ClusterId == clusterId {
				return entry.Id, nil
			}
		}
	}

	return "", errors.New("no cluster with such ID found")
}

type dedicatedKubernetesEngineList struct {
	Data []struct {
		ClusterName string `json:"cluster_name"`
		ClusterId   string `json:"cluster_id,omitempty"`
		Id          string `json:"id,omitempty"`
	} `json:"data"`
	Total int `json:"total"`
}
