package fptcloud_dfke

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_vpc "terraform-provider-fptcloud/fptcloud/vpc"
)

type TenancyApiClient struct {
	*commons.Client
}

func NewTenancyApiClient(c *commons.Client) *TenancyApiClient {
	return &TenancyApiClient{c}
}

func (t *TenancyApiClient) GetTenancy(ctx context.Context) (*EnabledTenants, error) {
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

func (t *TenancyApiClient) GetRegions(ctx context.Context, tenantId string) ([]Region, error) {
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

func (t *TenancyApiClient) ListVpcs(ctx context.Context, tenantId string, userId string, region string) ([]fptcloud_vpc.VPC, error) {
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

func (t *TenancyApiClient) GetVpcPlatform(ctx context.Context, vpcId string) (string, error) {
	tenants, err := t.GetTenancy(ctx)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("/v1/vmware/vpc/%s/user/%s/vpc_user", vpcId, tenants.UserId)

	tflog.Info(ctx, "Getting platform for VPC "+vpcId)
	res, err := t.SendGetRequest(path)
	if err != nil {
		return "", err
	}

	var ret vpcUserResponse
	err = json.Unmarshal(res, &ret)
	if err != nil {
		return "", err
	}

	tflog.Info(ctx, "Platform for VPC "+vpcId+" is "+ret.User.Platform)

	return strings.ToUpper(ret.User.Platform), nil
}

type vpcUserResponse struct {
	User vpcUser `json:"data"`
}

type vpcUser struct {
	UserId   string `json:"user_id"`
	VpcId    string `json:"vpc_id"`
	Platform string `json:"platform"`
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
