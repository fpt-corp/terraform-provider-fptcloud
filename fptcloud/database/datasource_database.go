package fptcloud_database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &datasourceDatabase{}
	_ datasource.DataSourceWithConfigure = &datasourceDatabase{}
)

// maxErrorBodyLength limits how much of an unexpected response body is echoed back in a diagnostic
const maxErrorBodyLength = 2000

// nodeWaitInterval is how long to pause between two attempts while waiting for the nodes
// of a cluster to show up. A variable so that tests do not have to sit through it
var nodeWaitInterval = 15 * time.Second

type datasourceDatabase struct {
	client         *common.Client
	dataBaseClient *databaseApiClient
}

func NewDataSourceDatabase() datasource.DataSource {
	return &datasourceDatabase{}
}

func (d *datasourceDatabase) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database"
}

func (d *datasourceDatabase) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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
	d.dataBaseClient = newDatabaseApiClient(client)
}

func (d *datasourceDatabase) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Retrieves the detail of a FPT Cloud database cluster by its cluster id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster to look up.",
			},
			"vpc_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The VPC Id of the database cluster. Provide it so that the node information is refreshed before the cluster is read.",
			},
			"wait_for_nodes_timeout": schema.StringAttribute{
				Optional:    true,
				Description: "How long to wait for the nodes of the cluster to become available, as a duration such as \"20m\". Nodes only exist once the cluster has finished provisioning, so this is what a freshly created cluster needs. Omit it to read the cluster once.",
			},
			"vdc_name": schema.StringAttribute{
				Computed:    true,
				Description: "The VDC name of the database cluster.",
			},
			"org_name": schema.StringAttribute{
				Computed:    true,
				Description: "The organization name of the database cluster.",
			},
			"vcd_url": schema.StringAttribute{
				Computed:    true,
				Description: "The VCD URL of the database cluster.",
			},
			"network_id": schema.StringAttribute{
				Computed:    true,
				Description: "The network Id of the database cluster.",
			},
			"vm_network": schema.StringAttribute{
				Computed:    true,
				Description: "The VM network of the database cluster.",
			},
			"storage_profile": schema.StringAttribute{
				Computed:    true,
				Description: "The storage profile of the database cluster.",
			},
			"edge_id": schema.StringAttribute{
				Computed:    true,
				Description: "The edge Id of the database cluster.",
			},
			"edge_gateway_id": schema.StringAttribute{
				Computed:    true,
				Description: "The edge gateway Id of the database cluster.",
			},
			"cluster_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the database cluster.",
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The version of the database engine.",
			},
			"is_beta_version": schema.Int64Attribute{
				Computed:    true,
				Description: "Whether the running version is a beta version (1: true, 0: false).",
			},
			"type_config": schema.StringAttribute{
				Computed:    true,
				Description: "The type of configuration of the database cluster (short-config or custom-config).",
			},
			"type_db": schema.StringAttribute{
				Computed:    true,
				Description: "The type of database of the database cluster.",
			},
			"engine_db": schema.StringAttribute{
				Computed:    true,
				Description: "The database engine of the database cluster.",
			},
			"engine_name": schema.StringAttribute{
				Computed:    true,
				Description: "The display name of the database engine.",
			},
			"edition": schema.StringAttribute{
				Computed:    true,
				Description: "The edition of the database engine.",
			},
			"port_db": schema.StringAttribute{
				Computed:    true,
				Description: "The port the database is listening on.",
			},
			"end_point": schema.StringAttribute{
				Computed:    true,
				Description: "The connection endpoint of the database cluster.",
			},
			"master_count": schema.Int64Attribute{
				Computed:    true,
				Description: "The number of master nodes in the database cluster.",
			},
			"worker_count": schema.Int64Attribute{
				Computed:    true,
				Description: "The number of worker nodes in the database cluster.",
			},
			"is_cluster": schema.StringAttribute{
				Computed:    true,
				Description: "Whether the database is deployed as a cluster (yes/no).",
			},
			"is_monitor": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether monitoring is enabled for the database cluster.",
			},
			"is_backup": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether backup is enabled for the database cluster.",
			},
			"node_cpu": schema.Int64Attribute{
				Computed:    true,
				Description: "The number of CPUs in each node of the database cluster.",
			},
			"node_core": schema.Int64Attribute{
				Computed:    true,
				Description: "The number of cores in each node of the database cluster.",
			},
			"node_ram": schema.Int64Attribute{
				Computed:    true,
				Description: "The amount of RAM in each node of the database cluster.",
			},
			"data_disk_size": schema.Int64Attribute{
				Computed:    true,
				Description: "The size of the data disk in each node of the database cluster.",
			},
			"limit_disk_size": schema.Int64Attribute{
				Computed:    true,
				Description: "The maximum data disk size the database cluster can be scaled up to.",
			},
			"ip_public": schema.StringAttribute{
				Computed:    true,
				Description: "The public IP of the database cluster.",
			},
			"ip_vip": schema.StringAttribute{
				Computed:    true,
				Description: "The virtual IP of the database cluster.",
			},
			"engine_active_pitr": schema.Int64Attribute{
				Computed:    true,
				Description: "Whether point in time recovery is active (1: true, 0: false).",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The current status of the database cluster.",
			},
			"database_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the default database in the database cluster.",
			},
			"vhost_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the RabbitMQ virtual host.",
			},
			"is_public": schema.StringAttribute{
				Computed:    true,
				Description: "Whether the database is public or not.",
			},
			"is_ops": schema.StringAttribute{
				Computed:    true,
				Description: "Whether the database runs on OpenStack (yes) or VMware (no).",
			},
			"admin_password": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The admin password of the database cluster (masked by the API).",
			},
			"source_cluster_id": schema.StringAttribute{
				Computed:    true,
				Description: "The Id of the cluster this cluster was restored/cloned from.",
			},
			"number_of_shard": schema.Int64Attribute{
				Computed:    true,
				Description: "The number of shards in the database cluster.",
			},
			"is_new_version": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the database cluster runs a new version.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The creation time of the database cluster.",
			},
			"is_alert": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether alerting is enabled for the database cluster.",
			},
			"is_autoscaling": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether autoscaling is enabled for the database cluster.",
			},
			"list_service": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The list of services available for the database cluster.",
			},
			"backup_service_status": schema.Int64Attribute{
				Computed:    true,
				Description: "The status of the backup service of the database cluster.",
			},
			"need_reboot": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the database cluster needs to be rebooted.",
			},
			"need_restart": schema.Int64Attribute{
				Computed:    true,
				Description: "Whether the database cluster needs to be restarted (1: true, 0: false).",
			},
			"is_beta": schema.Int64Attribute{
				Computed:    true,
				Description: "Whether the database cluster is a beta product (1: true, 0: false).",
			},
			"is_default_version": schema.Int64Attribute{
				Computed:    true,
				Description: "Whether the running version is the default version (1: true, 0: false).",
			},
			"is_deprecated_version": schema.Int64Attribute{
				Computed:    true,
				Description: "Whether the running version is deprecated (1: true, 0: false).",
			},
			"is_eos_version": schema.Int64Attribute{
				Computed:    true,
				Description: "Whether the running version reached end of support (1: true, 0: false).",
			},
			"end_of_support": schema.StringAttribute{
				Computed:    true,
				Description: "The end of support date of the running version.",
			},
			"flavor_id": schema.StringAttribute{
				Computed:    true,
				Description: "The flavor Id of the database cluster.",
			},
			"flavor": schema.StringAttribute{
				Computed:    true,
				Description: "The flavor name of the database cluster.",
			},
			"vm_sync": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the VM information of the database cluster is synced.",
			},
			"nodes_total": schema.Int64Attribute{
				Computed:    true,
				Description: "The total number of nodes (VMs) in the database cluster.",
			},
			// The provider is served over protocol version 5, which does not support
			// nested attributes, hence the object element type
			"nodes": schema.ListAttribute{
				Computed:    true,
				Description: "The list of nodes (VMs) in the database cluster.",
				ElementType: types.ObjectType{AttrTypes: detailNodeAttrTypes()},
			},
		},
	}
}

func (d *datasourceDatabase) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state databaseDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	clusterId := strings.TrimSpace(state.Id.ValueString())
	if clusterId == "" {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(
			"Missing id",
			"id must be a non empty string",
		))
		return
	}

	// The detail endpoint reads node information out of BSS, and BSS is only filled by
	// the sync calls below. When the caller tells us which VPC the cluster lives in we
	// can sync straight away, otherwise we have to read the cluster once to find out.
	configuredVpcId := strings.TrimSpace(state.VpcId.ValueString())
	if configuredVpcId != "" {
		syncVpcInfrastructure(ctx, d.client, configuredVpcId)
	}

	detail, diagErr := d.getDatabaseDetail(ctx, clusterId)
	if diagErr != nil {
		if configuredVpcId == "" {
			response.Diagnostics.Append(diag2.NewWarningDiagnostic(
				"Reading the database cluster failed before it could be synced",
				"Set vpc_id on this data source so the provider syncs the VPC infrastructure into BSS before reading the cluster.",
			))
		}
		response.Diagnostics.Append(diagErr)
		return
	}

	// Read succeeded but the cluster came back without nodes: now that the response told
	// us the VPC, sync and give it one more try
	if configuredVpcId == "" && detail.Nodes.Total == 0 {
		responseVpcId := strings.TrimSpace(strVal(detail.Cluster.VpcId))
		if responseVpcId != "" {
			tflog.Info(ctx, "Database cluster has no node yet, syncing VPC "+responseVpcId+" and reading again")
			syncVpcInfrastructure(ctx, d.client, responseVpcId)

			retried, retryDiagErr := d.getDatabaseDetail(ctx, clusterId)
			if retryDiagErr != nil {
				response.Diagnostics.Append(retryDiagErr)
				return
			}
			detail = retried
		}
	}

	// The caller asked us to keep trying until the nodes are there
	if detail.Nodes.Total == 0 && !state.WaitForNodesTimeout.IsNull() {
		waitTimeout, err := time.ParseDuration(state.WaitForNodesTimeout.ValueString())
		if err != nil {
			response.Diagnostics.Append(diag2.NewErrorDiagnostic(
				"Invalid wait_for_nodes_timeout",
				fmt.Sprintf("%q is not a valid duration: %v. Use a value such as \"20m\".",
					state.WaitForNodesTimeout.ValueString(), err),
			))
			return
		}

		vpcId := configuredVpcId
		if vpcId == "" {
			vpcId = strings.TrimSpace(strVal(detail.Cluster.VpcId))
		}

		waited, waitDiagErr := d.waitForNodes(ctx, clusterId, vpcId, detail, waitTimeout)
		if waitDiagErr != nil {
			response.Diagnostics.Append(waitDiagErr)
			return
		}
		detail = waited

		if detail.Nodes.Total == 0 {
			response.Diagnostics.Append(diag2.NewWarningDiagnostic(
				"Database cluster still has no node",
				fmt.Sprintf("Waited %s for the nodes of cluster %s to show up and none did. The cluster may still be provisioning, read it again later.",
					waitTimeout, clusterId),
			))
		}
	}

	d.mapDetailToState(ctx, detail, &state)

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

// waitForNodes re-reads the cluster while it still has no node. Every attempt syncs the
// VPC first, because BSS is what the detail endpoint reads from and it never refreshes on
// its own. Read errors are tolerated while waiting: the detail endpoint answers with an
// internal error for a while when a cluster is still being provisioned
func (d *datasourceDatabase) waitForNodes(
	ctx context.Context,
	clusterId string,
	vpcId string,
	detail *databaseDetailData,
	waitTimeout time.Duration,
) (*databaseDetailData, *diag2.ErrorDiagnostic) {
	deadline := time.Now().Add(waitTimeout)
	lastDetail := detail
	var lastErr *diag2.ErrorDiagnostic

	tflog.Info(ctx, fmt.Sprintf("Waiting up to %s for the nodes of cluster %s", waitTimeout, clusterId))

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			tflog.Info(ctx, "Giving up on the nodes, the read was cancelled")
			return lastDetail, nil
		case <-time.After(nodeWaitInterval):
		}

		syncVpcInfrastructure(ctx, d.client, vpcId)

		refreshed, diagErr := d.getDatabaseDetail(ctx, clusterId)
		if diagErr != nil {
			tflog.Info(ctx, fmt.Sprintf("Cluster %s is not readable yet: %s", clusterId, diagErr.Detail()))
			lastErr = diagErr
			continue
		}

		lastDetail, lastErr = refreshed, nil
		if lastDetail.Nodes.Total > 0 {
			tflog.Info(ctx, fmt.Sprintf("Cluster %s now reports %d node(s)", clusterId, lastDetail.Nodes.Total))
			return lastDetail, nil
		}
		tflog.Info(ctx, "Cluster "+clusterId+" still has no node, trying again")
	}

	// Only surface a read error when it left us with nothing usable at all
	if lastDetail == nil {
		return nil, lastErr
	}
	return lastDetail, nil
}

// getDatabaseDetail calls the cluster detail API and validates both the HTTP status
// and the status carried inside the response body
func (d *datasourceDatabase) getDatabaseDetail(ctx context.Context, clusterId string) (*databaseDetailData, *diag2.ErrorDiagnostic) {
	path := common.ApiPath.DatabaseGet(clusterId)
	tflog.Debug(ctx, "Calling path "+path)

	body, err := d.dataBaseClient.sendGet(path)
	if err != nil {
		return nil, httpErrorDiagnostic(clusterId, path, err)
	}

	var detailResponse databaseDetailResponse
	if err = json.Unmarshal(body, &detailResponse); err != nil {
		res := diag2.NewErrorDiagnostic(
			"Error unmarshalling response",
			fmt.Sprintf("Failed to parse the response of %s: %v. Response body was %s", path, err, truncateBody(body)),
		)
		return nil, &res
	}

	// The API answers with HTTP 200 even for business errors, the real status lives in the body
	code := string(detailResponse.Code)
	if code != "" && code != "200" {
		res := diag2.NewErrorDiagnostic(
			fmt.Sprintf("Error getting detail of database cluster %s", clusterId),
			fmt.Sprintf("API returned code %s: %s", code, apiMessage(detailResponse.Message, body)),
		)
		return nil, &res
	}
	if strings.EqualFold(string(detailResponse.Type), "error") {
		res := diag2.NewErrorDiagnostic(
			fmt.Sprintf("Error getting detail of database cluster %s", clusterId),
			apiMessage(detailResponse.Message, body),
		)
		return nil, &res
	}

	if strVal(detailResponse.Data.Cluster.ClusterId) == "" {
		res := diag2.NewErrorDiagnostic(
			"Database cluster not found",
			fmt.Sprintf("No database cluster with id %s was found", clusterId),
		)
		return nil, &res
	}

	return &detailResponse.Data, nil
}

// httpErrorDiagnostic turns a transport/HTTP level failure into a readable diagnostic
func httpErrorDiagnostic(clusterId string, path string, err error) *diag2.ErrorDiagnostic {
	var httpErr common.HTTPError
	if !errors.As(err, &httpErr) {
		res := diag2.NewErrorDiagnostic(
			errorCallingApi,
			fmt.Sprintf("failed calling path %s: %v", path, err),
		)
		return &res
	}

	detail := apiMessage("", []byte(httpErr.Reason))
	if detail == "" {
		detail = httpErr.Status
	}

	var summary, hint string
	switch httpErr.Code {
	case 401, 403:
		summary = "Unauthorized when calling the database API"
	case 404:
		summary = "Database cluster not found"
		if detail == httpErr.Status {
			detail = fmt.Sprintf("No database cluster with id %s was found", clusterId)
		}
	case 429:
		summary = "Rate limited by the database API"
	default:
		if httpErr.Code >= 500 {
			summary = "Database API server error"
		} else {
			summary = errorCallingApi
			// The detail endpoint answers 400 with an internal error message when the
			// cluster does not exist, so point at the most likely cause
			if httpErr.Code == 400 {
				hint = fmt.Sprintf("Check that the database cluster %s exists and belongs to the configured region and VPC", clusterId)
			}
		}
	}

	message := fmt.Sprintf("HTTP %d when calling %s: %s", httpErr.Code, path, detail)
	if hint != "" {
		message += ". " + hint
	}

	res := diag2.NewErrorDiagnostic(summary, message)
	return &res
}

// apiMessage picks the most useful error text: the message field of the response when it
// is present, otherwise the message found in the raw body, otherwise the raw body itself
func apiMessage(message apiString, body []byte) string {
	if msg := strings.TrimSpace(string(message)); msg != "" {
		return msg
	}

	if len(body) > 0 {
		// "detail" is what the backend uses for authentication and validation failures
		var parsed struct {
			Message apiString `json:"message"`
			Error   apiString `json:"error"`
			Detail  apiString `json:"detail"`
		}
		if err := json.Unmarshal(body, &parsed); err == nil {
			for _, candidate := range []apiString{parsed.Message, parsed.Error, parsed.Detail} {
				if msg := strings.TrimSpace(string(candidate)); msg != "" {
					return msg
				}
			}
		}
	}

	return truncateBody(body)
}

func truncateBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) > maxErrorBodyLength {
		return trimmed[:maxErrorBodyLength] + "... (truncated)"
	}
	return trimmed
}

func (d *datasourceDatabase) mapDetailToState(ctx context.Context, detail *databaseDetailData, state *databaseDataSourceModel) {
	cluster := detail.Cluster

	state.Id = stringOrNull(cluster.ClusterId)
	state.VpcId = stringOrNull(cluster.VpcId)
	state.VdcName = stringOrNull(firstNodeVdcName(detail.Nodes.Items))
	state.OrgName = stringOrNull(cluster.OrgName)
	state.VcdUrl = stringOrNull(cluster.VcdUrl)
	state.NetworkId = stringOrNull(cluster.NetworkId)
	state.VmNetwork = stringOrNull(cluster.VmNetwork)
	state.StorageProfile = stringOrNull(cluster.StorageProfile)
	state.EdgeId = stringOrNull(cluster.EdgeId)
	state.EdgeGatewayId = stringOrNull(cluster.EdgeGatewayId)
	state.ClusterName = stringOrNull(cluster.ClusterName)
	state.Version = stringOrNull(cluster.Version)
	state.IsBetaVersion = types.Int64Value(int64(cluster.IsBetaVersion))
	state.TypeConfig = stringOrNull(cluster.TypeConfig)
	state.TypeDb = stringOrNull(cluster.TypeDb)
	state.EngineDb = stringOrNull(cluster.EngineDb)
	state.EngineName = stringOrNull(cluster.EngineName)
	state.Edition = stringOrNull(cluster.EngineEdition)
	state.PortDb = stringOrNull(cluster.PortDb)
	state.EndPoint = stringOrNull(cluster.EndPoint)
	state.MasterCount = types.Int64Value(int64(cluster.MasterCount))
	state.WorkerCount = types.Int64Value(int64(cluster.WorkerCount))
	state.IsCluster = stringOrNull(cluster.IsCluster)
	state.IsMonitor = types.BoolValue(cluster.IsMonitor)
	state.IsBackup = types.BoolValue(cluster.IsBackup)
	state.NodeCpu = types.Int64Value(int64(cluster.NodeCpu))
	state.NodeCore = types.Int64Value(int64(cluster.NodeCore))
	state.NodeRam = types.Int64Value(int64(cluster.NodeRam))
	state.DataDiskSize = types.Int64Value(int64(cluster.DataDiskSize))
	state.LimitDiskSize = types.Int64Value(int64(cluster.LimitDiskSize))
	state.IpPublic = stringOrNull(cluster.IpPublic)
	state.IpVip = stringOrNull(cluster.IpVip)
	state.EngineActivePitr = types.Int64Value(int64(cluster.EngineActivePitr))
	state.Status = stringOrNull(cluster.Status)
	state.DatabaseName = stringOrNull(cluster.DatabaseName)
	state.VhostName = stringOrNull(cluster.VhostName)
	state.IsPublic = stringOrNull(cluster.IsPublic)
	state.AdminPassword = stringOrNull(cluster.AdminPassword)
	state.SourceClusterId = stringOrNull(cluster.SourceClusterId)
	state.NumberOfShard = types.Int64Value(int64(cluster.NumberOfShard))
	state.IsNewVersion = types.BoolValue(cluster.IsNewVersion)
	state.CreatedAt = stringOrNull(cluster.CreatedAt)
	state.IsAlert = types.BoolValue(cluster.IsAlert)
	state.IsAutoscaling = types.BoolValue(cluster.IsAutoscaling)
	state.BackupServiceStatus = types.Int64Value(int64(cluster.BackupServiceStatus))
	state.NeedReboot = types.BoolValue(cluster.NeedReboot)
	state.NeedRestart = types.Int64Value(int64(cluster.NeedRestart))
	state.IsBeta = types.Int64Value(int64(cluster.IsBeta))
	state.IsDefaultVersion = types.Int64Value(int64(cluster.IsDefaultVersion))
	state.IsDeprecatedVersion = types.Int64Value(int64(cluster.IsDeprecatedVersion))
	state.IsEosVersion = types.Int64Value(int64(cluster.IsEosVersion))
	state.EndOfSupport = stringOrNull(cluster.EndOfSupport)
	state.FlavorId = stringOrNull(cluster.FlavorId)
	state.VmSync = types.BoolValue(cluster.VmSync)

	// The detail endpoint returns neither is_ops nor the flavor name, so derive the
	// first one from the nodes and use it to look the second one up
	isOps := strings.TrimSpace(strVal(cluster.IsOps))
	if isOps == "" {
		isOps = isOpsFromNodes(detail.Nodes.Items)
	}
	state.IsOps = stringValueOrNull(isOps)

	flavor := strings.TrimSpace(strVal(cluster.Flavor))
	if flavor == "" {
		flavor = d.lookupFlavorName(ctx, strVal(cluster.VpcId), isOps, strVal(cluster.FlavorId))
	}
	state.Flavor = stringValueOrNull(flavor)

	state.ListService = buildServiceList(ctx, cluster.ListService)
	state.NodesTotal = types.Int64Value(detail.Nodes.Total)
	state.Nodes = buildDetailNodesList(ctx, detail.Nodes.Items)
}

// lookupFlavorName resolves a flavor id into its name using the flavor list of the VPC.
// It returns an empty string when the lookup is not possible, since the flavor name is
// extra information and must not make the whole read fail
func (d *datasourceDatabase) lookupFlavorName(ctx context.Context, vpcId string, isOps string, flavorId string) string {
	if vpcId == "" || isOps == "" || flavorId == "" {
		return ""
	}

	path := common.ApiPath.DatabaseFlavor(vpcId, isOps)
	tflog.Debug(ctx, "Calling path "+path)

	body, err := d.dataBaseClient.sendGet(path)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Could not resolve the name of flavor %s: %v", flavorId, err))
		return ""
	}

	var flavorResponse databaseFlavorListResponse
	if err = json.Unmarshal(body, &flavorResponse); err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Could not parse the flavor list: %v", err))
		return ""
	}

	for _, flavor := range flavorResponse.Data {
		if flavor.FlavorId == flavorId {
			return flavor.FlavorName
		}
	}

	tflog.Warn(ctx, fmt.Sprintf("Flavor %s was not found in the flavor list of VPC %s", flavorId, vpcId))
	return ""
}

// isOpsFromNodes derives the is_ops flag from the platform the nodes run on
func isOpsFromNodes(items []databaseDetailNodeItem) string {
	for _, item := range items {
		platform := strings.TrimSpace(strVal(item.Platform))
		if platform == "" {
			continue
		}
		if strings.EqualFold(platform, "OSP") {
			return "yes"
		}
		return "no"
	}
	return ""
}

// firstNodeVdcName returns the VDC name of the first node that carries one, since the
// cluster part of the response does not include it
func firstNodeVdcName(items []databaseDetailNodeItem) *string {
	for _, item := range items {
		if strVal(item.VdcName) != "" {
			return item.VdcName
		}
	}
	return nil
}

func buildServiceList(ctx context.Context, services []string) types.List {
	if len(services) == 0 {
		return types.ListValueMust(types.StringType, []attr.Value{})
	}

	values := make([]attr.Value, 0, len(services))
	for _, service := range services {
		values = append(values, types.StringValue(service))
	}

	result, diags := types.ListValue(types.StringType, values)
	if diags.HasError() {
		tflog.Warn(ctx, fmt.Sprintf("Failed to build list_service, returning empty list: %v", diags.Errors()))
		return types.ListValueMust(types.StringType, []attr.Value{})
	}
	return result
}

// detailNodeAttrTypes returns the attribute types of a node of the database detail data source
func detailNodeAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":             types.StringType,
		"name":           types.StringType,
		"cluster_id":     types.StringType,
		"cluster_name":   types.StringType,
		"vpc_id":         types.StringType,
		"vdc_name":       types.StringType,
		"org_name":       types.StringType,
		"vmw_id":         types.StringType,
		"status":         types.StringType,
		"guest_os":       types.StringType,
		"ip_address":     types.StringType,
		"ip_private":     types.StringType,
		"network_name":   types.StringType,
		"database_role":  types.StringType,
		"node_type":      types.StringType,
		"number_of_cpus": types.Int64Type,
		"memory_mb":      types.Int64Type,
		"data_disk_size": types.Int64Type,
		"is_deployed":    types.BoolType,
		"platform":       types.StringType,
		"shard_id":       types.StringType,
		"replica_set":    types.StringType,
		"created_at":     types.StringType,
		"updated_at":     types.StringType,
	}
}

func emptyDetailNodesList() types.List {
	return types.ListValueMust(types.ObjectType{AttrTypes: detailNodeAttrTypes()}, []attr.Value{})
}

// buildDetailNodesList converts the nodes of the API response into a types.List,
// returning an empty list when there is nothing to convert
func buildDetailNodesList(ctx context.Context, items []databaseDetailNodeItem) types.List {
	attrTypes := detailNodeAttrTypes()
	if len(items) == 0 {
		return emptyDetailNodesList()
	}

	nodeObjects := make([]attr.Value, 0, len(items))
	for _, item := range items {
		nodeObj, diags := types.ObjectValue(attrTypes, map[string]attr.Value{
			"id":             stringOrNull(item.Id),
			"name":           stringOrNull(item.Name),
			"cluster_id":     stringOrNull(item.ClusterId),
			"cluster_name":   stringOrNull(item.ClusterName),
			"vpc_id":         stringOrNull(item.VpcId),
			"vdc_name":       stringOrNull(item.VdcName),
			"org_name":       stringOrNull(item.OrgName),
			"vmw_id":         stringOrNull(item.VmwId),
			"status":         stringOrNull(item.Status),
			"guest_os":       stringOrNull(item.GuestOs),
			"ip_address":     stringOrNull(item.IpAddress),
			"ip_private":     stringOrNull(item.IpPrivate),
			"network_name":   stringOrNull(item.NetworkName),
			"database_role":  stringOrNull(item.DatabaseRole),
			"node_type":      stringOrNull(item.NodeType),
			"number_of_cpus": types.Int64Value(int64(item.NumberOfCpus)),
			"memory_mb":      types.Int64Value(int64(item.MemoryMb)),
			"data_disk_size": types.Int64Value(int64(item.DataDiskSize)),
			"is_deployed":    types.BoolValue(item.IsDeployed),
			"platform":       stringOrNull(item.Platform),
			"shard_id":       stringOrNull(item.ShardId),
			"replica_set":    stringOrNull(item.ReplicaSet),
			"created_at":     stringOrNull(item.CreatedAt),
			"updated_at":     stringOrNull(item.UpdatedAt),
		})
		if diags.HasError() {
			tflog.Warn(ctx, fmt.Sprintf("Failed to build node object for item %s, skipping: %v", strVal(item.Id), diags.Errors()))
			continue
		}
		nodeObjects = append(nodeObjects, nodeObj)
	}

	result, diags := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, nodeObjects)
	if diags.HasError() {
		tflog.Warn(ctx, fmt.Sprintf("Failed to build nodes list, returning empty list: %v", diags.Errors()))
		return emptyDetailNodesList()
	}
	return result
}

// stringOrNull maps a nullable JSON string to its Terraform counterpart
func stringOrNull(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func strVal(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// stringValueOrNull maps a derived value to null when it could not be determined
func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

// apiString stores the textual form of a JSON value that the API may encode
// as a string, a number or an object depending on whether the call succeeded
type apiString string

func (s *apiString) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = apiString(str)
		return nil
	}

	raw := strings.TrimSpace(string(b))
	if raw == "null" {
		raw = ""
	}
	*s = apiString(raw)
	return nil
}

type databaseDataSourceModel struct {
	Id                  types.String `tfsdk:"id"`
	VpcId               types.String `tfsdk:"vpc_id"`
	VdcName             types.String `tfsdk:"vdc_name"`
	OrgName             types.String `tfsdk:"org_name"`
	VcdUrl              types.String `tfsdk:"vcd_url"`
	NetworkId           types.String `tfsdk:"network_id"`
	VmNetwork           types.String `tfsdk:"vm_network"`
	StorageProfile      types.String `tfsdk:"storage_profile"`
	EdgeId              types.String `tfsdk:"edge_id"`
	EdgeGatewayId       types.String `tfsdk:"edge_gateway_id"`
	ClusterName         types.String `tfsdk:"cluster_name"`
	Version             types.String `tfsdk:"version"`
	IsBetaVersion       types.Int64  `tfsdk:"is_beta_version"`
	TypeConfig          types.String `tfsdk:"type_config"`
	TypeDb              types.String `tfsdk:"type_db"`
	EngineDb            types.String `tfsdk:"engine_db"`
	EngineName          types.String `tfsdk:"engine_name"`
	Edition             types.String `tfsdk:"edition"`
	PortDb              types.String `tfsdk:"port_db"`
	EndPoint            types.String `tfsdk:"end_point"`
	MasterCount         types.Int64  `tfsdk:"master_count"`
	WorkerCount         types.Int64  `tfsdk:"worker_count"`
	IsCluster           types.String `tfsdk:"is_cluster"`
	IsMonitor           types.Bool   `tfsdk:"is_monitor"`
	IsBackup            types.Bool   `tfsdk:"is_backup"`
	NodeCpu             types.Int64  `tfsdk:"node_cpu"`
	NodeCore            types.Int64  `tfsdk:"node_core"`
	NodeRam             types.Int64  `tfsdk:"node_ram"`
	DataDiskSize        types.Int64  `tfsdk:"data_disk_size"`
	LimitDiskSize       types.Int64  `tfsdk:"limit_disk_size"`
	IpPublic            types.String `tfsdk:"ip_public"`
	IpVip               types.String `tfsdk:"ip_vip"`
	EngineActivePitr    types.Int64  `tfsdk:"engine_active_pitr"`
	Status              types.String `tfsdk:"status"`
	DatabaseName        types.String `tfsdk:"database_name"`
	VhostName           types.String `tfsdk:"vhost_name"`
	IsPublic            types.String `tfsdk:"is_public"`
	IsOps               types.String `tfsdk:"is_ops"`
	WaitForNodesTimeout types.String `tfsdk:"wait_for_nodes_timeout"`
	AdminPassword       types.String `tfsdk:"admin_password"`
	SourceClusterId     types.String `tfsdk:"source_cluster_id"`
	NumberOfShard       types.Int64  `tfsdk:"number_of_shard"`
	IsNewVersion        types.Bool   `tfsdk:"is_new_version"`
	CreatedAt           types.String `tfsdk:"created_at"`
	IsAlert             types.Bool   `tfsdk:"is_alert"`
	IsAutoscaling       types.Bool   `tfsdk:"is_autoscaling"`
	ListService         types.List   `tfsdk:"list_service"`
	BackupServiceStatus types.Int64  `tfsdk:"backup_service_status"`
	NeedReboot          types.Bool   `tfsdk:"need_reboot"`
	NeedRestart         types.Int64  `tfsdk:"need_restart"`
	IsBeta              types.Int64  `tfsdk:"is_beta"`
	IsDefaultVersion    types.Int64  `tfsdk:"is_default_version"`
	IsDeprecatedVersion types.Int64  `tfsdk:"is_deprecated_version"`
	IsEosVersion        types.Int64  `tfsdk:"is_eos_version"`
	EndOfSupport        types.String `tfsdk:"end_of_support"`
	FlavorId            types.String `tfsdk:"flavor_id"`
	Flavor              types.String `tfsdk:"flavor"`
	VmSync              types.Bool   `tfsdk:"vm_sync"`
	NodesTotal          types.Int64  `tfsdk:"nodes_total"`
	Nodes               types.List   `tfsdk:"nodes"`
}

// Response from API when requesting the detail of a database cluster
type databaseDetailResponse struct {
	Code    apiString          `json:"code"`
	Message apiString          `json:"message"`
	Type    apiString          `json:"type"`
	Data    databaseDetailData `json:"data"`
}

type databaseDetailData struct {
	Cluster databaseDetailCluster `json:"cluster"`
	Nodes   databaseDetailNodes   `json:"nodes"`
}

type databaseDetailCluster struct {
	VpcId               *string  `json:"vpc_id"`
	OrgName             *string  `json:"org_name"`
	VcdUrl              *string  `json:"vcd_url"`
	NetworkId           *string  `json:"network_id"`
	VmNetwork           *string  `json:"vm_network"`
	StorageProfile      *string  `json:"storage_profile"`
	EdgeId              *string  `json:"edge_id"`
	EdgeGatewayId       *string  `json:"edge_gateway_id"`
	ClusterId           *string  `json:"cluster_id"`
	ClusterName         *string  `json:"cluster_name"`
	Version             *string  `json:"version"`
	IsBetaVersion       int      `json:"is_beta_version"`
	TypeConfig          *string  `json:"type_config"`
	TypeDb              *string  `json:"type_db"`
	EngineDb            *string  `json:"engine_db"`
	EngineName          *string  `json:"engine_name"`
	EngineEdition       *string  `json:"engine_edition"`
	PortDb              *string  `json:"port_db"`
	EndPoint            *string  `json:"end_point"`
	MasterCount         int      `json:"master_count"`
	WorkerCount         int      `json:"worker_count"`
	IsCluster           *string  `json:"is_cluster"`
	IsMonitor           bool     `json:"is_monitor"`
	IsBackup            bool     `json:"is_backup"`
	NodeCpu             int      `json:"node_cpu"`
	NodeCore            int      `json:"node_core"`
	NodeRam             int      `json:"node_ram"`
	DataDiskSize        int      `json:"data_disk_size"`
	LimitDiskSize       int      `json:"limit_disk_size"`
	IpPublic            *string  `json:"ip_public"`
	IpVip               *string  `json:"ip_vip"`
	EngineActivePitr    int      `json:"engine_active_pitr"`
	Status              *string  `json:"status"`
	DatabaseName        *string  `json:"database_name"`
	VhostName           *string  `json:"vhost_name"`
	IsPublic            *string  `json:"is_public"`
	IsOps               *string  `json:"is_ops"`
	AdminPassword       *string  `json:"admin_password"`
	SourceClusterId     *string  `json:"source_cluster_id"`
	NumberOfShard       int      `json:"number_of_shard"`
	IsNewVersion        bool     `json:"is_new_version"`
	CreatedAt           *string  `json:"created_at"`
	IsAlert             bool     `json:"is_alert"`
	IsAutoscaling       bool     `json:"is_autoscaling"`
	ListService         []string `json:"list_service"`
	BackupServiceStatus int      `json:"backup_service_status"`
	NeedReboot          bool     `json:"need_reboot"`
	NeedRestart         int      `json:"need_restart"`
	IsBeta              int      `json:"is_beta"`
	IsDefaultVersion    int      `json:"is_default_version"`
	IsDeprecatedVersion int      `json:"is_deprecated_version"`
	IsEosVersion        int      `json:"is_eos_version"`
	EndOfSupport        *string  `json:"end_of_support"`
	FlavorId            *string  `json:"flavor_id"`
	Flavor              *string  `json:"flavor"`
	VmSync              bool     `json:"vm_sync"`
}

// Response from API when listing the database flavors of a VPC, used to resolve a
// flavor id into a flavor name
type databaseFlavorListResponse struct {
	Data []struct {
		FlavorId   string `json:"flavor_id"`
		FlavorName string `json:"flavor_name"`
	} `json:"data"`
}

type databaseDetailNodes struct {
	Total int64                    `json:"total"`
	Items []databaseDetailNodeItem `json:"items"`
}

type databaseDetailNodeItem struct {
	Id           *string `json:"id"`
	Name         *string `json:"name"`
	ClusterId    *string `json:"cluster_id"`
	ClusterName  *string `json:"cluster_name"`
	VpcId        *string `json:"vpc_id"`
	VdcName      *string `json:"vdc_name"`
	OrgName      *string `json:"org_name"`
	VmwId        *string `json:"vmw_id"`
	Status       *string `json:"status"`
	GuestOs      *string `json:"guest_os"`
	IpAddress    *string `json:"ip_address"`
	IpPrivate    *string `json:"ip_private"`
	NetworkName  *string `json:"network_name"`
	DatabaseRole *string `json:"database_role"`
	NodeType     *string `json:"node_type"`
	NumberOfCpus int     `json:"number_of_cpus"`
	MemoryMb     int     `json:"memory_mb"`
	DataDiskSize int     `json:"data_disk_size"`
	IsDeployed   bool    `json:"is_deployed"`
	Platform     *string `json:"platform"`
	ShardId      *string `json:"shard_id"`
	ReplicaSet   *string `json:"replica_set"`
	CreatedAt    *string `json:"created_at"`
	UpdatedAt    *string `json:"updated_at"`
}
