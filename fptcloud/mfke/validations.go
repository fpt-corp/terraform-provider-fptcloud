package fptcloud_mfke

import (
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"

	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validatePool(pools []*managedKubernetesEnginePool) *diag2.ErrorDiagnostic {
	if len(pools) == 0 {
		d := diag2.NewErrorDiagnostic("Invalid configuration", "At least a worker pool must be configured")
		return &d
	}

	groupNames := map[string]bool{}
	for _, pool := range pools {
		name := pool.WorkerPoolID.ValueString()
		if name == "worker-new" {
			d := diag2.NewErrorDiagnostic("Invalid worker group name", "Worker group name \"worker-new\" is reserved")
			return &d
		}

		if _, ok := groupNames[name]; ok {
			d := diag2.NewErrorDiagnostic("Duplicate worker group name", "Worker group name "+name+" is used twice")
			return &d
		}

		groupNames[name] = true

		// Validate worker_disk_size >= 40GB
		if pool.WorkerDiskSize.ValueInt64() < 40 {
			d := diag2.NewErrorDiagnostic("Invalid worker_disk_size", "worker_disk_size must be greater than or equal to 40GB for pool '"+name+"'")
			return &d
		}

		// Validate scale_max >= scale_min
		if pool.ScaleMax.ValueInt64() < pool.ScaleMin.ValueInt64() {
			d := diag2.NewErrorDiagnostic("Invalid scale_max", "scale_max must be greater than or equal to scale_min for pool '"+name+"'")
			return &d
		}
	}

	return nil
}

func validateK8sVersion(version string) *diag2.ErrorDiagnostic {
	allowed := []string{"1.32.5", "1.31.4", "1.30.8", "1.29.8", "1.28.13"}
	for _, v := range allowed {
		if version == v {
			return nil
		}
	}
	d := diag2.NewErrorDiagnostic("Invalid Kubernetes version", "k8s_version must be one of: "+strings.Join(allowed, ", "))
	return &d
}

func parseK8sMinorVersion(version string) (int, error) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid version format: %s", version)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	return minor, nil
}

func validateK8sVersionUpdate(planVersion, stateVersion types.String) diag2.Diagnostic {
	allowedVersions := []string{"1.32.5", "1.31.4", "1.30.8", "1.29.8", "1.28.13"}
	if planVersion.IsNull() || planVersion.IsUnknown() || planVersion.ValueString() == "" {
		return nil
	}
	plan := planVersion.ValueString()
	found := false
	for _, v := range allowedVersions {
		if plan == v {
			found = true
			break
		}
	}
	if !found {
		return diag2.NewErrorDiagnostic(
			"Invalid k8s_version",
			fmt.Sprintf("k8s_version must be one of: %s", strings.Join(allowedVersions, ", ")),
		)
	}
	if stateVersion.IsNull() || stateVersion.IsUnknown() || stateVersion.ValueString() == "" {
		return nil
	}
	planMinor, err1 := parseK8sMinorVersion(plan)
	stateMinor, err2 := parseK8sMinorVersion(stateVersion.ValueString())
	if err1 != nil || err2 != nil {
		return diag2.NewErrorDiagnostic(
			"Invalid k8s_version format",
			fmt.Sprintf("Failed to parse k8s_version: plan=%s, state=%s", plan, stateVersion.ValueString()),
		)
	}
	if planMinor < stateMinor {
		return diag2.NewErrorDiagnostic(
			"k8s_version downgrade not allowed",
			fmt.Sprintf("Cannot downgrade k8s_version from %s to %s", stateVersion.ValueString(), plan),
		)
	}
	if planMinor > stateMinor+1 {
		return diag2.NewErrorDiagnostic(
			"k8s_version upgrade too large",
			fmt.Sprintf("Can only upgrade k8s_version by one minor version at a time (from %s to %s)", stateVersion.ValueString(), plan),
		)
	}
	return nil
}

// Call this from validateNetwork (or validatePool if more appropriate)
func validateNetwork(state *managedKubernetesEngine, platform string) *diag2.ErrorDiagnostic {
	// Use network_id as input; network_name is no longer used
	if strings.ToLower(platform) == "osp" {
		if state.NetworkID.ValueString() == "" {
			d := diag2.NewErrorDiagnostic(
				"Global network ID must be specified",
				"Network ID must be specified globally and each worker group's network ID must match",
			)
			return &d
		}

		network := state.NetworkID.ValueString()
		for _, pool := range state.Pools {
			fmt.Println("pool.NetworkID.ValueString(): " + pool.NetworkID.ValueString())
			fmt.Printf("state.Pools: %v\n", state.Pools)
			if pool.NetworkID.ValueString() != network {
				d := diag2.NewErrorDiagnostic(
					fmt.Sprintf("Worker network ID mismatch (%s and %s)", network, pool.NetworkID.ValueString()),
					fmt.Sprintf("Network ID of worker group \"%s\" must match global one", pool.WorkerPoolID.ValueString()),
				)
				return &d
			}
		}

		if state.EdgeGatewayId.ValueString() != "" {
			d := diag2.NewErrorDiagnostic("Edge gateway specification is not supported", "Edge gateway ID must be left empty")
			return &d
		}
	} else {
		if state.NetworkID.ValueString() != "" {
			d := diag2.NewErrorDiagnostic(
				"Global network ID is not supported",
				"Network ID must be specified per worker group, not globally",
			)
			return &d
		}
	}

	networkOverlayAllowed := []string{"Always", "CrossSubnet"}
	if !slices.Contains(networkOverlayAllowed, state.NetworkOverlay.ValueString()) {
		d := diag2.NewErrorDiagnostic(
			"Invalid Network Overlay configuration",
			fmt.Sprintf("Network overlay allowed values are: %s", strings.Join(networkOverlayAllowed, ", ")),
		)
		return &d
	}

	return nil
}

func validateNetworkType(networkType string) *diag2.ErrorDiagnostic {
	allowed := []string{"calico", "cilium"}
	for _, v := range allowed {
		if networkType == v {
			return nil
		}
	}
	d := diag2.NewErrorDiagnostic("Invalid network_type", "network_type must be one of: "+strings.Join(allowed, ", "))
	return &d
}

func validateNetworkOverlay(networkOverlay string) *diag2.ErrorDiagnostic {
	allowed := []string{"Always", "CrossSubnet"}
	for _, v := range allowed {
		if networkOverlay == v {
			return nil
		}
	}
	d := diag2.NewErrorDiagnostic("Invalid network_overlay", "network_overlay must be one of: "+strings.Join(allowed, ", "))
	return &d
}

func validatePoolNames(pool []*managedKubernetesEnginePool) ([]string, error) {
	var poolNames []string

	if len(pool) != 0 {
		existingPool := map[string]*managedKubernetesEnginePool{}
		for _, pool := range pool {
			name := pool.WorkerPoolID.ValueString()
			if _, ok := existingPool[name]; ok {
				return nil, fmt.Errorf("pool %s already exists", name)
			}

			existingPool[name] = pool
			poolNames = append(poolNames, name)
		}
	}

	return poolNames, nil
}

func validatePurpose(purpose string) *diag2.ErrorDiagnostic {
	allowed := []string{"public", "private"}
	for _, v := range allowed {
		if purpose == v {
			return nil
		}
	}
	d := diag2.NewErrorDiagnostic("Invalid purpose", "purpose must be one of: "+strings.Join(allowed, ", "))
	return &d
}

func validatePurposeUpdate(planPurpose, statePurpose types.String) diag2.Diagnostic {
	if planPurpose.IsNull() || planPurpose.IsUnknown() || planPurpose.ValueString() == "" {
		return nil
	}
	if planPurpose.ValueString() != statePurpose.ValueString() {
		return diag2.NewErrorDiagnostic("Purpose cannot be changed", "The 'purpose' field is immutable and cannot be updated.")
	}
	return nil
}

func validateExpander(expander string) *diag2.ErrorDiagnostic {
	allowed := []string{"Random", "Least-waste", "Most-pods", "Priority"}
	for _, v := range allowed {
		if expander == v {
			return nil
		}
	}
	d := diag2.NewErrorDiagnostic("Invalid expander", "expander must be one of: "+strings.Join(allowed, ", "))
	return &d
}

func validateClusterEndpointAccess(accessType string) *diag2.ErrorDiagnostic {
	allowed := []string{"public", "private", "mixed"}
	for _, v := range allowed {
		if accessType == v {
			return nil
		}
	}
	d := diag2.NewErrorDiagnostic("Invalid clusterEndpointAccess type", "clusterEndpointAccess.type must be one of: "+strings.Join(allowed, ", "))
	return &d
}

func ValidateCreate(state *managedKubernetesEngine, response *resource.CreateResponse) bool {
	// Validate k8s_version
	if diag := validateK8sVersion(state.K8SVersion.ValueString()); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Validate purpose
	if diag := validatePurpose(state.Purpose.ValueString()); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Validate network_type
	if diag := validateNetworkType(state.NetworkType.ValueString()); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Validate network_overlay
	if diag := validateNetworkOverlay(state.NetworkOverlay.ValueString()); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Validate cluster_endpoint_access
	accessType := state.ClusterEndpointAccess.Type.ValueString()
	if diag := validateClusterEndpointAccess(accessType); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	if diag := validateAllowCidr(state.ClusterEndpointAccess.AllowCidr); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Validate cluster_autoscaler expander
	if !state.ClusterAutoscaler.IsNull() && !state.ClusterAutoscaler.IsUnknown() {
		autoscalerAttrs := state.ClusterAutoscaler.Attributes()
		expander := autoscalerAttrs["expander"].(types.String).ValueString()
		if diag := validateExpander(expander); diag != nil {
			response.Diagnostics.Append(diag)
			return false
		}
	}
	// Validate pool
	if err := validatePool(state.Pools); err != nil {
		response.Diagnostics.Append(err)
		return false
	}
	return true
}

func validateNetworkTypeUpdate(planNetworkType, stateNetworkType types.String) diag2.Diagnostic {
	if planNetworkType.IsNull() || planNetworkType.IsUnknown() || planNetworkType.ValueString() == "" {
		return nil
	}
	if planNetworkType.ValueString() != stateNetworkType.ValueString() {
		return diag2.NewErrorDiagnostic("Network type cannot be changed", "The 'network_type' field is immutable and cannot be updated.")
	}
	return nil
}

func validateClusterAutoscalerUpdate(plan *managedKubernetesEngine) diag2.Diagnostic {
	if !plan.ClusterAutoscaler.IsNull() && !plan.ClusterAutoscaler.IsUnknown() {
		autoscalerAttrs := plan.ClusterAutoscaler.Attributes()
		expander := autoscalerAttrs["expander"].(types.String).ValueString()
		if diag := validateExpander(expander); diag != nil {
			return diag
		}
	}
	return nil
}

func validateAllowCidr(allowCidrs []types.String) *diag2.ErrorDiagnostic {
	for _, cidr := range allowCidrs {
		if cidr.IsNull() || cidr.IsUnknown() {
			d := diag2.NewErrorDiagnostic("Invalid allowCidr entry", "allowCidr entry is null or unknown")
			return &d
		}
		if _, _, err := net.ParseCIDR(cidr.ValueString()); err != nil {
			d := diag2.NewErrorDiagnostic("Invalid allowCidr entry", "allowCidr entry '"+cidr.ValueString()+"' is not a valid CIDR: "+err.Error())
			return &d
		}
	}
	return nil
}

func validateClusterEndpointAccessUpdate(plan, state *managedKubernetesEngine) diag2.Diagnostic {
	if plan == nil || plan.ClusterEndpointAccess == nil {
		return nil
	}
	// If plan.ClusterEndpointAccess is not set (null/unknown), skip
	if plan.ClusterEndpointAccess.Type.IsNull() || plan.ClusterEndpointAccess.Type.IsUnknown() {
		return nil
	}
	if state == nil || state.ClusterEndpointAccess == nil {
		return nil
	}
	if state.ClusterEndpointAccess.Type.IsNull() || state.ClusterEndpointAccess.Type.IsUnknown() {
		return nil
	}
	planType := plan.ClusterEndpointAccess.Type.ValueString()
	stateType := state.ClusterEndpointAccess.Type.ValueString()
	// Only allow transitions: private <-> mixed, but not public <-> private/mixed
	if stateType == "public" && planType != "public" {
		return diag2.NewErrorDiagnostic(
			"Invalid cluster_endpoint_access.type transition",
			"Cannot change cluster_endpoint_access.type from 'public' to 'private' or 'mixed' after creation.",
		)
	}
	if (stateType == "private" || stateType == "mixed") && planType == "public" {
		return diag2.NewErrorDiagnostic(
			"Invalid cluster_endpoint_access.type transition",
			"Cannot change cluster_endpoint_access.type from 'private' or 'mixed' to 'public' after creation.",
		)
	}
	// Validate allow_cidr here
	if diag := validateAllowCidr(plan.ClusterEndpointAccess.AllowCidr); diag != nil {
		return diag
	}
	return nil
}

func validateImmutableStringField(fieldName string, plan, state types.String) diag2.Diagnostic {
	if !plan.IsNull() && !plan.IsUnknown() && plan.ValueString() != "" &&
		!state.IsNull() && !state.IsUnknown() && state.ValueString() != "" &&
		plan.ValueString() != state.ValueString() {
		return diag2.NewErrorDiagnostic(
			fmt.Sprintf("%s cannot be changed", fieldName),
			fmt.Sprintf("The '%s' field is immutable and cannot be updated.", fieldName),
		)
	}
	return nil
}

func validateImmutableInt64Field(fieldName string, plan, state types.Int64) diag2.Diagnostic {
	if !plan.IsNull() && !plan.IsUnknown() &&
		!state.IsNull() && !state.IsUnknown() &&
		plan.ValueInt64() != state.ValueInt64() {
		return diag2.NewErrorDiagnostic(
			fmt.Sprintf("%s cannot be changed", fieldName),
			fmt.Sprintf("The '%s' field is immutable and cannot be updated.", fieldName),
		)
	}
	return nil
}

func validateImmutablePoolStringField(planPools, statePools []*managedKubernetesEnginePool) diag2.Diagnostic {
	if len(planPools) != len(statePools) {
		return diag2.NewErrorDiagnostic(
			"Pool count mismatch",
			"The number of pools in plan and state do not match.",
		)
	}
	for i := range planPools {
		planVal := planPools[i].ContainerRuntime
		stateVal := statePools[i].ContainerRuntime
		if !planVal.IsNull() && !planVal.IsUnknown() && planVal.ValueString() != "" &&
			!stateVal.IsNull() && !stateVal.IsUnknown() && stateVal.ValueString() != "" &&
			planVal.ValueString() != stateVal.ValueString() {
			return diag2.NewErrorDiagnostic(
				"container_runtime cannot be changed in worker pool",
				"The 'container_runtime' field in worker pool is immutable and cannot be updated.",
			)
		}
	}
	return nil
}

func ValidateUpdate(state, plan *managedKubernetesEngine, response *resource.UpdateResponse) bool {
	// Validate k8s_version and prevent downgrade
	if diag := validateK8sVersionUpdate(plan.K8SVersion, state.K8SVersion); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Deny changing purpose
	if diag := validatePurposeUpdate(plan.Purpose, state.Purpose); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Deny changing network_type
	if diag := validateNetworkTypeUpdate(plan.NetworkType, state.NetworkType); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Validate expander in cluster_autoscaler
	if diag := validateClusterAutoscalerUpdate(plan); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Validate clusterEndpointAccess.type and allow_cidr
	if diag := validateClusterEndpointAccessUpdate(plan, state); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Deny changing PodNetwork, PodPrefix, ServiceNetwork, ServicePrefix, K8SMaxPod
	if diag := validateImmutableStringField("pod_network", plan.PodNetwork, state.PodNetwork); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	if diag := validateImmutableStringField("pod_prefix", plan.PodPrefix, state.PodPrefix); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	if diag := validateImmutableStringField("service_network", plan.ServiceNetwork, state.ServiceNetwork); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	if diag := validateImmutableStringField("service_prefix", plan.ServicePrefix, state.ServicePrefix); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	if diag := validateImmutableInt64Field("k8s_max_pod", plan.K8SMaxPod, state.K8SMaxPod); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Deny changing container_runtime in worker pool
	if diag := validateImmutablePoolStringField(plan.Pools, state.Pools); diag != nil {
		response.Diagnostics.Append(diag)
		return false
	}
	// Add other update-time validations here as needed
	return true
}
