package fptcloud_vpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

type Service interface {
	GetTenant(ctx context.Context) (*Tenant, error)
	FindVPC(ctx context.Context, tenantId string, search FindVPCParam) (*VPC, error)
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
