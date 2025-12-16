package fptcloud_database_flavors

import (
	"encoding/json"
	"fmt"
	"net/http"
	common "terraform-provider-fptcloud/commons"
)

type DatabaseFlavor struct {
	ID         string  `json:"flavor_id"`
	Name       string  `json:"flavor_name"`
	Cpu        int     `json:"flavor_vcpu"`
	MemoryMb   int     `json:"flavor_ram"`
	IsScale    *int    `json:"is_scale"`
	FlavorSite string  `json:"flavor_site"`
}

type DatabaseFlavorResponse struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Data    []DatabaseFlavor `json:"data"`
}

type DatabaseFlavorService interface {
	ListDatabaseFlavor(vpcId string, isOps string) (*[]DatabaseFlavor, error)
}

type DatabaseFlavorServiceImpl struct {
	client *common.Client
}

func NewDatabaseFlavorService(client *common.Client) DatabaseFlavorService {
	if client == nil {
		panic("client cannot be nil")
	}
	
	return &DatabaseFlavorServiceImpl{
		client: client,
	}
}

func (s *DatabaseFlavorServiceImpl) ListDatabaseFlavor(vpcId string, isOps string) (*[]DatabaseFlavor, error) {
	if s == nil {
		return nil, fmt.Errorf("database flavor service is nil")
	}
	if s.client == nil {
		return nil, fmt.Errorf("database flavor service client is nil")
	}
	
	if vpcId == "" {
		return nil, fmt.Errorf("vpc_id cannot be empty")
	}
	if isOps == "" {
		return nil, fmt.Errorf("is_ops cannot be empty")
	}
	
	fmt.Printf("[DEBUG] Calling ListDatabaseFlavor with vpc_id: %s, is_ops: %s\n", vpcId, isOps)

	apiPath := common.ApiPath.DatabaseFlavor(vpcId, isOps)
	fmt.Printf("[DEBUG] API Path: %s\n", apiPath)
	
	u := s.client.PrepareClientURL(apiPath)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	
	switch s.client.Region {
	case "VN/HAN":
		req.Header.Set("fpt-region", "hanoi-vn")
	case "VN/SGN":
		req.Header.Set("fpt-region", "saigon-vn")
	case "VN/HAN2":
		req.Header.Set("fpt-region", "hanoi-2-vn")
	case "JP/JCSI2":
		req.Header.Set("fpt-region", "JP/JCSI2")
	default:
		req.Header.Set("fpt-region", s.client.Region)
	}
	
	resp, err := s.client.SendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	
	if resp == nil {
		return nil, fmt.Errorf("empty response from API")
	}
	
	fmt.Printf("[DEBUG] Raw API response: %s\n", string(resp))

	var responseModel DatabaseFlavorResponse
	err = json.Unmarshal(resp, &responseModel)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if responseModel.Code != "200" {
		return nil, fmt.Errorf("[ERR] Failed to retrieve database flavors: %s", responseModel.Message)
	}

	fmt.Printf("[DEBUG] Found %d database flavors\n", len(responseModel.Data))
	
	return &responseModel.Data, nil
}