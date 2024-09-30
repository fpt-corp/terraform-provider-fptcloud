package fptcloud_flavor

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
)

type FlavorInfo struct {
	Vcpu        int `json:"vcpu"`
	MemoryMb    int `json:"memory_mb"`
	GpuMemoryGb int `json:"gpu_memory_gb"`
}

// Flavor represents a flavor model
type Flavor struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Cpu         int    `json:"cpu"`
	MemoryMb    int    `json:"memory_mb"`
	GpuMemoryGb *int   `json:"gpu_memory_gb"`
	Type        string `json:"type"`
}

type FlavorResponse struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	Info FlavorInfo `json:"info"`
	Type string     `json:"type"`
}

// FlavorService defines the interface for flavor service
type FlavorService interface {
	ListFlavor(vpcId string) (*[]Flavor, error)
}

// FlavorServiceImpl is the implementation of FlavorService
type FlavorServiceImpl struct {
	client *common.Client
}

// NewFlavorService creates a new instance of flavor service with the given client
func NewFlavorService(client *common.Client) FlavorService {
	return &FlavorServiceImpl{client: client}
}

// ListFlavor get list flavor
func (s *FlavorServiceImpl) ListFlavor(vpcId string) (*[]Flavor, error) {
	var apiPath = common.ApiPath.Flavor(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var responseModel struct {
		Data []FlavorResponse `json:"data"`
	}
	err = json.Unmarshal(resp, &responseModel)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	flavors := make([]Flavor, len(responseModel.Data))

	for i, flavor := range responseModel.Data {
		flavorCopy := flavor
		flavors[i] = Flavor{
			ID:          flavorCopy.ID,
			Name:        flavorCopy.Name,
			Cpu:         flavorCopy.Info.Vcpu,
			MemoryMb:    flavorCopy.Info.MemoryMb,
			GpuMemoryGb: &flavorCopy.Info.GpuMemoryGb,
			Type:        flavorCopy.Type,
		}
	}
	return &flavors, nil
}
