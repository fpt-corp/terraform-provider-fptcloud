package fptcloud_user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	common "terraform-provider-fptcloud/commons"
)

type Service interface {
	ListUsers(ctx context.Context, orgId string) ([]User, error)
}

type serviceImpl struct {
	client *common.Client
}

func NewService(client *common.Client) Service {
	return &serviceImpl{client: client}
}

func (s *serviceImpl) ListUsers(ctx context.Context, orgId string) ([]User, error) {
	reqURL := common.ApiPath.UsersByOrg(orgId)
	fmt.Println("Debug: Get Users URL:", reqURL)
	resp, err := s.client.SendGetRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var listResponse ListUsersResponse
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&listResponse); err != nil {
		return nil, err
	}

	if !listResponse.Status {
		return nil, errors.New(listResponse.Message)
	}

	return listResponse.Data.Data, nil
}
