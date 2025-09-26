package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourcePool() *schema.Resource {
	return &schema.Resource{
		CreateContext: createPool,
		ReadContext:   readPool,
		UpdateContext: updatePool,
		DeleteContext: deletePool,
		Schema:        resourcePool,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 4 || parts[0] != "vpc" || parts[2] != "pool" {
					return nil, fmt.Errorf("invalid import id format, expected vpc/<vpc_id>/pool/<pool_id>")
				}
				vpcId := parts[1]
				poolId := parts[3]

				if err := d.Set("vpc_id", vpcId); err != nil {
					return nil, fmt.Errorf("error setting vpc id: %s", err)
				}
				d.SetId(poolId)

				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func readPool(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	poolId := d.Id()
	response, err := service.GetPool(vpcId, poolId)
	if err != nil {
		return diag.FromErr(err)
	}
	pool := response.Pool
	d.Set("name", pool.Name)
	d.Set("load_balancer_id", pool.LoadBalancerId)
	d.Set("protocol", pool.Protocol)
	d.Set("description", pool.Description)
	d.Set("algorithm", pool.Algorithm)
	healthMonitor := []interface{}{
		map[string]interface{}{
			"type":             pool.HealthMonitor.Type,
			"delay":            strconv.Itoa(pool.HealthMonitor.Delay),
			"max_retries":      strconv.Itoa(pool.HealthMonitor.MaxRetries),
			"max_retries_down": strconv.Itoa(pool.HealthMonitor.MaxRetriesDown),
			"timeout":          strconv.Itoa(pool.HealthMonitor.Timeout),
			"http_method":      pool.HealthMonitor.HttpMethod,
			"url_path":         pool.HealthMonitor.UrlPath,
			"expected_codes":   pool.HealthMonitor.ExpectedCodes,
		},
	}
	d.Set("health_monitor", healthMonitor)
	var members []interface{}
	for _, v := range pool.Members {
		port, _ := strconv.Atoi(v.Port)
		weight, _ := strconv.Atoi(v.Weight)
		members = append(members, map[string]interface{}{
			"name":          v.VmName,
			"vm_id":         v.VmId,
			"ip_address":    v.IpAddress,
			"network_id":    v.Network.Id,
			"protocol_port": port,
			"weight":        weight,
			"is_external":   v.IsExternal,
		})
	}
	d.Set("pool_members", members)
	d.Set("persistence_type", pool.PersistenceType)
	d.Set("persistence_cookie_name", pool.PersistenceCookieName)
	d.Set("alpn_protocols", pool.AlpnProtocols)
	d.Set("tls_enabled", pool.TlsEnabled)
	return nil
}

func createPool(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	loadBalancerId := d.Get("load_balancer_id").(string)

	var payload PoolCreateModel

	healthMonitor := d.Get("health_monitor").([]interface{})[0].(map[string]interface{})
	healthMonitorPayload := InputHealthMonitor{
		Type:           healthMonitor["type"].(string),
		Delay:          healthMonitor["delay"].(string),
		MaxRetries:     healthMonitor["max_retries"].(string),
		MaxRetriesDown: healthMonitor["max_retries_down"].(string),
		Timeout:        healthMonitor["timeout"].(string),
		HttpMethod:     healthMonitor["http_method"].(string),
		UrlPath:        healthMonitor["url_path"].(string),
		ExpectedCodes:  healthMonitor["expected_codes"].(string),
	}

	var membersPayload []InputPoolMember
	for _, member := range d.Get("pool_members").([]interface{}) {
		memberMap := member.(map[string]interface{})
		memberPayload := InputPoolMember{
			VmId:         memberMap["vm_id"].(string),
			IpAddress:    memberMap["ip_address"].(string),
			NetworkId:    memberMap["network_id"].(string),
			ProtocolPort: memberMap["protocol_port"].(int),
			Weight:       memberMap["weight"].(int),
			Name:         memberMap["name"].(string),
			IsExternal:   memberMap["is_external"].(bool),
		}
		membersPayload = append(membersPayload, memberPayload)
	}

	alpnProtocols := []string{}
	for _, v := range d.Get("alpn_protocols").([]interface{}) {
		alpnProtocols = append(alpnProtocols, v.(string))
	}

	payload.Name = d.Get("name").(string)
	payload.Description = d.Get("description").(string)
	payload.Algorithm = d.Get("algorithm").(string)
	payload.Protocol = d.Get("protocol").(string)
	payload.PersistenceType = d.Get("persistence_type").(string)
	payload.PersistenceCookieName = d.Get("persistence_cookie_name").(string)
	payload.HealthMonitor = healthMonitorPayload
	payload.PoolMembers = membersPayload
	payload.AlpnProtocols = alpnProtocols
	payload.TlsEnabled = d.Get("tls_enabled").(bool)

	response, err := service.CreatePool(vpcId, loadBalancerId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(response.Data.Id)
	return nil
}

func updatePool(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	poolId := d.Id()

	var payload PoolUpdateModel

	healthMonitor := d.Get("health_monitor").([]interface{})[0].(map[string]interface{})
	healthMonitorPayload := InputHealthMonitor{
		Type:           healthMonitor["type"].(string),
		Delay:          healthMonitor["delay"].(string),
		MaxRetries:     healthMonitor["max_retries"].(string),
		MaxRetriesDown: healthMonitor["max_retries_down"].(string),
		Timeout:        healthMonitor["timeout"].(string),
		HttpMethod:     healthMonitor["http_method"].(string),
		UrlPath:        healthMonitor["url_path"].(string),
		ExpectedCodes:  healthMonitor["expected_codes"].(string),
	}

	var membersPayload []InputPoolMember
	for _, member := range d.Get("pool_members").([]interface{}) {
		memberMap := member.(map[string]interface{})
		memberPayload := InputPoolMember{
			VmId:         memberMap["vm_id"].(string),
			IpAddress:    memberMap["ip_address"].(string),
			NetworkId:    memberMap["network_id"].(string),
			ProtocolPort: memberMap["protocol_port"].(int),
			Weight:       memberMap["weight"].(int),
			Name:         memberMap["name"].(string),
			IsExternal:   memberMap["is_external"].(bool),
		}
		membersPayload = append(membersPayload, memberPayload)
	}

	alpnProtocols := []string{}
	for _, v := range d.Get("alpn_protocols").([]interface{}) {
		alpnProtocols = append(alpnProtocols, v.(string))
	}

	payload.Name = d.Get("name").(string)
	payload.Description = d.Get("description").(string)
	payload.Algorithm = d.Get("algorithm").(string)
	payload.PersistenceType = d.Get("persistence_type").(string)
	payload.PersistenceCookieName = d.Get("persistence_cookie_name").(string)
	payload.PoolMembers = membersPayload
	payload.HealthMonitor = healthMonitorPayload
	payload.AlpnProtocols = alpnProtocols
	payload.TlsEnabled = d.Get("tls_enabled").(bool)

	_, err := service.UpdatePool(vpcId, poolId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	return readPool(ctx, d, m)
}

func deletePool(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	poolId := d.Id()

	_, err := service.DeletePool(vpcId, poolId)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
