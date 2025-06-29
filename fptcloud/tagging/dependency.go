package fptcloud_tagging

import (
	common "terraform-provider-fptcloud/commons"
	fptcloud_vpc "terraform-provider-fptcloud/fptcloud/vpc"
)

// DependencyService defines the interface for dependent services
type DependencyService interface {
	GetVPCService() fptcloud_vpc.Service
}

// dependencyServiceImpl implements DependencyService
type dependencyServiceImpl struct {
	vpcService fptcloud_vpc.Service
}

// NewDependencyService creates a new dependency service
func NewDependencyService(client *common.Client) DependencyService {
	return &dependencyServiceImpl{
		vpcService: fptcloud_vpc.NewService(client),
	}
}

// GetVPCService returns the VPC service implementation
func (s *dependencyServiceImpl) GetVPCService() fptcloud_vpc.Service {
	return s.vpcService
}
