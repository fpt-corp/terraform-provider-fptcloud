package fptcloud_database

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	common "terraform-provider-fptcloud/commons"
)

type databaseApiClient struct {
	*common.Client
}

func newDatabaseApiClient(c *common.Client) *databaseApiClient {
	return &databaseApiClient{c}
}

func (m *databaseApiClient) sendGet(requestURL string) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	return m.sendRequestWithHeader(req)
}

func (m *databaseApiClient) sendDelete(requestURL string) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)
	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, err
	}
	return m.sendRequestWithHeader(req)
}

func (m *databaseApiClient) sendPost(requestURL string, params interface{}) ([]byte, error) {
	u := m.Client.PrepareClientURL(requestURL)
	// Create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)
	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}
	return m.sendRequestWithHeader(req)
}

type databaseApiAnswer struct {
	StatusCode int
	Body       []byte
}

func (a databaseApiAnswer) StatusText() string {
	if a.StatusCode == 0 {
		return "2xx"
	}
	return strconv.Itoa(a.StatusCode)
}

func (m *databaseApiClient) sendPostForAnswer(requestURL string, params interface{}) (databaseApiAnswer, error) {
	body, err := m.sendPost(requestURL, params)
	if err == nil {
		return databaseApiAnswer{Body: body}, nil
	}

	var httpErr common.HTTPError
	if errors.As(err, &httpErr) {
		return databaseApiAnswer{StatusCode: httpErr.Code, Body: []byte(httpErr.Reason)}, err
	}

	return databaseApiAnswer{}, err
}

func (m *databaseApiClient) sendRequestWithHeader(request *http.Request) ([]byte, error) {
	m.setRegionHeader(request)
	return m.Client.SendRequest(request)
}

func (m *databaseApiClient) setRegionHeader(request *http.Request) {
	switch m.Client.Region {
	case "VN/HAN":
		request.Header.Set("fpt-region", "hanoi-vn")
	case "VN/SGN":
		request.Header.Set("fpt-region", "saigon-vn")
	case "VN/HAN2":
		request.Header.Set("fpt-region", "hanoi-2-vn")
	case "JP/JCSI2":
		request.Header.Set("fpt-region", "tokyo-jp")
	case "VN/SGN2":
		request.Header.Set("fpt-region", "saigon-02-vn")
	default:
		request.Header.Set("fpt-region", m.Client.Region)
	}
}
