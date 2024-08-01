package commons

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewClientWithURL_ValidURL(t *testing.T) {
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant")
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "https://api.example.com", client.BaseURL.String())
}

func TestNewClientWithURL_InvalidURL(t *testing.T) {
	client, err := NewClientWithURL("apiKey", ":", "region", "tenant")
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
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant")
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
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant")
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
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant")
	assert.NoError(t, err)

	component := &Component{ID: "123", Name: "TestComponent", Version: "1.0"}
	client.SetUserAgent(component)
	assert.Contains(t, client.UserAgent, "TestComponent/1.0-123")
}

func TestDecodeSimpleResponse_ValidResponse(t *testing.T) {
	client, err := NewClientWithURL("apiKey", "https://api.example.com", "region", "tenant")
	assert.NoError(t, err)

	resp := []byte(`{"Data": "success", "Status": "ok"}`)
	simpleResp, err := client.DecodeSimpleResponse(resp)
	assert.NoError(t, err)
	assert.Equal(t, "success", simpleResp.Data)
	assert.Equal(t, "ok", simpleResp.Status)
}
