package fptcloud_dfke

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-fptcloud/commons"
	fptcloud_vpc "terraform-provider-fptcloud/fptcloud/vpc"
)

type tenancyApiClient struct {
	*commons.Client
}

func newTenancyApiClient(c *commons.Client) *tenancyApiClient {
	return &tenancyApiClient{c}
}

func (t *tenancyApiClient) GetTenancy(ctx context.Context) (*EnabledTenants, error) {
	tflog.Info(ctx, "Getting enabled tenants")

	path := "/v1/vmware/user/tenants/enabled"
	res, err := t.SendGetRequest(path)
	if err != nil {
		return nil, err
	}

	var ret EnabledTenants
	err = json.Unmarshal(res, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (t *tenancyApiClient) GetRegions(ctx context.Context, tenantId string) ([]Region, error) {
	tflog.Info(ctx, "Getting regions under tenant "+tenantId)
	path := fmt.Sprintf("/v1/vmware/org/%s/list/regions", tenantId)
	res, err := t.SendGetRequest(path)
	if err != nil {
		return nil, err
	}
	var ret RegionResponse
	err = json.Unmarshal(res, &ret)
	if err != nil {
		return nil, err
	}

	return ret.Regions, nil
}

func (t *tenancyApiClient) ListVpcs(ctx context.Context, tenantId string, userId string, region string) ([]fptcloud_vpc.VPC, error) {
	tflog.Info(ctx, "Getting regions under tenant "+tenantId+", user "+userId+", region "+region)

	path := fmt.Sprintf("/v1/vmware/org/%s/user/%s/list/vpc?regionId=%s", tenantId, userId, region)
	res, err := t.SendGetRequest(path)

	if err != nil {
		return nil, err
	}

	var ret ListVpcResponse
	err = json.Unmarshal(res, &ret)
	if err != nil {
		return nil, err
	}

	return ret.VpcList, nil
}

type EnabledTenants struct {
	UserId  string                `json:"id"`
	Tenants []fptcloud_vpc.Tenant `json:"tenants"`
}

type Region struct {
	Id   string `json:"id"`
	Abbr string `json:"abbreviation_name"`
}

type RegionResponse struct {
	Regions []Region `json:"data"`
}

type ListVpcResponse struct {
	VpcList []fptcloud_vpc.VPC `json:"data"`
}
