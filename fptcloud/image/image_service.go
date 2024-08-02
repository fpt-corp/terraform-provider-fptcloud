package fptcloud_image

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
)

// Image represents a image model
type Image struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Catalog string `json:"catalog"`
	IsGpu   bool   `json:"is_gpu"`
}

// ImageService defines the interface for image service
type ImageService interface {
	ListImage(vpcId string) (*[]Image, error)
}

// ImageServiceImpl is the implementation of ImageService
type ImageServiceImpl struct {
	client *common.Client
}

// NewImageService creates a new instance of image service with the given client
func NewImageService(client *common.Client) ImageService {
	return &ImageServiceImpl{client: client}
}

// ListImage get list image
func (s *ImageServiceImpl) ListImage(vpcId string) (*[]Image, error) {
	var apiPath = common.ApiPath.Image(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, err
	}

	var imageResponse struct {
		Data []Image `json:"data"`
	}
	err = json.Unmarshal(resp, &imageResponse)

	if err != nil {
		return nil, err
	}
	return &imageResponse.Data, nil
}
