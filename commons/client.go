package commons

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"
)

// Client is the means of connecting to the Fpt cloud API service
type Client struct {
	BaseURL          *url.URL
	UserAgent        string
	APIKey           string
	TenantName       string
	Region           string
	LastJSONResponse string

	httpClient *http.Client
}

// Component is a struct to define a User-Agent from a client
type Component struct {
	ID, Name, Version string
}

// HTTPError is the error returned when the API fails with an HTTP error
type HTTPError struct {
	Code   int
	Status string
	Reason string
}

// SimpleResponse is a structure that returns success and/or any error
type SimpleResponse struct {
	Data   string
	Status string
}

// ConfigAdvanceClientForTesting initializes a Client connecting to a local test server and allows for specifying methods
type ConfigAdvanceClientForTesting struct {
	Method string
	Value  []ValueAdvanceClientForTesting
}

// ValueAdvanceClientForTesting is a struct that holds the URL and the request body
type ValueAdvanceClientForTesting struct {
	RequestBody  string
	URL          string
	ResponseBody string
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("%d: %s, %s", e.Code, e.Status, e.Reason)
}

// NewClientWithURL initializes a Client with a specific API URL
func NewClientWithURL(apiKey, apiUrl, region string, tenantName string) (*Client, error) {
	parsedURL, err := url.Parse(apiUrl)
	if err != nil {
		return nil, err
	}

	var httpTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	client := &Client{
		BaseURL:    parsedURL,
		APIKey:     apiKey,
		Region:     region,
		TenantName: tenantName,
		httpClient: &http.Client{
			Transport: httpTransport,
			Timeout:   5 * time.Minute,
		},
	}
	return client, nil
}

func (c *Client) PrepareClientURL(requestURL string) *url.URL {
	u, _ := url.Parse(c.BaseURL.String() + requestURL)
	return u
}

func (c *Client) SendRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	c.httpClient.Transport = &http.Transport{
		DisableCompression: false,
	}

	if req.Method == "GET" || req.Method == "DELETE" {
		// add the region param
		param := req.URL.Query()
		req.URL.RawQuery = param.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	c.LastJSONResponse = string(body)

	if resp.StatusCode >= 300 {
		return nil, HTTPError{Code: resp.StatusCode, Status: resp.Status, Reason: string(body)}
	}

	return body, err
}

// SendGetRequest sends a correctly authenticated get request to the API server
func (c *Client) SendGetRequest(requestURL string) ([]byte, error) {
	u := c.PrepareClientURL(requestURL)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return c.SendRequest(req)
}

// SendPostRequest sends a correctly authenticated post request to the API server
func (c *Client) SendPostRequest(requestURL string, params interface{}) ([]byte, error) {
	u := c.PrepareClientURL(requestURL)

	// we create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}
	return c.SendRequest(req)
}

// SendPutRequest sends a correctly authenticated put request to the API server
func (c *Client) SendPutRequest(requestURL string, params interface{}) ([]byte, error) {
	u := c.PrepareClientURL(requestURL)

	// we create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)

	req, err := http.NewRequest("PUT", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}
	return c.SendRequest(req)
}

// SendDeleteRequest sends a correctly authenticated delete request to the API server
func (c *Client) SendDeleteRequest(requestURL string) ([]byte, error) {
	u := c.PrepareClientURL(requestURL)
	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return c.SendRequest(req)
}

// SendDeleteRequestWithBody sends a correctly authenticated delete request to the API server
func (c *Client) SendDeleteRequestWithBody(requestURL string, params interface{}) ([]byte, error) {
	u := c.PrepareClientURL(requestURL)

	// we create a new buffer and encode everything to json to send it in the request
	jsonValue, _ := json.Marshal(params)

	req, err := http.NewRequest("DELETE", u.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}

	return c.SendRequest(req)
}

// SetUserAgent sets the user agent for the client
func (c *Client) SetUserAgent(component *Component) {
	if component.ID == "" {
		c.UserAgent = fmt.Sprintf("%s/%s %s", component.Name, component.Version, c.UserAgent)
	} else {
		c.UserAgent = fmt.Sprintf("%s/%s-%s %s", component.Name, component.Version, component.ID, c.UserAgent)
	}
}

// DecodeSimpleResponse parses a response body in to a SimpleResponse object
func (c *Client) DecodeSimpleResponse(resp []byte) (*SimpleResponse, error) {
	response := SimpleResponse{}
	err := json.NewDecoder(bytes.NewReader(resp)).Decode(&response)
	return &response, err
}

// NewClientForTestingWithServer initializes a Client connecting to a passed-in local test server
func NewClientForTestingWithServer(server *httptest.Server) (*Client, error) {
	client, err := NewClientWithURL("TEST-API-KEY", server.URL, "TEST", "TEST")
	if err != nil {
		return nil, err
	}
	client.httpClient = server.Client()
	return client, err
}

// NewClientForTesting initializes a Client connecting to a local test server
func NewClientForTesting(responses map[string]string) (*Client, *httptest.Server, error) {
	var responseSent bool

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		for reqUrl, response := range responses {
			if strings.Contains(req.URL.String(), reqUrl) {
				responseSent = true
				_, err := rw.Write([]byte(response))
				if err != nil {
					return
				}
			}
		}

		if !responseSent {
			fmt.Println("Failed to find a matching request!")
			fmt.Println("URL:", req.URL.String())

			_, err := rw.Write([]byte(`{"result": "failed to find a matching request"}`))
			if err != nil {
				return
			}
		}
	}))

	client, err := NewClientForTestingWithServer(server)

	return client, server, err
}
