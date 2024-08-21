package fptcloud_image_test

import (
	"terraform-provider-fptcloud/fptcloud/image"
	"testing"

	"github.com/stretchr/testify/assert"
	common "terraform-provider-fptcloud/commons"
)

func TestListImage_ReturnsImages(t *testing.T) {
	mockResponse := `{"data": [{"id": "1", "name": "image-name", "catalog": "windows", "is_gpu": true}]}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/images": mockResponse,
	})
	defer server.Close()
	service := fptcloud_image.NewImageService(mockClient)
	images, err := service.ListImage("vpc_id")
	assert.NoError(t, err)
	assert.NotNil(t, images)
	assert.Equal(t, 1, len(*images))
	assert.Equal(t, "1", (*images)[0].ID)
	assert.Equal(t, "image-name", (*images)[0].Name)
	assert.Equal(t, "windows", (*images)[0].Catalog)
	assert.True(t, (*images)[0].IsGpu)
}

func TestListImage_ReturnsErrorOnInvalidJSON(t *testing.T) {
	mockResponse := `invalid json`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/images": mockResponse,
	})
	defer server.Close()
	service := fptcloud_image.NewImageService(mockClient)
	images, err := service.ListImage("vpc_id")
	assert.Error(t, err)
	assert.Nil(t, images)
}

func TestListImage_ReturnsEmptyListOnEmptyResponse(t *testing.T) {
	mockResponse := `{"data": []}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/terraform/vpc/vpc_id/images": mockResponse,
	})
	defer server.Close()
	service := fptcloud_image.NewImageService(mockClient)
	images, err := service.ListImage("vpc_id")
	assert.NoError(t, err)
	assert.NotNil(t, images)
	assert.Equal(t, 0, len(*images))
}
