package fptcloud_tagging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	common "terraform-provider-fptcloud/commons"
)

// TaggingService defines the interface for tagging service
type TaggingService interface {
	Get(ctx context.Context, tagId string) (*Tag, error)
	List(ctx context.Context, key string, value string) (*ListTag, error)
	ListProjectVpc(ctx context.Context) (*ListTagProjectVpcResponse, error)
	Create(ctx context.Context, input *CreateTagInput) (*TagResponse, error)
	Update(ctx context.Context, tagId string, input *UpdateTagInput) (*TagResponse, error)
	Delete(ctx context.Context, tagId string) (*common.SimpleResponse, error)
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
func (s *TaggingServiceImpl) Get(ctx context.Context, tagId string) (*Tag, error) {
	tenant, err := s.dependency.GetVPCService().GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	apiPath := common.ApiPath.GetTag(tenant.Id, tagId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var response TagGetResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	if !response.Status {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
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
func (s *TaggingServiceImpl) List(ctx context.Context, key string, value string) (*ListTag, error) {
	tenant, err := s.dependency.GetVPCService().GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	if key != "" {
		params.Set("key", key)
	}
	if value != "" {
		params.Set("value", value)
	}

	apiPath := common.ApiPath.ListTags(tenant.Id)
	if encoded := params.Encode(); encoded != "" {
		apiPath = fmt.Sprintf("%s?%s", apiPath, encoded)
	}

	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var response TagListResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	if !response.Status {
		return nil, errors.New(response.Message)
	}
	if response.Data == nil || len(response.Data.Data) == 0 {
		return nil, errors.New("Tagging list not found")
	}

	return response.Data, nil
}

// ListTagProjectVpcResponse represents the API response for list-project-vpc
type ListTagProjectVpcResponse struct {
	Status  bool                    `json:"status"`
	Message string                  `json:"message"`
	Data    *ListTagProjectVpcData  `json:"data"`
}

// ListTagProjectVpcData contains projects and vpcs
type ListTagProjectVpcData struct {
	Projects []TagProjectVpcProject `json:"projects"`
	Vpcs     []TagProjectVpcVpc     `json:"vpcs"`
}

// TagProjectVpcProject represents a project item
type TagProjectVpcProject struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsDefault   bool    `json:"is_default"`
}

// TagProjectVpcVpc represents a VPC item
type TagProjectVpcVpc struct {
	ID           string  `json:"id"`
	VdcUUID      *string `json:"vdc_uuid"`
	TenantID     string  `json:"tenant_id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	EndedAt      *string `json:"ended_at"`
	Data         *string `json:"data"`
	DisplayName  *string `json:"display_name"`
	VdcGroupID   *string `json:"vdc_group_id"`
	InfraVpcID   string  `json:"infra_vpc_id"`
}

// ListProjectVpc retrieves projects and VPCs for tag scope selection
func (s *TaggingServiceImpl) ListProjectVpc(ctx context.Context) (*ListTagProjectVpcResponse, error) {
	tenant, err := s.dependency.GetVPCService().GetTenant(ctx)
	if err != nil {
		return nil, err
	}

	apiPath := common.ApiPath.ListTagProjectVpc(tenant.Id)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var response ListTagProjectVpcResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	if !response.Status && response.Message != "" {
		return nil, errors.New(response.Message)
	}

	return &response, nil
}
