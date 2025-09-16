package fptcloud_mfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_edge_gateway "terraform-provider-fptcloud/fptcloud/edge_gateway"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"
	"time"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// getNetworkInfoByPlatform network_id, network name, error
func getNetworkInfoByPlatform(ctx context.Context, client fptcloud_subnet.SubnetService, vpcId, platform string, w *managedKubernetesEngineDataWorker, data *managedKubernetesEngineData) (string, string, error) {
	if strings.ToLower(platform) == "vmw" {
		// For VMW platform, bypass network lookup and use the network info from API response
		// The API response doesn't provide network_id, so we'll use empty values
		return "", w.ProviderConfig.NetworkName, nil
	} else {
		return getNetworkByIdOrName(ctx, client, vpcId, "", data.Spec.Provider.InfrastructureConfig.Networks.Id)
	}
}

func getNetworkByIdOrName(ctx context.Context, client fptcloud_subnet.SubnetService, vpcId string, networkName string, networkId string) (string, string, error) {
	if networkName != "" && networkId != "" {
		return "", "", errors.New("only specify network name or id")
	}

	if networkName != "" {
		tflog.Info(ctx, "Resolving network ID for VPC "+vpcId+", network "+networkName)

		networks, err := client.FindSubnetByName(fptcloud_subnet.FindSubnetDTO{
			NetworkName: networkName,
			NetworkID:   networkId,
			VpcId:       vpcId,
		})
		if err != nil {
			return "", "", err
		}

		return networks.NetworkID, networks.NetworkName, nil
	} else {
		tflog.Info(ctx, "Resolving network ID for VPC "+vpcId+", network_id "+networkId)

		networks, err := client.ListSubnet(vpcId)
		if err != nil {
			return "", "", err
		}

		for _, n := range *networks {
			if n.NetworkID == networkId {
				return n.NetworkID, n.NetworkName, nil
			}
		}

		return "", "", errors.New("no such network found")
	}
}

func TopFields() map[string]schema.Attribute {
	topLevelAttributes := map[string]schema.Attribute{}
	// Required string fields
	requiredStrings := []string{
		"vpc_id", "cluster_name", "network_id",
	}
	// Optional string fields
	optionalStrings := []string{
		"k8s_version", "internal_subnet_lb", "edge_gateway_name", "auto_upgrade_timezone", "edge_gateway_id", "network_type",
		"network_overlay", "purpose", "pod_network", "pod_prefix", "service_network", "service_prefix",
	}
	// Required int fields
	requiredInts := []string{}
	// Optional int fields
	optionalInts := []string{"k8s_max_pod"}
	// Optional bool fields
	optionalBools := []string{"is_enable_auto_upgrade"}
	// Special handling for is_running - it should be user-configurable for hibernation/wake-up
	// Optional list fields
	optionalLists := []string{"auto_upgrade_expression"}

	for _, attribute := range requiredStrings {
		topLevelAttributes[attribute] = schema.StringAttribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersString,
			Description:   descriptions[attribute],
		}
	}
	for _, attribute := range optionalStrings {
		topLevelAttributes[attribute] = schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range requiredInts {
		topLevelAttributes[attribute] = schema.Int64Attribute{
			Required:      true,
			PlanModifiers: forceNewPlanModifiersInt,
			Description:   descriptions[attribute],
		}
	}
	for _, attribute := range optionalInts {
		topLevelAttributes[attribute] = schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalBools {
		topLevelAttributes[attribute] = schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: descriptions[attribute],
		}
	}
	// Special handling for is_running - make it computed to avoid inconsistent state
	topLevelAttributes["is_running"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: descriptions["is_running"],
	}
	for _, attribute := range optionalLists {
		topLevelAttributes[attribute] = schema.ListAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: descriptions[attribute],
		}
	}

	return topLevelAttributes
}

func PoolFields() map[string]schema.Attribute {
	poolLevelAttributes := map[string]schema.Attribute{}
	// Required string fields
	requiredStrings := []string{
		"name", "storage_profile", "worker_type",
	}
	// Optional string fields
	optionalStrings := []string{"gpu_sharing_client", "driver_installation_type", "network_name", "network_id", "container_runtime", "gpu_driver_version", "vgpu_id"}
	// Required int fields
	requiredInts := []string{"worker_disk_size", "scale_min", "scale_max"}
	// Optional int fields
	optionalInts := []string{"max_client"}
	// Required bool fields
	requiredBools := []string{}
	// Optional bool fields
	optionalBools := []string{"worker_base", "is_enable_auto_repair"}
	// Optional list fields
	optionalLists := []string{"tags"}

	for _, attribute := range requiredStrings {
		poolLevelAttributes[attribute] = schema.StringAttribute{
			Required:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalStrings {
		poolLevelAttributes[attribute] = schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range requiredInts {
		poolLevelAttributes[attribute] = schema.Int64Attribute{
			Required:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalInts {
		poolLevelAttributes[attribute] = schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range requiredBools {
		poolLevelAttributes[attribute] = schema.BoolAttribute{
			Required:    true,
			Description: descriptions[attribute],
		}
	}

	for _, attribute := range optionalBools {
		poolLevelAttributes[attribute] = schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: descriptions[attribute],
		}
	}
	for _, attribute := range optionalLists {
		poolLevelAttributes[attribute] = schema.ListAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: descriptions[attribute],
		}
	}

	return poolLevelAttributes
}

func MapTerraformToJson(r *resourceManagedKubernetesEngine, ctx context.Context, from *managedKubernetesEngine, to *managedKubernetesEngineJson, vpcId string) *diag2.ErrorDiagnostic {
	to.ClusterName = from.ClusterName.ValueString()
	to.K8SVersion = from.K8SVersion.ValueString()
	to.Purpose = from.Purpose.ValueString()
	defaultNetworkID, defaultNetworkName, err := getNetworkByIdOrName(ctx, r.subnetClient, vpcId, "", from.NetworkID.ValueString())
	fmt.Println("defaultNetworkID: " + defaultNetworkID)
	fmt.Println("defaultNetworkName: " + defaultNetworkName)
	if err != nil {
		d := diag2.NewErrorDiagnostic("Error getting default network", err.Error())
		return &d
	}

	pools := make([]*managedKubernetesEnginePoolJson, 0)
	for _, item := range from.Pools {
		name := item.WorkerPoolID.ValueString()

		kvs := make([]map[string]string, 0)

		if len(item.Kv) > 0 {
			// Sort KV blocks by key name for consistent ordering during plan
			sortedKv := sortKVByKey(item.Kv)

			for _, kv := range sortedKv {
				if kv.Name.IsNull() && kv.Value.IsNull() {
					continue
				}
				key := kv.Name.ValueString()
				val := kv.Value.ValueString()
				if key == "" && val == "" {
					continue
				}

				// Skip system-generated keys when sending request
				if isSystemGeneratedKey(key) {
					continue
				}

				kvs = append(kvs, map[string]string{key: val})
			}
		}

		// Automatically add required system-generated keys for GPU pools
		if item.VGpuID.ValueString() != "" {
			// Check which required GPU labels are already present
			hasMigConfig := false
			hasWorkerType := false
			migConfigValue := "all-1g.6gb" // default value

			for _, kv := range item.Kv {
				if kv.Name.ValueString() == "nvidia.com/mig.config" {
					hasMigConfig = true
					migConfigValue = kv.Value.ValueString() // use user-specified value
				}
				if kv.Name.ValueString() == "worker.fptcloud/type" {
					hasWorkerType = true
				}
			}

			// Add nvidia.com/mig.config if not present, or use user-specified value
			if !hasMigConfig {
				kvs = append(kvs, map[string]string{"nvidia.com/mig.config": migConfigValue})
			}

			// Add worker.fptcloud/type if not present
			if !hasWorkerType {
				kvs = append(kvs, map[string]string{"worker.fptcloud/type": "gpu"})
			}
		}

		taints := make([]map[string]interface{}, 0)
		if len(item.Taints) > 0 {
			for _, taint := range item.Taints {
				if taint.Key.IsNull() && taint.Value.IsNull() && taint.Effect.IsNull() {
					continue
				}
				key := taint.Key.ValueString()
				val := taint.Value.ValueString()
				effect := taint.Effect.ValueString()
				if key == "" && val == "" && effect == "" {
					continue
				}
				taintMap := map[string]interface{}{
					key: map[string]string{
						"value":  val,
						"effect": effect,
					},
				}
				taints = append(taints, taintMap)
			}
		}

		newItem := &managedKubernetesEnginePoolJson{
			StorageProfile:         item.StorageProfile.ValueString(),
			WorkerType:             item.WorkerType.ValueString(),
			WorkerDiskSize:         item.WorkerDiskSize.ValueInt64(),
			ScaleMin:               item.ScaleMin.ValueInt64(),
			ScaleMax:               item.ScaleMax.ValueInt64(),
			IsEnableAutoRepair:     item.IsEnableAutoRepair.ValueBool(),
			WorkerPoolID:           &name,
			VGpuID:                 item.VGpuID.ValueString(),
			MaxClient:              item.MaxClient.ValueInt64(),
			GpuSharingClient:       item.GpuSharingClient.ValueString(),
			GpuDriverVersion:       item.GpuDriverVersion.ValueString(),
			DriverInstallationType: item.DriverInstallationType.ValueString(),
			Tags:                   listToTagsString(item.Tags),
			IsCreate:               true,
			IsScale:                false,
			IsOthers:               false,
			ContainerRuntime:       item.ContainerRuntime.ValueString(),
			Kv:                     kvs,
			Taints:                 taints,
		}
		if item.VGpuID.ValueString() != "" {
			newItem.IsDisplayGPU = true
		} else {
			newItem.IsDisplayGPU = false
		}

		if item.ScaleMin.ValueInt64() == item.ScaleMax.ValueInt64() {
			newItem.AutoScale = false
		} else {
			newItem.AutoScale = true
		}

		if item.NetworkName.ValueString() == "" && item.NetworkID.ValueString() == "" {
			newItem.NetworkName = defaultNetworkName
			newItem.NetworkID = defaultNetworkID
		} else {
			if item.NetworkName.ValueString() == "" {
				_, networkName, err := getNetworkByIdOrName(ctx, r.subnetClient, vpcId, "", item.NetworkID.ValueString())
				if err != nil {
					d := diag2.NewErrorDiagnostic("Error getting network by id", err.Error())
					return &d
				}
				newItem.NetworkName = networkName
				newItem.NetworkID = item.NetworkID.ValueString()
			} else {
				networkID, _, err := getNetworkByIdOrName(ctx, r.subnetClient, vpcId, item.NetworkName.ValueString(), "")
				if err != nil {
					d := diag2.NewErrorDiagnostic("Error getting network by name", err.Error())
					return &d
				}
				newItem.NetworkID = networkID
				newItem.NetworkName = item.NetworkName.ValueString()
			}
		}

		pools = append(pools, newItem)
	}
	to.Pools = pools

	to.NetworkID = from.NetworkID.ValueString()

	to.PodNetwork = from.PodNetwork.ValueString()
	to.PodPrefix = from.PodPrefix.ValueString()
	to.ServiceNetwork = from.ServiceNetwork.ValueString()
	to.ServicePrefix = from.ServicePrefix.ValueString()
	to.K8SMaxPod = from.K8SMaxPod.ValueInt64()
	to.NetworkType = from.NetworkType.ValueString()
	to.NetworkOverlay = from.NetworkOverlay.ValueString()
	to.EdgeGatewayId = from.EdgeGatewayId.ValueString()

	platform, e := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if e != nil {
		d := diag2.NewErrorDiagnostic("Error getting platform for VPC "+vpcId, e.Error())
		return &d
	}

	if strings.ToLower(platform) == "osp" {
		to.EdgeGatewayId = ""
		to.EdgeGatewayName = ""
		to.InternalSubnetLb = ""
	} else {
		// get edge gateway name
		edgeGatewayId := to.EdgeGatewayId
		edge, err := r.GetEdgeGateway(ctx, edgeGatewayId, vpcId)
		if err != nil {
			return err
		}
		to.EdgeGatewayName = edge.Name
	}

	if !from.ClusterEndpointAccess.IsNull() && !from.ClusterEndpointAccess.IsUnknown() {
		attrs := from.ClusterEndpointAccess.Attributes()

		typeStr := attrs["type"].(types.String).ValueString()

		allowCidrsAttr := attrs["allow_cidr"].(types.List)
		var allowCidrs []string
		for _, v := range allowCidrsAttr.Elements() {
			allowCidrs = append(allowCidrs, v.(types.String).ValueString())
		}

		to.ClusterEndpointAccess = &ClusterEndpointAccessJson{
			Type:      typeStr,
			AllowCidr: allowCidrs,
		}
	}

	if !from.IsEnableAutoUpgrade.IsNull() && !from.IsEnableAutoUpgrade.IsUnknown() {
		to.IsEnableAutoUpgrade = from.IsEnableAutoUpgrade.ValueBool()
	}

	if !from.AutoUpgradeExpression.IsNull() && !from.AutoUpgradeExpression.IsUnknown() {
		var exprs []string
		for _, e := range from.AutoUpgradeExpression.Elements() {
			if str, ok := e.(types.String); ok && !str.IsNull() && !str.IsUnknown() {
				exprs = append(exprs, str.ValueString())
			}
		}
		to.AutoUpgradeExpression = exprs
	}

	if !from.AutoUpgradeTimezone.IsNull() && !from.AutoUpgradeTimezone.IsUnknown() {
		to.AutoUpgradeTimezone = from.AutoUpgradeTimezone.ValueString()
	}

	// Map cluster_autoscaler to JSON
	if !from.ClusterAutoscaler.IsNull() && !from.ClusterAutoscaler.IsUnknown() {
		autoscalerAttrs := from.ClusterAutoscaler.Attributes()

		clusterAutoscaler := map[string]interface{}{
			"isEnableAutoScaling":           autoscalerAttrs["is_enable_auto_scaling"].(types.Bool).ValueBool(),
			"scaleDownDelayAfterAdd":        autoscalerAttrs["scale_down_delay_after_add"].(types.Int64).ValueInt64(),
			"scaleDownDelayAfterDelete":     autoscalerAttrs["scale_down_delay_after_delete"].(types.Int64).ValueInt64(),
			"scaleDownDelayAfterFailure":    autoscalerAttrs["scale_down_delay_after_failure"].(types.Int64).ValueInt64(),
			"scaleDownUnneededTime":         autoscalerAttrs["scale_down_unneeded_time"].(types.Int64).ValueInt64(),
			"scaleDownUtilizationThreshold": autoscalerAttrs["scale_down_utilization_threshold"].(types.Float64).ValueFloat64(),
			"scanInterval":                  autoscalerAttrs["scan_interval"].(types.Int64).ValueInt64(),
			"expander":                      strings.ToLower(autoscalerAttrs["expander"].(types.String).ValueString()),
		}

		to.ClusterAutoscaler = clusterAutoscaler
	}

	to.TypeCreate = "create"

	return nil
}

// remapPools
func (r *resourceManagedKubernetesEngine) remapPools(item *managedKubernetesEnginePool, name string, clusterNetworkID string, clusterNetworkName string) *managedKubernetesEnginePoolJson {

	var workerPoolID *string
	if name == "" || name == "worker-new" || item.WorkerPoolID.IsNull() || item.WorkerPoolID.IsUnknown() {
		workerPoolID = nil // new pool
	} else {
		workerPoolID = &name // existing pool
	}

	kvs := make([]map[string]string, 0)
	if len(item.Kv) > 0 {
		// Sort KV blocks by key name for consistent ordering during plan
		sortedKv := sortKVByKey(item.Kv)
		for _, kv := range sortedKv {
			if kv.Name.IsNull() && kv.Value.IsNull() {
				continue
			}
			key := kv.Name.ValueString()
			val := kv.Value.ValueString()
			if key == "" && val == "" {
				continue
			}

			// Skip system-generated keys when sending request
			if isSystemGeneratedKey(key) {
				continue
			}

			kvs = append(kvs, map[string]string{key: val})
		}
	}

	taints := make([]map[string]interface{}, 0)
	if len(item.Taints) > 0 {
		for _, taint := range item.Taints {
			if taint.Key.IsNull() && taint.Value.IsNull() && taint.Effect.IsNull() {
				continue
			}
			key := taint.Key.ValueString()
			val := taint.Value.ValueString()
			effect := taint.Effect.ValueString()
			if key == "" && val == "" && effect == "" {
				continue
			}
			taintMap := map[string]interface{}{
				key: map[string]string{
					"value":  val,
					"effect": effect,
				},
			}
			taints = append(taints, taintMap)
		}
	}

	// Handle network ID and name for VMW platform
	networkID := item.NetworkID.ValueString()
	networkName := item.NetworkName.ValueString()

	// If network ID is empty (common for VMW platform), use cluster's network ID and name
	if networkID == "" {
		networkID = clusterNetworkID
		networkName = clusterNetworkName
	}

	newItem := &managedKubernetesEnginePoolJson{
		WorkerPoolID:           workerPoolID,
		StorageProfile:         item.StorageProfile.ValueString(),
		WorkerType:             item.WorkerType.ValueString(),
		WorkerDiskSize:         item.WorkerDiskSize.ValueInt64(),
		ScaleMin:               item.ScaleMin.ValueInt64(),
		ScaleMax:               item.ScaleMax.ValueInt64(),
		MaxClient:              item.MaxClient.ValueInt64(),
		NetworkID:              networkID,
		NetworkName:            networkName,
		VGpuID:                 item.VGpuID.ValueString(),
		DriverInstallationType: item.DriverInstallationType.ValueString(),
		GpuDriverVersion:       item.GpuDriverVersion.ValueString(),
		Tags:                   listToTagsString(item.Tags),
		GpuSharingClient:       item.GpuSharingClient.ValueString(),
		ContainerRuntime:       item.ContainerRuntime.ValueString(),
		Kv:                     kvs,
		Taints:                 taints,
		AutoScale:              item.ScaleMin.ValueInt64() != item.ScaleMax.ValueInt64(),
		IsDisplayGPU:           false,
		IsCreate:               false,
		IsScale:                false,
		IsOthers:               false,
		IsEnableAutoRepair:     item.IsEnableAutoRepair.ValueBool(),
		WorkerBase:             item.WorkerBase.ValueBool(),
	}

	// Set IsDisplayGPU if VGpuID is set
	if item.VGpuID.ValueString() != "" {
		newItem.IsDisplayGPU = true
	}
	// Set IsCreate for new pool
	if workerPoolID == nil {
		newItem.IsCreate = true
	}

	return newItem
}

// checkForError
func (r *resourceManagedKubernetesEngine) CheckForError(a []byte) *diag2.ErrorDiagnostic {
	var re map[string]interface{}
	err := json.Unmarshal(a, &re)
	if err != nil {
		res := diag2.NewErrorDiagnostic("Error unmarshalling response", err.Error())
		return &res
	}
	if e, ok := re["error"]; ok {
		if e == true {
			res := diag2.NewErrorDiagnostic("Response contained an error field", "Response body was "+string(a))
			return &res
		}
	}
	return nil
}

// diff
func (r *resourceManagedKubernetesEngine) Diff(ctx context.Context, from *managedKubernetesEngine, to *managedKubernetesEngine) *diag2.ErrorDiagnostic {
	// Handle Version changes
	if from.K8SVersion != to.K8SVersion {
		if err := r.UpgradeVersion(ctx, from, to); err != nil {
			return err
		}
	}

	// Handle is_running changes
	if from.IsRunning.ValueBool() != to.IsRunning.ValueBool() {
		vpcId := from.VpcId.ValueString()
		platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error getting platform", err.Error())
			return &d
		}

		platform = strings.ToLower(platform)
		isWakeup := to.IsRunning.ValueBool()
		path := commons.ApiPath.ManagedFKEHibernate(vpcId, platform, from.Id.ValueString(), isWakeup)

		resp, err := r.mfkeClient.sendPatch(ctx, path, platform, nil)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error calling hibernate API", err.Error())
			return &d
		}
		if diagErr := fptcloud_dfke.CheckForError(resp); diagErr != nil {
			return diagErr
		}
	}

	// Handle hibernation schedules changes
	if !to.HibernationSchedules.Equal(from.HibernationSchedules) {
		err := r.updateHibernationSchedules(ctx, to, from)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error updating hibernation schedules", err.Error())
			return &d
		}
	}

	// Handle auto upgrade version changes
	if from.IsEnableAutoUpgrade.ValueBool() != to.IsEnableAutoUpgrade.ValueBool() ||
		!to.AutoUpgradeTimezone.Equal(from.AutoUpgradeTimezone) ||
		!to.AutoUpgradeExpression.Equal(from.AutoUpgradeExpression) {

		exprs := []string{}
		if !to.AutoUpgradeExpression.IsNull() && !to.AutoUpgradeExpression.IsUnknown() {
			for _, e := range to.AutoUpgradeExpression.Elements() {
				if str, ok := e.(types.String); ok && !str.IsNull() && !str.IsUnknown() {
					exprs = append(exprs, str.ValueString())
				}
			}
		}

		body := map[string]interface{}{
			"is_enable_auto_upgrade":  to.IsEnableAutoUpgrade.ValueBool(),
			"auto_upgrade_expression": exprs,
			"auto_upgrade_timezone":   to.AutoUpgradeTimezone.ValueString(),
		}

		vpcId := from.VpcId.ValueString()
		clusterId := from.Id.ValueString()
		platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error getting platform", err.Error())
			return &d
		}
		platform = strings.ToLower(platform)

		path := commons.ApiPath.ManagedFKEAutoUpgradeVersion(vpcId, platform, clusterId)

		res, err := r.mfkeClient.sendPatch(ctx, path, platform, body)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error calling auto upgrade API", err.Error())
			return &d
		}
		if diagErr := r.CheckForError(res); diagErr != nil {
			return diagErr
		}
	}

	// Handle cluster endpoint CIDR changes
	if !to.ClusterEndpointAccess.Equal(from.ClusterEndpointAccess) {
		err := r.updateClusterEndpointCIDR(ctx, to, from)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error updating cluster endpoint CIDR", err.Error())
			return &d
		}
	}

	if !to.ClusterAutoscaler.Equal(from.ClusterAutoscaler) {
		err := r.updateClusterAutoscaler(ctx, to, from)
		if err != nil {
			d := diag2.NewErrorDiagnostic("Error updating cluster autoscaler", err.Error())
			return &d
		}
	}

	// Worker pool changes
	if r.DiffPool(ctx, from, to) {
		if err := r.updateWorkerPools(ctx, from, to); err != nil {
			return err
		}
	}

	return nil
}

// upgradeVersion
func (r *resourceManagedKubernetesEngine) UpgradeVersion(ctx context.Context, from *managedKubernetesEngine, to *managedKubernetesEngine) *diag2.ErrorDiagnostic {
	vpcId := from.VpcId.ValueString()
	cluster := from.Id.ValueString()
	targetVersion := to.K8SVersion.ValueString()
	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		d := diag2.NewErrorDiagnostic(platformVpcErrorPrefix+vpcId, err.Error())
		return &d
	}
	platform = strings.ToLower(platform)
	path := fmt.Sprintf(
		"/v1/xplat/fke/vpc/%s/m-fke/%s/upgrade_version_cluster/shoots/%s/k8s-version/%s",
		vpcId,
		platform,
		cluster,
		targetVersion,
	)
	body, err := r.mfkeClient.sendPatch(ctx, path, platform, struct{}{})
	if err != nil {
		d := diag2.NewErrorDiagnostic(
			fmt.Sprintf("Error upgrading version to %s", to.K8SVersion.ValueString()),
			err.Error(),
		)
		return &d
	}
	if diagErr2 := r.CheckForError(body); diagErr2 != nil {
		return diagErr2
	}
	return nil
}

// diffPool
func (r *resourceManagedKubernetesEngine) DiffPool(ctx context.Context, from *managedKubernetesEngine, to *managedKubernetesEngine) bool {
	fromPool := map[string]*managedKubernetesEnginePool{}
	toPool := map[string]*managedKubernetesEnginePool{}

	kvMap := func(p *managedKubernetesEnginePool) map[string]string {
		m := map[string]string{}
		// Sort KV blocks by key name for consistent comparison
		sortedKv := sortKVByKey(p.Kv)
		for _, kv := range sortedKv {
			k := kv.Name.ValueString()
			v := kv.Value.ValueString()
			if k != "" || v != "" {
				m[k] = v
			}
		}
		return m
	}

	taintMap := func(p *managedKubernetesEnginePool) map[string]interface{} {
		m := map[string]interface{}{}
		for _, taint := range p.Taints {
			k := taint.Key.ValueString()
			v := taint.Value.ValueString()
			effect := taint.Effect.ValueString()
			if k != "" || v != "" || effect != "" {
				m[k] = map[string]string{
					"value":  v,
					"effect": effect,
				}
			}
		}
		return m
	}
	for _, pool := range from.Pools {
		fromPool[pool.WorkerPoolID.ValueString()] = pool
		fmt.Printf("fromPool[%s]: %+v\n", pool.WorkerPoolID.ValueString(), *pool)
	}
	for _, pool := range to.Pools {
		toPool[pool.WorkerPoolID.ValueString()] = pool
		fmt.Printf("toPool[%s]: %+v\n", pool.WorkerPoolID.ValueString(), *pool)
	}
	if len(fromPool) != len(toPool) {
		return true
	}
	for _, pool := range from.Pools {
		f := fromPool[pool.WorkerPoolID.ValueString()]
		t := toPool[pool.WorkerPoolID.ValueString()]

		// Debug logging for MaxClient comparison
		if f.MaxClient.ValueInt64() != t.MaxClient.ValueInt64() {
			fmt.Printf("DEBUG: MaxClient changed from %d to %d for pool %s\n",
				f.MaxClient.ValueInt64(), t.MaxClient.ValueInt64(), pool.WorkerPoolID.ValueString())
		}

		// Skip KV comparison for system-generated labels (like nvidia.com/device-plugin.config)
		userDefinedKvMap := filterUserDefinedKV(kvMap(f))
		userDefinedTvMap := filterUserDefinedKV(kvMap(t))

		if f.ScaleMin != t.ScaleMin ||
			f.ScaleMax != t.ScaleMax ||
			f.WorkerBase != t.WorkerBase ||
			f.IsEnableAutoRepair != t.IsEnableAutoRepair ||
			!f.Tags.Equal(t.Tags) ||
			f.MaxClient != t.MaxClient ||
			f.GpuSharingClient.ValueString() != t.GpuSharingClient.ValueString() ||
			!reflect.DeepEqual(userDefinedKvMap, userDefinedTvMap) ||
			!reflect.DeepEqual(taintMap(f), taintMap(t)) {
			return true
		}
	}
	return false
}

// internalRead
func (r *resourceManagedKubernetesEngine) InternalRead(ctx context.Context, id string, state *managedKubernetesEngine) (*managedKubernetesEngineReadResponse, error) {
	vpcId := state.VpcId.ValueString()
	tflog.Info(ctx, "Reading state of cluster ID "+id+", VPC ID "+vpcId)
	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return nil, err
	}
	platform = strings.ToLower(platform)
	path := commons.ApiPath.ManagedFKEGet(vpcId, platform, id)
	a, err := r.mfkeClient.sendGet(path, platform)
	if err != nil {
		return nil, err
	}
	var d managedKubernetesEngineReadResponse
	err = json.Unmarshal(a, &d)
	if err != nil {
		return nil, err
	}
	if d.Error {
		return nil, fmt.Errorf("error: %v", d.Mess)
	}
	data := d.Data
	state.Id = types.StringValue(data.Metadata.Name)
	state.ClusterName = types.StringValue(getClusterName(data.Metadata.Name))
	state.VpcId = types.StringValue(vpcId)
	state.K8SVersion = types.StringValue(data.Spec.Kubernetes.Version)
	if strings.Contains(data.Spec.SeedSelector.MatchLabels.GardenerCloudPurpose, "public") {
		state.Purpose = types.StringValue("public")
	} else if strings.Contains(data.Spec.SeedSelector.MatchLabels.GardenerCloudPurpose, "firewall") {
		state.Purpose = types.StringValue("firewall")
	} else {
		state.Purpose = types.StringValue("private")
	}

	apiPools := make([]*managedKubernetesEnginePool, 0)

	for _, worker := range data.Spec.Provider.Workers {
		flavorPoolKey := "fptcloud.com/flavor_pool_" + worker.Name
		flavorId, ok := data.Metadata.Labels[flavorPoolKey]
		if !ok {
			return nil, errors.New("missing flavor ID on label " + flavorPoolKey)
		}
		autoRepair := worker.AutoRepair()
		networkId, networkName, e := getNetworkInfoByPlatform(ctx, r.subnetClient, vpcId, platform, worker, &data)
		if e != nil {
			return nil, e
		}

		item := &managedKubernetesEnginePool{
			WorkerPoolID:   types.StringValue(worker.Name),
			StorageProfile: types.StringValue(worker.Volume.Type),
			WorkerType:     types.StringValue(flavorId),
			WorkerDiskSize: types.Int64Value(int64(parseNumber(worker.Volume.Size))),
			ScaleMin:       types.Int64Value(int64(worker.Minimum)),
			ScaleMax:       types.Int64Value(int64(worker.Maximum)),
			NetworkID:      types.StringValue(networkId),
			// NetworkName:            types.StringValue(networkName),
			IsEnableAutoRepair:     types.BoolValue(autoRepair),
			ContainerRuntime:       types.StringValue(worker.Cri.Name),
			Tags:                   tagsStringToList(worker.Tags()),
			VGpuID:                 types.StringValue(worker.ProviderConfig.VGpuID),
			DriverInstallationType: types.StringValue(worker.Machine.Image.DriverInstallationType),
			GpuDriverVersion:       types.StringValue(worker.Machine.Image.GpuDriverVersion),
			WorkerBase:             types.BoolValue(worker.IsWorkerBase()),
		}

		// For GPU pools, read values from addons configuration
		if worker.ProviderConfig.VGpuID != "" {
			// Read MaxClient from addons configuration
			maxClientFromAPI := r.MaxClientFromAddons(&data.Spec, worker.Name)
			item.MaxClient = types.Int64Value(maxClientFromAPI)

			// Read GpuSharingClient from addons configuration
			gpuSharingClientFromAPI := r.GpuSharingClientFromAddons(&data.Spec, worker.Name)
			item.GpuSharingClient = types.StringValue(gpuSharingClientFromAPI)
		} else {
			// Non-GPU pools: set default values
			item.MaxClient = types.Int64Value(0)
			item.GpuSharingClient = types.StringValue("")
		}

		if len(worker.Labels) > 0 {
			// Convert labels to map for filtering
			labelMap := make(map[string]string)
			for _, l := range worker.Labels {
				switch m := l.(type) {
				case map[string]interface{}:
					for k, v := range m {
						vs := fmt.Sprint(v)
						labelMap[k] = vs
					}
				case map[string]string:
					for k, v := range m {
						labelMap[k] = v
					}
				}
			}

			// Filter out system-generated labels
			userDefinedLabels := filterUserDefinedKV(labelMap)

			// Convert back to KV slice
			kvs := make([]KV, 0)
			for k, v := range userDefinedLabels {
				kvs = append(kvs, KV{
					Name:  types.StringValue(k),
					Value: types.StringValue(v),
				})
			}
			item.Kv = sortKVByKey(kvs)
		} else {
			item.Kv = []KV{}
		}

		if len(worker.Taints) > 0 {
			taints := make([]Taint, 0)
			for _, t := range worker.Taints {
				switch taintData := t.(type) {
				case map[string]interface{}:
					for key, taintValue := range taintData {
						if taintMap, ok := taintValue.(map[string]interface{}); ok {
							value := ""
							effect := ""
							if v, exists := taintMap["value"]; exists {
								value = fmt.Sprint(v)
							}
							if e, exists := taintMap["effect"]; exists {
								effect = fmt.Sprint(e)
							}
							taints = append(taints, Taint{
								Key:    types.StringValue(key),
								Value:  types.StringValue(value),
								Effect: types.StringValue(effect),
							})
						}
					}
				}
			}
			item.Taints = taints
		} else {
			item.Taints = []Taint{}
		}

		item.NetworkID = types.StringValue(networkId)
		item.NetworkName = types.StringValue(networkName)

		apiPools = append(apiPools, item)
	}

	// Assign the fully reconstructed list of pools to the state
	state.Pools = apiPools

	podNetwork := strings.Split(data.Spec.Networking.Pods, "/")
	state.PodNetwork = types.StringValue(podNetwork[0])
	state.PodPrefix = types.StringValue(podNetwork[1])
	serviceNetwork := strings.Split(data.Spec.Networking.Services, "/")
	state.ServiceNetwork = types.StringValue(serviceNetwork[0])
	state.ServicePrefix = types.StringValue(serviceNetwork[1])
	state.K8SMaxPod = types.Int64Value(int64(data.Spec.Kubernetes.Kubelet.MaxPods))
	state.NetworkOverlay = types.StringValue(data.Spec.Networking.ProviderConfig.Ipip)

	// Parse cluster_autoscaler from API response
	autoscalerMap := map[string]attr.Value{
		"is_enable_auto_scaling":           types.BoolValue(true),
		"scale_down_delay_after_add":       types.Int64Value(3600),
		"scale_down_delay_after_delete":    types.Int64Value(0),
		"scale_down_delay_after_failure":   types.Int64Value(180),
		"scale_down_unneeded_time":         types.Int64Value(1800),
		"scale_down_utilization_threshold": types.Float64Value(0.5),
		"scan_interval":                    types.Int64Value(10),
		"expander":                         types.StringValue("least-waste"),
	}

	// Parse actual values from API response
	if data.Spec.Kubernetes.ClusterAutoscaler.Expander != "" {
		autoscalerMap["expander"] = types.StringValue(data.Spec.Kubernetes.ClusterAutoscaler.Expander)
	}
	autoscalerMap["scale_down_utilization_threshold"] = types.Float64Value(data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownUtilizationThreshold)

	// Parse duration strings to seconds
	if data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownDelayAfterAdd != "" {
		if duration, err := time.ParseDuration(data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownDelayAfterAdd); err == nil {
			autoscalerMap["scale_down_delay_after_add"] = types.Int64Value(int64(duration.Seconds()))
		}
	}
	if data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownDelayAfterDelete != "" {
		if duration, err := time.ParseDuration(data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownDelayAfterDelete); err == nil {
			autoscalerMap["scale_down_delay_after_delete"] = types.Int64Value(int64(duration.Seconds()))
		}
	}
	if data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownDelayAfterFailure != "" {
		if duration, err := time.ParseDuration(data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownDelayAfterFailure); err == nil {
			autoscalerMap["scale_down_delay_after_failure"] = types.Int64Value(int64(duration.Seconds()))
		}
	}
	if data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownUnneededTime != "" {
		if duration, err := time.ParseDuration(data.Spec.Kubernetes.ClusterAutoscaler.ScaleDownUnneededTime); err == nil {
			autoscalerMap["scale_down_unneeded_time"] = types.Int64Value(int64(duration.Seconds()))
		}
	}
	if data.Spec.Kubernetes.ClusterAutoscaler.ScanInterval != "" {
		if duration, err := time.ParseDuration(data.Spec.Kubernetes.ClusterAutoscaler.ScanInterval); err == nil {
			autoscalerMap["scan_interval"] = types.Int64Value(int64(duration.Seconds()))
		}
	}

	state.ClusterAutoscaler, _ = types.ObjectValue(
		map[string]attr.Type{
			"is_enable_auto_scaling":           types.BoolType,
			"scale_down_delay_after_add":       types.Int64Type,
			"scale_down_delay_after_delete":    types.Int64Type,
			"scale_down_delay_after_failure":   types.Int64Type,
			"scale_down_unneeded_time":         types.Int64Type,
			"scale_down_utilization_threshold": types.Float64Type,
			"scan_interval":                    types.Int64Type,
			"expander":                         types.StringType,
		},
		autoscalerMap,
	)
	// Parse cluster_endpoint_access from extensions and metadata labels
	accessMap := map[string]attr.Value{
		"type": types.StringValue("public"),
		"allow_cidr": types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("0.0.0.0/0"),
		}),
	}

	// Determine cluster type from metadata labels
	if _, hasACL := data.Metadata.Labels["extensions.extensions.gardener.cloud/acl"]; hasACL {
		accessMap["type"] = types.StringValue("public")
	} else if _, hasPrivateNetwork := data.Metadata.Labels["extensions.extensions.gardener.cloud/private-network"]; hasPrivateNetwork {
		// For private-network, need to check privateCluster flag in extensions
		for _, extension := range data.Spec.Extensions {
			if extension.Type == "private-network" && extension.ProviderConfig != nil {
				if privateCluster, exists := extension.ProviderConfig["privateCluster"].(bool); exists {
					if privateCluster {
						accessMap["type"] = types.StringValue("private")
					} else {
						accessMap["type"] = types.StringValue("mixed")
					}
				}
				break
			}
		}
	}

	// Parse extensions field to get CIDR configuration
	if len(data.Spec.Extensions) > 0 {
		for _, extension := range data.Spec.Extensions {
			if extension.ProviderConfig != nil {
				var cidrValues []attr.Value

				// Handle ACL extension type (public clusters)
				if extension.Type == "acl" {
					if rule, ok := extension.ProviderConfig["rule"].(map[string]interface{}); ok {
						// Extract allow_cidr values from the ACL rule
						if cidrs, exists := rule["cidrs"].([]interface{}); exists {
							for _, cidr := range cidrs {
								if cidrStr, ok := cidr.(string); ok {
									cidrValues = append(cidrValues, types.StringValue(cidrStr))
								}
							}
						}
					}
				}

				// Handle private-network extension type (private/mixed clusters)
				if extension.Type == "private-network" {
					// Extract allow_cidr values from allowCIDRs field
					if allowCIDRs, exists := extension.ProviderConfig["allowCIDRs"].([]interface{}); exists {
						for _, cidr := range allowCIDRs {
							if cidrStr, ok := cidr.(string); ok {
								cidrValues = append(cidrValues, types.StringValue(cidrStr))
							}
						}
					}
				}

				// Update allow_cidr if we found any CIDR values
				if len(cidrValues) > 0 {
					accessMap["allow_cidr"] = types.ListValueMust(types.StringType, cidrValues)
				}
			}
		}
	}

	state.ClusterEndpointAccess, _ = types.ObjectValue(
		map[string]attr.Type{
			"type":       types.StringType,
			"allow_cidr": types.ListType{ElemType: types.StringType},
		},
		accessMap,
	)

	// Default auto_upgrade_expression if missing
	if state.AutoUpgradeExpression.IsNull() || state.AutoUpgradeExpression.IsUnknown() {
		state.AutoUpgradeExpression = types.ListNull(types.StringType)
	}

	// Use the same logic as the power state resource to determine if cluster is running
	isRunning := false
	if len(data.Status.Conditions) > 0 {
		isRunning = data.Status.Conditions[0].Status == "True"
	}
	if data.Spec.Hibernate != nil {
		isRunning = !data.Spec.Hibernate.Enabled
	}
	state.IsRunning = types.BoolValue(isRunning)

	if d.Data.Spec.Hibernate != nil && d.Data.Spec.Hibernate.Schedules != nil {
		var schedulesFromAPI []HibernationSchedule

		for _, apiSchedule := range d.Data.Spec.Hibernate.Schedules {
			schedulesFromAPI = append(schedulesFromAPI, HibernationSchedule{
				Start:    types.StringValue(apiSchedule.Start),
				End:      types.StringValue(apiSchedule.End),
				Location: types.StringValue(apiSchedule.Location),
			})
		}

		hibernationScheduleObjectType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"start":    types.StringType,
				"end":      types.StringType,
				"location": types.StringType,
			},
		}

		var diags diag2.Diagnostics
		state.HibernationSchedules, diags = types.ListValueFrom(ctx, hibernationScheduleObjectType, schedulesFromAPI)
		if diags.HasError() {
			return nil, fmt.Errorf("error creating hibernation schedules list for state: %v", diags)
		}
	} else {
		state.HibernationSchedules = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"start":    types.StringType,
				"end":      types.StringType,
				"location": types.StringType,
			},
		})
	}

	if d.Data.Spec.AutoUpgrade != nil {
		autoUpgradeInfo := d.Data.Spec.AutoUpgrade

		isEnabled := len(autoUpgradeInfo.TimeUpgrade) > 0
		state.IsEnableAutoUpgrade = types.BoolValue(isEnabled)

		state.AutoUpgradeTimezone = types.StringValue(autoUpgradeInfo.TimeZone)

		if isEnabled {
			listVal, diags := types.ListValueFrom(ctx, types.StringType, autoUpgradeInfo.TimeUpgrade)
			if diags.HasError() {
				return nil, fmt.Errorf("error creating auto_upgrade_expression list for state: %v", diags)
			}
			state.AutoUpgradeExpression = listVal
		} else {
			state.AutoUpgradeExpression = types.ListNull(types.StringType)
		}
	} else {
		state.IsEnableAutoUpgrade = types.BoolValue(false)
		state.AutoUpgradeTimezone = types.StringNull()
		state.AutoUpgradeExpression = types.ListNull(types.StringType)
	}

	return &d, nil
}

// getEdgeGateway
func (r *resourceManagedKubernetesEngine) GetEdgeGateway(_ context.Context, edgeId string, vpcId string) (*fptcloud_edge_gateway.EdgeGatewayData, *diag2.ErrorDiagnostic) {
	path := commons.ApiPath.EdgeGatewayList(vpcId)
	res, err := r.client.SendGetRequest(path)
	if err != nil {
		d := diag2.NewErrorDiagnostic(errorCallingApi(path), err.Error())
		return nil, &d
	}
	var resp fptcloud_edge_gateway.EdgeGatewayResponse
	if err = json.Unmarshal(res, &resp); err != nil {
		diag := diag2.NewErrorDiagnostic(errorCallingApi(path), err.Error())
		return nil, &diag
	}
	for _, item := range resp.Data {
		if item.EdgeGatewayId == edgeId {
			return &item, nil
		}
	}
	diag := diag2.NewErrorDiagnostic("No such Edge Gateway in this VPC", fmt.Sprintf("No edge gateway with ID %s was found in VPC %s", edgeId, vpcId))
	return nil, &diag
}

// getClusterName
func getClusterName(name string) string {
	var indices []int
	for i, c := range name {
		if c == '-' {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		return name
	}
	last := indices[len(indices)-1]
	clusterName := string([]rune(name)[:last])
	return clusterName
}

// parseNumber
func parseNumber(s string) int {
	out := ""
	for _, c := range s {
		if unicode.IsDigit(c) {
			out += string(c)
		}
	}
	if out == "" {
		out = "0"
	}
	f, _ := strconv.Atoi(out)
	return f
}

// errorCallingApi
func errorCallingApi(s string) string {
	return fmt.Sprintf("Error calling path: %s", s)
}

// Helper function to convert types.List to string with \n separator
func listToTagsString(tagsList types.List) string {
	if tagsList.IsNull() || tagsList.IsUnknown() {
		return ""
	}

	var tags []string
	for _, element := range tagsList.Elements() {
		if stringVal, ok := element.(types.String); ok {
			tags = append(tags, stringVal.ValueString())
		}
	}

	return strings.Join(tags, "\n")
}

// Helper function to convert string with \n separator to types.List
func tagsStringToList(tagsString string) types.List {
	if tagsString == "" {
		return types.ListValueMust(types.StringType, []attr.Value{})
	}

	tags := strings.Split(tagsString, "\n")
	var elements []attr.Value
	for _, tag := range tags {
		if strings.TrimSpace(tag) != "" {
			elements = append(elements, types.StringValue(strings.TrimSpace(tag)))
		}
	}

	return types.ListValueMust(types.StringType, elements)
}

func (w *managedKubernetesEngineDataWorker) AutoRepair() bool {
	autoRepair := false
	if label, ok := w.Annotations["worker.fptcloud.com/node-auto-repair"]; ok {
		autoRepair = label == "true"
	}

	return autoRepair
}

func (w *managedKubernetesEngineDataWorker) Tags() string {
	return w.Annotations["tagging.fke.fptcloud.com/worker-tags"]
}

// MaxClient reads the maxClient value from the addons configuration
// The maxClient is stored in spec.addons.gpuOperator.timeSliceConfig.maxClient
// Format: ["pool-name:value"] e.g. ["gpu-test:2"]
func (r *resourceManagedKubernetesEngine) MaxClientFromAddons(spec *managedKubernetesEngineDataSpec, poolName string) int64 {
	if spec.Addons == nil || spec.Addons.GpuOperator == nil || spec.Addons.GpuOperator.TimeSliceConfig == nil {
		return 0
	}

	for _, maxClientStr := range spec.Addons.GpuOperator.TimeSliceConfig.MaxClient {
		// Parse format "pool-name:value" e.g. "gpu-test:2"
		if strings.HasPrefix(maxClientStr, poolName+":") {
			parts := strings.Split(maxClientStr, ":")
			if len(parts) == 2 {
				if value, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					return value
				}
			}
		}
	}
	return 0
}

func (r *resourceManagedKubernetesEngine) GpuSharingClientFromAddons(spec *managedKubernetesEngineDataSpec, poolName string) string {
	if spec.Addons == nil || spec.Addons.GpuOperator == nil || spec.Addons.GpuOperator.TimeSliceConfig == nil {
		return ""
	}

	// Check if this pool has TimeSliceConfig (maxClient configuration)
	for _, maxClientStr := range spec.Addons.GpuOperator.TimeSliceConfig.MaxClient {
		if strings.HasPrefix(maxClientStr, poolName+":") {
			// If pool has TimeSliceConfig, it means gpu_sharing_client = "timeSlicing"
			return "timeSlicing"
		}
	}

	// If no TimeSliceConfig found for this pool, gpu_sharing_client = "" (empty)
	return ""
}

func (w *managedKubernetesEngineDataWorker) IsWorkerBase() bool {
	return w.SystemComponents.Allow
}

func (r *resourceManagedKubernetesEngine) updateHibernationSchedules(ctx context.Context, plan *managedKubernetesEngine, state *managedKubernetesEngine) error {
	vpcId := state.VpcId.ValueString()

	var hibernationSchedulesFromPlan []HibernationSchedule

	diags := plan.HibernationSchedules.ElementsAs(ctx, &hibernationSchedulesFromPlan, false)
	if diags.HasError() {
		return fmt.Errorf("error parsing hibernation schedules from plan: %v", diags.Errors())
	}

	// Convert to JSON format for API
	var schedulesForJson []HibernationScheduleJson
	for _, scheduleData := range hibernationSchedulesFromPlan {
		if scheduleData.Start.IsNull() || scheduleData.End.IsNull() || scheduleData.Location.IsNull() {
			tflog.Warn(ctx, "Skipping one hibernation schedule because it has null values.")
			continue
		}
		schedulesForJson = append(schedulesForJson, HibernationScheduleJson{
			Start:    scheduleData.Start.ValueString(),
			End:      scheduleData.End.ValueString(),
			Location: scheduleData.Location.ValueString(),
		})
	}

	requestBody := HibernationSchedulesRequest{
		Schedules: schedulesForJson,
	}

	if len(requestBody.Schedules) == 0 {
		requestBody.Schedules = []HibernationScheduleJson{}
	}

	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return fmt.Errorf("error getting platform: %v", err)
	}

	platform = strings.ToLower(platform)
	path := commons.ApiPath.ManagedFKEHibernationSchedules(vpcId, platform, state.Id.ValueString())

	resp, err := r.mfkeClient.sendPatch(ctx, path, platform, requestBody)
	if err != nil {
		return fmt.Errorf("error calling hibernation schedules API: %v", err)
	}

	tflog.Info(ctx, "Hibernation schedules API response: "+string(resp))

	tflog.Info(ctx, "Successfully updated hibernation schedules.")

	return nil
}

func (r *resourceManagedKubernetesEngine) updateClusterEndpointCIDR(
	ctx context.Context,
	plan *managedKubernetesEngine,
	state *managedKubernetesEngine,
) error {
	vpcId := state.VpcId.ValueString()
	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return fmt.Errorf("error getting platform: %v", err)
	}
	platform = strings.ToLower(platform)
	path := commons.ApiPath.ManagedFKEUpdateEndpointCIDR(vpcId, platform, state.Id.ValueString())

	// Lấy type từ state
	endpointAccess := state.ClusterEndpointAccess.Attributes()
	endpointType := endpointAccess["type"].(types.String).ValueString()

	// Convert allow_cidr (types.List -> []string)
	allowCidrsList := plan.ClusterEndpointAccess.Attributes()["allow_cidr"].(types.List)
	var allowCidrs []string
	for _, e := range allowCidrsList.Elements() {
		if s, ok := e.(types.String); ok && !s.IsNull() {
			allowCidrs = append(allowCidrs, s.ValueString())
		}
	}

	requestBody := map[string]interface{}{
		"type":      endpointType, // lấy từ state
		"allowCidr": allowCidrs,   // đúng key name theo API
	}

	resp, err := r.mfkeClient.sendPatch(ctx, path, platform, requestBody)
	if err != nil {
		return fmt.Errorf("error calling update cluster endpoint CIDR API: %v", err)
	}

	tflog.Info(ctx, "Cluster endpoint CIDR API response: "+string(resp))
	tflog.Info(ctx, "Successfully updated cluster endpoint CIDR.")

	return nil
}

func (r *resourceManagedKubernetesEngine) updateClusterAutoscaler(ctx context.Context, plan *managedKubernetesEngine, state *managedKubernetesEngine) error {
	vpcId := state.VpcId.ValueString()
	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return fmt.Errorf("error getting platform: %v", err)
	}
	platform = strings.ToLower(platform)
	path := commons.ApiPath.ManagedFKEUpdateClusterAutoscaler(vpcId, platform, state.Id.ValueString())

	// Get cluster autoscaler attributes from plan
	autoscalerAttrs := plan.ClusterAutoscaler.Attributes()

	requestBody := map[string]interface{}{
		"isEnableAutoScaling":           autoscalerAttrs["is_enable_auto_scaling"].(types.Bool).ValueBool(),
		"scaleDownDelayAfterAdd":        autoscalerAttrs["scale_down_delay_after_add"].(types.Int64).ValueInt64(),
		"scaleDownDelayAfterDelete":     autoscalerAttrs["scale_down_delay_after_delete"].(types.Int64).ValueInt64(),
		"scaleDownDelayAfterFailure":    autoscalerAttrs["scale_down_delay_after_failure"].(types.Int64).ValueInt64(),
		"scaleDownUnneededTime":         autoscalerAttrs["scale_down_unneeded_time"].(types.Int64).ValueInt64(),
		"scaleDownUtilizationThreshold": autoscalerAttrs["scale_down_utilization_threshold"].(types.Float64).ValueFloat64(),
		"scanInterval":                  autoscalerAttrs["scan_interval"].(types.Int64).ValueInt64(),
		"expander":                      strings.ToLower(autoscalerAttrs["expander"].(types.String).ValueString()),
	}

	resp, err := r.mfkeClient.sendPatch(ctx, path, platform, requestBody)
	if err != nil {
		return fmt.Errorf("error calling update cluster autoscaler API: %v", err)
	}

	tflog.Info(ctx, "Cluster autoscaler API response: "+string(resp))

	// Check for API errors in response
	if diagErr := r.CheckForError(resp); diagErr != nil {
		return fmt.Errorf("API returned error: %s", diagErr.Summary())
	}

	tflog.Info(ctx, "Successfully updated cluster autoscaler.")

	return nil
}

// isSystemGeneratedKey checks if a key is system-generated
func isSystemGeneratedKey(key string) bool {
	systemKeys := []string{
		"nvidia.com/device-plugin.config", // System auto-generates this for GPU pools
		// Add more system-generated keys here if needed
	}

	for _, systemKey := range systemKeys {
		if key == systemKey {
			return true
		}
	}
	return false
}

// System-generated keys like "nvidia.com/device-plugin.config" should be ignored
func filterUserDefinedKV(kvMap map[string]string) map[string]string {
	userDefined := make(map[string]string)

	for k, v := range kvMap {
		if !isSystemGeneratedKey(k) {
			userDefined[k] = v
		}
	}

	return userDefined
}

// sortKVByKey sorts KV blocks by key name to ensure consistent ordering
func sortKVByKey(kvs []KV) []KV {
	if len(kvs) <= 1 {
		return kvs
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]KV, len(kvs))
	copy(sorted, kvs)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name.ValueString() < sorted[j].Name.ValueString()
	})

	return sorted
}

func (r *resourceManagedKubernetesEngine) updateWorkerPools(ctx context.Context, from *managedKubernetesEngine, to *managedKubernetesEngine) *diag2.ErrorDiagnostic {
	// Read current cluster state
	d, err := r.InternalRead(ctx, from.Id.ValueString(), from)
	if err != nil {
		di := diag2.NewErrorDiagnostic("Error reading cluster state", err.Error())
		return &di
	}

	vpcId := from.VpcId.ValueString()
	platform, err := r.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		d := diag2.NewErrorDiagnostic(platformVpcErrorPrefix+vpcId, err.Error())
		return &d
	}

	platform = strings.ToLower(platform)

	// Get cluster's network name for VMW platform
	clusterNetworkName := ""
	if strings.ToLower(platform) == "vmw" {
		// For VMW platform, try to get network name from subnet service
		subnets, err := r.subnetClient.ListSubnet(vpcId)
		if err == nil {
			for _, subnet := range *subnets {
				if subnet.NetworkID == from.NetworkID.ValueString() {
					clusterNetworkName = subnet.Name
					break
				}
			}
		}
	}

	// Prepare pools data
	pools := []*managedKubernetesEnginePoolJson{}
	for _, pool := range to.Pools {
		item := r.remapPools(pool, pool.WorkerPoolID.ValueString(), from.NetworkID.ValueString(), clusterNetworkName)
		pools = append(pools, item)
	}

	// Prepare request body
	body := managedKubernetesEngineEditWorker{
		K8sVersion:        to.K8SVersion.ValueString(),
		CurrentNetworking: d.Data.Spec.Networking.Nodes,
		Pools:             pools,
		TypeConfigure:     "configure",
	}

	// Call API to configure workers
	path := commons.ApiPath.ManagedFKEConfigWorker(vpcId, platform, from.Id.ValueString())
	res, err := r.mfkeClient.sendPatch(ctx, path, platform, body)
	if err != nil {
		d := diag2.NewErrorDiagnostic("Error configuring worker", err.Error())
		return &d
	}
	if e2 := r.CheckForError(res); e2 != nil {
		return e2
	}

	return nil
}
