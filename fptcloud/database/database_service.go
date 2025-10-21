package fptcloud_database

import (
	"bytes"
	"encoding/json"
	"net/http"
	"terraform-provider-fptcloud/commons"
)

type databaseApiClient struct {
	*commons.Client
}

func newDatabaseApiClient(c *commons.Client) *databaseApiClient {
	return &databaseApiClient{c}
}

func (m *databaseApiClient) sendGet(requestURL string) ([]byte, error) {
	u := m.PrepareClientURL(requestURL)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	return m.sendRequestWithHeader(req)
}

func (m *databaseApiClient) sendDelete(requestURL string) ([]byte, error) {
	u := m.PrepareClientURL(requestURL)
	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, err
	}
	return m.sendRequestWithHeader(req)
}

func (m *databaseApiClient) sendPost(requestURL string, params interface{}) ([]byte, error) {
	u := m.PrepareClientURL(requestURL)
	// Create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)
	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}
	return m.sendRequestWithHeader(req)
}

func (m *databaseApiClient) sendRequestWithHeader(request *http.Request) ([]byte, error) {
	request.Header.Set("fpt-region", m.Region)
	return m.SendRequest(request)
}
