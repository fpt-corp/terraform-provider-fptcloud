package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceListener() *schema.Resource {
	return &schema.Resource{
		CreateContext: createListener,
		ReadContext:   readListener,
		UpdateContext: updateListener,
		DeleteContext: deleteListener,
		Schema:        resourceListener,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 4 || parts[0] != "vpc" || parts[2] != "listener" {
					return nil, fmt.Errorf("invalid import id format, expected vpc/<vpc_id>/listener/<listener_id>")
				}
				vpcId := parts[1]
				listenerId := parts[3]

				if err := d.Set("vpc_id", vpcId); err != nil {
					return nil, fmt.Errorf("error setting vpc id: %s", err)
				}
				d.SetId(listenerId)

				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func readListener(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Id()

	response, err := service.GetListener(vpcId, listenerId)
	if err != nil {
		return diag.FromErr(err)
	}
	listener := response.Listener
	var sniCertificateIds []string
	for _, v := range listener.SniCertificates {
		sniCertificateIds = append(sniCertificateIds, v.Id)
	}
	insertHeaders := map[string]interface{}{
		"x_forwarded_for":   listener.InsertHeaders.XForwardedFor,
		"x_forwarded_port":  listener.InsertHeaders.XForwardedPort,
		"x_forwarded_proto": listener.InsertHeaders.XForwardedProto,
	}
	if err := d.Set("name", listener.Name); err != nil {
		return diag.FromErr(fmt.Errorf("error setting name: %v", err))
	}
	if err := d.Set("description", listener.Description); err != nil {
		return diag.FromErr(fmt.Errorf("error setting description: %v", err))
	}
	if err := d.Set("protocol", listener.Protocol); err != nil {
		return diag.FromErr(fmt.Errorf("error setting protocol: %v", err))
	}
	if err := d.Set("protocol_port", listener.Port); err != nil {
		return diag.FromErr(fmt.Errorf("error setting protocol port: %v", err))
	}
	if err := d.Set("provisioning_status", listener.ProvisioningStatus); err != nil {
		return diag.FromErr(fmt.Errorf("error setting provisioning status: %v", err))
	}
	if err := d.Set("insert_headers", []interface{}{insertHeaders}); err != nil {
		return diag.FromErr(fmt.Errorf("error setting insert headers: %v", err))
	}
	if err := d.Set("default_pool_id", listener.DefaultPool.Id); err != nil {
		return diag.FromErr(fmt.Errorf("error setting default pool id: %v", err))
	}
	if err := d.Set("certificate_id", listener.Certificate.Id); err != nil {
		return diag.FromErr(fmt.Errorf("error setting certificate id: %v", err))
	}
	if err := d.Set("sni_certificate_ids", sniCertificateIds); err != nil {
		return diag.FromErr(fmt.Errorf("error setting sni certificate ids: %v", err))
	}
	if err := d.Set("hsts_max_age", listener.HstsMaxAge); err != nil {
		return diag.FromErr(fmt.Errorf("error setting hsts max age: %v", err))
	}
	if err := d.Set("hsts_include_subdomains", listener.HstsIncludeSubdomains); err != nil {
		return diag.FromErr(fmt.Errorf("error setting hsts include subdomains: %v", err))
	}
	if err := d.Set("hsts_preload", listener.HstsPreload); err != nil {
		return diag.FromErr(fmt.Errorf("error setting hsts preload: %v", err))
	}
	if err := d.Set("connection_limit", listener.ConnectionLimit); err != nil {
		return diag.FromErr(fmt.Errorf("error setting connection limit: %v", err))
	}
	if err := d.Set("client_data_timeout", listener.ClientDataTimeout); err != nil {
		return diag.FromErr(fmt.Errorf("error setting client data timeout: %v", err))
	}
	if err := d.Set("member_connection_timeout", listener.MemberConnectTimeout); err != nil {
		return diag.FromErr(fmt.Errorf("error setting member connection timeout: %v", err))
	}
	if err := d.Set("member_data_timeout", listener.MemberDataTimeout); err != nil {
		return diag.FromErr(fmt.Errorf("error setting member data timeout: %v", err))
	}
	if err := d.Set("tcp_inspect_timeout", listener.TcpInspectTimeout); err != nil {
		return diag.FromErr(fmt.Errorf("error setting tcp inspect timeout: %v", err))
	}
	if err := d.Set("alpn_protocols", listener.AlpnProtocols); err != nil {
		return diag.FromErr(fmt.Errorf("error setting alpn protocols: %v", err))
	}
	if err := d.Set("allowed_cidrs", listener.AllowedCidrs); err != nil {
		return diag.FromErr(fmt.Errorf("error setting allowed cidrs: %v", err))
	}
	return nil
}

func createListener(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	loadBalancerId := d.Get("load_balancer_id").(string)

	var payload ListenerCreateModel

	headers := d.Get("insert_headers").([]interface{})[0].(map[string]interface{})
	insertHeaders := map[string]bool{
		"X-Forwarded-For":   headers["x_forwarded_for"].(bool),
		"X-Forwarded-Port":  headers["x_forwarded_port"].(bool),
		"X-Forwarded-Proto": headers["x_forwarded_proto"].(bool),
	}

	allowedCidrs := []string{}
	for _, v := range d.Get("allowed_cidrs").([]interface{}) {
		allowedCidrs = append(allowedCidrs, v.(string))
	}

	protocol := d.Get("protocol").(string)
	alpnProtocols := []string{}
	value, ok := d.GetOk("alpn_protocols")
	if ok {
		if protocol != "TERMINATED_HTTPS" {
			return diag.Errorf("alpn_protocols must be null or empty when protocol is not TERMINATED_HTTPS")
		}
		alpnRaw := value.([]interface{})
		for _, v := range alpnRaw {
			alpnProtocols = append(alpnProtocols, v.(string))
		}
		payload.AlpnProtocols = alpnProtocols
	}

	sniCertificateIds := []string{}
	for _, v := range d.Get("sni_certificate_ids").([]interface{}) {
		sniCertificateIds = append(sniCertificateIds, v.(string))
	}

	payload.Name = d.Get("name").(string)
	payload.Description = d.Get("description").(string)
	payload.Protocol = protocol
	payload.ProtocolPort = d.Get("protocol_port").(string)
	payload.DefaultPoolId = d.Get("default_pool_id").(string)
	payload.CertificateId = d.Get("certificate_id").(string)
	payload.SniCertificateIds = sniCertificateIds
	payload.ConnectionLimit = d.Get("connection_limit").(int)
	payload.ClientDataTimeout = d.Get("client_data_timeout").(int)
	payload.MemberConnectTimeout = d.Get("member_connect_timeout").(int)
	payload.MemberDataTimeout = d.Get("member_data_timeout").(int)
	payload.TcpInspectTimeout = d.Get("tcp_inspect_timeout").(int)
	payload.InsertHeaders = insertHeaders
	payload.HstsMaxAge = d.Get("hsts_max_age").(int)
	payload.HstsIncludeSubdomains = d.Get("hsts_include_subdomains").(bool)
	payload.HstsPreload = d.Get("hsts_preload").(bool)
	payload.AllowedCidrs = allowedCidrs

	response, err := service.CreateListener(vpcId, loadBalancerId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(response.Data.Id)
	return nil
}

func updateListener(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Id()

	var payload ListenerUpdateModel

	allowedCidrs := []string{}
	for _, v := range d.Get("allowed_cidrs").([]interface{}) {
		allowedCidrs = append(allowedCidrs, v.(string))
	}

	protocol := d.Get("protocol").(string)
	alpnProtocols := []string{}
	value, ok := d.GetOk("alpn_protocols")
	if ok {
		if protocol != "TERMINATED_HTTPS" {
			return diag.Errorf("alpn_protocols must be null or empty when protocol is not TERMINATED_HTTPS")
		}
		alpnRaw := value.([]interface{})
		for _, v := range alpnRaw {
			alpnProtocols = append(alpnProtocols, v.(string))
		}
		payload.AlpnProtocols = alpnProtocols
	}

	sniCertificateIds := []string{}
	for _, v := range d.Get("sni_certificate_ids").([]interface{}) {
		sniCertificateIds = append(sniCertificateIds, v.(string))
	}
	headers := d.Get("insert_headers").([]interface{})[0].(map[string]interface{})
	insertHeaders := map[string]bool{
		"X-Forwarded-For":   headers["x_forwarded_for"].(bool),
		"X-Forwarded-Port":  headers["x_forwarded_port"].(bool),
		"X-Forwarded-Proto": headers["x_forwarded_proto"].(bool),
	}
	payload.Name = d.Get("name").(string)
	payload.Description = d.Get("description").(string)
	payload.DefaultPoolId = d.Get("default_pool_id").(string)
	payload.CertificateId = d.Get("certificate_id").(string)
	payload.SniCertificateIds = sniCertificateIds
	payload.ConnectionLimit = d.Get("connection_limit").(int)
	payload.ClientDataTimeout = d.Get("client_data_timeout").(int)
	payload.MemberConnectTimeout = d.Get("member_connect_timeout").(int)
	payload.MemberDataTimeout = d.Get("member_data_timeout").(int)
	payload.TcpInspectTimeout = d.Get("tcp_inspect_timeout").(int)
	payload.InsertHeaders = insertHeaders
	payload.HstsMaxAge = d.Get("hsts_max_age").(int)
	payload.HstsIncludeSubdomains = d.Get("hsts_include_subdomains").(bool)
	payload.HstsPreload = d.Get("hsts_preload").(bool)
	payload.AllowedCidrs = allowedCidrs

	_, err := service.UpdateListener(vpcId, listenerId, payload)
	if err != nil {
		return diag.FromErr(err)
	}

	return readListener(ctx, d, m)
}

func deleteListener(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	listenerId := d.Id()

	_, err := service.DeleteListener(vpcId, listenerId)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
