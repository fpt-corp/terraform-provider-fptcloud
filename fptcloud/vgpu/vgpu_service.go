package fptcloud_vgpu

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
)

// VGpu represents a vGPU model
type VGpu struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	DisplayName   string  `json:"display_name"`
	CreatedAt     string  `json:"created_at"`
	Memory        int     `json:"memory"`
	Status        string  `json:"status"`
	IsDedicated   bool    `json:"is_dedicated"`
	ServiceTypeID *string `json:"service_type_id"`
	Platform      string  `json:"platform"`
	ParentID      string  `json:"parent_id"`
	EnableNvme    bool    `json:"enable_nvme"`
}

// VGpuService defines the interface for vGPU service
type VGpuService interface {
	ListVGpu(vpcId string) (*[]VGpu, error)
}

// VGpuServiceImpl is the implementation of VGpuService
type VGpuServiceImpl struct {
	client *common.Client
}

// NewVGpuService creates a new instance of vGPU service with the given client
func NewVGpuService(client *common.Client) VGpuService {
	return &VGpuServiceImpl{client: client}
}

// ListVGpu get list vGPU
func (s *VGpuServiceImpl) ListVGpu(vpcId string) (*[]VGpu, error) {
	var apiPath = common.ApiPath.GetGPUInfo(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var responseModel struct {
		Data []VGpu `json:"data"`
	}
	err = json.Unmarshal(resp, &responseModel)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &responseModel.Data, nil
}
