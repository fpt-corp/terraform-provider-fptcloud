package fptcloud_dfke

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"terraform-provider-fptcloud/commons"
)

type dfkeApiClient struct {
	*commons.Client
	edgeClient *commons.Client
}

func newDfkeApiClient(c *commons.Client) (*dfkeApiClient, error) {
	serviceToken := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzb21lIjoicGF5bG9hZCJ9.Joh1R2dYzkRvDkqv3sygm5YyK8Gi4ShZqbhK2gxcs2U"
	edgeClient, err := commons.NewClientWithURL(
		serviceToken,
		c.BaseURL.String(),
		c.Region,
		c.TenantName,
	)

	if err != nil {
		return nil, err
	}

	return &dfkeApiClient{
		Client:     edgeClient,
		edgeClient: edgeClient,
	}, nil
}

type EdgeGateway struct {
	Id            string `json:"id"`
	VpcId         string `json:"vpc_id"`
	EdgeGatewayId string `json:"edge_gateway_id"`
}

type edgeResponse struct {
	EdgeGateway EdgeGateway `json:"edgeGateway"`
}

func (a *dfkeApiClient) FindEdgeById(vpcId string, id string) (*EdgeGateway, error) {
	path := fmt.Sprintf("internal/vpc/%s/find_edge_by_id/%s/false", vpcId, id)
	r, err := a.internalFindEdge(path)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (a *dfkeApiClient) FindEdgeByEdgeGatewayId(vpcId string, edgeId string) (*EdgeGateway, error) {
	if !strings.HasPrefix(edgeId, "urn:vcloud:gateway") {
		return nil, errors.New("edge gateway id must be prefixed with \"urn:vcloud:gateway\"")
	}
	path := fmt.Sprintf("internal/vpc/%s/find_edge_by_id/%s/true", vpcId, edgeId)
	r, err := a.internalFindEdge(path)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (a *dfkeApiClient) internalFindEdge(endpoint string) (*EdgeGateway, error) {
	r, err := a.edgeClient.SendGetRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var edge edgeResponse
	err = json.Unmarshal(r, &edge)
	if err != nil {
		return nil, err
	}

	return &edge.EdgeGateway, nil
}
