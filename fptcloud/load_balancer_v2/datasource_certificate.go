package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceCertificates() *schema.Resource {
	return &schema.Resource{
		ReadContext: listCertificates,
		Schema:      dataSourceCertificates,
	}
}

func DataSourceCertificate() *schema.Resource {
	return &schema.Resource{
		ReadContext: getCertificate,
		Schema:      dataSourceCertificate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func listCertificates(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	response, err := service.ListCertificates(vpcId, 1, 1000)
	if err != nil {
		return diag.FromErr(err)
	}
	certificates := response.Certificates
	var formattedData []interface{}
	for _, certificate := range certificates {
		formattedData = append(formattedData, map[string]interface{}{
			"certificate_id": certificate.Id,
			"name":           certificate.Name,
			"created_at":     certificate.CreatedAt,
			"expired_at":     certificate.ExpiredAt,
		})
	}
	if err := d.Set("certificates", formattedData); err != nil {
		return diag.FromErr(fmt.Errorf("error setting certificate list: %v", err))
	}
	d.SetId(vpcId)

	return nil
}

func getCertificate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	certificateId := d.Get("certificate_id").(string)
	response, err := service.GetCertificate(vpcId, certificateId)
	if err != nil {
		return diag.FromErr(err)
	}
	certificate := response.Certificate
	if err := d.Set("certificate_id", certificate.Id); err != nil {
		diag.FromErr(fmt.Errorf("error setting certificate id: %v", err))
	}
	if err := d.Set("name", certificate.Name); err != nil {
		diag.FromErr(fmt.Errorf("error setting certificate name: %v", err))
	}
	if err := d.Set("created_at", certificate.CreatedAt); err != nil {
		diag.FromErr(fmt.Errorf("error setting certificate create date: %v", err))
	}
	if err := d.Set("expired_at", certificate.ExpiredAt); err != nil {
		diag.FromErr(fmt.Errorf("error setting certificate expire date: %v", err))
	}
	d.SetId(certificateId)
	return nil
}
