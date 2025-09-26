package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourcePools() *schema.Resource {
	return &schema.Resource{
		ReadContext: listPools,
		Schema:      dataSourcePools,
	}
}

func DataSourcePool() *schema.Resource {
	return &schema.Resource{
		ReadContext: getPool,
		Schema:      dataSourcePool,
	}
}

func listPools(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	loadBalancerId := d.Get("load_balancer_id").(string)
	response, err := service.ListPools(vpcId, loadBalancerId, 1, 1000)
	if err != nil {
		return diag.FromErr(err)
	}
	pools := response.Pools
	var formattedData []interface{}
	for _, pool := range pools {
		health_monitor := []interface{}{
			map[string]interface{}{
				"type":             pool.HealthMonitor.Type,
				"delay":            pool.HealthMonitor.Delay,
				"max_retries":      pool.HealthMonitor.MaxRetries,
				"max_retries_down": pool.HealthMonitor.MaxRetriesDown,
				"timeout":          pool.HealthMonitor.Timeout,
				"http_method":      pool.HealthMonitor.HttpMethod,
				"url_path":         pool.HealthMonitor.UrlPath,
				"expected_codes":   pool.HealthMonitor.ExpectedCodes,
			},
		}
		var members []interface{}
		for _, v := range pool.Members {
			network := []interface{}{
				map[string]interface{}{
					"id":   v.Network.Id,
					"name": v.Network.Name,
				},
			}
			members = append(members, map[string]interface{}{
				"id":                  v.Id,
				"vm_id":               v.VmId,
				"vm_name":             v.VmName,
				"ip_address":          v.IpAddress,
				"network":             network,
				"port":                v.Port,
				"weight":              v.Weight,
				"operating_status":    v.OperatingStatus,
				"provisioning_status": v.ProvisioningStatus,
				"created_at":          v.CreatedAt,
				"is_external":         v.IsExternal,
			})
		}
		var tags []interface{}
		for _, v := range pool.Tags {
			tags = append(tags, v)
		}
		var alpnProtocols []interface{}
		for _, v := range pool.AlpnProtocols {
			alpnProtocols = append(alpnProtocols, v)
		}
		formattedData = append(formattedData, map[string]interface{}{
			"id":                      pool.Id,
			"name":                    pool.Name,
			"description":             pool.Description,
			"load_balancer_id":        pool.LoadBalancerId,
			"operating_status":        pool.OperatingStatus,
			"provisioning_status":     pool.ProvisioningStatus,
			"protocol":                pool.Protocol,
			"algorithm":               pool.Algorithm,
			"health_monitor":          health_monitor,
			"members":                 members,
			"persistence_type":        pool.PersistenceType,
			"persistence_cookie_name": pool.PersistenceCookieName,
			"alpn_protocols":          alpnProtocols,
			"tls_enabled":             pool.TlsEnabled,
			"created_at":              pool.CreatedAt,
			"tags":                    tags,
		})
	}
	if err := d.Set("pools", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool list: %v", err))
	}
	d.SetId(loadBalancerId)

	return nil
}

func getPool(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	poolId := d.Get("pool_id").(string)
	data, err := service.GetPool(vpcId, poolId)
	if err != nil {
		return diag.FromErr(err)
	}
	pool := data.Pool
	healthMonitor := []interface{}{
		map[string]interface{}{
			"type":             pool.HealthMonitor.Type,
			"delay":            pool.HealthMonitor.Delay,
			"max_retries":      pool.HealthMonitor.MaxRetries,
			"max_retries_down": pool.HealthMonitor.MaxRetriesDown,
			"timeout":          pool.HealthMonitor.Timeout,
			"http_method":      pool.HealthMonitor.HttpMethod,
			"url_path":         pool.HealthMonitor.UrlPath,
			"expected_codes":   pool.HealthMonitor.ExpectedCodes,
		},
	}
	var members []interface{}
	for _, v := range pool.Members {
		members = append(members, map[string]interface{}{
			"id":                  v.Id,
			"vm_id":               v.VmId,
			"vm_name":             v.VmName,
			"ip_address":          v.IpAddress,
			"network":             v.Network,
			"port":                v.Port,
			"weight":              v.Weight,
			"operating_status":    v.OperatingStatus,
			"provisioning_status": v.ProvisioningStatus,
			"created_at":          v.CreatedAt,
			"is_external":         v.IsExternal,
		})
	}
	if err := d.Set("pool_id", pool.Id); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool id: %v", err))
	}
	if err := d.Set("name", pool.Name); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool name: %v", err))
	}
	if err := d.Set("description", pool.Description); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool description: %v", err))
	}
	if err := d.Set("load_balancer_id", pool.LoadBalancerId); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer id of pool: %v", err))
	}
	if err := d.Set("operating_status", pool.OperatingStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool operating status: %v", err))
	}
	if err := d.Set("provisioning_status", pool.ProvisioningStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool provisioning status: %v", err))
	}
	if err := d.Set("protocol", pool.Protocol); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool protocol: %v", err))
	}
	if err := d.Set("algorithm", pool.Algorithm); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool algorithm: %v", err))
	}
	if err := d.Set("health_monitor", healthMonitor); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool health monitor: %v", err))
	}
	if err := d.Set("members", members); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool members: %v", err))
	}
	if err := d.Set("persistence_type", pool.PersistenceType); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool persistence type: %v", err))
	}
	if err := d.Set("persistence_cookie_name", pool.PersistenceCookieName); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool persistence cookie name: %v", err))
	}
	if err := d.Set("alpn_protocols", pool.AlpnProtocols); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool alpn protocols: %v", err))
	}
	if err := d.Set("created_at", pool.CreatedAt); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool create date: %v", err))
	}
	if err := d.Set("tags", pool.Tags); err != nil {
		return diag.FromErr(fmt.Errorf("error setting pool tags: %v", err))
	}
	d.SetId(poolId)

	return nil
}
