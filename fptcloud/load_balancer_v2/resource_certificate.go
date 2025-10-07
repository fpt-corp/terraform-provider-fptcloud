package fptcloud_load_balancer_v2

import (
	"context"
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceCertificate() *schema.Resource {
	return &schema.Resource{
		CreateContext: createCertificate,
		ReadContext:   readCertificate,
		UpdateContext: nil,
		DeleteContext: deleteCertificate,
		Schema:        resourceCertificate,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 4 || parts[0] != "vpc" || parts[2] != "certificate" {
					return nil, fmt.Errorf("invalid import id format, expected vpc/<vpc_id>/certificate/<certificate_id>")
				}
				vpcId := parts[1]
				certificateId := parts[3]

				if err := d.Set("vpc_id", vpcId); err != nil {
					return nil, fmt.Errorf("error setting vpc id: %s", err)
				}
				d.SetId(certificateId)

				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func readCertificate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	certificateId := d.Id()
	response, err := service.GetCertificate(vpcId, certificateId)
	if err != nil {
		return diag.FromErr(err)
	}
	certificate := response.Certificate
	if err := d.Set("name", certificate.Name); err != nil {
		return diag.FromErr(fmt.Errorf("error setting certificate name: %v", err))
	}
	return nil
}

func createCertificate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)

	var payload CertificateCreateModel
	payload.Name = d.Get("name").(string)
	payload.Certificate = d.Get("certificate").(string)
	payload.PrivateKey = d.Get("private_key").(string)
	payload.CertChain = d.Get("cert_chain").(string)

	response, err := service.CreateCertificate(vpcId, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(response.Data.Id)
	return nil
}

func deleteCertificate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client)
	service := NewLoadBalancerV2Service(client)
	vpcId := d.Get("vpc_id").(string)
	certificateId := d.Id()

	_, err := service.DeleteCertificate(vpcId, certificateId)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
