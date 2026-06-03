package fptcloud_mfke_kubeconfig

import (
	"context"
	"fmt"
	"strings"
	"time"

	common "terraform-provider-fptcloud/commons"
	fptcloud_dfke "terraform-provider-fptcloud/fptcloud/dfke"
	fptcloud_mfke "terraform-provider-fptcloud/fptcloud/mfke"

	"gopkg.in/yaml.v3"
)

// MfkeKubeconfig holds the parsed fields returned to Terraform
type MfkeKubeconfig struct {
	Endpoint                 string
	CertificateAuthorityData string
	Token                    string
}

// MfkeKubeconfigService defines the interface
type MfkeKubeconfigService interface {
	GetKubeconfig(ctx context.Context, vpcId, clusterId string) (*MfkeKubeconfig, error)
}

// MfkeKubeconfigServiceImpl is the implementation
type MfkeKubeconfigServiceImpl struct {
	mfkeClient    *fptcloud_mfke.MfkeApiClient
	tenancyClient *fptcloud_dfke.TenancyApiClient
}

// NewMfkeKubeconfigService creates a new instance
func NewMfkeKubeconfigService(client *common.Client) MfkeKubeconfigService {
	return &MfkeKubeconfigServiceImpl{
		mfkeClient:    fptcloud_mfke.NewMfkeApiClient(client),
		tenancyClient: fptcloud_dfke.NewTenancyApiClient(client),
	}
}

const (
	kubeconfigPollInterval = 60 * time.Second
	kubeconfigPollTimeout  = 15 * time.Minute
)

// GetKubeconfig fetches and parses the kubeconfig YAML for a cluster, retrying until available or timeout.
func (s *MfkeKubeconfigServiceImpl) GetKubeconfig(ctx context.Context, vpcId, clusterId string) (*MfkeKubeconfig, error) {
	platform, err := s.tenancyClient.GetVpcPlatform(ctx, vpcId)
	if err != nil {
		return nil, fmt.Errorf("failed to detect platform: %s", err)
	}
	platform = strings.ToLower(platform)

	apiPath := common.ApiPath.ManagedFKEKubeconfig(vpcId, platform, clusterId)
	deadline := time.Now().Add(kubeconfigPollTimeout)

	for {
		resp, err := s.mfkeClient.SendGetWithInfraType(apiPath, platform)
		if err != nil {
			if !isKubeconfigNotReady(err) {
				return nil, err
			}
			if time.Now().After(deadline) {
				return nil, fmt.Errorf(
					"kubeconfig for cluster %q is not yet available. "+
						"The cluster may still be provisioning — please try again later",
					clusterId,
				)
			}
			time.Sleep(kubeconfigPollInterval)
			continue
		}

		var raw struct {
			Clusters []struct {
				Cluster struct {
					CertificateAuthorityData string `yaml:"certificate-authority-data"`
					Server                   string `yaml:"server"`
				} `yaml:"cluster"`
			} `yaml:"clusters"`
			Users []struct {
				User struct {
					Token string `yaml:"token"`
				} `yaml:"user"`
			} `yaml:"users"`
		}

		if err := yaml.Unmarshal(resp, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse kubeconfig: %s", err)
		}

		if len(raw.Clusters) == 0 {
			return nil, fmt.Errorf("kubeconfig contains no clusters")
		}
		if len(raw.Users) == 0 {
			return nil, fmt.Errorf("kubeconfig contains no users")
		}

		return &MfkeKubeconfig{
			Endpoint:                 raw.Clusters[0].Cluster.Server,
			CertificateAuthorityData: raw.Clusters[0].Cluster.CertificateAuthorityData,
			Token:                    raw.Users[0].User.Token,
		}, nil
	}
}

// isKubeconfigNotReady returns true when the API signals the kubeconfig does not exist yet (cluster still provisioning).
func isKubeconfigNotReady(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "not found")
}
