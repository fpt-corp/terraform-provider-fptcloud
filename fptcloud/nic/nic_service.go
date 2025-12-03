// Package fptcloud_nic provides a service for managing network interfaces (NICs) in FPT Cloud.
package fptcloud_nic

import (
	"encoding/json"
	common "terraform-provider-fptcloud/commons"
)

type Nic struct {
	ID         string `json:"id"`
	InstanceId string `json:"instance_id"`
	SubnetId   string `json:"subnet_id"`
	MacAddress string `json:"mac_address"`
	PrivateIp  string `json:"private_ip"`
	Status     string `json:"status"`
	SubnetName string `json:"subnet_name"`
	IsPrimary  bool   `json:"is_primary"`
	VpcId      string `json:"vpc_id"`
}

type FindNicResponse struct {
	Data Nic `json:"data"`
}

type CreateNicDto struct {
	InstanceId string `json:"instance_id"`
	SubnetId   string `json:"subnet_id"`
	VpcId      string `json:"vpc_id"`
	IsPrimary  bool   `json:"is_primary"`
}

type UpdateNicDto struct {
	InstanceId string `json:"instance_id"`
	PrivateIp  string `json:"private_ip"`
	IsPrimary  bool   `json:"is_primary"`
}

type DeleteNicDto struct {
	InstanceId string `json:"instance_id"`
	SubnetId   string `json:"subnet_id"`
}

type NicService interface {
	Find(vpcId string, instanceId string, nicId string) (*Nic, error)
	Create(vpcId string, createdModel CreateNicDto) (*Nic, error)
	Delete(vpcId string, nicId string, deleteModel DeleteNicDto) (*common.SimpleResponse, error)
	Update(vpcId string, nicId string, updatedModel UpdateNicDto) (*Nic, error)
}

type NicServiceImpl struct {
	client *common.Client
}

func NewNicService(client *common.Client) NicService {
	return &NicServiceImpl{client: client}
}

func (s *NicServiceImpl) Find(vpcId string, instanceId string, nicId string) (*Nic, error) {
	var apiPath = common.ApiPath.FindNic(vpcId, instanceId, nicId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		if err.(common.HTTPError).Code == 403 {
			return nil, common.HttpError.WrapString("Permission denied: missing instance:ViewNIC")
		}
		return nil, common.DecodeError(err)
	}

	response := FindNicResponse{}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, common.DecodeError(err)
	}
	return &response.Data, nil
}

// Create creates a new NIC
func (s *NicServiceImpl) Create(vpcId string, createdModel CreateNicDto) (*Nic, error) {
	var apiPath = common.ApiPath.CreateNic(vpcId)
	resp, err := s.client.SendPostRequest(apiPath, createdModel)
	if err != nil {
		if err.(common.HTTPError).Code == 403 {
			return nil, common.HttpError.WrapString("Permission denied: missing instance:AddNIC")
		}
		return nil, common.DecodeError(err)
	}

	var createdResponse struct {
		Data Nic `json:"data"`
	}

	err = json.Unmarshal(resp, &createdResponse)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &createdResponse.Data, nil
}

// Delete deletes a NIC
func (s *NicServiceImpl) Delete(vpcId string, nicId string, deleteModel DeleteNicDto) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.Nic(vpcId, nicId)
	_, err := s.client.SendDeleteRequestWithBody(apiPath, deleteModel)
	if err != nil {
		if err.(common.HTTPError).Code == 403 {
			return nil, common.HttpError.WrapString("Permission denied: missing instance:DeleteNIC")
		}
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}

// Update updates a NIC
func (s *NicServiceImpl) Update(vpcId string, nicId string, updatedModel UpdateNicDto) (*Nic, error) {
	var apiPath = common.ApiPath.Nic(vpcId, nicId)
	resp, err := s.client.SendPutRequest(apiPath, updatedModel)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var updatedResponse struct {
		Data Nic `json:"data"`
	}

	err = json.Unmarshal(resp, &updatedResponse)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	return &updatedResponse.Data, nil
}
