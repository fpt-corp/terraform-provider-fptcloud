package fptcloud_database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Status of a proxy. It uses the same enum as a cluster.
const (
	proxyStatusRunning     = "running"
	proxyStatusWarning     = "warning"
	proxyStatusFailed      = "failed"
	proxyStatusCreateError = "create_error"
	proxyStatusDeleteError = "delete_error"
	proxyStatusDeleting    = "deleting"
)

// Intervals are variables so that tests do not have to wait.
var (
	proxyPollInterval  = 10 * time.Second
	proxyCreateTimeout = 45 * time.Minute
	proxyUpdateTimeout = 45 * time.Minute
	proxyDeleteTimeout = 30 * time.Minute
	// proxyAppearTimeout is how long create waits for the new proxy to show up in the list.
	proxyAppearTimeout = 3 * time.Minute
	// proxyRestartGrace is how long a restart waits for the proxy to leave "running" before it takes
	// the proxy that stayed "running" as already restarted.
	proxyRestartGrace = 60 * time.Second
)

// errProxyNotFound is returned when the portal has no proxy with the id.
var errProxyNotFound = errors.New("proxy not found")

// errProxyDeleteFellBack is returned by waitProxyDeleted when a proxy that was deleting is "failed" again. The portal
// does this when the delete came while the proxy was still busy; the delete has to be sent again.
var errProxyDeleteFellBack = errors.New("the proxy is failed again after the delete")

// proxyDeleteAttempts is how many times a delete is sent before the portal's fall back to "failed" is reported.
const proxyDeleteAttempts = 3

// proxySettled says whether a status is one a proxy rests in, as opposed to one it passes through.
func proxySettled(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case proxyStatusRunning, proxyStatusWarning:
		return true
	}
	return false
}

// proxyFailedStatus says whether a status is a failure of the last operation.
func proxyFailedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case proxyStatusFailed, proxyStatusCreateError, proxyStatusDeleteError:
		return true
	}
	return false
}

// proxyCanChange says whether the portal allows a resize or a restart in this status.
func proxyCanChange(status string) bool {
	return proxySettled(status)
}

// proxyIsHA reads is_lb_cluster, which the list returns as "True"/"False" and the detail as a boolean.
func proxyIsHA(v apiString) bool {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case "true", "yes", "1":
		return true
	}
	return false
}

// proxyListItem is a row of load-balancing/list.
type proxyListItem struct {
	LbClusterId    string    `json:"lb_cluster_id"`
	LbClusterName  apiString `json:"lb_cluster_name"`
	IsLbCluster    apiString `json:"is_lb_cluster"`
	LbType         apiString `json:"lb_type"`
	NodeCpu        flexInt   `json:"node_cpu"`
	NodeRam        flexInt   `json:"node_ram"`
	StorageProfile apiString `json:"storage_profile"`
	MasterCount    flexInt   `json:"master_count"`
	SlaveCount     flexInt   `json:"slave_count"`
	VmSync         bool      `json:"vm_sync"`
	Status         apiString `json:"status"`
	DbClusterId    apiString `json:"db_cluster_id"`
	CreatedAt      apiString `json:"created_at"`
}

// proxyDetail is the cluster part of load-balancing/detail plus the number of nodes.
type proxyDetail struct {
	DbClusterId    apiString `json:"db_cluster_id"`
	LbClusterId    string    `json:"lb_cluster_id"`
	LbClusterName  apiString `json:"lb_cluster_name"`
	IsLbCluster    apiString `json:"is_lb_cluster"`
	LbType         apiString `json:"lb_type"`
	NodeCpu        flexInt   `json:"node_cpu"`
	NodeRam        flexInt   `json:"node_ram"`
	StorageProfile apiString `json:"storage_profile"`
	// EndPoint is null right after creation. When set it has one line per port, for example
	// "R-W ://10.0.0.1:4006/, \nR-O ://10.0.0.1:4008/".
	EndPoint  apiString `json:"end_point"`
	Status    apiString `json:"status"`
	Flavor    apiString `json:"flavor"`
	CreatedAt apiString `json:"created_at"`
	NodeCount int64     `json:"-"`
}

// proxyCreateRequest is the body of load-balancing/create. The portal sends no flavor_id.
type proxyCreateRequest struct {
	DbClusterId    string `json:"db_cluster_id"`
	LbClusterName  string `json:"lb_cluster_name"`
	IsLbCluster    string `json:"is_lb_cluster"` // "yes" or "no"
	LbType         string `json:"lb_type"`
	NodeCpu        int64  `json:"node_cpu"`
	NodeRam        int64  `json:"node_ram"` // GB
	Flavor         string `json:"flavor"`   // name of the flavor, for example "2C4G"
	StorageProfile string `json:"storage_profile"`
}

// proxyResizeRequest is the body of load-balancing/{id}/resize.
type proxyResizeRequest struct {
	LbClusterId string `json:"lb_cluster_id"`
	NodeCpu     int64  `json:"node_cpu"`
	NodeRam     int64  `json:"node_ram"` // GB
	Flavor      string `json:"flavor"`
	FlavorId    string `json:"flavor_id"`
}

// proxyFlavor is a flavor a proxy can be created or resized with.
type proxyFlavor struct {
	FlavorId   string  `json:"flavor_id"`
	FlavorName string  `json:"flavor_name"`
	Vcpu       flexInt `json:"flavor_vcpu"`
	RamMb      flexInt `json:"flavor_ram"`
}

func (f proxyFlavor) ramGb() int64 { return int64(f.RamMb) / 1024 }

// listProxies returns every proxy of a cluster.
func (m *databaseApiClient) listProxies(clusterId string) ([]proxyListItem, error) {
	return getPaged[proxyListItem](m, func(page, size int) string {
		return common.ApiPath.DatabaseProxyList(clusterId, page, size)
	}, 0)
}

// listProxyTypes returns the lb_type values the cluster accepts.
func (m *databaseApiClient) listProxyTypes(clusterId string) ([]string, error) {
	path := common.ApiPath.DatabaseProxyTypes(clusterId)
	body, err := m.getWithRetry(path)
	if err != nil {
		return nil, fmt.Errorf("failed calling path %s: %v", path, err)
	}
	if berr := checkBusinessError(body); berr != nil {
		return nil, berr
	}
	var resp struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing %s: %v, body=%s", path, err, truncateBody(body))
	}
	return resp.Data, nil
}

// listProxyFlavors returns the flavors of the VPC. The portal asks with is_ops=yes on every platform.
func (m *databaseApiClient) listProxyFlavors(vpcId string) ([]proxyFlavor, error) {
	path := common.ApiPath.DatabaseFlavor(vpcId, "yes")
	body, err := m.getWithRetry(path)
	if err != nil {
		return nil, fmt.Errorf("failed calling path %s: %v", path, err)
	}
	var resp struct {
		Data []proxyFlavor `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing %s: %v, body=%s", path, err, truncateBody(body))
	}
	return resp.Data, nil
}

// getProxy reads a proxy. It returns errProxyNotFound when the portal does not know the id: the portal
// answers HTTP 400 ("'NoneType' object has no attribute ...") for an unknown id, not 404.
func (m *databaseApiClient) getProxy(lbClusterId string) (*proxyDetail, error) {
	path := common.ApiPath.DatabaseProxyDetail(lbClusterId)
	var (
		body []byte
		err  error
	)
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(proxyPollInterval)
		}
		if body, err = m.sendGet(path); err == nil {
			break
		}
		var httpErr common.HTTPError
		if errors.As(err, &httpErr) && (httpErr.Code == 400 || httpErr.Code == 404) {
			return nil, errProxyNotFound
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed calling path %s: %v", path, err)
	}
	if berr := checkBusinessError(body); berr != nil {
		return nil, berr
	}
	var resp struct {
		Data struct {
			Cluster proxyDetail `json:"cluster"`
			Nodes   struct {
				Total flexInt `json:"total"`
			} `json:"nodes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing %s: %v, body=%s", path, err, truncateBody(body))
	}
	if resp.Data.Cluster.LbClusterId == "" {
		return nil, errProxyNotFound
	}
	d := resp.Data.Cluster
	d.NodeCount = int64(resp.Data.Nodes.Total)
	return &d, nil
}

// createProxy sends the create call and returns the id of the new proxy. The call does not return the id, so the
// proxy is looked up by name in the list, among the ones that were not there before the call.
func (m *databaseApiClient) createProxy(ctx context.Context, req proxyCreateRequest) (string, error) {
	before, err := m.listProxies(req.DbClusterId)
	if err != nil {
		return "", err
	}
	known := make(map[string]bool, len(before))
	for _, p := range before {
		known[p.LbClusterId] = true
	}

	if err := m.mutate("POST", common.ApiPath.DatabaseProxyCreate(), req); err != nil {
		return "", err
	}

	start := time.Now()
	for {
		items, err := m.listProxies(req.DbClusterId)
		if err != nil {
			tflog.Warn(ctx, "Error listing proxies, retrying: "+err.Error())
		}
		for _, p := range items {
			if !known[p.LbClusterId] && string(p.LbClusterName) == req.LbClusterName {
				return p.LbClusterId, nil
			}
		}
		if time.Since(start) >= proxyAppearTimeout {
			return "", fmt.Errorf("the portal accepted the proxy %q but it did not appear in the proxy list of cluster %s within %s",
				req.LbClusterName, req.DbClusterId, proxyAppearTimeout)
		}
		time.Sleep(proxyPollInterval)
	}
}

func (m *databaseApiClient) restartProxy(lbClusterId string) error {
	return m.mutate("POST", common.ApiPath.DatabaseProxyRestart(), map[string]string{"lb_cluster_id": lbClusterId})
}

func (m *databaseApiClient) resizeProxy(req proxyResizeRequest) error {
	return m.mutate("POST", common.ApiPath.DatabaseProxyResize(req.LbClusterId), req)
}

func (m *databaseApiClient) deleteProxy(lbClusterId string) error {
	return m.mutate("DELETE", common.ApiPath.DatabaseProxyDelete(lbClusterId), nil)
}

// pollProxy syncs the VPC, as the cluster waiters do, then reads the proxy. A read that fails is logged and
// reported as (nil, nil) so the caller tries again; a proxy that is gone returns errProxyNotFound.
func (m *databaseApiClient) pollProxy(ctx context.Context, vpcId, lbClusterId string) (*proxyDetail, error) {
	syncVpcInfrastructure(ctx, m.Client, vpcId)
	d, err := m.getProxy(lbClusterId)
	if errors.Is(err, errProxyNotFound) {
		return nil, err
	}
	if err != nil {
		tflog.Warn(ctx, "Error polling proxy, retrying: "+err.Error())
		return nil, nil
	}
	return d, nil
}

// waitProxyReady polls until the proxy is running and has an end point. The portal reports "running" before the
// nodes and the end point exist, so running alone is not enough.
func (m *databaseApiClient) waitProxyReady(ctx context.Context, vpcId, lbClusterId string, waitTimeout time.Duration) (*proxyDetail, error) {
	start := time.Now()
	for time.Since(start) < waitTimeout {
		time.Sleep(proxyPollInterval)
		d, err := m.pollProxy(ctx, vpcId, lbClusterId)
		if err != nil {
			return nil, fmt.Errorf("proxy %s disappeared while it was being created", lbClusterId)
		}
		if d == nil {
			continue
		}
		status := string(d.Status)
		tflog.Info(ctx, fmt.Sprintf("Waiting for proxy %s: status=%s, end_point=%q", lbClusterId, status, d.EndPoint))
		if proxyFailedStatus(status) {
			return d, fmt.Errorf("proxy %s entered %s status while it was being created", lbClusterId, status)
		}
		if proxySettled(status) && strings.TrimSpace(string(d.EndPoint)) != "" {
			return d, nil
		}
	}
	return nil, fmt.Errorf("timed out waiting for proxy %s to be running", lbClusterId)
}

// waitProxyResized polls until the proxy is settled with the new cpu and ram. If it went through a busy status
// and came back settled twice with the old values, the portal did not apply the resize.
func (m *databaseApiClient) waitProxyResized(ctx context.Context, vpcId, lbClusterId string, cpu, ramGb int64, waitTimeout time.Duration) error {
	start := time.Now()
	sawBusy := false
	settledWithoutChange := 0
	for time.Since(start) < waitTimeout {
		time.Sleep(proxyPollInterval)
		d, err := m.pollProxy(ctx, vpcId, lbClusterId)
		if err != nil {
			return fmt.Errorf("proxy %s disappeared while it was being resized", lbClusterId)
		}
		if d == nil {
			continue
		}
		status := string(d.Status)
		tflog.Info(ctx, fmt.Sprintf("Waiting for proxy %s resize: status=%s, cpu=%d, ram=%d", lbClusterId, status, d.NodeCpu, d.NodeRam))
		if proxyFailedStatus(status) {
			return fmt.Errorf("proxy %s entered %s status while it was being resized", lbClusterId, status)
		}
		if !proxySettled(status) {
			if strings.TrimSpace(status) != "" {
				sawBusy = true
			}
			settledWithoutChange = 0
			continue
		}
		if int64(d.NodeCpu) == cpu && int64(d.NodeRam) == ramGb {
			return nil
		}
		if sawBusy {
			settledWithoutChange++
			if settledWithoutChange >= 2 {
				return fmt.Errorf("proxy %s is %s again after the resize, but it still has %d vCPU and %d GB RAM: the portal did not apply the resize. "+
					"Check the proxy in the portal, then apply again", lbClusterId, status, d.NodeCpu, d.NodeRam)
			}
		}
	}
	return fmt.Errorf("timed out waiting for proxy %s to be resized", lbClusterId)
}

// waitProxyRestarted polls until the proxy is running again after a restart. The call returns before the status
// changes, so a "running" read only counts once the proxy was seen busy or proxyRestartGrace has passed.
func (m *databaseApiClient) waitProxyRestarted(ctx context.Context, vpcId, lbClusterId string, waitTimeout time.Duration) (*proxyDetail, error) {
	start := time.Now()
	sawBusy := false
	for time.Since(start) < waitTimeout {
		time.Sleep(proxyPollInterval)
		d, err := m.pollProxy(ctx, vpcId, lbClusterId)
		if err != nil {
			return nil, fmt.Errorf("proxy %s disappeared while it was being restarted", lbClusterId)
		}
		if d == nil {
			continue
		}
		status := string(d.Status)
		tflog.Info(ctx, fmt.Sprintf("Waiting for proxy %s restart: status=%s", lbClusterId, status))
		if proxyFailedStatus(status) {
			return d, fmt.Errorf("proxy %s entered %s status while it was being restarted", lbClusterId, status)
		}
		if !proxySettled(status) {
			if strings.TrimSpace(status) != "" {
				sawBusy = true
			}
			continue
		}
		if sawBusy || time.Since(start) >= proxyRestartGrace {
			return d, nil
		}
	}
	return nil, fmt.Errorf("timed out waiting for proxy %s to be restarted", lbClusterId)
}

// waitProxyDeleted polls until the proxy is gone from the list of its cluster or the detail no longer finds it.
// A delete_error ends the wait. So does a proxy that was deleting and is "failed" again (errProxyDeleteFellBack):
// the portal gives up that way, and the delete has to be sent again.
func (m *databaseApiClient) waitProxyDeleted(ctx context.Context, vpcId, clusterId, lbClusterId string, waitTimeout time.Duration) error {
	start := time.Now()
	sawDeleting := false
	for time.Since(start) < waitTimeout {
		time.Sleep(proxyPollInterval)
		syncVpcInfrastructure(ctx, m.Client, vpcId)
		items, err := m.listProxies(clusterId)
		if err != nil {
			tflog.Warn(ctx, "Error listing proxies, retrying: "+err.Error())
			continue
		}
		var found *proxyListItem
		for i := range items {
			if items[i].LbClusterId == lbClusterId {
				found = &items[i]
				break
			}
		}
		if found == nil {
			return nil
		}
		status := strings.ToLower(string(found.Status))
		tflog.Info(ctx, fmt.Sprintf("Waiting for proxy %s to be deleted: status=%s", lbClusterId, status))
		if status == proxyStatusDeleting {
			sawDeleting = true
		}
		if sawDeleting && status == proxyStatusFailed {
			return fmt.Errorf("proxy %s could not be deleted: its status is %s: %w", lbClusterId, status, errProxyDeleteFellBack)
		}
		if status == proxyStatusDeleteError {
			return fmt.Errorf("proxy %s could not be deleted: its status is %s. Check it in the portal, then apply again", lbClusterId, status)
		}
	}
	return fmt.Errorf("timed out waiting for proxy %s to be deleted", lbClusterId)
}

// proxyIdleStatus says whether a status is one the proxy rests in and takes a delete in: the portal lets a
// running, warning, stopped, failed or disabled proxy be deleted; a proxy already deleting needs nothing more.
func proxyIdleStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case proxyStatusRunning, proxyStatusWarning, "stopped", "disabled", proxyStatusFailed, proxyStatusCreateError, proxyStatusDeleteError, proxyStatusDeleting:
		return true
	}
	return false
}

// waitProxyIdle polls until the proxy rests in a status that takes a delete (see proxyIdleStatus). The portal
// takes a delete sent while the proxy is busy, then puts the proxy back to "failed" without deleting it.
func (m *databaseApiClient) waitProxyIdle(ctx context.Context, vpcId, lbClusterId string, waitTimeout time.Duration) (*proxyDetail, error) {
	start := time.Now()
	for time.Since(start) < waitTimeout {
		d, err := m.pollProxy(ctx, vpcId, lbClusterId)
		if err != nil {
			return nil, err
		}
		if d != nil {
			status := string(d.Status)
			if proxyIdleStatus(status) {
				return d, nil
			}
			tflog.Info(ctx, fmt.Sprintf("Waiting for proxy %s to be idle: status=%s", lbClusterId, status))
		}
		time.Sleep(proxyPollInterval)
	}
	return nil, fmt.Errorf("timed out waiting for proxy %s to be idle", lbClusterId)
}

// resolveProxyFlavor picks the flavor of a proxy. The flavor id, the flavor name or the cpu and ram can be given;
// when several are, they must agree.
func resolveProxyFlavor(flavors []proxyFlavor, cpu, ramGb int64, name, id string) (*proxyFlavor, error) {
	var f *proxyFlavor
	switch {
	case id != "":
		for i := range flavors {
			if flavors[i].FlavorId == id {
				f = &flavors[i]
				break
			}
		}
		if f == nil {
			return nil, fmt.Errorf("flavor_id %q is not a flavor of this VPC", id)
		}
		if name != "" && f.FlavorName != name {
			return nil, fmt.Errorf("flavor_id %q is the flavor %q, not %q", id, f.FlavorName, name)
		}
	case name != "":
		for i := range flavors {
			if flavors[i].FlavorName == name {
				f = &flavors[i]
				break
			}
		}
		if f == nil {
			return nil, fmt.Errorf("flavor %q is not a flavor of this VPC", name)
		}
	default:
		for i := range flavors {
			if int64(flavors[i].Vcpu) == cpu && flavors[i].ramGb() == ramGb {
				f = &flavors[i]
				break
			}
		}
		if f == nil {
			return nil, fmt.Errorf("no flavor has %d vCPU and %d GB RAM; set node_cpu and node_ram to those of a flavor of the VPC", cpu, ramGb)
		}
	}
	if int64(f.Vcpu) != cpu || f.ramGb() != ramGb {
		return nil, fmt.Errorf("flavor %q has %d vCPU and %d GB RAM, not %d vCPU and %d GB RAM: set node_cpu and node_ram to match the flavor",
			f.FlavorName, f.Vcpu, f.ramGb(), cpu, ramGb)
	}
	return f, nil
}
