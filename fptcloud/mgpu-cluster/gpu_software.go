package fptcloud_mgpu_cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_vpc "terraform-provider-fptcloud/fptcloud/vpc"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// A bare-metal cluster lives in two backends that must be kept in step: the
// m-fke cluster (create-cluster, get-shoot-specific, configure-worker-cluster)
// and the GPU software installed on top of it (this file). Creating only the
// first leaves a cluster with no GPU operator, driver, sharing or MIG
// configuration, so create-cluster is always followed by installGpuSoftware.
//
// The two calls are not atomic and the backend offers no rollback: if the
// second fails the cluster still exists, so callers must persist the cluster
// id before surfacing the error, or Terraform loses track of it entirely.

const (
	// gpuSoftwareClusterTypeBareMetal is the only cluster_type this resource
	// creates — mgpu is bare metal by definition.
	gpuSoftwareClusterTypeBareMetal = "BM"

	// gpuSoftwareInitialStatus is the status the console sends when installing
	// GPU software; the backend drives it forward from there.
	gpuSoftwareInitialStatus = "NOTREADY"

	// gpuValueNone is what the API expects for "not configured" across the
	// GPU enums (mig_mode, gpu_scheduler, sharing_client_type).
	gpuValueNone = "NONE"

	// gpuMigDisabled is the mig_profile a pool carries when MIG is off. Note
	// this is a profile name, not an enum value: the profile field says
	// "partitioned into nothing" rather than being omitted.
	gpuMigDisabled = "all-disabled"

	// gpuSharingDisabled is the sharing_client_type reported for a pool whose
	// driver is USER_INSTALL, where sharing is not available at all.
	gpuSharingDisabled = "all-disabled"
)

// gpuSoftwareRegion is the region value the GPU-software body carries. It is
// the same value the fpt-region header uses, so both stay derived from one
// mapping rather than drifting apart.
func gpuSoftwareRegion(region string) string {
	switch region {
	case "VN/HAN":
		return "hanoi-vn"
	case "VN/SGN":
		return "saigon-vn"
	case "VN/HAN2":
		return "hanoi-2-vn"
	case "VN/SGN2":
		return "saigon-02-vn"
	case "JP/JCSI2":
		return "tokyo-jp"
	default:
		return region
	}
}

// buildGpuSoftwareRequest assembles the GPU-software body from cluster state.
// clusterName is the real, suffixed name create-cluster returned — the GPU
// software is addressed by it, not by the name the user typed.
func buildGpuSoftwareRequest(state *managedGpuCluster, clusterName, platform, region, tenantId string, isV2 bool) *gpuSoftwareRequest {
	operatorVersion, migStrategy := softwareToOperatorVersion(state.Software)

	workerGroups := make([]*gpuSoftwareWorkerJson, 0, len(state.Pools))
	for _, pool := range state.Pools {
		if pool == nil {
			continue
		}
		workerGroups = append(workerGroups, gpuSoftwareWorkerFromPool(pool))
	}

	return &gpuSoftwareRequest{
		IsV2:              isV2,
		Name:              clusterName,
		InfraType:         strings.ToUpper(platform),
		Region:            region,
		TenantId:          tenantId,
		KubernetesVersion: state.K8SVersion.ValueString(),
		OperatorVersion:   operatorVersion,
		ClusterType:       gpuSoftwareClusterTypeBareMetal,
		MigStrategy:       migStrategy,
		WorkerGroups:      workerGroups,
		Status:            gpuSoftwareInitialStatus,
	}
}

// softwareAttrTypes is the software set element's attribute type map.
var softwareAttrTypes = map[string]attr.Type{
	"software_type":        types.StringType,
	"software_version":     types.StringType,
	"cluster_mig_strategy": types.StringType,
}

// softwareToOperatorVersion turns the software set into the operator_version
// map the API expects, plus the cluster-level mig_strategy that only the GPU
// operator carries. The API spells the strategy in upper case where the
// Terraform config spells it lower case.
func softwareToOperatorVersion(software types.Set) (map[string]string, *string) {
	operatorVersion := map[string]string{}
	var migStrategy *string

	if software.IsNull() || software.IsUnknown() {
		return operatorVersion, nil
	}

	for _, element := range software.Elements() {
		entry, ok := element.(types.Object)
		if !ok {
			continue
		}
		attrs := entry.Attributes()

		softwareType := objectString(attrs, "software_type")
		if softwareType == "" {
			continue
		}
		operatorVersion[softwareType] = objectString(attrs, "software_version")

		if softwareType == softwareTypeGpuOperator {
			if strategy := objectString(attrs, "cluster_mig_strategy"); strategy != "" {
				s := strings.ToUpper(strategy)
				migStrategy = &s
			}
		}
	}

	return operatorVersion, migStrategy
}

// softwareSetValue rebuilds the software set from what the API reports.
// Operators the cluster does not have come back as an empty version — that is
// how the console uninstalls one, by blanking the value rather than dropping
// the key — so those entries are skipped.
func softwareSetValue(operatorVersion map[string]string, migStrategy string) types.Set {
	elementType := types.ObjectType{AttrTypes: softwareAttrTypes}

	names := make([]string, 0, len(operatorVersion))
	for name, version := range operatorVersion {
		if version != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return types.SetNull(elementType)
	}
	sort.Strings(names)

	elements := make([]attr.Value, 0, len(names))
	for _, name := range names {
		strategy := ""
		if name == softwareTypeGpuOperator {
			strategy = strings.ToLower(migStrategy)
		}
		elements = append(elements, types.ObjectValueMust(softwareAttrTypes, map[string]attr.Value{
			"software_type":        types.StringValue(name),
			"software_version":     types.StringValue(operatorVersion[name]),
			"cluster_mig_strategy": stringOrNull(strategy),
		}))
	}

	return types.SetValueMust(elementType, elements)
}

// gpuSoftwareWorkerFromPool maps one Terraform pool onto its worker_groups
// entry.
//
// A USER_INSTALL pool installs its own driver, so the platform manages neither
// MIG nor sharing for it: every such field collapses to its disabled value
// regardless of what the pool configured.
//
// For every other pool, an unconfigured mig or gpu_sharing block is left out
// of the payload rather than sent as a placeholder — that is what the console
// does, and mig_profile alone carries "all-disabled" to say MIG is off.
func gpuSoftwareWorkerFromPool(pool *managedGpuClusterPool) *gpuSoftwareWorkerJson {
	driverType, driverVersion := gpuDriverFields(pool.GpuDriver)
	migMode, migProfile, sharingClientType, maxClient := gpuSharingFields(pool.GpuSharing)

	if driverType == driverInstallationTypeUserInstall {
		migModeNone := gpuValueNone
		sharingDisabled := gpuSharingDisabled
		return &gpuSoftwareWorkerJson{
			Name:              pool.WorkerPoolID.ValueString(),
			MigMode:           &migModeNone,
			MigProfile:        gpuValueNone,
			SharingClientType: &sharingDisabled,
			MaxClient:         0,
			GpuScheduler:      gpuValueNone,
			DriverType:        driverType,
			DriverVersion:     "",
			EnableOperand:     true,
			GpuType:           pool.GpuType.ValueString(),
		}
	}

	worker := &gpuSoftwareWorkerJson{
		Name: pool.WorkerPoolID.ValueString(),
		// No MIG configured: "all-disabled" is how the API spells that, and
		// mig_mode is left out of the payload entirely.
		MigProfile: migProfileForRequest(migProfile),
		MaxClient:  maxClient,
		// gpu_scheduler has no Terraform field: the console only ever sends
		// NONE and no other valid value is documented yet.
		GpuScheduler:  gpuValueNone,
		DriverType:    driverType,
		DriverVersion: driverVersion,
		EnableOperand: true,
		GpuType:       pool.GpuType.ValueString(),
	}

	if migMode != "" {
		worker.MigMode = &migMode
	}
	if sharingClientType != "" {
		worker.SharingClientType = &sharingClientType
	}

	return worker
}

// normalizeGpuNone maps the API's "not configured" values back to an empty
// string, so a pool that never configured sharing or MIG reads back as a null
// block rather than one full of NONE — otherwise every such config would show
// a permanent diff.
//
// "all-disabled" counts as not-configured for both fields it appears in, but
// for different reasons: as a mig_profile it means MIG is off, and as a
// sharing_client_type it means sharing is unavailable on that pool.
func normalizeGpuNone(value string) string {
	if value == gpuValueNone || value == gpuMigDisabled {
		return ""
	}
	return value
}

// stringOrNull keeps an absent API value null in state rather than turning it
// into an empty string, which would differ from an unset optional attribute.
func stringOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

// gpuSoftwareState is what a read of the GPU-software backend contributes to
// Terraform state: the per-pool settings the shoot does not report, and the
// cluster's installed operators.
type gpuSoftwareState struct {
	workers  map[string]gpuSoftwareWorkerRead
	software types.Set
}

// readGpuSoftwareState reads a cluster's GPU software into the shape state
// needs. Errors are swallowed on purpose: the GPU software is a second,
// independent backend, and a cluster whose install never completed simply has
// no record there — that must not make the cluster itself unreadable.
//
// A 404 is exactly that case, and it self-heals: activating recreates the
// record, the same thing the console does when its detail page finds none.
func readGpuSoftwareState(
	ctx context.Context,
	mgpuClient *MgpuClusterApiClient,
	vpcClient fptcloud_vpc.Service,
	region, vpcId, clusterName, platform, k8sVersion string,
) gpuSoftwareState {
	out := gpuSoftwareState{
		workers:  map[string]gpuSoftwareWorkerRead{},
		software: types.SetNull(types.ObjectType{AttrTypes: softwareAttrTypes}),
	}

	tenant, err := vpcClient.GetTenant(ctx)
	if err != nil || tenant == nil || tenant.Id == "" {
		tflog.Info(ctx, "Skipping GPU software read: could not resolve tenant id")
		return out
	}

	isV2 := requiresV2API(k8sVersion)
	apiRegion := gpuSoftwareRegion(region)

	resp, err := mgpuClient.fetchGpuSoftware(ctx, vpcId, clusterName, platform, tenant.Id, apiRegion, isV2)
	if err != nil && isNotFoundError(err) {
		tflog.Info(ctx, "No GPU software record for cluster "+clusterName+", activating")
		if activateErr := mgpuClient.activateGpuSoftware(ctx, vpcId, clusterName, platform, apiRegion, tenant.Id, isV2); activateErr != nil {
			tflog.Info(ctx, "Could not activate GPU software for cluster "+clusterName+": "+activateErr.Error())
			return out
		}
		resp, err = mgpuClient.fetchGpuSoftware(ctx, vpcId, clusterName, platform, tenant.Id, apiRegion, isV2)
	}
	if err != nil {
		tflog.Info(ctx, "Skipping GPU software read for cluster "+clusterName+": "+err.Error())
		return out
	}

	for _, w := range resp.WorkerGroups {
		out.workers[w.Name] = w
	}
	out.software = softwareSetValue(resp.OperatorVersion, resp.MigStrategy)

	return out
}

// syncGpuSoftware pushes a cluster's operator selection and worker groups to
// the GPU-software backend after its pools or software changed. It is a
// read-modify-write: the endpoint replaces the whole record, so the body is
// whatever the API just returned with only the two fields this provider owns
// swapped in.
//
// Uninstalling an operator means setting its version to the empty string, not
// dropping its key — that is how the console removes one, and dropping the key
// instead leaves the operator installed.
func syncGpuSoftware(
	ctx context.Context,
	mgpuClient *MgpuClusterApiClient,
	vpcClient fptcloud_vpc.Service,
	region string,
	state *managedGpuCluster,
	clusterName, platform string,
) error {
	tenant, err := vpcClient.GetTenant(ctx)
	if err != nil {
		return fmt.Errorf("error resolving tenant: %w", err)
	}
	if tenant == nil || tenant.Id == "" {
		return fmt.Errorf("could not resolve tenant id")
	}

	isV2 := requiresV2API(state.K8SVersion.ValueString())
	apiRegion := gpuSoftwareRegion(region)
	vpcId := state.VpcId.ValueString()

	_, body, err := mgpuClient.fetchGpuSoftwareRaw(ctx, vpcId, clusterName, platform, tenant.Id, apiRegion, isV2)
	if err != nil {
		return fmt.Errorf("error reading GPU software before update: %w", err)
	}

	desired, migStrategy := softwareToOperatorVersion(state.Software)

	// Operators the config no longer lists have to be blanked rather than
	// omitted, so start from the keys the backend already knows about.
	operatorVersion := map[string]string{}
	if existing, ok := body["operator_version"].(map[string]interface{}); ok {
		for name := range existing {
			operatorVersion[name] = ""
		}
	}
	for name, version := range desired {
		operatorVersion[name] = version
	}
	body["operator_version"] = operatorVersion

	if migStrategy != nil {
		body["mig_strategy"] = *migStrategy
	} else {
		body["mig_strategy"] = nil
	}

	workerGroups := make([]*gpuSoftwareWorkerJson, 0, len(state.Pools))
	for _, pool := range state.Pools {
		if pool == nil {
			continue
		}
		workerGroups = append(workerGroups, gpuSoftwareWorkerFromPool(pool))
	}
	body["worker_groups"] = workerGroups

	return mgpuClient.updateGpuSoftware(ctx, vpcId, clusterName, platform, isV2, body)
}

// fetchOperatorVersions reads the catalog of installable operators and the
// versions each offers, keyed by operator type. This is the authority for
// validating gpu_software — hardcoding the list means new versions are
// rejected until the provider is rebuilt.
func (m *MgpuClusterApiClient) fetchOperatorVersions(ctx context.Context, vpcId, platform string) (map[string][]string, error) {
	path := commons.ApiPath.ManagedGpuClusterOperatorVersions(vpcId)
	tflog.Info(ctx, "Fetching operator versions: "+path)

	a, err := m.sendGet(path, strings.ToUpper(platform))
	if err != nil {
		return nil, err
	}

	var resp map[string]operatorVersionEntry
	if err := json.Unmarshal(a, &resp); err != nil {
		return nil, fmt.Errorf("error unmarshalling operator-versions response: %w", err)
	}

	out := make(map[string][]string, len(resp))
	for name, entry := range resp {
		out[name] = entry.SoftwareVersion
	}
	return out, nil
}

// installGpuSoftware performs the second half of cluster creation: installing
// the GPU operator, driver, sharing and MIG configuration onto the cluster
// create-cluster just made.
func (m *MgpuClusterApiClient) installGpuSoftware(ctx context.Context, vpcId, clusterName, platform string, isV2 bool, body *gpuSoftwareRequest) error {
	path := commons.ApiPath.ManagedGpuClusterGpuSoftwareInstall(vpcId, clusterName)
	if isV2 {
		path = commons.ApiPath.ManagedGpuClusterGpuSoftwareInstallV2(vpcId, clusterName)
	}
	tflog.Info(ctx, "Installing GPU software: "+path)

	a, err := m.sendPost(ctx, path, strings.ToUpper(platform), body)
	if err != nil {
		return err
	}

	return gpuSoftwareResponseError(a)
}

// fetchGpuSoftware reads a cluster's GPU software back. It is the only source
// for a pool's gpu_type, sharing and MIG settings — get-shoot-specific does
// not report any of them.
func (m *MgpuClusterApiClient) fetchGpuSoftware(ctx context.Context, vpcId, clusterName, platform, tenantId, region string, isV2 bool) (*gpuSoftwareReadResponse, error) {
	resp, _, err := m.fetchGpuSoftwareRaw(ctx, vpcId, clusterName, platform, tenantId, region, isV2)
	return resp, err
}

// fetchGpuSoftwareRaw is fetchGpuSoftware plus the response decoded into a
// plain map. Updates are replace-the-whole-record, so they have to start from
// everything the API returned — including fields this provider does not model
// — rather than from the typed struct, which would silently drop them.
func (m *MgpuClusterApiClient) fetchGpuSoftwareRaw(ctx context.Context, vpcId, clusterName, platform, tenantId, region string, isV2 bool) (*gpuSoftwareReadResponse, map[string]interface{}, error) {
	path := commons.ApiPath.ManagedGpuClusterGpuSoftwareGet(vpcId, clusterName, tenantId, region)
	if isV2 {
		path = commons.ApiPath.ManagedGpuClusterGpuSoftwareGetV2(vpcId, clusterName, tenantId, region)
	}
	tflog.Info(ctx, "Fetching GPU software: "+path)

	a, err := m.sendGet(path, strings.ToUpper(platform))
	if err != nil {
		return nil, nil, err
	}

	var resp gpuSoftwareReadResponse
	if err := json.Unmarshal(a, &resp); err != nil {
		return nil, nil, fmt.Errorf("error unmarshalling gpu-clusters response: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(a, &raw); err != nil {
		return nil, nil, fmt.Errorf("error unmarshalling gpu-clusters response: %w", err)
	}

	return &resp, raw, nil
}

// updateGpuSoftware writes a modified GPU-software object back. The endpoint
// replaces the whole record, so callers must build body from what
// fetchGpuSoftware returned and change only the fields they mean to — never
// assemble one from scratch, or fields this provider does not model would be
// wiped.
func (m *MgpuClusterApiClient) updateGpuSoftware(ctx context.Context, vpcId, clusterName, platform string, isV2 bool, body map[string]interface{}) error {
	path := commons.ApiPath.ManagedGpuClusterGpuSoftwareInstall(vpcId, clusterName)
	if isV2 {
		path = commons.ApiPath.ManagedGpuClusterGpuSoftwareInstallV2(vpcId, clusterName)
	}
	tflog.Info(ctx, "Updating GPU software: "+path)

	a, err := m.sendPut(ctx, path, strings.ToUpper(platform), body)
	if err != nil {
		return err
	}

	return gpuSoftwareResponseError(a)
}

// activateGpuSoftware recreates a cluster's GPU-software record. Cluster
// creation spans two backends with no rollback, so a cluster can end up
// existing with no GPU-software record at all — reads then 404. The console
// self-heals that by activating, and so does this provider.
func (m *MgpuClusterApiClient) activateGpuSoftware(ctx context.Context, vpcId, clusterName, platform, region, tenantId string, isV2 bool) error {
	path := commons.ApiPath.ManagedGpuClusterGpuSoftwareActivate(vpcId, clusterName)
	if isV2 {
		path = commons.ApiPath.ManagedGpuClusterGpuSoftwareActivateV2(vpcId, clusterName)
	}
	tflog.Info(ctx, "Activating GPU software: "+path)

	// Activation takes only the cluster's identity — unlike install, which
	// carries the full operator and worker-group configuration.
	body := map[string]string{
		"name":       clusterName,
		"infra_type": strings.ToUpper(platform),
		"region":     region,
		"tenant_id":  tenantId,
	}

	a, err := m.sendPost(ctx, path, strings.ToUpper(platform), body)
	if err != nil {
		return err
	}

	return gpuSoftwareResponseError(a)
}

// deleteGpuSoftware removes a cluster's GPU software. It runs after the
// cluster itself is deleted, mirroring the console's ordering.
func (m *MgpuClusterApiClient) deleteGpuSoftware(ctx context.Context, vpcId, clusterName, platform, tenantId string, isV2 bool) error {
	path := commons.ApiPath.ManagedGpuClusterGpuSoftwareDelete(vpcId, clusterName, tenantId)
	if isV2 {
		path = commons.ApiPath.ManagedGpuClusterGpuSoftwareDeleteV2(vpcId, clusterName, tenantId)
	}
	tflog.Info(ctx, "Deleting GPU software: "+path)

	a, err := m.sendDelete(path, strings.ToUpper(platform))
	if err != nil {
		return err
	}

	return gpuSoftwareResponseError(a)
}

// gpuSoftwareResponseError reports an error carried in a 200-level response
// body. The GPU-software endpoints signal some failures that way rather than
// through the HTTP status alone.
func gpuSoftwareResponseError(a []byte) error {
	if len(a) == 0 {
		return nil
	}

	var re map[string]interface{}
	if err := json.Unmarshal(a, &re); err != nil {
		// A non-JSON body from a successful call is not an error in itself.
		return nil
	}

	if hasError, ok := re["error"].(bool); ok && hasError {
		if message, ok := re["message"]; ok {
			return fmt.Errorf("%v", message)
		}
		if mess, ok := re["mess"]; ok {
			return fmt.Errorf("%v", mess)
		}
		return fmt.Errorf("gpu software request failed: %s", string(a))
	}

	return nil
}
