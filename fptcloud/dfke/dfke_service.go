package fptcloud_dfke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-fptcloud/commons"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"
)

type dfkeApiClient struct {
	*commons.Client
}

func newDfkeApiClient(c *commons.Client) (*dfkeApiClient, error) {
	return &dfkeApiClient{
		Client: c,
	}, nil
}

type edgeListResponse struct {
	Data []fptcloud_subnet.EdgeGateway `json:"data"`
}

func (a *dfkeApiClient) FindEdgeByEdgeGatewayId(ctx context.Context, vpcId string, edgeId string) (string, error) {
	if !strings.HasPrefix(edgeId, "urn:vcloud:gateway") {
		return "", errors.New("edge gateway id must be prefixed with \"urn:vcloud:gateway\"")
	}

	tflog.Info(ctx, "Resolving edge by gateway ID "+edgeId)

	path := fmt.Sprintf("/v1/vmware/vpc/%s/edge_gateway/list", vpcId)
	r, err := a.Client.SendGetRequest(path)
	if err != nil {
		return "", err
	}

	var edgeList edgeListResponse
	err = json.Unmarshal(r, &edgeList)
	if err != nil {
		return "", err
	}

	for _, edge := range edgeList.Data {
		if edge.EdgeGatewayId == edgeId {
			return edge.ID, nil
		}
	}

	return "", errors.New("edge gateway not found")
}
