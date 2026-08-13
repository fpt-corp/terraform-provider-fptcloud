package commons

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientWithURL_ValidURL(t *testing.T) {
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant", 5)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "https://api.example.com", client.BaseURL.String())
}

func TestNewClientWithURL_InvalidURL(t *testing.T) {
	client, err := NewClientWithURL("apiKey", ":", "region", "tenant", 5)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestSendGetRequest_ValidResponse(t *testing.T) {
	client, server, err := NewClientForTesting(map[string]string{
		"/test": `{"data": "success"}`,
	})
	assert.NoError(t, err)
	defer server.Close()

	resp, err := client.SendGetRequest("/test")
	assert.NoError(t, err)
	assert.Contains(t, string(resp), "success")
}

func TestSendGetRequest_InvalidURL(t *testing.T) {
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant", 5)
	assert.NoError(t, err)

	resp, err := client.SendGetRequest("://invalid-url")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSendPostRequest_ValidResponse(t *testing.T) {
	client, server, err := NewClientForTesting(map[string]string{
		"/test": `{"data": "success"}`,
	})
	assert.NoError(t, err)
	defer server.Close()

	resp, err := client.SendPostRequest("/test", map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.Contains(t, string(resp), "success")
}

func TestSendPostRequest_InvalidJSON(t *testing.T) {
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant", 5)
	assert.NoError(t, err)

	resp, err := client.SendPostRequest("/test", make(chan int))
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSendDeleteRequest_ValidResponse(t *testing.T) {
	client, server, err := NewClientForTesting(map[string]string{
		"/test": `{"data": "deleted"}`,
	})
	assert.NoError(t, err)
	defer server.Close()

	resp, err := client.SendDeleteRequest("/test")
	assert.NoError(t, err)
	assert.Contains(t, string(resp), "deleted")
}

func TestSendDeleteRequestWithBody_ValidResponse(t *testing.T) {
	client, server, err := NewClientForTesting(map[string]string{
		"/test": `{"data": "deleted"}`,
	})
	assert.NoError(t, err)
	defer server.Close()

	resp, err := client.SendDeleteRequestWithBody("/test", map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.Contains(t, string(resp), "deleted")
}

func TestSetUserAgent_SetsCorrectly(t *testing.T) {
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant", 5)
	assert.NoError(t, err)

	component := &Component{ID: "123", Name: "TestComponent", Version: "1.0"}
	client.SetUserAgent(component)
	assert.Contains(t, client.UserAgent, "TestComponent/1.0-123")
}

func TestDecodeSimpleResponse_ValidResponse(t *testing.T) {
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant", 5)
	assert.NoError(t, err)

	resp := []byte(`{"Data": "success", "Status": "ok"}`)
	simpleResp, err := client.DecodeSimpleResponse(resp)
	assert.NoError(t, err)
	assert.Equal(t, "success", simpleResp.Data)
	assert.Equal(t, "ok", simpleResp.Status)
}

func TestSendRequestWithStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "200", statusCode: 200, body: `{"code":"200"}`, wantErr: false},
		{name: "201", statusCode: 201, body: `{"successes":true}`, wantErr: false},
		{name: "400", statusCode: 400, body: `{"message":"bad request"}`, wantErr: true},
		{name: "500", statusCode: 500, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(tt.statusCode)
				_, _ = rw.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClientForTestingWithServer(server)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", server.URL+"/test", nil)
			assert.NoError(t, err)

			body, statusCode, err := client.SendRequestWithStatus(req)
			assert.Equal(t, tt.statusCode, statusCode)
			// The body comes back even on failure, that is the point of this method
			assert.Equal(t, tt.body, string(body))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// SendRequest has to keep answering exactly like it did before it started delegating
func TestSendRequestKeepsItsContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(`{"message":"boom"}`))
	}))
	defer server.Close()

	client, err := NewClientForTestingWithServer(server)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", server.URL+"/test", nil)
	assert.NoError(t, err)

	body, err := client.SendRequest(req)
	assert.Nil(t, body)

	var httpErr HTTPError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, 500, httpErr.Code)
	assert.Contains(t, httpErr.Reason, "boom")
}

// The transport carries the proxy configuration and the connection pool, so it must be
// built once and never replaced per request
func TestClientKeepsItsTransport(t *testing.T) {
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant", 5)
	assert.NoError(t, err)

	transport, ok := client.httpClient.Transport.(*http.Transport)
	assert.True(t, ok)
	assert.NotNil(t, transport.Proxy, "the transport must keep resolving the proxy from the environment")
}
