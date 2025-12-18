package fptcloud_database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strconv"
	"strings"
	common "terraform-provider-fptcloud/commons"
	"time"
)

var (
	_ resource.Resource                = &resourceDatabase{}
	_ resource.ResourceWithConfigure   = &resourceDatabase{}
	_ resource.ResourceWithImportState = &resourceDatabase{}

	forceNewPlanModifiersString = []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	}

	forceNewPlanModifiersInt = []planmodifier.Int64{
		int64planmodifier.RequiresReplace(),
	}
)

const (
	errorCallingApi = "Error calling API"
)

type resourceDatabase struct {
	client         *common.Client
	dataBaseClient *databaseApiClient
}

type databaseResourceModel struct {
	Id             types.String `tfsdk:"id" json:"id,omitempty"`
	VpcId          types.String `tfsdk:"vpc_id" json:"vpc_id"`
	NetworkId      types.String `tfsdk:"network_id" json:"network_id"`
	VmNetwork      types.String `tfsdk:"vm_network" json:"vm_network"`
	TypeConfig     types.String `tfsdk:"type_config" json:"type_config"`
	TypeDb         types.String `tfsdk:"type_db" json:"type_db"`
	Version        types.String `tfsdk:"version" json:"version"`
	VdcName        types.String `tfsdk:"vdc_name" json:"vdc_name"`
	IsCluster      types.String `tfsdk:"is_cluster" json:"is_cluster"`
	MasterCount    types.Int64  `tfsdk:"master_count" json:"master_count"`
	WorkerCount    types.Int64  `tfsdk:"worker_count" json:"worker_count"`
	NodeCpu        types.Int64  `tfsdk:"node_cpu" json:"node_cpu"`
	NodeCore       types.Int64  `tfsdk:"node_core" json:"node_core"`
	NodeRam        types.Int64  `tfsdk:"node_ram" json:"node_ram"`
	DataDiskSize   types.Int64  `tfsdk:"data_disk_size" json:"data_disk_size"`
	ClusterName    types.String `tfsdk:"cluster_name" json:"cluster_name"`
	DatabaseName   types.String `tfsdk:"database_name" json:"database_name"`
	VhostName      types.String `tfsdk:"vhost_name" json:"vhost_name"`
	IsPublic       types.String `tfsdk:"is_public" json:"is_public"`
	AdminPassword  types.String `tfsdk:"admin_password" json:"admin_password"`
	StorageProfile types.String `tfsdk:"storage_profile" json:"storage_profile"`
	EdgeId         types.String `tfsdk:"edge_id" json:"edge_id"`
	Edition        types.String `tfsdk:"edition" json:"edition"`
	FlavorId       types.String `tfsdk:"flavor_id" json:"flavor_id"`
	IsOps          types.String `tfsdk:"is_ops" json:"is_ops"`
	Flavor         types.String `tfsdk:"flavor" json:"flavor"`
	NumberOfNode   types.Int64  `tfsdk:"number_of_node" json:"number_of_node"`
	NumberOfShard  types.Int64  `tfsdk:"number_of_shard" json:"number_of_shard"`
	DomainName     types.String `tfsdk:"domain_name" json:"domain_name"`
	TagIds         types.String `tfsdk:"tag_ids"`
}

var timeout = 1800 * time.Second

func NewResourceDatabase() resource.Resource {
	return &resourceDatabase{}
}

func (r *resourceDatabase) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database"
}

func (m *databaseApiClient) applyTagToCluster(clusterId string, tagIds string) error {
	body := map[string]interface{}{
		"cluster_id": clusterId,
		"tag_ids":    tagIds,
	}

	path := common.ApiPath.DatabaseApplyTags()
	_, err := m.sendPost(path, body)
	return err
}

func (r *resourceDatabase) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	// Get current state of the resource
	var currentState databaseResourceModel
	diags := request.Plan.Get(ctx, &currentState)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	f := databaseJson{}
	r.remap(&currentState, &f)
	_, err := json.Marshal(f)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error marshalling JSON", err.Error()))
		return
	}

	path := common.ApiPath.DatabaseCreate()
	tflog.Debug(ctx, "Calling path "+path)
	a, err := r.dataBaseClient.sendPost(path, f)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi, fmt.Sprintf("failed calling path %s: %v", path, err)))
		return
	}
	errorResponse := r.checkForError(a)
	if errorResponse != nil {
		response.Diagnostics.Append(errorResponse)
		return
	}

	var createResponse databaseCreateResponse
	if err = json.Unmarshal(a, &createResponse); err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error()))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Create response: Type=%s, Message=%s", createResponse.Type, createResponse.Message))

	respType := strings.ToLower(createResponse.Type)
	switch respType {
	case "error":
		msg := createResponse.Message
		if msg == "" {
			msg = "unknown error from API"
		}
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error creating database", msg))
		response.State.RemoveResource(ctx)
		return

	case "success":
		msg := createResponse.Message
		if msg == "" {
			msg = "database created successfully"
		}
		tflog.Info(ctx, msg)
		clusterId := createResponse.Data.ClusterId
		currentState.Id = types.StringValue(clusterId)
		if currentState.Id.IsNull() || currentState.Id.ValueString() == "" {
			currentState.Id = types.StringValue("temp-" + strconv.FormatInt(time.Now().Unix(), 10))
		}
		// ===== APPLY TAG AFTER CREATE =====
		if !currentState.TagIds.IsNull() && !currentState.TagIds.IsUnknown() {
			tagIds := strings.TrimSpace(currentState.TagIds.ValueString())
			response.Diagnostics.Append(diags...)
			if response.Diagnostics.HasError() {
				return
			}

			tflog.Info(ctx, "Applying tags to database cluster after create")
			err = r.dataBaseClient.applyTagToCluster(
				currentState.Id.ValueString(),
				tagIds,
			)
			if err != nil {
				response.Diagnostics.Append(
					diag2.NewErrorDiagnostic(
						"Error applying tag to database",
						err.Error(),
					),
				)
				return
			}
		}
		diags = response.State.Set(ctx, &currentState)
		response.Diagnostics.Append(diags...)
	}
}

func (r *resourceDatabase) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	tflog.Info(ctx, "Calling Read method")
	var state databaseResourceModel
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		tflog.Info(ctx, "Error getting state in read method")
		return
	}

	// Keep the value of number_of_node from the state
	originalFlavorId := state.FlavorId
	originalAdminPassword := state.AdminPassword
	tflog.Info(ctx, "Original FlavorId: "+originalFlavorId.ValueString())

	var timeStart = time.Now()
	var timeout = 10 * time.Minute
	var err2 = errors.New("init error (read)")

	for time.Since(timeStart) < timeout && err2 != nil {
		err2 = r.internalRead(ctx, state.Id.ValueString(), &state)
		tflog.Info(ctx, "state_id"+state.Id.ValueString())
		if err2 != nil {
			tflog.Info(ctx, "err2: "+err2.Error())
			time.Sleep(10 * time.Second)
			continue
		}
	}
	if err2 != nil && err2.Error() != "init error (read)" {
		tflog.Info(ctx, "Error reading database currentState2")
		tflog.Info(ctx, err2.Error())
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error reading database currentState", err2.Error()))
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi, err2.Error()))
	}

	// If the NumberOfNode field is reset or has no value after calling internalRead, restore it from the old state.
	if state.FlavorId.IsNull() || state.FlavorId.IsUnknown() {
		state.FlavorId = originalFlavorId
		tflog.Info(ctx, "Restored original FlavorId from previous state")
	}
	if !originalAdminPassword.IsNull() && !originalAdminPassword.IsUnknown() {
		if state.AdminPassword.IsNull() ||
			state.AdminPassword.IsUnknown() ||
			state.AdminPassword.ValueString() == "********" {

			state.AdminPassword = originalAdminPassword
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("READING: number of node is %d (%d master, %d worker)", state.NumberOfNode, state.MasterCount, state.WorkerCount))
	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceDatabase) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan databaseResourceModel
	var state databaseResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	// Only handle tag_ids update
	if !plan.TagIds.IsNull() && !plan.TagIds.IsUnknown() {
		tagIds := strings.TrimSpace(plan.TagIds.ValueString())

		tflog.Info(ctx, "Applying tags to existing database cluster")
		err := r.dataBaseClient.applyTagToCluster(
			state.Id.ValueString(),
			tagIds,
		)
		if err != nil {
			response.Diagnostics.Append(
				diag2.NewErrorDiagnostic(
					"Error applying tag to database",
					err.Error(),
				),
			)
			return
		}
	}
	plan.Id = state.Id
	response.State.Set(ctx, &plan)
}

func (r *resourceDatabase) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state databaseResourceModel
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	path := common.ApiPath.DatabaseDelete(state.Id.ValueString())
	tflog.Debug(ctx, "Calling path "+path)
	_, err := r.dataBaseClient.sendDelete(path)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic(errorCallingApi, fmt.Sprintf("failed calling path %s: %v", path, err)))
		return
	}
}

func (r *resourceDatabase) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Provides a Fpt database cluster which can be used to store data.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Id of the database cluster.",
			},
			"vpc_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The VPC Id of the database cluster.",
			},
			"network_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The network Id of the database cluster.",
			},
			"vm_network": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The VM network of the database cluster.",
			},
			"type_config": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The type of configuration of the database cluster (short-config or custom-config).",
			},
			"type_db": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The type of database of the database cluster",
			},
			"version": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The version of the database cluster.",
			},
			"vdc_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The VDC name of the database cluster.",
			},
			"is_cluster": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The cluster status of the database cluster.",
			},
			"master_count": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
				Description:   "The number of master nodes in the database cluster.",
			},
			"worker_count": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
				Description:   "The number of worker nodes in the database cluster.",
			},
			"node_cpu": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
				Description:   "The number of CPUs in each node of the database cluster.",
			},
			"node_core": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
				Description:   "The number of cores in each node of the database cluster.",
			},
			"node_ram": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
				Description:   "The amount of RAM in each node of the database cluster.",
			},
			"data_disk_size": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
				Description:   "The size of the data disk in each node of the database cluster.",
			},
			"cluster_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The name of the database cluster.",
			},
			"database_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The name of the database in the database cluster.",
			},
			"vhost_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The name of the RabbitMQ database.",
			},
			"is_public": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "Whether the database is public or not.",
			},
			"admin_password": schema.StringAttribute{
				Required:      true,
				Sensitive:     true,
				Description:   "The admin password of the database cluster.",
			},
			"storage_profile": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The storage profile of the database cluster.",
			},
			"edge_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The edge Id of the database cluster.",
			},
			"edition": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The edition of the database cluster.",
			},
			"flavor_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The flavor_id of the database cluster.",
			},
			"is_ops": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "Whether the database is OpenStack or VMware",
			},
			"flavor": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The flavor of the database cluster.",
			},
			"number_of_node": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
				Description:   "The number of nodes in the database cluster.",
			},
			"number_of_shard": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersInt,
				Description:   "The number of shards in the database cluster.",
			},
			"domain_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The domain name of the database cluster.",
			},
			"tag_ids": schema.StringAttribute{
				Optional:    true,
				Description: "List of tag IDs applied to the database",
			},
		},
	}
}

func (r *resourceDatabase) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	var state databaseResourceModel

	state.Id = types.StringValue(request.ID)
	err := r.internalRead(ctx, request.ID, &state)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Error calling API in Import State Method", err.Error()))
		return
	}

	diags := response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceDatabase) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	tflog.Info(ctx, "Configuring")
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*common.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *internal.ClientV1, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	}

	r.client = client
	r.dataBaseClient = newDatabaseApiClient(r.client)
}

// Get resource data from API, then update to terrafrom state
func (r *resourceDatabase) internalRead(ctx context.Context, databaseId string, state *databaseResourceModel) error {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of Database Id "+databaseId+", VPC Id "+vpcId)

	var nodeTotal = 0
	var timeStart = time.Now()
	var status = "undefined"
	var node databaseNode
	var cluster databaseData

	for nodeTotal == 0 && time.Since(timeStart) < timeout && status != "failed" {
		tflog.Info(ctx, "Getting database detail from API")
		path := common.ApiPath.DatabaseGet(databaseId)
		tflog.Debug(ctx, "Calling path "+path)
		a, err := r.dataBaseClient.sendGet(path)
		if err != nil {
			return fmt.Errorf("failed calling path %s: %v", path, err)
		}
		// Convert response to Go struct
		var d databaseReadResponse
		err = json.Unmarshal(a, &d)
		if d.Code == "400" {
			// Syncing VM information when creating database successfully but not yet ready
			if d.Message == "'node-role.database.node'" {
				continue
			}
			// Sometimes the connection pool gets clogged
			if strings.HasPrefix(d.Message, "HTTPConnectionPool:") {
				continue
			}
		}

		cluster = d.Data.Cluster
		node = d.Data.Node
		status = cluster.Status
		if err != nil {
			return err
		}

		// Update node_total
		nodeTotal = int(node.Total)
		if node.Total == 0 {
			tflog.Info(ctx, "Waiting for nodes to be provisioned. Time waited: "+strconv.Itoa(int(time.Since(timeStart).Seconds()))+" seconds.")
			time.Sleep(30 * time.Second)
		}
	}

	if status == "failed" {
		return fmt.Errorf("failed to provision nodes for database! Server error")
	} else if nodeTotal == 0 {
		return fmt.Errorf("Request time out! Can not provision nodes for database")
	} else {
		tflog.Info(ctx, "Provisioned nodes for database successfully!")
		// Update resource status to state
		state.VpcId = types.StringValue(cluster.VpcId)
		state.NetworkId = types.StringValue(cluster.NetworkId)
		state.VmNetwork = types.StringValue(cluster.VmNetwork)
		state.Id = types.StringValue(cluster.ClusterId)
		state.TypeConfig = types.StringValue(cluster.TypeConfig)
		state.TypeDb = types.StringValue(cluster.TypeDb)
		state.Version = types.StringValue(cluster.Version)
		state.IsCluster = types.StringValue(cluster.IsCluster)
		state.MasterCount = types.Int64Value(int64(cluster.MasterCount))
		state.WorkerCount = types.Int64Value(int64(cluster.WorkerCount))
		state.NodeCore = types.Int64Value(int64(cluster.NodeCore))
		state.NodeCpu = types.Int64Value(int64(cluster.NodeCpu))
		state.NodeRam = types.Int64Value(int64(cluster.NodeRam))
		state.DataDiskSize = types.Int64Value(int64(cluster.DataDiskSize))
		state.ClusterName = types.StringValue(cluster.ClusterName)
		state.DatabaseName = types.StringValue(cluster.DatabaseName)
		state.AdminPassword = types.StringValue(cluster.AdminPassword)
		state.StorageProfile = types.StringValue(cluster.StorageProfile)
		state.EdgeId = types.StringValue(cluster.EdgeId)
		state.Edition = types.StringValue(cluster.EngineEdition)
		state.DomainName = types.StringValue("")
		state.VdcName = types.StringValue(node.Items[0].VdcName)
	}
	return nil
}

// Map data from databaseResourceModel to databaseJson
func (r *resourceDatabase) remap(from *databaseResourceModel, to *databaseJson) {
	to.VpcId = from.VpcId.ValueString()
	to.NetworkId = from.NetworkId.ValueString()
	to.VmNetwork = from.VmNetwork.ValueString()
	to.TypeConfig = from.TypeConfig.ValueString()
	to.TypeDb = from.TypeDb.ValueString()
	to.Version = from.Version.ValueString()
	to.VdcName = from.VdcName.ValueString()
	to.IsCluster = from.IsCluster.ValueString()
	to.MasterCount = int(from.MasterCount.ValueInt64())
	to.WorkerCount = int(from.WorkerCount.ValueInt64())
	to.NodeCore = int(from.NodeCore.ValueInt64())
	to.NodeCpu = int(from.NodeCpu.ValueInt64())
	to.NodeRam = int(from.NodeRam.ValueInt64())
	to.DataDiskSize = int(from.DataDiskSize.ValueInt64())
	to.ClusterName = from.ClusterName.ValueString()
	to.DatabaseName = from.DatabaseName.ValueString()
	to.VhostName = from.VhostName.ValueString()
	to.IsPublic = from.IsPublic.ValueString()
	to.AdminPassword = from.AdminPassword.ValueString()
	to.StorageProfile = from.StorageProfile.ValueString()
	to.EdgeId = from.EdgeId.ValueString()
	to.Edition = from.Edition.ValueString()
	to.IsOps = from.IsOps.ValueString()
	to.Flavor = from.Flavor.ValueString()
	to.FlavorId = from.FlavorId.ValueString()
	to.NumberOfNode = int(from.NumberOfNode.ValueInt64())
	to.NumberOfShard = int(from.NumberOfShard.ValueInt64())
	to.DomainName = from.DomainName.ValueString()
}

// Check if the response contains an error
func (r *resourceDatabase) checkForError(a []byte) *diag2.ErrorDiagnostic {
	var re map[string]interface{}
	err := json.Unmarshal(a, &re)
	if err != nil {
		res := diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error())
		return &res
	}
	if _, ok := re["error"]; ok {
		res := diag2.NewErrorDiagnostic("Response contained an error field", "Response body was "+string(a))
		return &res
	}
	return nil
}

type databaseJson struct {
	Id             string `json:"id,omitempty"`
	VpcId          string `json:"vpc_id"`
	NetworkId      string `json:"network_id"`
	VmNetwork      string `json:"vm_network"`
	TypeConfig     string `json:"type_config"`
	TypeDb         string `json:"type_db"`
	Version        string `json:"version"`
	VdcName        string `json:"vdc_name"`
	IsCluster      string `json:"is_cluster"`
	MasterCount    int    `json:"master_count"`
	WorkerCount    int    `json:"worker_count"`
	NodeCpu        int    `json:"node_cpu"`
	NodeCore       int    `json:"node_core"`
	NodeRam        int    `json:"node_ram"`
	DataDiskSize   int    `json:"data_disk_size"`
	ClusterName    string `json:"cluster_name"`
	DatabaseName   string `json:"database_name"`
	VhostName      string `json:"vhost_name"`
	IsPublic       string `json:"is_public"`
	AdminPassword  string `json:"admin_password"`
	StorageProfile string `json:"storage_profile"`
	EdgeId         string `json:"edge_id"`
	Edition        string `json:"edition"`
	FlavorId       string `json:"flavor_id"`
	IsOps          string `json:"is_ops"`
	Flavor         string `json:"flavor"`
	NumberOfNode   int    `json:"number_of_node"`
	NumberOfShard  int    `json:"number_of_shard"`
	DomainName     string `json:"domain_name"`
}

type databaseData struct {
	VpcId           string `json:"vpc_id"`
	OrgName         string `json:"org_name"`
	VcdUrl          string `json:"vcd_url"`
	NetworkId       string `json:"network_id"`
	VmNetwork       string `json:"vm_network"`
	StorageProfile  string `json:"storage_profile"`
	EdgeId          string `json:"edge_id"`
	Flavor          string `json:"flavor"`
	FlavorId        string `json:"flavor_id"`
	ClusterId       string `json:"cluster_id"`
	ClusterName     string `json:"cluster_name"`
	Version         string `json:"version"`
	TypeConfig      string `json:"type_config"`
	TypeDb          string `json:"type_db"`
	EngineDb        string `json:"engine_db"`
	PortDb          string `json:"port_db"`
	EndPoint        string `json:"end_point"`
	MasterCount     int    `json:"master_count"`
	WorkerCount     int    `json:"worker_count"`
	IsCluster       string `json:"is_cluster"`
	IsMonitor       bool   `json:"is_monitor"`
	IsBackup        bool   `json:"is_backup"`
	NodeCpu         int    `json:"node_cpu"`
	NodeCore        int    `json:"node_core"`
	NodeRam         int    `json:"node_ram"`
	DataDiskSize    int    `json:"data_disk_size"`
	IpPublic        string `json:"ip_public"`
	Status          string `json:"status"`
	DatabaseName    string `json:"database_name"`
	VhostName       string `json:"vhost_name"`
	IsPublic        string `json:"is_public"`
	IsOps           string `json:"is_ops"`
	AdminPassword   string `json:"admin_password"`
	SourceClusterId string `json:"source_cluster_id"`
	EngineEdition   string `json:"engine_edition"`
	NumberOfShard   int    `json:"number_of_shard"`
	IsNewVersion    bool   `json:"is_new_version"`
	CreatedAt       string `json:"created_at"`
	IsAlert         bool   `json:"is_alert"`
	IsAutoscaling   bool   `json:"is_autoscaling"`
}

type databaseNode struct {
	Total int64              `json:"total"`
	Items []databaseNodeItem `json:"items"`
}

type databaseNodeItem struct {
	VdcName string `json:"vdc_name"`
}

// Response from API when requesting a database's detail
type databaseReadResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Cluster databaseData `json:"cluster"`
		Node    databaseNode `json:"nodes"`
	} `json:"data,omitempty"`
}

type databaseCreateResponse struct {
	Message   string                     `json:"message"`
	Type      string                     `json:"type"`
	ErrorCode int                        `json:"error_code"`
	Data      databaseCreateResponseData `json:"data"`
}

// Response from API when creating a database
type databaseCreateResponseData struct {
	ClusterId      string `json:"cluster_id"`
	VpcId          string `json:"vpc_id"`
	NetworkId      string `json:"network_id"`
	VmNetwork      string `json:"vm_network"`
	TypeConfig     string `json:"type_config"`
	TypeDb         string `json:"type_db"`
	PortDb         string `json:"port_db"`
	Version        string `json:"version"`
	MasterCount    int    `json:"master_count"`
	WorkerCount    int    `json:"worker_count"`
	IsCluster      string `json:"is_cluster"`
	ClusterName    string `json:"cluster_name"`
	NodeCpu        int    `json:"node_cpu"`
	NodeCore       int    `json:"node_core"`
	NodeRam        int    `json:"node_ram"`
	DataDiskSize   int    `json:"data_disk_size"`
	VdcName        string `json:"vdc_name"`
	StorageProfile string `json:"storage_profile"`
	IsOps          string `json:"is_ops"`
	Flavor         string `json:"flavor"`
	FlavorId       string `json:"flavor_id"`
	NodeCount      int    `json:"node_count"`
	Status         string `json:"status"`
	Zone           string `json:"zone"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
