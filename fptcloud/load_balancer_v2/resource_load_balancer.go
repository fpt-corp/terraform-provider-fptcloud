package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceLoadBalancer() *schema.Resource {
	return &schema.Resource{
		CreateContext: createLoadBalancer,
		ReadContext:   readLoadBalancer,
		UpdateContext: updateLoadBalancer,
		DeleteContext: deleteLoadBalancer,
		Schema:        resourceLoadBalancer,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 4 || parts[0] != "vpc" || parts[2] != "load_balancer" {
					return nil, fmt.Errorf("invalid import id format, expected vpc/<vpc_id>/load_balancer/<load_balancer_id>")
				}
				vpcId := parts[1]
				loadBalancerId := parts[3]

				if err := d.Set("vpc_id", vpcId); err != nil {
					return nil, fmt.Errorf("error setting vpc id: %s", err)
				}
				d.SetId(loadBalancerId)

				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func readLoadBalancer(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	loadBalancerId := d.Id()

	response, err := service.GetLoadBalancer(vpcId, loadBalancerId)
	if err != nil {
		return diag.FromErr(err)
	}
	loadBalancer := response.LoadBalancer
	d.Set("name", loadBalancer.Name)
	d.Set("description", loadBalancer.Description)
	d.Set("size", loadBalancer.Size.Name)
	d.Set("floating_ip", loadBalancer.PublicIp.IpAddress)
	d.Set("network_id", loadBalancer.Network.Id)
	d.Set("vip_address", loadBalancer.PrivateIp)
	d.Set("cidr", loadBalancer.Cidr)
	d.Set("egw_id", loadBalancer.EgwId)
	return nil
}

func createLoadBalancer(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)

	var payload LoadBalancerCreateModel

	payload.Name = d.Get("name").(string)
	payload.Description = d.Get("description").(string)
	payload.Size = d.Get("size").(string)
	payload.FloatingIp = d.Get("floating_ip").(string)
	payload.NetworkId = d.Get("network_id").(string)
	payload.VipAddress = d.Get("vip_address").(string)
	payload.Cidr = d.Get("cidr").(string)
	payload.EgwId = d.Get("egw_id").(string)

	listener := d.Get("listener").(*schema.Set).List()[0].(map[string]interface{})
	listenerPayload := DefaultListener{
		Name:          listener["name"].(string),
		Protocol:      listener["protocol"].(string),
		ProtocolPort:  listener["protocol_port"].(string),
		CertificateId: listener["certificate_id"].(string),
	}
	payload.Listener = listenerPayload

	pool := d.Get("pool").(*schema.Set).List()[0].(map[string]interface{})
	poolPayload := InputDefaultServerPool{
		Name:                  pool["name"].(string),
		Protocol:              pool["protocol"].(string),
		Algorithm:             pool["algorithm"].(string),
		PersistenceType:       pool["persistence_type"].(string),
		PersistenceCookieName: pool["persistence_cookie_name"].(string),
	}

	healthMonitor := pool["health_monitor"].(*schema.Set).List()[0].(map[string]interface{})
	healthMonitorPayload := InputHealthMonitor{
		Type:           healthMonitor["type"].(string),
		UrlPath:        healthMonitor["url_path"].(string),
		HttpMethod:     healthMonitor["http_method"].(string),
		ExpectedCodes:  healthMonitor["expected_codes"].(string),
		MaxRetries:     healthMonitor["max_retries"].(string),
		MaxRetriesDown: healthMonitor["max_retries_down"].(string),
		Delay:          healthMonitor["delay"].(string),
		Timeout:        healthMonitor["timeout"].(string),
	}
	poolPayload.HealthMonitor = healthMonitorPayload

	poolMembers := pool["pool_members"].(*schema.Set).List()
	poolMembersPayload := make([]InputPoolMember, 0, len(poolMembers))
	for _, item := range poolMembers {
		member := item.(map[string]interface{})
		memberPayload := InputPoolMember{
			VmId:         member["vm_id"].(string),
			IpAddress:    member["ip_address"].(string),
			NetworkId:    member["network_id"].(string),
			ProtocolPort: member["protocol_port"].(int),
			Weight:       member["weight"].(int),
			Name:         member["name"].(string),
			IsExternal:   member["is_external"].(bool),
		}
		poolMembersPayload = append(poolMembersPayload, memberPayload)
	}
	poolPayload.PoolMembers = poolMembersPayload
	payload.Pool = poolPayload

	response, err := service.CreateLoadBalancer(vpcId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(response.Data.Id)
	return nil
}

func updateLoadBalancer(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	loadBalancerId := d.Id()

	if d.HasChange("size") {
		new_size := d.Get("size").(string)
		var payload LoadBalancerResizeModel
		payload.NewSize = new_size

		_, err := service.ResizeLoadBalancer(vpcId, loadBalancerId, payload)
		if err != nil {
			return diag.FromErr(err)
		}
		return readLoadBalancer(ctx, d, m)
	} else if d.HasChange("name") || d.HasChange("description") || d.HasChange("floating_ip") {
		var payload LoadBalancerUpdateModel
		payload.Name = d.Get("name").(string)
		payload.Description = d.Get("description").(string)
		payload.FloatingIp = d.Get("floating_ip").(string)

		_, err := service.UpdateLoadBalancer(vpcId, loadBalancerId, payload)
		if err != nil {
			return diag.FromErr(err)
		}
		return readLoadBalancer(ctx, d, m)
	}
	return nil
}

func deleteLoadBalancer(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	loadBalancerId := d.Id()

	_, err := service.DeleteLoadBalancer(vpcId, loadBalancerId)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
