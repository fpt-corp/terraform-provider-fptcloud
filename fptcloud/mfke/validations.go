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

		// Validate: if worker_base = true, taints must be empty
		if pool.WorkerBase.ValueBool() && len(pool.Taints) > 0 {
			d := diag2.NewErrorDiagnostic("Invalid taints configuration", "Worker pool '"+name+"' has worker_base = true, but taints are not allowed for base worker pools")
			return &d
		}

		// Validate taint effect values
		for _, taint := range pool.Taints {
			if !taint.Effect.IsNull() && !taint.Effect.IsUnknown() {
				effect := taint.Effect.ValueString()
				allowedEffects := []string{"NoSchedule", "PreferNoSchedule", "NoExecute"}
				isValid := false
				for _, allowed := range allowedEffects {
					if effect == allowed {
						isValid = true
						break
					}
				}
				if !isValid {
					d := diag2.NewErrorDiagnostic("Invalid taint effect", "Taint effect '"+effect+"' in pool '"+name+"' is not allowed. Must be one of: "+strings.Join(allowedEffects, ", "))
					return &d
				}
			}
		}

		// Only validate GPU-related fields if this is a GPU pool (has VGpuID)
		if !pool.VGpuID.IsNull() && !pool.VGpuID.IsUnknown() && pool.VGpuID.ValueString() != "" {
			// Validate gpu_sharing_client
			if !pool.GpuSharingClient.IsNull() && !pool.GpuSharingClient.IsUnknown() {
				gpuSharingClient := pool.GpuSharingClient.ValueString()
				allowedGpuSharingClients := []string{"", "timeSlicing"}
				isValid := false
				for _, allowed := range allowedGpuSharingClients {
					if gpuSharingClient == allowed {
						isValid = true
						break
					}
				}
				if !isValid {
					d := diag2.NewErrorDiagnostic("Invalid gpu_sharing_client", "gpu_sharing_client '"+gpuSharingClient+"' in pool '"+name+"' is not allowed. Must be one of: "+strings.Join(allowedGpuSharingClients, ", "))
					return &d
				}
			}

			// Validate max_client only when gpu_sharing_client = "timeSlicing"
			if !pool.MaxClient.IsNull() && !pool.MaxClient.IsUnknown() {
				// Only validate max_client if gpu_sharing_client is "timeSlicing"
				if !pool.GpuSharingClient.IsNull() && !pool.GpuSharingClient.IsUnknown() && pool.GpuSharingClient.ValueString() == "timeSlicing" {
					maxClient := pool.MaxClient.ValueInt64()
					if maxClient < 2 || maxClient > 48 {
						d := diag2.NewErrorDiagnostic("Invalid max_client", fmt.Sprintf("max_client must be between 2 and 48 for pool '%s' when gpu_sharing_client = 'timeSlicing', got: %d", name, maxClient))
						return &d
					}
				}
			}

			// Validate driver_installation_type (must be "pre-install")
			if !pool.DriverInstallationType.IsNull() && !pool.DriverInstallationType.IsUnknown() {
				driverInstallationType := pool.DriverInstallationType.ValueString()
				if driverInstallationType != "pre-install" {
					d := diag2.NewErrorDiagnostic("Invalid driver_installation_type", fmt.Sprintf("driver_installation_type must be 'pre-install' for pool '%s', got: '%s'", name, driverInstallationType))
					return &d
				}
			}

			// Validate gpu_driver_version (must be "default" or "latest")
			if !pool.GpuDriverVersion.IsNull() && !pool.GpuDriverVersion.IsUnknown() {
				gpuDriverVersion := pool.GpuDriverVersion.ValueString()
				allowedGpuDriverVersions := []string{"default", "latest"}
				isValid := false
				for _, allowed := range allowedGpuDriverVersions {
					if gpuDriverVersion == allowed {
						isValid = true
						break
					}
				}
				if !isValid {
					d := diag2.NewErrorDiagnostic("Invalid gpu_driver_version", fmt.Sprintf("gpu_driver_version must be one of: %s for pool '%s', got: '%s'", strings.Join(allowedGpuDriverVersions, ", "), name, gpuDriverVersion))
					return &d
				}
			}
		}
	}

	// Check: if more than one pool, at least one must have worker_base = true
	if len(pools) > 1 {
		hasBase := false
		for _, pool := range pools {
			if pool.WorkerBase.ValueBool() {
				hasBase = true
				break
			}
		}
		if !hasBase {
			d := diag2.NewErrorDiagnostic("Missing worker_base", "When you define more than one worker group, at least one must have worker_base = true.")
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
	if !state.ClusterEndpointAccess.IsNull() && !state.ClusterEndpointAccess.IsUnknown() {
		accessAttrs := state.ClusterEndpointAccess.Attributes()

		accessTypeAttr, ok := accessAttrs["type"].(types.String)
		if !ok || accessTypeAttr.IsNull() || accessTypeAttr.IsUnknown() {
			response.Diagnostics.AddError("Invalid cluster_endpoint_access.type", "Missing or invalid type field in cluster_endpoint_access")
			return false
		}

		if diag := validateClusterEndpointAccess(accessTypeAttr.ValueString()); diag != nil {
			response.Diagnostics.Append(diag)
			return false
		}

		allowCidrAttr, ok := accessAttrs["allow_cidr"].(types.List)
		if !ok || allowCidrAttr.IsNull() || allowCidrAttr.IsUnknown() {
			response.Diagnostics.AddError("Invalid cluster_endpoint_access.allow_cidr", "Missing or invalid allow_cidr field in cluster_endpoint_access")
			return false
		}

		if diag := validateAllowCidr(allowCidrAttr); diag != nil {
			response.Diagnostics.Append(diag)
			return false
		}
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

func validateAllowCidr(allowCidr types.List) diag2.Diagnostic {
	if allowCidr.IsNull() || allowCidr.IsUnknown() {
		return nil
	}

	for _, v := range allowCidr.Elements() {
		str, ok := v.(types.String)
		if !ok || str.IsNull() || str.IsUnknown() {
			continue // hoặc có thể báo lỗi nếu cần thiết
		}
		_, _, err := net.ParseCIDR(str.ValueString())
		if err != nil {
			return diag2.NewErrorDiagnostic(
				"Invalid CIDR format in allow_cidr",
				fmt.Sprintf("'%s' is not a valid CIDR string: %v", str.ValueString(), err),
			)
		}
	}

	return nil
}

func validateClusterEndpointAccessUpdate(plan, state *managedKubernetesEngine) diag2.Diagnostic {
	if plan == nil || state == nil {
		return nil
	}

	// Kiểm tra null/unknown của object
	if plan.ClusterEndpointAccess.IsNull() || plan.ClusterEndpointAccess.IsUnknown() ||
		state.ClusterEndpointAccess.IsNull() || state.ClusterEndpointAccess.IsUnknown() {
		return nil
	}

	planAttrs := plan.ClusterEndpointAccess.Attributes()
	stateAttrs := state.ClusterEndpointAccess.Attributes()

	planType, planOk := planAttrs["type"].(types.String)
	stateType, stateOk := stateAttrs["type"].(types.String)

	// Bỏ qua nếu không lấy được field type
	if !planOk || planType.IsNull() || planType.IsUnknown() ||
		!stateOk || stateType.IsNull() || stateType.IsUnknown() {
		return nil
	}

	planTypeVal := planType.ValueString()
	stateTypeVal := stateType.ValueString()

	if stateTypeVal == "public" && planTypeVal != "public" {
		return diag2.NewErrorDiagnostic(
			"Invalid cluster_endpoint_access.type transition",
			"Cannot change cluster_endpoint_access.type from 'public' to 'private' or 'mixed' after creation.",
		)
	}
	if (stateTypeVal == "private" || stateTypeVal == "mixed") && planTypeVal == "public" {
		return diag2.NewErrorDiagnostic(
			"Invalid cluster_endpoint_access.type transition",
			"Cannot change cluster_endpoint_access.type from 'private' or 'mixed' to 'public' after creation.",
		)
	}

	// Validate allow_cidr
	if allowCidrAttr, ok := planAttrs["allow_cidr"].(types.List); ok && !allowCidrAttr.IsNull() && !allowCidrAttr.IsUnknown() {
		if diag := validateAllowCidr(allowCidrAttr); diag != nil {
			return diag
		}
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
	// if len(planPools) != len(statePools) {
	// 	return diag2.NewErrorDiagnostic(
	// 		"Pool count mismatch",
	// 		"The number of pools in plan and state do not match.",
	// 	)
	// }
	for i := range planPools {
		if i >= len(statePools) {
			// New pool, nothing to check
			continue
		}
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

	// Validate taints configuration for worker pools during update
	for _, pool := range plan.Pools {
		if pool.WorkerBase.ValueBool() && len(pool.Taints) > 0 {
			response.Diagnostics.AddError(
				"Invalid taints configuration",
				fmt.Sprintf("Worker pool '%s' has worker_base = true, but taints are not allowed for base worker pools", pool.WorkerPoolID.ValueString()),
			)
			return false
		}
		// Validate taint effect values
		for _, taint := range pool.Taints {
			if !taint.Effect.IsNull() && !taint.Effect.IsUnknown() {
				effect := taint.Effect.ValueString()
				allowedEffects := []string{"NoSchedule", "PreferNoSchedule", "NoExecute"}
				isValid := false
				for _, allowed := range allowedEffects {
					if effect == allowed {
						isValid = true
						break
					}
				}
				if !isValid {
					response.Diagnostics.AddError(
						"Invalid taint effect",
						fmt.Sprintf("Taint effect '%s' in pool '%s' is not allowed. Must be one of: %s", effect, pool.WorkerPoolID.ValueString(), strings.Join(allowedEffects, ", ")),
					)
					return false
				}
			}
		}

		if !pool.VGpuID.IsNull() && !pool.VGpuID.IsUnknown() && pool.VGpuID.ValueString() != "" {
			if !pool.GpuSharingClient.IsNull() && !pool.GpuSharingClient.IsUnknown() {
				gpuSharingClient := pool.GpuSharingClient.ValueString()
				allowedGpuSharingClients := []string{"", "timeSlicing"}
				isValid := false
				for _, allowed := range allowedGpuSharingClients {
					if gpuSharingClient == allowed {
						isValid = true
						break
					}
				}
				if !isValid {
					response.Diagnostics.AddError(
						"Invalid gpu_sharing_client",
						fmt.Sprintf("gpu_sharing_client '%s' in pool '%s' is not allowed. Must be one of: %s", gpuSharingClient, pool.WorkerPoolID.ValueString(), strings.Join(allowedGpuSharingClients, ", ")),
					)
					return false
				}
			}

			// Validate max_client only when gpu_sharing_client = "timeSlicing"
			if !pool.MaxClient.IsNull() && !pool.MaxClient.IsUnknown() {
				// Only validate max_client if gpu_sharing_client is "timeSlicing"
				gpuSharingClientValue := ""
				if !pool.GpuSharingClient.IsNull() && !pool.GpuSharingClient.IsUnknown() {
					gpuSharingClientValue = pool.GpuSharingClient.ValueString()
				}

				// Debug logging
				fmt.Printf("DEBUG: Pool '%s' - gpu_sharing_client: '%s', max_client: %d\n",
					pool.WorkerPoolID.ValueString(), gpuSharingClientValue, pool.MaxClient.ValueInt64())

				if gpuSharingClientValue == "timeSlicing" {
					maxClient := pool.MaxClient.ValueInt64()
					if maxClient < 2 || maxClient > 48 {
						response.Diagnostics.AddError(
							"Invalid max_client",
							fmt.Sprintf("max_client must be between 2 and 48 for pool '%s' when gpu_sharing_client = 'timeSlicing', got: %d", pool.WorkerPoolID.ValueString(), maxClient),
						)
						return false
					}
				}
			}

			if !pool.DriverInstallationType.IsNull() && !pool.DriverInstallationType.IsUnknown() {
				driverInstallationType := pool.DriverInstallationType.ValueString()
				if driverInstallationType != "pre-install" {
					response.Diagnostics.AddError(
						"Invalid driver_installation_type",
						fmt.Sprintf("driver_installation_type must be 'pre-install' for pool '%s', got: '%s'", pool.WorkerPoolID.ValueString(), driverInstallationType),
					)
					return false
				}
			}

			if !pool.GpuDriverVersion.IsNull() && !pool.GpuDriverVersion.IsUnknown() {
				gpuDriverVersion := pool.GpuDriverVersion.ValueString()
				allowedGpuDriverVersions := []string{"default", "latest"}
				isValid := false
				for _, allowed := range allowedGpuDriverVersions {
					if gpuDriverVersion == allowed {
						isValid = true
						break
					}
				}
				if !isValid {
					response.Diagnostics.AddError(
						"Invalid gpu_driver_version",
						fmt.Sprintf("gpu_driver_version must be one of: %s for pool '%s', got: '%s'", allowedGpuDriverVersions, pool.WorkerPoolID.ValueString(), gpuDriverVersion),
					)
					return false
				}
			}
		}
	}

	// Add other update-time validations here as needed
	return true
}
