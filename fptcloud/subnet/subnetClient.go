package fptcloud_subnet

import (
	"encoding/json"
	"terraform-provider-fptcloud/commons"
)

type SubnetClient struct {
	*commons.Client
}

func NewSubnetClient(client *commons.Client) *SubnetClient {
	return &SubnetClient{client}
}

func (c *SubnetClient) ListNetworks(vpcId string) ([]SubnetData, error) {
	url := commons.ApiPath.Subnet(vpcId)
	res, err := c.SendGetRequest(url)

	if err != nil {
		return nil, err
	}

	var r subnetResponse
	if err = json.Unmarshal(res, &r); err != nil {
		return nil, err
	}

	return r.Data, nil
}

type SubnetData struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	DefaultGateway     string      `json:"defaultGateway"`
	SubnetPrefixLength int         `json:"subnetPrefixLength"`
	NetworkID          interface{} `json:"network_id"`
	NetworkType        string      `json:"networkType"`
}

type subnetResponse struct {
	Data []SubnetData `json:"data"`
}
