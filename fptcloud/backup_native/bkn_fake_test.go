package fptcloud_backup_native

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	common "terraform-provider-fptcloud/commons"
)

type fakeResponse struct {
	Status int
	Body   string
}

type recordedRequest struct {
	Method string
	Path   string
	Query  url.Values
	Body   map[string]interface{}
	Served int // index of the scripted response this request got
}

// fakeAPI answers requests by "METHOD /path" (query string excluded). Each
// route holds a sequence of responses: call n gets response n, and the last
// response repeats, which is how polling tests script a state change.
type fakeAPI struct {
	t        *testing.T
	mu       sync.Mutex
	routes   map[string][]fakeResponse
	calls    map[string]int
	requests []recordedRequest
}

func newFakeAPI(t *testing.T, routes map[string][]fakeResponse) (*fakeAPI, *common.Client) {
	t.Helper()
	api := &fakeAPI{t: t, routes: routes, calls: map[string]int{}}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	client, err := common.NewClientForTestingWithServer(server)
	if err != nil {
		t.Fatal(err)
	}
	return api, client
}

func (a *fakeAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := r.Method + " " + r.URL.Path
	raw, _ := io.ReadAll(r.Body)
	var body map[string]interface{}
	_ = json.Unmarshal(raw, &body)
	responses := a.routes[key]
	served := a.calls[key]
	if served >= len(responses) {
		served = len(responses) - 1
	}
	a.requests = append(a.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Body: body, Served: served})

	if len(responses) == 0 {
		a.t.Errorf("unexpected request %s?%s", key, r.URL.RawQuery)
		w.WriteHeader(http.StatusTeapot)
		return
	}
	i := a.calls[key]
	if i >= len(responses) {
		i = len(responses) - 1
	}
	a.calls[key]++
	resp := responses[i]
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(resp.Body))
}

// last returns the most recent request to key ("METHOD /path").
func (a *fakeAPI) last(key string) recordedRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := len(a.requests) - 1; i >= 0; i-- {
		if a.requests[i].Method+" "+a.requests[i].Path == key {
			return a.requests[i]
		}
	}
	a.t.Fatalf("no request to %s", key)
	return recordedRequest{}
}

// servedBefore returns which scripted response the last request to key got
// before the first request to next, or -1 when key was not called before it.
func (a *fakeAPI) servedBefore(key, next string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	served := -1
	for _, r := range a.requests {
		switch r.Method + " " + r.Path {
		case next:
			return served
		case key:
			served = r.Served
		}
	}
	a.t.Fatalf("no request to %s", next)
	return -1
}

func (a *fakeAPI) count(key string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls[key]
}

// setFastPolling shortens every poll in this package for one test.
func setFastPolling(t *testing.T) {
	old, oldHold := pollInterval, runFailureHold
	pollInterval = 10 * time.Millisecond
	runFailureHold = 50 * time.Millisecond
	t.Cleanup(func() { pollInterval, runFailureHold = old, oldHold })
}
