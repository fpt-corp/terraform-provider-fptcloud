package fptcloud_tagging

import (
	"context"
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"
)

// TaggingService defines the interface for tagging service
type TaggingService interface {
	Create(ctx context.Context, input *CreateTagInput) (*TagResponse, error)
	Get(ctx context.Context, tagId string) (*TagDetail, error)
	Update(ctx context.Context, tagId string, input *UpdateTagInput) (*TagResponse, error)
	Delete(ctx context.Context, tagId string) (*common.SimpleResponse, error)
	List(ctx context.Context, name string) (*TagListResponse, error)
}

// TaggingServiceImpl is the implementation of TaggingService
type TaggingServiceImpl struct {
	client     *common.Client
	dependency DependencyService
}

// NewTaggingService creates a new tagging service with the given client
func NewTaggingService(client *common.Client) TaggingService {
	return &TaggingServiceImpl{
		client:     client,
		dependency: NewDependencyService(client),
	}
}

// Create creates a new tag
func (s *TaggingServiceImpl) Create(ctx context.Context, input *CreateTagInput) (*TagResponse, error) {
	tenant, err := s.dependency.GetVPCService().GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	apiPath := common.ApiPath.CreateTag(tenant.Id)
	resp, err := s.client.SendPostRequest(apiPath, input)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var response TagResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &response, nil
}

// Get retrieves tag details
func (s *TaggingServiceImpl) Get(ctx context.Context, tagId string) (*TagDetail, error) {
	tenant, err := s.dependency.GetVPCService().GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	apiPath := common.ApiPath.GetTag(tenant.Id, tagId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var tagDetail TagDetail
	err = json.Unmarshal(resp, &tagDetail)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &tagDetail, nil
}

// Update updates an existing tag
func (s *TaggingServiceImpl) Update(ctx context.Context, tagId string, input *UpdateTagInput) (*TagResponse, error) {
	tenant, err := s.dependency.GetVPCService().GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	apiPath := common.ApiPath.UpdateTag(tenant.Id, tagId)
	resp, err := s.client.SendPutRequest(apiPath, input)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var response TagResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &response, nil
}

// Delete deletes a tag
func (s *TaggingServiceImpl) Delete(ctx context.Context, tagId string) (*common.SimpleResponse, error) {
	tenant, err := s.dependency.GetVPCService().GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	apiPath := common.ApiPath.DeleteTag(tenant.Id, tagId)
	_, err = s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data: "Successfully",
	}

	return result, nil
}

// List retrieves all tags with optional name filter
func (s *TaggingServiceImpl) List(ctx context.Context, name string) (*TagListResponse, error) {
	tenant, err := s.dependency.GetVPCService().GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	queryParams := ""
	if name != "" {
		queryParams = fmt.Sprintf("?name=%s", name)
	}

	apiPath := common.ApiPath.ListTags(tenant.Id) + queryParams
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var tagList TagListResponse
	err = json.Unmarshal(resp, &tagList)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &tagList, nil
}
