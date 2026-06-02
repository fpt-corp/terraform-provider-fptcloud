package fptcloud_mfke_kubeconfig

import (
	"context"
	"fmt"
	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceMfkeKubeconfig returns the schema.Resource for the MFKE kubeconfig datasource.
func DataSourceMfkeKubeconfig() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves the kubeconfig for a Managed FKE cluster, exposing the cluster endpoint, CA certificate, and authentication token.",
		Read:        dataSourceMfkeKubeconfigRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The VPC ID that the MFKE cluster belongs to.",
			},
			"cluster_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "The cluster ID (name) of the MFKE cluster.",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Kubernetes API server endpoint URL.",
			},
			"certificate_authority_data": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Base64-encoded certificate authority data for the cluster.",
			},
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The bearer token used to authenticate against the Kubernetes API server.",
			},
		},
	}
}

func dataSourceMfkeKubeconfigRead(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*common.Client)
	service := NewMfkeKubeconfigService(apiClient)

	vpcId := d.Get("vpc_id").(string)
	clusterId := d.Get("cluster_id").(string)

	kc, err := service.GetKubeconfig(context.Background(), vpcId, clusterId)
	if err != nil {
		return fmt.Errorf("[ERR] Failed to retrieve MFKE kubeconfig: %s", err)
	}

	d.SetId(clusterId)
	_ = d.Set("endpoint", kc.Endpoint)
	_ = d.Set("certificate_authority_data", kc.CertificateAuthorityData)
	_ = d.Set("token", kc.Token)

	return nil
}
