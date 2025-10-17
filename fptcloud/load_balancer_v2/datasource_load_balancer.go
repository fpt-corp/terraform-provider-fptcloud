package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceLoadBalancers() *schema.Resource {
	return &schema.Resource{
		ReadContext: listLoadBalancers,
		Schema:      dataSourceLoadBalancers,
	}
}

func DataSourceLoadBalancer() *schema.Resource {
	return &schema.Resource{
		ReadContext: getLoadBalancer,
		Schema:      dataSourceLoadBalancer,
	}
}

func listLoadBalancers(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	response, err := service.ListLoadBalancers(vpcId, 1, 1000)
	if err != nil {
		return diag.FromErr(err)
	}
	loadBalancers := response.LoadBalancers
	var formattedData []interface{}
	for _, lb := range loadBalancers {
		size := []interface{}{
			map[string]interface{}{
				"id":                     lb.Size.Id,
				"name":                   lb.Size.Name,
				"vip_amount":             lb.Size.VipAmount,
				"active_connection":      lb.Size.ActiveConnection,
				"application_throughput": lb.Size.ApplicationThroughput,
			},
		}
		publicIp := []interface{}{
			map[string]interface{}{
				"id":         lb.PublicIp.Id,
				"ip_address": lb.PublicIp.IpAddress,
			},
		}
		network := []interface{}{
			map[string]interface{}{
				"id":   lb.Network.Id,
				"name": lb.Network.Name,
			},
		}
		var tags []interface{}
		for _, v := range lb.Tags {
			tags = append(tags, v)
		}
		formattedData = append(formattedData, map[string]interface{}{
			"id":                  lb.Id,
			"name":                lb.Name,
			"description":         lb.Description,
			"operating_status":    lb.OperatingStatus,
			"provisioning_status": lb.ProvisioningStatus,
			"public_ip":           publicIp,
			"private_ip":          lb.PrivateIp,
			"network":             network,
			"cidr":                lb.Cidr,
			"size":                size,
			"created_at":          lb.CreatedAt,
			"tags":                tags,
			"egw_name":            lb.EgwName,
		})
	}
	if err := d.Set("loadbalancers", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer list: %v", err))
	}
	d.SetId(vpcId)

	return nil
}

func getLoadBalancer(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	loadBalancerId := d.Get("load_balancer_id").(string)
	vpcId := d.Get("vpc_id").(string)
	response, err := service.GetLoadBalancer(vpcId, loadBalancerId)
	if err != nil {
		return diag.FromErr(err)
	}
	loadBalancer := response.LoadBalancer
	size := []interface{}{
		map[string]interface{}{
			"id":                     loadBalancer.Size.Id,
			"name":                   loadBalancer.Size.Name,
			"vip_amount":             loadBalancer.Size.VipAmount,
			"active_connection":      loadBalancer.Size.ActiveConnection,
			"application_throughput": loadBalancer.Size.ApplicationThroughput,
		},
	}
	publicIp := []interface{}{
		map[string]interface{}{
			"id":         loadBalancer.PublicIp.Id,
			"ip_address": loadBalancer.PublicIp.IpAddress,
		},
	}
	network := []interface{}{
		map[string]interface{}{
			"id":   loadBalancer.Network.Id,
			"name": loadBalancer.Network.Name,
		},
	}
	if err := d.Set("load_balancer_id", loadBalancer.Id); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer id: %v", err))
	}
	if err := d.Set("name", loadBalancer.Name); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer name: %v", err))
	}
	if err := d.Set("description", loadBalancer.Description); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer description: %v", err))
	}
	if err := d.Set("operating_status", loadBalancer.OperatingStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer operating status: %v", err))
	}
	if err := d.Set("provisioning_status", loadBalancer.ProvisioningStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer provisioning status: %v", err))
	}
	if err := d.Set("private_ip", loadBalancer.PrivateIp); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer private ip: %v", err))
	}
	if err := d.Set("cidr", loadBalancer.Cidr); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer cidr: %v", err))
	}
	if err := d.Set("created_at", loadBalancer.CreatedAt); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer create date: %v", err))
	}
	if err := d.Set("tags", loadBalancer.Tags); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer tags: %v", err))
	}
	if err := d.Set("egw_name", loadBalancer.EgwName); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer edge gateway name: %v", err))
	}
	if err := d.Set("size", size); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer size: %v", err))
	}
	if err := d.Set("public_ip", publicIp); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer public ip: %v", err))
	}
	if err := d.Set("network", network); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer network: %v", err))
	}
	d.SetId(loadBalancerId)

	return nil
}
