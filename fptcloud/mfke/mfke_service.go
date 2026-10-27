package fptcloud_mfke

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type MfkeApiClient struct {
	*commons.Client
}

func newMfkeApiClient(c *commons.Client) *MfkeApiClient {
	return &MfkeApiClient{c}
}

// NewMfkeApiClient creates an exported MfkeApiClient for use by other packages
func NewMfkeApiClient(c *commons.Client) *MfkeApiClient {
	return &MfkeApiClient{c}
}

// SendGetWithInfraType sends a GET request with the infra-type and fpt-region headers set
func (m *MfkeApiClient) SendGetWithInfraType(requestURL string, infraType string) ([]byte, error) {
	return m.sendGet(requestURL, infraType)
}

func (m *MfkeApiClient) sendGet(requestURL string, infraType string) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return m.sendRequestWithHeader(req, infraType)
}

func (m *MfkeApiClient) sendPost(ctx context.Context, requestURL string, infraType string, params interface{}) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)

	// we create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, "sendPost Body JSON: "+string(jsonValue))

	return m.sendRequestWithHeader(req, infraType)
}

func (m *MfkeApiClient) sendPatch(ctx context.Context, requestURL string, infraType string, params interface{}) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)

	// we create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)

	req, err := http.NewRequest("PATCH", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, "sendPatch Body JSON: "+string(jsonValue))

	return m.sendRequestWithHeader(req, infraType)
}

func (m *MfkeApiClient) sendDelete(requestURL string, infraType string) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)
	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return m.sendRequestWithHeader(req, infraType)
}

// isNotFoundError reports whether err is an HTTP 404 response from the API.
func isNotFoundError(err error) bool {
	var httpErr commons.HTTPError
	return errors.As(err, &httpErr) && httpErr.Code == http.StatusNotFound
}

// sendGetV2Aware issues a GET against primaryPath, falling back to
// fallbackPath on a 404. The API routes a cluster to its v1 or v2 API family
// based on how it was originally created, which callers can't always
// determine up front from Terraform state alone (e.g. import, data source
// reads before k8s_version is known) — the fallback keeps those cases working.
func (m *MfkeApiClient) sendGetV2Aware(primaryPath, fallbackPath, infraType string) ([]byte, error) {
	a, err := m.sendGet(primaryPath, infraType)
	if err != nil && isNotFoundError(err) {
		return m.sendGet(fallbackPath, infraType)
	}
	return a, err
}

// sendPatchV2Aware issues a PATCH against primaryPath, falling back to
// fallbackPath on a 404. See sendGetV2Aware for why the fallback is needed.
func (m *MfkeApiClient) sendPatchV2Aware(ctx context.Context, primaryPath, fallbackPath, infraType string, params interface{}) ([]byte, error) {
	a, err := m.sendPatch(ctx, primaryPath, infraType, params)
	if err != nil && isNotFoundError(err) {
		return m.sendPatch(ctx, fallbackPath, infraType, params)
	}
	return a, err
}

// sendDeleteV2Aware issues a DELETE against primaryPath, falling back to
// fallbackPath on a 404. See sendGetV2Aware for why the fallback is needed.
func (m *MfkeApiClient) sendDeleteV2Aware(primaryPath, fallbackPath, infraType string) ([]byte, error) {
	a, err := m.sendDelete(primaryPath, infraType)
	if err != nil && isNotFoundError(err) {
		return m.sendDelete(fallbackPath, infraType)
	}
	return a, err
}

func (m *MfkeApiClient) sendRequestWithHeader(request *http.Request, infraType string) ([]byte, error) {
	switch m.Client.Region {
	case "VN/HAN":
		request.Header.Set("fpt-region", "hanoi-vn")
	case "VN/SGN":
		request.Header.Set("fpt-region", "saigon-vn")
	case "VN/HAN2":
		request.Header.Set("fpt-region", "hanoi-2-vn")
	case "VN/SGN2":
		request.Header.Set("fpt-region", "saigon-02-vn")
	case "JP/JCSI2":
		request.Header.Set("fpt-region", "tokyo-jp")
	default:
		request.Header.Set("fpt-region", m.Client.Region)
	}
	request.Header.Set("infra-type", strings.ToUpper(infraType))

	return m.Client.SendRequest(request)
}

// managedKubernetesEngineListPageSize is the page size used when listing the
// clusters of a VPC. A VPC rarely holds more than a handful of clusters, so
// one request is normally enough.
const managedKubernetesEngineListPageSize = 1000

// listClusters returns every managed FKE cluster in the VPC, v1 and v2 alike.
// Pages are requested until one comes back short. A cluster that shows up on
// two pages is kept once, and paging stops if a page adds nothing new, so an
// API that ignores the page parameter cannot loop forever.
func (m *MfkeApiClient) listClusters(vpcId string, platform string) ([]managedKubernetesEngineShoot, error) {
	all := []managedKubernetesEngineShoot{}
	seen := map[string]bool{}

	for page := 1; ; page++ {
		path := commons.ApiPath.ManagedFKEList(vpcId, page, managedKubernetesEngineListPageSize, platform)
		a, err := m.sendGet(path, platform)
		if err != nil {
			return nil, err
		}

		var response managedKubernetesEngineListResponse
		if err = json.Unmarshal(a, &response); err != nil {
			return nil, err
		}
		if response.Error {
			return nil, fmt.Errorf("error listing clusters of VPC %s: %v", vpcId, response.Mess)
		}

		items := response.Data.Items
		added := 0
		for _, item := range items {
			if seen[item.Metadata.Name] {
				continue
			}
			seen[item.Metadata.Name] = true
			all = append(all, item)
			added++
		}

		if len(items) < managedKubernetesEngineListPageSize || added == 0 {
			return all, nil
		}
	}
}
