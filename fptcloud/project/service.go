package fptcloud_project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	common "terraform-provider-fptcloud/commons"
)

type Service interface {
	GetTenant(ctx context.Context) (*Tenant, error)
	FindProject(ctx context.Context, tenantId string, search FindProjectParam) (*Project, error)
}

type Tenant struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type GetTenantResponse struct {
	Response
	Data *Tenant `json:"data,omitempty"`
}

type serviceImpl struct {
	client *common.Client
}

func NewService(client *common.Client) Service {
	return &serviceImpl{
		client: client,
	}
}

func (s *serviceImpl) GetTenant(ctx context.Context) (*Tenant, error) {
	reqURL := common.ApiPath.Tenant(s.client.TenantName)
	resp, err := s.client.SendGetRequest(reqURL)
	if err != nil {
		return nil, err
	}
	response := GetTenantResponse{}
	err = json.NewDecoder(bytes.NewReader(resp)).Decode(&response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}
	return response.Data, nil
}

func (s *serviceImpl) FindProject(ctx context.Context, tenantId string, search FindProjectParam) (*Project, error) {
	// List all projects and filter by name
	reqURL := common.ApiPath.Projects(tenantId)
	fmt.Println("Debug : reqURL ", reqURL)
	resp, err := s.client.SendGetRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var listResponse struct {
		Status  bool      `json:"status"`
		Message string    `json:"message"`
		Data    []Project `json:"data"`
	}

	err = json.NewDecoder(bytes.NewReader(resp)).Decode(&listResponse)
	if err != nil {
		return nil, err
	}

	if !listResponse.Status {
		return nil, errors.New(listResponse.Message)
	}

	// Find project by name
	for _, project := range listResponse.Data {
		if project.Name == search.Name {
			return &project, nil
		}
	}

	return nil, fmt.Errorf("project with name '%s' not found", search.Name)
}
