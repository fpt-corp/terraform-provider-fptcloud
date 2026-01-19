package fptcloud_vpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

type Service interface {
	GetTenant(ctx context.Context) (*Tenant, error)
	FindVPC(ctx context.Context, tenantId string, search FindVPCParam) (*VPC, error)
	CreateVPC(ctx context.Context, orgId string, createModel CreateVPCDTO) (*CreateVPCData, error)
	GetUserByEmail(ctx context.Context, orgId string, email string) (string, error)
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

func (s *serviceImpl) FindVPC(ctx context.Context, tenantId string, search FindVPCParam) (*VPC, error) {
	reqURL := common.ApiPath.Vpc(tenantId) + utils.ToQueryParams(search)
	resp, err := s.client.SendGetRequest(reqURL)
	if err != nil {
		return nil, err
	}
	response := FindVPCResponse{}
	err = json.NewDecoder(bytes.NewReader(resp)).Decode(&response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}
	return response.Data, nil
}

func (s *serviceImpl) CreateVPC(ctx context.Context, orgId string, createModel CreateVPCDTO) (*CreateVPCData, error) {
	reqURL := common.ApiPath.CreateVpc(orgId)
	fmt.Println("Debug: Create VPC URL:", reqURL)
	get_, _ := json.Marshal(createModel)
	fmt.Println("Debug: Create VPC Payload:", string(get_))
	resp, err := s.client.SendPostRequest(reqURL, createModel)

	fmt.Println("Debug: Create VPC Response:", string(resp))
	fmt.Println("Debug: Create VPC Error:", err)
	if err != nil {
		return nil, common.DecodeError(err)
	}
	response := CreateVPCResponse{}
	err = json.NewDecoder(bytes.NewReader(resp)).Decode(&response)
	if err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, errors.New(response.Message)
	}
	return response.Data, nil
}

type User struct {
	Id    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
}

type ListUsersResponse struct {
	Response
	Data struct {
		Items []User `json:"items,omitempty"`
	} `json:"data,omitempty"`
}

func (s *serviceImpl) GetUserByEmail(ctx context.Context, orgId string, email string) (string, error) {
	reqURL := common.ApiPath.UsersByOrg(orgId)
	fmt.Println("Debug: Get Users URL:", reqURL)
	resp, err := s.client.SendGetRequest(reqURL)
	if err != nil {
		return "", err
	}

	var listResponse ListUsersResponse
	err = json.NewDecoder(bytes.NewReader(resp)).Decode(&listResponse)
	if err != nil {
		return "", err
	}

	if !listResponse.Status {
		return "", errors.New(listResponse.Message)
	}

	// Find user by email
	for _, user := range listResponse.Data.Items {
		if user.Email == email {
			return user.Id, nil
		}
	}

	return "", fmt.Errorf("user with email '%s' not found", email)
}
