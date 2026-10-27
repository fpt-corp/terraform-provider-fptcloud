package fptcloud_database
 
import (
  "net/http"
  "os"
  "strings"
  "testing"
)
 
func TestGetVpcResources(t *testing.T) {
  body, err := os.ReadFile("testdata/vpc_resources.json")
  if err != nil {
    t.Fatal(err)
  }
  api := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/v1/xplat/database/management/cluster/resource/vpc-1" {
      http.NotFound(w, r)
      return
    }
    _, _ = w.Write(body)
  }))
 
  res, err := api.getVpcResources("vpc-1")
  if err != nil {
    t.Fatal(err)
  }
  if res.Total.ClusterCount != 15 || len(res.Clusters) != 15 {
    t.Fatalf("clusters: total %d, listed %d", res.Total.ClusterCount, len(res.Clusters))
  }
  if res.Total.Proxy.NodeCount != 2 || res.Total.Database.RamGb != 268 {
    t.Fatalf("unexpected totals: %+v", res.Total)
  }
  var withProxy *vpcCluster
  for i, c := range res.Clusters {
    if c.Proxy.Items == nil {
      t.Fatalf("cluster %s: proxy items must not be nil", c.ClusterId)
    }
    if c.ClusterId == "ckgdzwct" {
      withProxy = &res.Clusters[i]
    }
  }
  if withProxy == nil || len(withProxy.Proxy.Items) != 2 || withProxy.Proxy.Items[0].LbType != "haproxy" {
    t.Fatalf("cluster ckgdzwct proxies: %+v", withProxy)
  }
  if res.Clusters[0].Backup.UsedGb != nil {
    t.Fatalf("used_gb null must stay null, got %v", *res.Clusters[0].Backup.UsedGb)
  }
}
 
func TestGetVpcResourcesBusinessError(t *testing.T) {
  api := newTestApi(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    _, _ = w.Write([]byte(`{"message":"vpc not found","type":"error","error_code":404001}`))
  }))
  if _, err := api.getVpcResources("vpc-x"); err == nil || !strings.Contains(err.Error(), "vpc not found") {
    t.Fatalf("expected the API message, got %v", err)
  }
}
