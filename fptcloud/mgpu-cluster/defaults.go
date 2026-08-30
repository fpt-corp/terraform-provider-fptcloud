package fptcloud_mgpu_cluster

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SetDefaults(state *managedGpuCluster) {
	if state.Purpose.IsNull() || state.Purpose.IsUnknown() || state.Purpose.ValueString() == "" {
		state.Purpose = types.StringValue("public")
	}
	if state.NetworkType.IsNull() || state.NetworkType.IsUnknown() || state.NetworkType.ValueString() == "" {
		state.NetworkType = types.StringValue("calico")
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
	if state.EdgeGatewayName.IsNull() || state.EdgeGatewayName.IsUnknown() {
		state.EdgeGatewayName = types.StringValue("")
	}
	if state.EdgeGatewayId.IsNull() || state.EdgeGatewayId.IsUnknown() {
		state.EdgeGatewayId = types.StringValue("")
	}
	// internal_subnet_lb is now Required — no default needed.
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
		state.K8SVersion = types.StringValue("1.34.6")
	}
	if !state.ClusterEndpointAccess.IsNull() && !state.ClusterEndpointAccess.IsUnknown() {
		attrs := state.ClusterEndpointAccess.Attributes()

		// Xử lý field `type`
		typeAttr := attrs["type"]
		if typeAttr.IsNull() || typeAttr.IsUnknown() || typeAttr.(types.String).ValueString() == "" {
			attrs["type"] = types.StringValue("public")
		}

		// Xử lý field `allow_cidr`
		needSet := false
		cidrAttr := attrs["allow_cidr"]
		if cidrAttr.IsNull() || cidrAttr.IsUnknown() {
			needSet = true
		} else {
			cidrList := cidrAttr.(types.List)
			elements := cidrList.Elements()
			if len(elements) == 0 {
				needSet = true
			}
		}
		if needSet {
			attrs["allow_cidr"], _ = types.ListValue(
				types.StringType,
				[]attr.Value{types.StringValue("0.0.0.0/0")},
			)
		}

		state.ClusterEndpointAccess, _ = types.ObjectValue(
			map[string]attr.Type{
				"type":       types.StringType,
				"allow_cidr": types.ListType{ElemType: types.StringType},
			},
			attrs,
		)
	} else {
		state.ClusterEndpointAccess, _ = types.ObjectValue(
			map[string]attr.Type{
				"type":       types.StringType,
				"allow_cidr": types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"type":       types.StringValue("public"),
				"allow_cidr": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("0.0.0.0/0")}),
			},
		)
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
			"expander":                         types.StringValue("least-waste"),
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
		// Set worker_base: first pool defaults to true, others default to false
		if pool.WorkerBase.IsNull() || pool.WorkerBase.IsUnknown() {
			if i == 0 {
				pool.WorkerBase = types.BoolValue(true) // First pool is worker_base by default
			} else {
				pool.WorkerBase = types.BoolValue(false)
			}
		}

		// Handle MaxClient for GPU pools
		if pool.MaxClient.IsNull() || pool.MaxClient.IsUnknown() {
			pool.MaxClient = types.Int64Value(0)
		}

		// gpu_driver is left as-is: null when the user does not set it, so
		// nothing is sent to the API for either of its fields.
	}

	if state.IsRunning.IsNull() || state.IsRunning.IsUnknown() {
		state.IsRunning = types.BoolValue(true)
	}
}

func SetDefaultsUpdate(plan, state *managedGpuCluster) {
	if plan.Id.IsNull() || plan.Id.IsUnknown() || plan.Id.ValueString() == "" {
		plan.Id = state.Id
	}
	if plan.Purpose.IsNull() || plan.Purpose.IsUnknown() || plan.Purpose.ValueString() == "" {
		plan.Purpose = state.Purpose
	}
	if plan.NetworkType.IsNull() || plan.NetworkType.IsUnknown() || plan.NetworkType.ValueString() == "" {
		plan.NetworkType = state.NetworkType
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
	// internal_subnet_lb is now Required — no default needed.
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

	if plan.IsRunning.IsNull() || plan.IsRunning.IsUnknown() {
		plan.IsRunning = state.IsRunning
	}

	// cluster_endpoint_access: always set default when not provided in plan
	if plan.ClusterEndpointAccess.IsNull() || plan.ClusterEndpointAccess.IsUnknown() {
		plan.ClusterEndpointAccess = state.ClusterEndpointAccess
	}
	// cluster_autoscaler: always set default when not provided in plan
	if plan.ClusterAutoscaler.IsNull() || plan.ClusterAutoscaler.IsUnknown() {
		defaultMap := map[string]attr.Value{
			"is_enable_auto_scaling":           types.BoolValue(true),
			"scale_down_delay_after_add":       types.Int64Value(3600),
			"scale_down_delay_after_delete":    types.Int64Value(0),
			"scale_down_delay_after_failure":   types.Int64Value(180),
			"scale_down_unneeded_time":         types.Int64Value(1800),
			"scale_down_utilization_threshold": types.Float64Value(0.5),
			"scan_interval":                    types.Int64Value(10),
			"expander":                         types.StringValue("least-waste"),
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

	// For hibernation_schedules, if not provided in config, keep it as null
	// This allows users to explicitly remove schedules by not specifying them
	if plan.HibernationSchedules.IsNull() || plan.HibernationSchedules.IsUnknown() {
		plan.HibernationSchedules = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"start":    types.StringType,
				"end":      types.StringType,
				"location": types.StringType,
			},
		})
	}
	// Pools: default optional fields in each pool
	for i := range plan.Pools {
		if plan.Pools[i].NetworkID.IsNull() || plan.Pools[i].NetworkID.IsUnknown() || plan.Pools[i].NetworkID.ValueString() == "" {
			if i < len(state.Pools) && state.Pools[i] != nil {
				plan.Pools[i].NetworkID = state.Pools[i].NetworkID
			} else {
				plan.Pools[i].NetworkID = state.NetworkID
			}
		}
		if plan.Pools[i].ContainerRuntime.IsNull() || plan.Pools[i].ContainerRuntime.IsUnknown() || plan.Pools[i].ContainerRuntime.ValueString() == "" {
			if i < len(state.Pools) && state.Pools[i] != nil {
				plan.Pools[i].ContainerRuntime = state.Pools[i].ContainerRuntime
			} else {
				plan.Pools[i].ContainerRuntime = types.StringValue("containerd")
			}
		}
		// Additional fields
		if plan.Pools[i].NetworkName.IsNull() || plan.Pools[i].NetworkName.IsUnknown() || plan.Pools[i].NetworkName.ValueString() == "" {
			if i < len(state.Pools) && state.Pools[i] != nil {
				plan.Pools[i].NetworkName = state.Pools[i].NetworkName
			} else {
				plan.Pools[i].NetworkName = types.StringValue("")
			}
		}
		if plan.Pools[i].Tags.IsNull() || plan.Pools[i].Tags.IsUnknown() {
			if i < len(state.Pools) && state.Pools[i] != nil {
				plan.Pools[i].Tags = state.Pools[i].Tags
			} else {
				plan.Pools[i].Tags = types.ListValueMust(types.StringType, []attr.Value{})
			}
		}

		// Note: kv is a Set, so plan.Pools[i].Kv doesn't need any reordering
		// here - Terraform compares Set values regardless of element order.
		// Don't change null/unknown values - let them stay as they are
		if plan.Pools[i].MaxClient.IsNull() || plan.Pools[i].MaxClient.IsUnknown() {
			if i < len(state.Pools) && state.Pools[i] != nil {
				plan.Pools[i].MaxClient = state.Pools[i].MaxClient
			} else {
				plan.Pools[i].MaxClient = types.Int64Value(0)
			}
		}
		// gpu_driver is left as-is: not computed, so a null plan value means
		// the user genuinely left it unset (or is clearing it).
		if plan.Pools[i].WorkerBase.IsNull() || plan.Pools[i].WorkerBase.IsUnknown() {
			if i < len(state.Pools) && state.Pools[i] != nil {
				plan.Pools[i].WorkerBase = state.Pools[i].WorkerBase
			} else {
				plan.Pools[i].WorkerBase = types.BoolValue(false)
			}
		}
	}
}
