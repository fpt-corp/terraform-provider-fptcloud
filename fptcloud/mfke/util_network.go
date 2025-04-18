package fptcloud_mfke

import (
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"
)

func getNetworkIdByPlatform(ctx context.Context, client fptcloud_subnet.SubnetService, vpcId, platform string, w *managedKubernetesEngineDataWorker, data *managedKubernetesEngineData) (string, error) {
	if strings.ToLower(platform) == "vmw" {
		return getNetworkId(ctx, client, vpcId, w.ProviderConfig.NetworkName, "")
	} else {
		return getNetworkId(ctx, client, vpcId, "", data.Spec.Provider.InfrastructureConfig.Networks.Id)
	}
}

func getNetworkId(ctx context.Context, client fptcloud_subnet.SubnetService, vpcId string, networkName string, networkId string) (string, error) {
	if networkName != "" && networkId != "" {
		return "", errors.New("only specify network name or id")
	}

	if networkName != "" {
		tflog.Info(ctx, "Resolving network ID for VPC "+vpcId+", network "+networkName)
	} else {
		tflog.Info(ctx, "Resolving network ID for VPC "+vpcId+", network ID "+networkId)
	}

	networks, err := client.FindSubnetByName(fptcloud_subnet.FindSubnetDTO{
		NetworkName: networkName,
		NetworkID:   networkId,
		VpcId:       vpcId,
	})
	if err != nil {
		return "", err
	}

	return networks.NetworkID, nil
}
