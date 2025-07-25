package fptcloud_mfke

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SetDefaults(state *managedKubernetesEngine) {
	if state.Purpose.IsNull() || state.Purpose.IsUnknown() || state.Purpose.ValueString() == "" {
		state.Purpose = types.StringValue("public")
	}
	if state.NetworkType.IsNull() || state.NetworkType.IsUnknown() || state.NetworkType.ValueString() == "" {
		state.NetworkType = types.StringValue("calico")
	}
	if state.NetworkOverlay.IsNull() || state.NetworkOverlay.IsUnknown() || state.NetworkOverlay.ValueString() == "" {
		state.NetworkOverlay = types.StringValue("CrossSubnet")
	}
	if state.IsEnableAutoUpgrade.IsNull() || state.IsEnableAutoUpgrade.IsUnknown() {
		state.IsEnableAutoUpgrade = types.BoolValue(false)
	}
	if state.AutoUpgradeExpression.IsNull() || state.AutoUpgradeExpression.IsUnknown() {
		state.AutoUpgradeExpression, _ = types.ListValue(types.StringType, []attr.Value{})
	}
	if state.AutoUpgradeTimezone.IsNull() || state.AutoUpgradeTimezone.IsUnknown() {
		state.AutoUpgradeTimezone = types.StringValue("Asia/Saigon")
	}
	if state.InternalSubnetLb.IsNull() || state.InternalSubnetLb.IsUnknown() {
		state.InternalSubnetLb = types.StringValue("")
	}
	if state.EdgeGatewayName.IsNull() || state.EdgeGatewayName.IsUnknown() {
		state.EdgeGatewayName = types.StringValue("")
	}
	if state.EdgeGatewayId.IsNull() || state.EdgeGatewayId.IsUnknown() {
		state.EdgeGatewayId = types.StringValue("")
	}
	if state.InternalSubnetLb.IsNull() || state.InternalSubnetLb.IsUnknown() {
		state.InternalSubnetLb = types.StringValue("")
	}
	if state.PodNetwork.IsNull() || state.PodNetwork.IsUnknown() || state.PodNetwork.ValueString() == "" {
		state.PodNetwork = types.StringValue("100.96.0.0")
	}
	if state.PodPrefix.IsNull() || state.PodPrefix.IsUnknown() || state.PodPrefix.ValueString() == "" {
		state.PodPrefix = types.StringValue("11")
	}
	if state.ServiceNetwork.IsNull() || state.ServiceNetwork.IsUnknown() || state.ServiceNetwork.ValueString() == "" {
		state.ServiceNetwork = types.StringValue("100.64.0.0")
	}
	if state.ServicePrefix.IsNull() || state.ServicePrefix.IsUnknown() || state.ServicePrefix.ValueString() == "" {
		state.ServicePrefix = types.StringValue("13")
	}
	if state.K8SMaxPod.IsNull() || state.K8SMaxPod.IsUnknown() || state.K8SMaxPod.ValueInt64() == 0 {
		state.K8SMaxPod = types.Int64Value(110)
	}
	if state.K8SVersion.IsNull() || state.K8SVersion.IsUnknown() || state.K8SVersion.ValueString() == "" {
		state.K8SVersion = types.StringValue("1.31.4")
	}
	if state.ClusterEndpointAccess == nil {
		state.ClusterEndpointAccess = &ClusterEndpointAccess{}
	}
	if state.ClusterEndpointAccess.Type.IsNull() || state.ClusterEndpointAccess.Type.IsUnknown() || state.ClusterEndpointAccess.Type.ValueString() == "" {
		state.ClusterEndpointAccess.Type = types.StringValue("public")
	}
	if state.ClusterEndpointAccess.AllowCidr == nil {
		state.ClusterEndpointAccess.AllowCidr = make([]types.String, 1)
		state.ClusterEndpointAccess.AllowCidr[0] = types.StringValue("0.0.0.0/0")
	}
	if state.ClusterAutoscaler.IsNull() || state.ClusterAutoscaler.IsUnknown() {
		defaultMap := map[string]attr.Value{
			"is_enable_auto_scaling":           types.BoolValue(true),
			"scale_down_delay_after_add":       types.Int64Value(3600),
			"scale_down_delay_after_delete":    types.Int64Value(0),
			"scale_down_delay_after_failure":   types.Int64Value(180),
			"scale_down_unneeded_time":         types.Int64Value(1800),
			"scale_down_utilization_threshold": types.Float64Value(0.5),
			"scan_interval":                    types.Int64Value(10),
			"expander":                         types.StringValue("Least-waste"),
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
			defaultMap,
		)
	}

	// Set default network_id and network_name for each worker pool
	for i := range state.Pools {
		pool := state.Pools[i]
		if pool == nil {
			continue
		}
		if pool.NetworkID.IsNull() || pool.NetworkID.IsUnknown() || pool.NetworkID.ValueString() == "" {
			pool.NetworkID = state.NetworkID
		}
		if pool.ContainerRuntime.IsNull() || pool.ContainerRuntime.IsUnknown() || pool.ContainerRuntime.ValueString() == "" {
			pool.ContainerRuntime = types.StringValue("containerd")
		}
	}

}

func SetDefaultsUpdate(plan, state *managedKubernetesEngine) {
	if plan.Id.IsNull() || plan.Id.IsUnknown() || plan.Id.ValueString() == "" {
		plan.Id = state.Id
	}
	if plan.Purpose.IsNull() || plan.Purpose.IsUnknown() || plan.Purpose.ValueString() == "" {
		plan.Purpose = state.Purpose
	}
	if plan.NetworkType.IsNull() || plan.NetworkType.IsUnknown() || plan.NetworkType.ValueString() == "" {
		plan.NetworkType = state.NetworkType
	}
	if plan.NetworkOverlay.IsNull() || plan.NetworkOverlay.IsUnknown() || plan.NetworkOverlay.ValueString() == "" {
		plan.NetworkOverlay = state.NetworkOverlay
	}
	if plan.IsEnableAutoUpgrade.IsNull() || plan.IsEnableAutoUpgrade.IsUnknown() {
		plan.IsEnableAutoUpgrade = state.IsEnableAutoUpgrade
	}
	if plan.AutoUpgradeExpression.IsNull() || plan.AutoUpgradeExpression.IsUnknown() {
		plan.AutoUpgradeExpression = state.AutoUpgradeExpression
	}
	if plan.AutoUpgradeTimezone.IsNull() || plan.AutoUpgradeTimezone.IsUnknown() {
		plan.AutoUpgradeTimezone = state.AutoUpgradeTimezone
	}
	if plan.InternalSubnetLb.IsNull() || plan.InternalSubnetLb.IsUnknown() {
		plan.InternalSubnetLb = state.InternalSubnetLb
	}
	if plan.EdgeGatewayName.IsNull() || plan.EdgeGatewayName.IsUnknown() {
		plan.EdgeGatewayName = state.EdgeGatewayName
	}
	if plan.EdgeGatewayId.IsNull() || plan.EdgeGatewayId.IsUnknown() {
		plan.EdgeGatewayId = state.EdgeGatewayId
	}
	if plan.PodNetwork.IsNull() || plan.PodNetwork.IsUnknown() || plan.PodNetwork.ValueString() == "" {
		plan.PodNetwork = state.PodNetwork
	}
	if plan.PodPrefix.IsNull() || plan.PodPrefix.IsUnknown() || plan.PodPrefix.ValueString() == "" {
		plan.PodPrefix = state.PodPrefix
	}
	if plan.ServiceNetwork.IsNull() || plan.ServiceNetwork.IsUnknown() || plan.ServiceNetwork.ValueString() == "" {
		plan.ServiceNetwork = state.ServiceNetwork
	}
	if plan.ServicePrefix.IsNull() || plan.ServicePrefix.IsUnknown() || plan.ServicePrefix.ValueString() == "" {
		plan.ServicePrefix = state.ServicePrefix
	}
	if plan.K8SMaxPod.IsNull() || plan.K8SMaxPod.IsUnknown() || plan.K8SMaxPod.ValueInt64() == 0 {
		plan.K8SMaxPod = state.K8SMaxPod
	}
	if plan.K8SVersion.IsNull() || plan.K8SVersion.IsUnknown() || plan.K8SVersion.ValueString() == "" {
		plan.K8SVersion = state.K8SVersion
	}
	// cluster_endpoint_access: if not in state, fill default
	if state.ClusterEndpointAccess == nil || state.ClusterEndpointAccess.Type.IsNull() || state.ClusterEndpointAccess.Type.IsUnknown() || state.ClusterEndpointAccess.Type.ValueString() == "" {
		if plan.ClusterEndpointAccess == nil {
			plan.ClusterEndpointAccess = &ClusterEndpointAccess{}
		}
		plan.ClusterEndpointAccess.Type = types.StringValue("public")
		plan.ClusterEndpointAccess.AllowCidr = []types.String{types.StringValue("0.0.0.0/0")}
	} else if plan.ClusterEndpointAccess == nil {
		plan.ClusterEndpointAccess = state.ClusterEndpointAccess
	}
	// cluster_autoscaler: if not in state, fill default
	if state.ClusterAutoscaler.IsNull() || state.ClusterAutoscaler.IsUnknown() {
		if plan.ClusterAutoscaler.IsNull() || plan.ClusterAutoscaler.IsUnknown() {
			defaultMap := map[string]attr.Value{
				"is_enable_auto_scaling":           types.BoolValue(true),
				"scale_down_delay_after_add":       types.Int64Value(3600),
				"scale_down_delay_after_delete":    types.Int64Value(0),
				"scale_down_delay_after_failure":   types.Int64Value(180),
				"scale_down_unneeded_time":         types.Int64Value(1800),
				"scale_down_utilization_threshold": types.Float64Value(0.5),
				"scan_interval":                    types.Int64Value(10),
				"expander":                         types.StringValue("Least-waste"),
			}
			plan.ClusterAutoscaler, _ = types.ObjectValue(
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
				defaultMap,
			)
		}
	} else if plan.ClusterAutoscaler.IsNull() || plan.ClusterAutoscaler.IsUnknown() {
		plan.ClusterAutoscaler = state.ClusterAutoscaler
	}
	// Pools: default optional fields in each pool
	for i := range plan.Pools {
		if plan.Pools[i].NetworkID.IsNull() || plan.Pools[i].NetworkID.IsUnknown() || plan.Pools[i].NetworkID.ValueString() == "" {
			if state.Pools[i] != nil {
				plan.Pools[i].NetworkID = state.Pools[i].NetworkID
			} else {
				plan.Pools[i].NetworkID = state.NetworkID
			}
		}
		if plan.Pools[i].ContainerRuntime.IsNull() || plan.Pools[i].ContainerRuntime.IsUnknown() || plan.Pools[i].ContainerRuntime.ValueString() == "" {
			if state.Pools[i] != nil {
				plan.Pools[i].ContainerRuntime = state.Pools[i].ContainerRuntime
			} else {
				plan.Pools[i].ContainerRuntime = types.StringValue("containerd")
			}
		}
		// Additional fields
		if plan.Pools[i].NetworkName.IsNull() || plan.Pools[i].NetworkName.IsUnknown() || plan.Pools[i].NetworkName.ValueString() == "" {
			if state.Pools[i] != nil {
				plan.Pools[i].NetworkName = state.Pools[i].NetworkName
			} else {
				plan.Pools[i].NetworkName = types.StringValue("")
			}
		}
		if plan.Pools[i].Tags.IsNull() || plan.Pools[i].Tags.IsUnknown() || plan.Pools[i].Tags.ValueString() == "" {
			if state.Pools[i] != nil {
				plan.Pools[i].Tags = state.Pools[i].Tags
			} else {
				plan.Pools[i].Tags = types.StringValue("")
			}
		}
		if plan.Pools[i].Kv == nil || len(plan.Pools[i].Kv) == 0 {
			if state.Pools[i] != nil {
				plan.Pools[i].Kv = state.Pools[i].Kv
			} else {
				plan.Pools[i].Kv = nil
			}
		}
		if plan.Pools[i].VGpuID.IsNull() || plan.Pools[i].VGpuID.IsUnknown() || plan.Pools[i].VGpuID.ValueString() == "" {
			if state.Pools[i] != nil {
				plan.Pools[i].VGpuID = state.Pools[i].VGpuID
			} else {
				plan.Pools[i].VGpuID = types.StringValue("")
			}
		}
		if plan.Pools[i].GpuSharingClient.IsNull() || plan.Pools[i].GpuSharingClient.IsUnknown() || plan.Pools[i].GpuSharingClient.ValueString() == "" {
			if state.Pools[i] != nil {
				plan.Pools[i].GpuSharingClient = state.Pools[i].GpuSharingClient
			} else {
				plan.Pools[i].GpuSharingClient = types.StringValue("")
			}
		}
		if plan.Pools[i].MaxClient.IsNull() || plan.Pools[i].MaxClient.IsUnknown() || plan.Pools[i].MaxClient.ValueInt64() == 0 {
			if state.Pools[i] != nil {
				plan.Pools[i].MaxClient = state.Pools[i].MaxClient
			} else {
				plan.Pools[i].MaxClient = types.Int64Value(0)
			}
		}
		if plan.Pools[i].IsEnableAutoRepair.IsNull() || plan.Pools[i].IsEnableAutoRepair.IsUnknown() {
			if state.Pools[i] != nil {
				plan.Pools[i].IsEnableAutoRepair = state.Pools[i].IsEnableAutoRepair
			} else {
				plan.Pools[i].IsEnableAutoRepair = types.BoolValue(false)
			}
		}
		if plan.Pools[i].DriverInstallationType.IsNull() || plan.Pools[i].DriverInstallationType.IsUnknown() || plan.Pools[i].DriverInstallationType.ValueString() == "" {
			if state.Pools[i] != nil {
				plan.Pools[i].DriverInstallationType = state.Pools[i].DriverInstallationType
			} else {
				plan.Pools[i].DriverInstallationType = types.StringValue("")
			}
		}
		if plan.Pools[i].GpuDriverVersion.IsNull() || plan.Pools[i].GpuDriverVersion.IsUnknown() || plan.Pools[i].GpuDriverVersion.ValueString() == "" {
			if state.Pools[i] != nil {
				plan.Pools[i].GpuDriverVersion = state.Pools[i].GpuDriverVersion
			} else {
				plan.Pools[i].GpuDriverVersion = types.StringValue("")
			}
		}
		if plan.Pools[i].WorkerBase.IsNull() || plan.Pools[i].WorkerBase.IsUnknown() {
			if state.Pools[i] != nil {
				plan.Pools[i].WorkerBase = state.Pools[i].WorkerBase
			} else {
				plan.Pools[i].WorkerBase = types.BoolValue(false)
			}
		}
	}
}
