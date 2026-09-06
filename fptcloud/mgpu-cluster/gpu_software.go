package fptcloud_mgpu_cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_vpc "terraform-provider-fptcloud/fptcloud/vpc"

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
	operatorVersion := map[string]string{}
	var migStrategy *string

	if software := softwareToJson(state.Software); software != nil {
		operatorVersion[software.SoftwareType] = software.SoftwareVersion
		// mig_strategy is the cluster-level counterpart of the per-pool mig
		// block, and only the GPU operator has one. The API spells it in
		// upper case where the create-cluster body spells it lower case.
		if software.ClusterMigStrategy != "" {
			s := strings.ToUpper(software.ClusterMigStrategy)
			migStrategy = &s
		}
	}

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
	sharingClientType, maxClient := gpuSharingFields(pool.GpuSharing)
	migMode, migProfile := migFields(pool.Mig)

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

// fetchGpuSoftwareWorkers reads a cluster's GPU-software worker groups, keyed
// by pool name. Errors are swallowed on purpose: the GPU software is a second,
// independent backend, and a cluster whose install never completed simply has
// no record there — that must not make the cluster itself unreadable.
func fetchGpuSoftwareWorkers(
	ctx context.Context,
	mgpuClient *MgpuClusterApiClient,
	vpcClient fptcloud_vpc.Service,
	region, vpcId, clusterName, platform, k8sVersion string,
) map[string]gpuSoftwareWorkerRead {
	workers := map[string]gpuSoftwareWorkerRead{}

	tenant, err := vpcClient.GetTenant(ctx)
	if err != nil || tenant == nil || tenant.Id == "" {
		tflog.Info(ctx, "Skipping GPU software read: could not resolve tenant id")
		return workers
	}

	resp, err := mgpuClient.fetchGpuSoftware(
		ctx, vpcId, clusterName, platform,
		tenant.Id, gpuSoftwareRegion(region), requiresV2API(k8sVersion),
	)
	if err != nil {
		tflog.Info(ctx, "Skipping GPU software read for cluster "+clusterName+": "+err.Error())
		return workers
	}

	for _, w := range resp.WorkerGroups {
		workers[w.Name] = w
	}
	return workers
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
	path := commons.ApiPath.ManagedGpuClusterGpuSoftwareGet(vpcId, clusterName, tenantId, region)
	if isV2 {
		path = commons.ApiPath.ManagedGpuClusterGpuSoftwareGetV2(vpcId, clusterName, tenantId, region)
	}
	tflog.Info(ctx, "Fetching GPU software: "+path)

	a, err := m.sendGet(path, strings.ToUpper(platform))
	if err != nil {
		return nil, err
	}

	var resp gpuSoftwareReadResponse
	if err := json.Unmarshal(a, &resp); err != nil {
		return nil, fmt.Errorf("error unmarshalling gpu-clusters response: %w", err)
	}

	return &resp, nil
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
