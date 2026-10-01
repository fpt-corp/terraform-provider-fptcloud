package fptcloud_storage

import (
	"context"
	"sync"

	common "terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
)

const (
	PlatformOsp = "OSP"
	PlatformVmw = "VMW"
)

var vpcPlatforms sync.Map

// GetVpcPlatform get the platform of a vpc, "OSP" or "VMW"
func GetVpcPlatform(ctx context.Context, client *common.Client, vpcId string) (string, error) {
	cacheKey := client.BaseURL.String() + "|" + vpcId
	if cached, ok := vpcPlatforms.Load(cacheKey); ok {
		return cached.(string), nil
	}

	platform, err := fptcloud_dfke.NewTenancyApiClient(client).GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return "", err
	}

	vpcPlatforms.Store(cacheKey, platform)
	return platform, nil
}
