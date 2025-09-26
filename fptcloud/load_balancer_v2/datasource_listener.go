package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceListeners() *schema.Resource {
	return &schema.Resource{
		ReadContext: listListeners,
		Schema:      dataSourceListeners,
	}
}

func DataSourceListener() *schema.Resource {
	return &schema.Resource{
		ReadContext: getListener,
		Schema:      dataSourceListener,
	}
}

func listListeners(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	loadBalancerId := d.Get("load_balancer_id").(string)
	response, err := service.ListListeners(vpcId, loadBalancerId, 1, 1000)
	if err != nil {
		return diag.FromErr(err)
	}
	listeners := response.Listeners
	var formattedData []interface{}
	for _, listener := range listeners {
		insert_headers := []interface{}{
			map[string]interface{}{
				"x_forwarded_for":   listener.InsertHeaders.XForwardedFor,
				"x_forwarded_port":  listener.InsertHeaders.XForwardedPort,
				"x_forwarded_proto": listener.InsertHeaders.XForwardedProto,
			},
		}
		default_pool := []interface{}{
			map[string]interface{}{
				"id":       listener.DefaultPool.Id,
				"name":     listener.DefaultPool.Name,
				"protocol": listener.DefaultPool.Protocol,
			},
		}
		certificate := []interface{}{
			map[string]interface{}{
				"id":         listener.Certificate.Id,
				"name":       listener.Certificate.Name,
				"expired_at": listener.Certificate.ExpiredAt,
				"created_at": listener.Certificate.CreatedAt,
			},
		}
		var sniCertificates []interface{}
		for _, v := range listener.SniCertificates {
			sniCertificates = append(sniCertificates, map[string]interface{}{
				"id":         v.Id,
				"name":       v.Name,
				"expired_at": v.ExpiredAt,
				"created_at": v.CreatedAt,
			})
		}
		var tags []interface{}
		for _, v := range listener.Tags {
			tags = append(tags, v)
		}
		var allowedCidrs []interface{}
		for _, v := range listener.AllowedCidrs {
			allowedCidrs = append(allowedCidrs, v)
		}
		var alpnProtocols []interface{}
		for _, v := range listener.AlpnProtocols {
			alpnProtocols = append(alpnProtocols, v)
		}
		formattedData = append(formattedData, map[string]interface{}{
			"id":                      listener.Id,
			"name":                    listener.Name,
			"description":             listener.Description,
			"provisioning_status":     listener.ProvisioningStatus,
			"protocol":                listener.Protocol,
			"port":                    listener.Port,
			"insert_headers":          insert_headers,
			"default_pool":            default_pool,
			"certificate":             certificate,
			"sni_certificates":        sniCertificates,
			"hsts_max_age":            listener.HstsMaxAge,
			"hsts_include_subdomains": listener.HstsIncludeSubdomains,
			"hsts_preload":            listener.HstsPreload,
			"connection_limit":        listener.ConnectionLimit,
			"client_data_timeout":     listener.ClientDataTimeout,
			"member_connect_timeout":  listener.MemberConnectTimeout,
			"member_data_timeout":     listener.MemberDataTimeout,
			"tcp_inspect_timeout":     listener.TcpInspectTimeout,
			"alpn_protocols":          alpnProtocols,
			"created_at":              listener.CreatedAt,
			"allowed_cidrs":           allowedCidrs,
			"tags":                    tags,
		})
	}
	if err := d.Set("listeners", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener list: %v", err))
	}
	d.SetId(loadBalancerId)

	return nil
}

func getListener(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Get("listener_id").(string)
	response, err := service.GetListener(vpcId, listenerId)
	if err != nil {
		return diag.FromErr(err)
	}
	listener := response.Listener
	insertHeaders := []interface{}{
		map[string]interface{}{
			"x_forwarded_for":   listener.InsertHeaders.XForwardedFor,
			"x_forwarded_port":  listener.InsertHeaders.XForwardedPort,
			"x_forwarded_proto": listener.InsertHeaders.XForwardedProto,
		},
	}
	defaultPool := []interface{}{
		map[string]interface{}{
			"id":       listener.DefaultPool.Id,
			"name":     listener.DefaultPool.Name,
			"protocol": listener.DefaultPool.Protocol,
		},
	}
	certificate := []interface{}{
		map[string]interface{}{
			"id":         listener.Certificate.Id,
			"name":       listener.Certificate.Name,
			"expired_at": listener.Certificate.ExpiredAt,
			"created_at": listener.Certificate.CreatedAt,
		},
	}
	var sniCertificates []interface{}
	for _, v := range listener.SniCertificates {
		sniCertificates = append(sniCertificates, map[string]interface{}{
			"id":         v.Id,
			"name":       v.Name,
			"expired_at": v.ExpiredAt,
			"created_at": v.CreatedAt,
		})
	}
	if err := d.Set("listener_id", listener.Id); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener id: %v", err))
	}
	if err := d.Set("name", listener.Name); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener name: %v", err))
	}
	if err := d.Set("description", listener.Description); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener description: %v", err))
	}
	if err := d.Set("provisioning_status", listener.ProvisioningStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener provisioning status: %v", err))
	}
	if err := d.Set("protocol", listener.Protocol); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener protocol: %v", err))
	}
	if err := d.Set("port", listener.Port); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener port: %v", err))
	}
	if err := d.Set("load_balancer_id", listener.LoadBalancerId); err != nil {
		return diag.FromErr(fmt.Errorf("error setting load balancer id of listener: %v", err))
	}
	if err := d.Set("insert_headers", insertHeaders); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener insert headers: %v", err))
	}
	if err := d.Set("default_pool", defaultPool); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener default pool: %v", err))
	}
	if err := d.Set("certificate", certificate); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener certificate: %v", err))
	}
	if err := d.Set("sni_certificates", sniCertificates); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener sni certificates: %v", err))
	}
	if err := d.Set("hsts_max_age", listener.HstsMaxAge); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener hsts max age: %v", err))
	}
	if err := d.Set("hsts_include_subdomains", listener.HstsIncludeSubdomains); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener hsts include subdomain: %v", err))
	}
	if err := d.Set("hsts_preload", listener.HstsPreload); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener hsts preload: %v", err))
	}
	if err := d.Set("connection_limit", listener.ConnectionLimit); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener connection limit %v", err))
	}
	if err := d.Set("client_data_timeout", listener.ClientDataTimeout); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener client data timeout: %v", err))
	}
	if err := d.Set("member_connect_timeout", listener.MemberConnectTimeout); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener member connection timeout: %v", err))
	}
	if err := d.Set("member_data_timeout", listener.MemberDataTimeout); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener member data timeout: %v", err))
	}
	if err := d.Set("tcp_inspect_timeout", listener.TcpInspectTimeout); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener tcp inspect timeout: %v", err))
	}
	if err := d.Set("alpn_protocols", listener.AlpnProtocols); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener alpn protocols: %v", err))
	}
	if err := d.Set("allowed_cidrs", listener.AllowedCidrs); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener allowed cidrs: %v", err))
	}
	if err := d.Set("created_at", listener.CreatedAt); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener create date: %v", err))
	}
	if err := d.Set("tags", listener.Tags); err != nil {
		return diag.FromErr(fmt.Errorf("error setting listener tags: %v", err))
	}
	d.SetId(listenerId)

	return nil
}
