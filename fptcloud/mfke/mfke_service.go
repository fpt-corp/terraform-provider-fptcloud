package fptcloud_mfke

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"terraform-provider-fptcloud/commons"
)

type MfkeApiClient struct {
	*commons.Client
}

func newMfkeApiClient(c *commons.Client) *MfkeApiClient {
	return &MfkeApiClient{c}
}

func (m *MfkeApiClient) sendGet(requestURL string, infraType string) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return m.sendRequestWithHeader(req, infraType)
}

func (m *MfkeApiClient) sendPost(requestURL string, infraType string, params interface{}) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)

	// we create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}
	return m.sendRequestWithHeader(req, infraType)
}

func (m *MfkeApiClient) sendPatch(requestURL string, infraType string, params interface{}) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)

	// we create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)

	req, err := http.NewRequest("PATCH", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}
	return m.sendRequestWithHeader(req, infraType)
}

func (m *MfkeApiClient) sendRequestWithHeader(request *http.Request, infraType string) ([]byte, error) {
	request.Header.Set("fpt-region", m.Client.Region)
	request.Header.Set("infra-type", strings.ToUpper(infraType))
	return m.Client.SendRequest(request)
}
