package fptcloud_database

import (
	"context"
	"encoding/json"
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strconv"
	common "terraform-provider-fptcloud/commons"

	"time"
)

var (
	_ resource.Resource                = &resourceDatabaseStatus{}
	_ resource.ResourceWithConfigure   = &resourceDatabaseStatus{}
	_ resource.ResourceWithImportState = &resourceDatabaseStatus{}
)

type resourceDatabaseStatus struct {
	client *common.Client
}

func NewResourceDatabaseStatus() resource.Resource {
	return &resourceDatabaseStatus{}
}

func (r *resourceDatabaseStatus) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database_status"
}

// Make sure that the database is in the appropriate state
func (r *resourceDatabaseStatus) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	// Get user state of the resource (from terraform)
	var currentState databaseStatusResourceModel
	diags := request.Plan.Get(ctx, &currentState)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Convert currentState to JSON
	var database databaseStatusJson
	r.remap(&currentState, &database)

	// Getting current status of database on the server
	status, err := r.getDatabaseCurrentStatus(ctx, database.Id)
	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Can't find matching database", err.Error()))
		return
	}
	if status == "failed" {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Database failed", err.Error()))
		return
	}

	// Nếu database đang running và khách hàng cần stopped
	if status == "running" && database.Status == "stopped" {
		err = r.stopDatabase(ctx, database.Id)
		if err != nil {
			response.Diagnostics.Append(diag2.NewErrorDiagnostic("Can't stop database", err.Error()))
			return
		}
	} else if status == "stopped" && database.Status == "running" {
		err = r.startDatabase(ctx, database.Id)
		if err != nil {
			response.Diagnostics.Append(diag2.NewErrorDiagnostic("Can't start database", err.Error()))
			return
		}
	}

	// Update new state of resource to terraform state
	currentState.Id = types.StringValue(database.Id)
	currentState.Status = types.StringValue(database.Status)

	diags = response.State.Set(ctx, &currentState)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceDatabaseStatus) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state databaseStatusResourceModel
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Get current status of database
	var err error
	status, err := r.getDatabaseCurrentStatus(ctx, state.Id.ValueString())

	if err != nil {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Can't find matching database", err.Error()))
		return
	} else if status == "failed" {
		response.Diagnostics.Append(diag2.NewErrorDiagnostic("Database failed", err.Error()))
		return
	}

	state.Id = types.StringValue(state.Id.ValueString())
	state.Status = types.StringValue(status)

	diags = response.State.Set(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceDatabaseStatus) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	//TODO implement me
	panic("implement me")
}

func (r *resourceDatabaseStatus) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state databaseStatusResourceModel
	diags := request.State.Get(ctx, &state)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *resourceDatabaseStatus) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Provides a Fpt database cluster status to temporarily stop or start a database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The Id of the database cluster.",
			},
			"status": schema.StringAttribute{
				Required:      true,
				PlanModifiers: forceNewPlanModifiersString,
				Description:   "The status of the database cluster, must be 'running' or 'stopped'.",
			},
		},
	}
}

func (r *resourceDatabaseStatus) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	var state databaseStatusResourceModel

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

func (r *resourceDatabaseStatus) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

	//a, err := newDatabaseApiClient(client)
	//if err != nil {
	//	response.Diagnostics.AddError(
	//		"Error configuring API client",
	//		fmt.Sprintf("%s", err.Error()),
	//	)
	//	return
	//}
	//r.databaseClient = a
}

// Get current status of database (running, stopped, failed)
func (r *resourceDatabaseStatus) getDatabaseCurrentStatus(ctx context.Context, databaseId string) (string, error) {
	status := ""
	var cluster databaseData

	// Get database detail from API by database Id
	path := common.ApiPath.DatabaseGet(databaseId)
	a, err := r.client.SendGetRequest(path)
	if err != nil {
		return status, err
	}

	// Convert response to Go struct
	var d databaseReadResponse
	err = json.Unmarshal(a, &d)
	if err != nil {
		return status, err
	}
	if d.Code != "200" {
		return status, fmt.Errorf("Database not found")
	}
	cluster = d.Data.Cluster

	// Wait for database to be provisioned
	timeStart := time.Now()
	for cluster.Status != "running" && cluster.Status != "stopped" && cluster.Status != "failed" && time.Since(timeStart) < timeout {
		path = common.ApiPath.DatabaseGet(databaseId)
		a, err = r.client.SendGetRequest(path)
		if err != nil {
			return status, err
		}
		err = json.Unmarshal(a, &d)
		if d.Code != "200" {
			return status, fmt.Errorf("Database not found")
		}
		if err != nil {
			return "", err
		}
		cluster = d.Data.Cluster

		tflog.Info(ctx, "Waiting for database to be provisioned. Time waited: "+time.Since(timeStart).String())
		time.Sleep(30 * time.Second)
	}

	if cluster.Status == "running" || cluster.Status == "stopped" || cluster.Status == "failed" {
		status = cluster.Status
	} else {
		return "not found", fmt.Errorf("Request time out! Can not provision database")
	}

	return status, nil
}

// Stop a running database
func (r *resourceDatabaseStatus) stopDatabase(ctx context.Context, databaseId string) error {
	body := map[string]string{
		"cluster_id": databaseId,
	}

	path := common.ApiPath.DatabaseStop()
	_, err := r.client.SendPostRequest(path, body)
	if err != nil {
		tflog.Error(ctx, "Error stopping database: "+err.Error())
		return err
	}

	timeStart := time.Now()
	for time.Since(timeStart) < timeout {
		status, err := r.getDatabaseCurrentStatus(ctx, databaseId)
		if err != nil {
			return err
		}
		if status == "stopped" {
			return nil
		}

		tflog.Info(ctx, "Waiting for nodes to be stopped. Time waited: "+time.Since(timeStart).String())
		time.Sleep(60 * time.Second)
	}

	return fmt.Errorf("Request time out! Can not stop database")
}

// Start a stopped database
func (r *resourceDatabaseStatus) startDatabase(ctx context.Context, databaseId string) error {
	body := map[string]string{
		"cluster_id": databaseId,
	}

	path := common.ApiPath.DatabaseStart()
	_, err := r.client.SendPostRequest(path, body)
	if err != nil {
		return err
	}

	timeStart := time.Now()
	for time.Since(timeStart) < timeout {
		status, err := r.getDatabaseCurrentStatus(ctx, databaseId)
		if err != nil {
			return err
		}
		if status == "running" {
			return nil
		}
		tflog.Info(ctx, "Waiting for nodes to be provisioned. Time waited: "+time.Since(timeStart).String())
		time.Sleep(60 * time.Second)
	}

	return fmt.Errorf("Request time out! Can not start database")
}

// Get resource data from API, then update to terraform state
func (r *resourceDatabaseStatus) internalRead(ctx context.Context, databaseId string, state *databaseStatusResourceModel) error {
	tflog.Info(ctx, "Reading state of Database Id "+databaseId+", VPC Id ")

	var nodeTotal = 0
	var timeStart = time.Now()
	var node databaseNode
	var cluster databaseData

	for nodeTotal == 0 && time.Since(timeStart) < timeout {
		// Get database detail from API by database Id
		a, err := r.client.SendGetRequest(fmt.Sprintf("xplat/database/management/cluster/detail/%s", databaseId))
		if err != nil {
			return err
		}

		// Convert response to Go struct
		var d databaseReadResponse
		err = json.Unmarshal(a, &d)
		cluster = d.Data.Cluster
		node = d.Data.Node
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

	if nodeTotal == 0 {
		return fmt.Errorf("Request time out! Can not provision nodes for database")
	} else {
		// Update resource status to state
		state.Id = types.StringValue(cluster.VpcId)
		state.Status = types.StringValue(cluster.Status)
	}

	return nil
}

// Map data from databaseResourceModel to databaseJson
func (r *resourceDatabaseStatus) remap(from *databaseStatusResourceModel, to *databaseStatusJson) {
	to.Id = from.Id.ValueString()
	to.Status = from.Status.ValueString()
}

// The database status json to send to API
type databaseStatusJson struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

// The database status managed in terraform
type databaseStatusResourceModel struct {
	Id     types.String `tfsdk:"id" json:"id"`
	Status types.String `tfsdk:"status" json:"status"`
}
