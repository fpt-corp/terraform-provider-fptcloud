package fptcloud_mfke

import (
	"fmt"
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"slices"
	"strings"
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
	}

	return nil
}

func validateNetwork(state *managedKubernetesEngine, platform string) *diag2.ErrorDiagnostic {
	if strings.ToLower(platform) == "osp" {
		if state.NetworkID.ValueString() == "" {
			d := diag2.NewErrorDiagnostic(
				"Global network ID must be specified",
				"VPC platform is OSP. Network ID must be specified globally and each worker group's network ID must match",
			)
			return &d
		}

		network := state.NetworkID.ValueString()
		for _, pool := range state.Pools {
			if pool.NetworkID.ValueString() != network {
				d := diag2.NewErrorDiagnostic(
					fmt.Sprintf("Worker network ID mismatch (%s and %s)", network, pool.NetworkID.ValueString()),
					fmt.Sprintf("VPC platform is OSP. Network ID of worker group \"%s\" must match global one", pool.WorkerPoolID.ValueString()),
				)
				return &d
			}
		}

		if state.EdgeGatewayId.ValueString() != "" {
			d := diag2.NewErrorDiagnostic("Edge gateway specification is not supported", "VPC platform is OSP. Edge gateway ID must be left empty")
			return &d
		}
	} else {
		if state.NetworkID.ValueString() != "" {
			d := diag2.NewErrorDiagnostic(
				"Global network ID is not supported",
				"VPC platform is VMW. Network ID must be specified per worker group, not globally",
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
